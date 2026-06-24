<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>面向 Agent 的知识搜索引擎 — 蒸馏原始资料为结构化 Markdown 知识库，通过 MCP 搜索和读取</p>
  <p><a href="../../README.md">English</a></p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="Platform">
  </p>
</div>

WikiLoop 是面向 Agent 的本地优先知识搜索引擎。它把原始文档蒸馏为结构化、可 review 的 Markdown 知识库，通过两个 MCP 工具——`kb_search` 和 `kb_page`——让 Agent 按自己的节奏搜索和深读。

## 设计理念

WikiLoop 基于一个核心观察：**不要你以为，我要我以为**——Agent 更愿意使用外部知识工具的方式和人用搜索引擎一样，用不同关键词多次查询，顺着关联链接展开，最终自己综合结论。它们不需要系统打包好的答案，需要的是能自主验证、自主汇总的原始材料。

因此 WikiLoop 的职责不是回答问题，而是确保 Agent 搜索时能找到正确的文档，并能完整读取。

```text
wikiloop-kb/
  raw/                  权威来源 — 任意格式的原始资料。
                        放入文件后 watcher 自动触发蒸馏。

  wiki/                 结构化 Markdown 知识层（LLM 维护）。
    source-notes/       每篇原始资料的蒸馏摘要，FTS 检索主体。
    concepts/           跨文档综合：概念解释 / 方法论。
    comparisons/        跨文档综合：多方案横向对比。
    decisions/          跨文档综合：技术决策 / 选型结论。
    _draft/             来源不足（< 2篇）的综合页，暂不索引。

  schema/               知识库本地撰写规范和页面模板。
                        编辑此目录可自定义蒸馏页面格式。

  index/                生成产物（SQLite FTS 索引、查询日志）。
                        自动管理，无需手动修改。
```

## Agent 如何使用 WikiLoop

Agent 通过两个 MCP 工具与 WikiLoop 交互：

**`kb_search(query, limit?)`** — 用关键词或自然语言搜索。每次返回最多 5 篇 source-note 和 3 篇 concept/comparison/decision 页面。每条结果包含 `related` 字段，列出关联文档供继续导航。用不同关键词多次搜索，从多个角度覆盖话题。

**`kb_page(ids, full?)`** — 按 ID（来自 `kb_search` 结果）获取一篇或多篇页面的完整内容。一次最多传 5 个 ID，或对单篇使用 `full=true` 获取不截断的全文。

推荐的 Agent 工作流：

```text
kb_search("关键词 A")              → 发现相关文档
kb_search("关键词 B")              → 换角度再搜一次
kb_page(["id1", "id2", "id3"])   → 深读最相关的几篇
Agent 从找到的材料中自己综合结论
```

Agent 应主动迭代搜索、跟随 `related` 链接、交叉验证、自行得出结论。WikiLoop 不生成答案。

## WikiLoop vs RAG

传统 RAG 检索上下文后交给 LLM 来回答，然后返回给Agent。WikiLoop 把原始材料交给 Agent，让 Agent 自己推理。

```text
RAG:       用户提问 → 检索上下文 → LLM 生成答案
WikiLoop:  Agent 搜索 → Agent 深读 → Agent 自己综合
```

| | RAG | WikiLoop |
|---|---|---|
| 知识形式 | 隐式（向量或切块） | 显式（Markdown，可审计） |
| Agent 角色 | 被动接收上下文 | 主动搜索和阅读 |
| 答案来源 | 系统生成 | Agent 自主汇总 |
| 可审计 | 否 | 是 — git diff、lint、冲突链接 |
| 多跳推理 | 依赖 LLM | `related` 字段的图展开 |
| 需要 Embedding | 是 | 否（纯 FTS） |

WikiLoop wiki bundle 遵循 [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) 规范。

## 知识流水线

原始文档经过蒸馏流水线处理后才能被 Agent 搜索到：

**第一步 — 蒸馏（自动）**

