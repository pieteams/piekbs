#!/usr/bin/env python3
"""
PieKBS vs Naive RAG 评估脚本
使用 RAGAS 框架对比四个指标：
  - Faithfulness（忠实度）
  - Answer Relevancy（答案相关性）
  - Context Precision（上下文精度）
  - Context Recall（上下文召回）

用法：
  cd <project_root>
  python3 eval/eval_piekbs.py
"""
import os, sys, json, re, yaml, requests, glob, random, time, threading
from concurrent.futures import ThreadPoolExecutor

_llm_lock = threading.Semaphore(3)  # 最多同时3个 LLM 请求（两路系统并行时总并发受控）
from pathlib import Path

# ── 读取配置 ──────────────────────────────────────────────────────────────────
KB_ROOT = Path.home() / ".hermes/piekbs-kb"
with open(KB_ROOT / "config.yaml") as f:
    cfg = yaml.safe_load(f)

LLM_BASE_URL = cfg["distill"]["base_url"].rstrip("/")
LLM_TOKEN    = cfg["distill"]["token"]
LLM_MODEL    = cfg["distill"]["model"]
MCP_URL      = f"http://127.0.0.1:{cfg['server']['port']}"
MCP_API_KEY  = cfg["server"].get("api_key", "")

print(f"LLM: {LLM_MODEL} @ {LLM_BASE_URL}")
print(f"MCP: {MCP_URL}")

# ── LLM 调用（Anthropic 兼容，指数退避重试） ─────────────────────────────────
def call_llm(system: str, user: str) -> str:
    payload = {
        "model": LLM_MODEL,
        "max_tokens": 8192,
        "system": system,
        "messages": [{"role": "user", "content": user}],
    }
    headers = {
        "x-api-key": LLM_TOKEN,
        "anthropic-version": "2023-06-01",
        "Content-Type": "application/json",
    }
    for attempt in range(5):
        try:
            with _llm_lock:
                resp = requests.post(
                    f"{LLM_BASE_URL}/v1/messages",
                    headers=headers, json=payload, timeout=120,
                )
            if resp.status_code == 429:
                wait = 2 ** attempt * 5  # 5s, 10s, 20s, 40s, 80s
                print(f"  [429] retry in {wait}s ...", flush=True)
                time.sleep(wait)
                continue
            resp.raise_for_status()
            break
        except requests.exceptions.Timeout:
            wait = 2 ** attempt * 5
            print(f"  [timeout] retry in {wait}s ...", flush=True)
            time.sleep(wait)
    else:
        raise RuntimeError("call_llm: exceeded max retries")
    data = resp.json()
    # 兼容 Anthropic 和 OpenAI 格式
    if "content" in data and isinstance(data["content"], list):
        # thinking 模型先返回 thinking block，再返回 text block
        for block in data["content"]:
            if isinstance(block, dict) and block.get("type") == "text":
                return block["text"].strip()
    if "choices" in data:
        return data["choices"][0]["message"]["content"].strip()
    raise ValueError(f"Unexpected LLM response: {json.dumps(data)[:200]}")

# ── PieKBS kb_context ───────────────────────────────────────────────────────
_mcp_session_id = None

def mcp_call(method: str, params: dict) -> dict:
    """调用 MCP 工具（HTTP streamable transport，维持 session）"""
    global _mcp_session_id
    headers = {"Content-Type": "application/json"}
    if MCP_API_KEY:
        headers["x-api-key"] = MCP_API_KEY

    # initialize session（每次脚本运行只做一次）
    if _mcp_session_id is None:
        r0 = requests.post(f"{MCP_URL}/mcp", headers=headers,
            json={"jsonrpc":"2.0","id":0,"method":"initialize",
                  "params":{"protocolVersion":"2024-11-05","capabilities":{},
                            "clientInfo":{"name":"eval","version":"1"}}},
            timeout=10)
        _mcp_session_id = r0.headers.get("Mcp-Session-Id", "")

    if _mcp_session_id:
        headers["Mcp-Session-Id"] = _mcp_session_id

    resp = requests.post(f"{MCP_URL}/mcp", headers=headers,
        json={"jsonrpc": "2.0", "id": 1, "method": method, "params": params},
        timeout=30)
    return resp.json().get("result", {})

NO_VEC = os.environ.get("WIKILOOP_NO_VEC", "").lower() in ("1", "true", "yes")

def piekbs_context(question: str) -> list[str]:
    """调用 PieKBS MCP kb_context 工具，返回 context 片段列表"""
    try:
        args = {"question": question, "limit": 10}
        if NO_VEC:
            args["no_vec"] = True
        result = mcp_call("tools/call", {
            "name": "kb_context",
            "arguments": args
        })
        content_text = ""
        for item in result.get("content", []):
            if item.get("type") == "text":
                content_text += item["text"]
        contexts = []
        try:
            parsed = json.loads(content_text)
            for page in parsed.get("wiki_pages", []):
                ctx = f"[{page.get('title','')}] {page.get('description','')}"
                if page.get("snippet"):
                    ctx += f"\n{page['snippet']}"
                contexts.append(ctx)
            for src in parsed.get("raw_sources", []) or []:
                ctx = f"[raw] {src.get('title','')} {src.get('description','')}"
                contexts.append(ctx)
        except Exception:
            if content_text:
                contexts = [content_text[:2000]]
        return contexts if contexts else ["(no context)"]
    except Exception as e:
        print(f"  MCP error: {e}")
        return ["(no context)"]

# ── Naive RAG：直接从 raw 文件关键词匹配 ──────────────────────────────────────
def naive_rag_context(question: str, top_k: int = 5) -> list[str]:
    """简单 BM25-like：从 raw/*.md 文件里找包含问题关键词的段落"""
    keywords = [w.lower() for w in re.split(r'\W+', question) if len(w) > 2]
    raw_files = list((KB_ROOT / "raw").rglob("*.md"))
    random.shuffle(raw_files)

    scored = []
    for fpath in raw_files[:200]:  # 限制扫描数量
        try:
            text = fpath.read_text(encoding="utf-8", errors="ignore")
            # 去掉 frontmatter
            text = re.sub(r'^---.*?---\s*', '', text, flags=re.DOTALL)
            score = sum(text.lower().count(kw) for kw in keywords)
            if score > 0:
                # 取前 500 字符作为 context chunk
                scored.append((score, text[:500]))
        except Exception:
            continue

    scored.sort(reverse=True)
    return [chunk for _, chunk in scored[:top_k]] if scored else ["(no context)"]

# ── 自动生成测试问题集 ─────────────────────────────────────────────────────────
def generate_questions(n: int = 10) -> list[dict]:
    """从 wiki source-notes 中随机采样，让 LLM 生成问题+参考答案"""
    note_files = list((KB_ROOT / "wiki" / "source-notes").rglob("*.md"))
    if not note_files:
        print("⚠ 没有 source-notes，使用预设问题")
        return PRESET_QUESTIONS

    samples = random.sample(note_files, min(n * 2, len(note_files)))
    questions = []
    for fpath in samples[:n]:
        try:
            text = fpath.read_text(encoding="utf-8", errors="ignore")[:1500]
            resp = call_llm(
                "你是一个知识库评估助手。根据给定的文档内容，生成一个具体的问题和对应的参考答案。"
                "输出 JSON 格式：{\"question\": \"...\", \"ground_truth\": \"...\"}",
                f"文档内容：\n{text}\n\n请生成一个问题和参考答案（JSON）："
            )
            # 提取 JSON
            m = re.search(r'\{[^{}]+\}', resp, re.DOTALL)
            if m:
                obj = json.loads(m.group())
                if obj.get("question") and obj.get("ground_truth"):
                    questions.append(obj)
                    print(f"  ✓ 生成问题: {obj['question'][:60]}...")
        except Exception as e:
            print(f"  ✗ 生成失败: {e}")
        if len(questions) >= n:
            break

    return questions if questions else PRESET_QUESTIONS

PRESET_QUESTIONS = [
    {"question": "什么是 RAG？", "ground_truth": "RAG 是检索增强生成，结合检索系统和大语言模型提升答案质量。"},
    {"question": "PieKBS 和 RAG 有什么区别？", "ground_truth": "PieKBS 在 RAG 基础上增加了显式的结构化 wiki 知识层，知识可审计可版本化。"},
    {"question": "什么是 RRF 算法？", "ground_truth": "RRF 是倒数排名融合算法，用于合并多个检索系统的排名结果。"},
]

# ── RAGAS 评估（手动实现简化版） ──────────────────────────────────────────────
def score_faithfulness(answer: str, contexts: list[str]) -> float:
    """忠实度：答案中每个声明是否都能从 context 中找到支撑"""
    ctx_text = "\n".join(contexts)
    resp = call_llm(
        "你是一个评估助手。判断答案中的每个声明是否都有 context 支撑。"
        "输出 0.0-1.0 的分数（1.0=完全有支撑，0.0=完全无支撑），只输出数字。",
        f"Context:\n{ctx_text[:4000]}\n\nAnswer:\n{answer}\n\n忠实度分数："
    )
    try:
        return float(re.search(r'[\d.]+', resp).group())
    except Exception:
        return 0.5