把任意 Markdown 文件放入 `raw/`，`wikiloop serve` 的 watcher 会自动完成蒸馏 + 建索引。LLM 把原始资料提取为结构化 source-note，写入 `wiki/source-notes/`，包含：
- `key_claims`：内嵌同义词和中英文变体（ALIAS RULE）——确保 FTS 能命中所有查询变体
- `【实体|类型】` 格式的命名实体标注
- `related_to`、`supports`、`contradicts` 关系链接——驱动搜索结果中的 `related` 字段
- `authority`（1–5 权威度）和 `doc_type` 元数据

**第二步 — 综合（按需）**

```bash
wikiloop synthesize --topic "RAG"
```

当某话题积累了足够的 source-note 后，生成 concept / comparison / decision 综合页。来源不足 2 篇的新页面会进入 `wiki/<type>/_draft/`，积累到阈值后自动晋级到正式目录并加入索引。

**第三步 — 搜索**

Agent 通过 MCP 使用 `kb_search` + `kb_page`。搜索基于纯 FTS（SQLite FTS5 + BM25 打分），不需要向量模型。

## 安装

从 Releases 页面下载对应平台的预编译包：

| 平台 | 文件 |
|---|---|
| macOS Apple Silicon | `WikiLoop-<version>-darwin-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `wikiloop-<version>-windows-amd64.zip` |

**macOS：** 打开 DMG，将 WikiLoop 拖入 Applications。App 以 menubar 图标形式运行。

**Linux：**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

## 从源码构建

需要 Go 1.25+，无需 CGO。

```bash
go build -tags fts5 -o wikiloop ./cmd/wikiloop/
```

或使用多平台构建脚本：

```bash
./scripts/build.sh [version] [target...]
```

| Target | 输出 | 平台 |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-darwin-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |

## 仓库结构

```text
wikiloop/
  cmd/wikiloop/        # 主入口
  internal/
    kb/                # FTS 索引、搜索、图展开、页面获取
    mcp/               # MCP server（stdio + HTTP）
    watcher/           # 文件监控，自动触发蒸馏 + 建索引
    distill/           # LLM 蒸馏流水线
    synthesize/        # concept/comparison/decision 页面生成
    convert/           # 原始文件转换
    service/           # 系统服务管理（launchd / systemd）
    webui/             # Web UI
    tray/              # macOS 系统托盘（仅 darwin）
    config/            # KB 配置（config.yaml）
  scripts/
    build.sh           # 多平台构建脚本
```

## Schema 与模板

`wikiloop init` 会把内置的撰写规范和页面模板复制到 KB 的 `schema/` 目录：

- `schema/templates/`：source-note / concept / comparison / decision 页面的 Markdown 模板。
- `schema/references/`：撰写规范——页面类型、引用规则、冲突规则、目录结构。

distill/synthesize 的 prompt 会读取这些模板，编辑它们即可按 KB 定制生成的 wiki 格式。

## 快速开始

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop init           # 初始化 KB 目录并复制 schema/模板
wikiloop serve          # 启动服务：MCP + Web UI + 文件监控
wikiloop index          # 构建/更新 FTS 索引
wikiloop status         # 索引统计
wikiloop lint           # 健康检查 wiki 页面
```

## 命令参考

所有命令都接受全局 `--kb <path>` 参数（默认取 `$WIKILOOP_KB`，再退回 `~/wikiloop-kb`）。

| 命令 | 说明 |
|---|---|
| `wikiloop init [--force]` | 初始化 KB 目录并复制内置 schema/模板。 |
| `wikiloop serve` | 启动常驻服务：HTTP MCP（`/mcp`）+ Web UI + 文件监控。无子命令时的默认行为。 |
| `wikiloop index` | 从 `wiki/` 和 `raw/` 的 markdown 构建/更新 FTS 索引。 |
| `wikiloop search <query>` | FTS 关键词搜索，输出带路径和摘要的排序结果。 |
| `wikiloop synthesize [--topic X] [--full]` | 从 source-notes 生成 concept/comparison/decision 页面。 |
| `wikiloop synthesize --gaps --topic X` | 对某主题做知识缺口分析。 |
| `wikiloop import-lark <URL>` | 导入飞书/Lark Wiki 页面及内嵌多维表格到 `raw/lark/`。需要已登录的 `lark-cli`。 |
| `wikiloop lint` | 健康检查 wiki 页面：缺失 frontmatter 字段、断裂的来源链接。 |
| `wikiloop status` | 打印索引统计（文档数、索引大小）。 |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | 管理系统服务（launchd / systemd）。 |