def score_answer_relevancy(question: str, answer: str) -> float:
    """答案相关性：答案是否回答了问题"""
    resp = call_llm(
        "你是一个评估助手。判断答案是否充分回答了问题。"
        "输出 0.0-1.0 的分数（1.0=完全回答，0.0=完全没回答），只输出数字。",
        f"Question: {question}\n\nAnswer: {answer}\n\n相关性分数："
    )
    try:
        return float(re.search(r'[\d.]+', resp).group())
    except Exception:
        return 0.5

def score_context_precision(question: str, contexts: list[str]) -> float:
    """上下文精度：一次调用批量评估所有 context 片段"""
    if not contexts:
        return 0.0
    numbered = "\n".join(f"[{i+1}] {ctx[:400]}" for i, ctx in enumerate(contexts))
    resp = call_llm(
        "你是一个评估助手。对于每个编号的 context 片段，判断它对回答问题是否有用。"
        f"输出一个长度为 {len(contexts)} 的 0/1 列表，用逗号分隔，例如：1,0,1,0,1",
        f"Question: {question}\n\nContexts:\n{numbered}\n\n有用性列表（{len(contexts)}个，逗号分隔）："
    )
    try:
        scores = [int(x.strip()) for x in resp.split(',') if x.strip() in ('0', '1')]
        if not scores:
            return 0.5
        return sum(scores) / len(contexts)
    except Exception:
        return 0.5

def score_context_recall(question: str, contexts: list[str], ground_truth: str) -> float:
    """上下文召回：ground truth 中的信息是否被 context 覆盖"""
    ctx_text = "\n".join(contexts)
    resp = call_llm(
        "判断 context 是否包含了足够的信息来回答问题（参考标准答案）。"
        "输出 0.0-1.0 的分数（1.0=完全覆盖，0.0=完全未覆盖），只输出数字。",
        f"Question: {question}\nGround Truth: {ground_truth}\nContext:\n{ctx_text[:4000]}\n\n召回率分数："
    )
    try:
        return float(re.search(r'[\d.]+', resp).group())
    except Exception:
        return 0.5

def generate_answer(question: str, contexts: list[str]) -> str:
    ctx_text = "\n".join(contexts)
    return call_llm(
        "你是一个知识库助手。根据给定的 context 回答问题，不要引入 context 之外的信息。",
        f"Context:\n{ctx_text[:4000]}\n\nQuestion: {question}\n\nAnswer:"
    )

# ── 主流程 ────────────────────────────────────────────────────────────────────
def score_hit_rate(contexts_meta: list[dict], expected_page: str) -> int:
    """Hit Rate：expected_page 是否出现在检索结果中（1=命中，0=未命中）"""
    if not expected_page:
        return 0
    for ctx in contexts_meta:
        path = ctx.get("path", "") or ctx.get("id", "")
        if expected_page in path or path in expected_page:
            return 1
    return 0

def score_mrr(contexts_meta: list[dict], expected_page: str) -> float:
    """MRR：expected_page 首次出现的排名倒数（1/rank），未命中为 0"""
    if not expected_page:
        return 0.0
    for rank, ctx in enumerate(contexts_meta, 1):
        path = ctx.get("path", "") or ctx.get("id", "")
        if expected_page in path or path in expected_page:
            return 1.0 / rank
    return 0.0

def piekbs_context_with_meta(question: str) -> tuple[list[str], list[dict]]:
    """返回 (context文本列表, metadata列表) 用于 Hit Rate 计算"""
    try:
        args = {"question": question, "limit": 10}
        if NO_VEC:
            args["no_vec"] = True
        result = mcp_call("tools/call", {
            "name": "kb_context",
            "arguments": args
        })
        content_text = ""
        for item in result.get("content", []):
            if item.get("type") == "text":
                content_text += item["text"]
        contexts = []
        meta = []
        try:
            parsed = json.loads(content_text)
            for page in parsed.get("wiki_pages", []):
                ctx = f"[{page.get('title','')}] {page.get('description','')}"
                if page.get("snippet"):
                    ctx += f"\n{page['snippet']}"
                contexts.append(ctx)
                meta.append({"path": page.get("path",""), "id": page.get("id","")})
            for src in parsed.get("raw_sources", []) or []:
                meta.append({"path": src.get("path",""), "id": src.get("id","")})
        except Exception:
            if content_text:
                contexts = [content_text[:2000]]
        return contexts if contexts else ["(no context)"], meta
    except Exception as e:
        print(f"  MCP error: {e}")
        return ["(no context)"], []

def evaluate(questions: list[dict], system_name: str, context_fn) -> dict:
    scores = {"faithfulness": [], "answer_relevancy": [], "context_precision": [],
              "context_recall": [], "hit_rate": [], "mrr": []}
    print(f"\n{'='*50}")
    print(f"评估系统: {system_name}")
    print(f"{'='*50}")

    use_meta = (context_fn == piekbs_context)

    for i, q in enumerate(questions, 1):
        question = q["question"]
        ground_truth = q["ground_truth"]
        expected_page = q.get("expected_page", "")
        print(f"\n[{i}/{len(questions)}] {question[:60]}...")

        if use_meta:
            # 单次调用，同时获取 context 和 meta（避免两次 MCP 调用）
            contexts, meta = piekbs_context_with_meta(question)
        else:
            contexts = context_fn(question)
            meta = []

        print(f"  检索到 {len(contexts)} 个 context 片段")

        # 并行5路：生成答案 + F + AR + CP + CR 同时跑
        with ThreadPoolExecutor(max_workers=5) as ex:
            f_ans = ex.submit(generate_answer, question, contexts)
            f_cp  = ex.submit(score_context_precision, question, contexts)
            f_cr  = ex.submit(score_context_recall, question, contexts, ground_truth)
            answer = f_ans.result()
            cp = f_cp.result()
            cr = f_cr.result()
            # F 和 AR 依赖 answer，串行跑
            f_f  = ex.submit(score_faithfulness, answer, contexts)
            f_ar = ex.submit(score_answer_relevancy, question, answer)
            f  = f_f.result()
            ar = f_ar.result()

        hr = score_hit_rate(meta, expected_page) if use_meta else 0
        mrr = score_mrr(meta, expected_page) if use_meta else 0.0

        scores["faithfulness"].append(f)
        scores["answer_relevancy"].append(ar)
        scores["context_precision"].append(cp)
        scores["context_recall"].append(cr)
        scores["hit_rate"].append(hr)
        scores["mrr"].append(mrr)
        print(f"  F={f:.2f} AR={ar:.2f} CP={cp:.2f} CR={cr:.2f} Hit={hr} MRR={mrr:.2f}")

    return {k: sum(v)/len(v) if v else 0 for k, v in scores.items()}

def main():
    print("\n=== PieKBS vs Naive RAG 评估 ===\n")

    # 支持 v2 问题集（concept/comparison/decision，含 expected_page + Hit Rate）
    # 优先使用 questions_v2.json，回退到 questions_rag.json
    v2_path = os.path.join(os.path.dirname(__file__), "questions_v2.json")
    preset_path = os.path.join(os.path.dirname(__file__), "questions_rag.json")
    if os.path.exists(v2_path):
        print(f"加载 v2 问题集：{v2_path}")
        with open(v2_path) as f:
            questions = json.load(f)
    elif os.path.exists(preset_path):
        print(f"加载预设问题集：{preset_path}")
        with open(preset_path) as f:
            questions = json.load(f)
    else:
        print("生成测试问题集（5题）...")
        questions = generate_questions(n=5)
    print(f"共 {len(questions)} 个问题\n")

    # 并行评估 PieKBS 和 Naive RAG
    with ThreadPoolExecutor(max_workers=2) as executor:
        f_piekbs = executor.submit(evaluate, questions, "PieKBS (kb_context)", piekbs_context)
        f_naive    = executor.submit(evaluate, questions, "Naive RAG (BM25 keyword)", naive_rag_context)
        piekbs_scores = f_piekbs.result()
        naive_scores    = f_naive.result()

    # 输出对比结果
    print("\n" + "="*60)
    print("📊 评估结果对比")
    print("="*60)
    metrics = ["faithfulness", "answer_relevancy", "context_precision", "context_recall", "hit_rate", "mrr"]
    labels  = ["忠实度", "答案相关性", "上下文精度", "上下文召回", "命中率(PieKBS)", "MRR(PieKBS)"]
    print(f"{'指标':<18} {'PieKBS':>10} {'Naive RAG':>10} {'提升':>8}")
    print("-"*50)
    for m, label in zip(metrics, labels):
        w = piekbs_scores[m]
        n = naive_scores[m]
        delta = w - n
        sign = "↑" if delta > 0 else ("↓" if delta < 0 else "→")
        print(f"{label:<14} {w:>10.3f} {n:>10.3f} {sign}{abs(delta):>6.3f}")
    print("="*60)

    # 保存结果
    out = {
        "questions": questions,
        "piekbs": piekbs_scores,
        "naive_rag": naive_scores,
    }
    from datetime import datetime
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    label = "no_vec" if NO_VEC else "vec"
    out_path = os.path.join(os.path.dirname(__file__), f"results_{ts}_{label}.json")
    with open(out_path, "w") as f:
        json.dump(out, f, ensure_ascii=False, indent=2)
    # 同时保存一份到 /tmp 方便快速访问
    with open("/tmp/eval_result.json", "w") as f:
        json.dump(out, f, ensure_ascii=False, indent=2)
    print(f"\n结果已保存到 {out_path}")

if __name__ == "__main__":
    main()