**LLM 配置**（KB 根目录的 `config.yaml` 的 `distill` 段）是 `distill` 和 `synthesize` 的必要条件。

### synthesize 工作流

原始资料来源不限：Agent 抓取的网页、用户放入的文档、调研报告、任意 Markdown 文件。

**第一步：资料进入 KB（自动）**

把 Markdown 放入 `raw/`（按来源分目录，如 `raw/wechat-tech/`、`raw/papers/`），`wikiloop serve` 的 watcher 自动完成蒸馏 + 建索引。

**第二步：按主题生成综合页（按需）**

```bash
# 对"芯片产业"话题生成 concept/comparison/decision 页面
wikiloop synthesize --topic "芯片产业"

# 全量重跑（忽略增量缓存）
wikiloop synthesize --topic "芯片产业" --full

# 不指定主题：处理所有新增/变更的 source-notes
wikiloop synthesize

# 知识缺口分析
wikiloop synthesize --gaps --topic "芯片产业"
# 输出：index/gaps/zhi-pian-chan-ye.md
```

`--topic` 按 source-note 的 `title` 或 `tags` 字段做大小写不敏感子串匹配。

综合页类型：

| 类型 | 输出目录 | 来源不足时 |
|---|---|---|
| concept | `wiki/concepts/` | 进入 `_draft/`，积累后自动晋级 |
| comparison | `wiki/comparisons/` | 同上 |
| decision | `wiki/decisions/` | 同上 |

## 系统服务（可选）

`wikiloop serve` 启动后内置 watcher 会自动监控 KB 目录变化、触发蒸馏和建索引，无需额外配置。

如果需要让 WikiLoop **开机自启、后台常驻**，可以安装为系统服务（macOS launchd / Linux systemd）：

```bash
wikiloop service install --kb /path/to/your-kb
wikiloop service status
wikiloop service uninstall
```

日志：`{WIKILOOP_KB}/index/watcher.log`

## MCP Server

WikiLoop 通过 MCP 协议对外暴露 KB 工具。

**可用 tools：** `kb_search`、`kb_page`、`kb_status`、`kb_reindex`、`kb_lint`

---

### 场景一：本机多 Agent 共享

推荐使用 HTTP 方式：一个 WikiLoop 进程，所有 Agent 共用——Claude Code、Cursor、VS Code（Copilot）、Windsurf、Trae、Codex、Hermes、OpenClaw 等均可接入。

**第一步：启动 WikiLoop**

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
wikiloop serve
```

> macOS 也可直接双击 WikiLoop.app 启动（menubar 图标）。App 启动时自动读取 `~/.zshenv`、`~/.bashrc` 等 shell 配置文件中的 `WIKILOOP_*` 环境变量。

**第二步：各 Agent 配置 HTTP MCP**

在 `~/.claude.json` 的 `mcpServers` 中添加：

```json
{
  "mcpServers": {
    "wikiloop": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${WIKILOOP_API_KEY}"
      }
    }
  }
}
```

`x-api-key` 对应 `config.yaml` 中 `server.api_key`，未配置时可省略 headers。

---

### 场景二：托管 Agent 环境（Hermes / OpenClaw 等）

托管环境的容器无法访问用户本地进程，需将 WikiLoop 安装在 Agent 所在环境的**持久卷**中，通过 stdio 本地调用。

以 NAS 挂载的 OpenClaw/Hermes 为例（挂载点 `/root/.openclaw`）：

**1. 安装到持久卷（一次性）：**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop
```

容器重建后 NAS 重新挂载，二进制和 KB 数据均保留。

**2. MCP 配置：**

Claude Code (`~/.claude.json`)：

```json
{
  "mcpServers": {
    "wikiloop": {
      "command": "/root/.openclaw/wikiloop/wikiloop",
      "args": ["serve"],
      "env": {
        "WIKILOOP_KB": "/root/.openclaw/wikiloop-kb"
      }
    }
  }
}
```

Hermes (`mcp_servers` in agent config)：

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [serve]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
```
