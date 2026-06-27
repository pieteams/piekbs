<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>面向 Agent 的知識搜尋引擎 — 蒸餾原始資料為結構化 Markdown 知識庫，透過 MCP 搜尋和讀取</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <strong>繁體中文</strong> |
    <a href="README.ru.md">Русский</a> |
    <a href="README.de.md">Deutsch</a> |
    <a href="README.fr.md">Français</a> |
    <a href="README.es.md">Español</a> |
    <a href="README.ko.md">한국어</a>
  </p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

WikiLoop 是面向 Agent 的本機優先知識搜尋引擎。它把原始文件蒸餾為結構化、可 review 的 Markdown 知識庫，透過兩個 MCP 工具——`kb_search` 和 `kb_page`——讓 Agent 依自己的節奏搜尋和深讀。

![WikiLoop Screenshot](image-001.png)

## 設計理念

WikiLoop 基於一個核心觀察：**Agent 使用外部知識工具的方式和人用搜尋引擎一樣**——用不同關鍵詞多次查詢，順著關聯連結展開，最終自己綜合結論。它們不需要系統打包好的答案，需要的是能自主驗證、自主彙總的原始材料。

因此 WikiLoop 的職責不是回答問題，而是確保 Agent 搜尋時能找到正確的文件，並能完整讀取。

```text
wikiloop-kb/
  raw/                  權威來源 — 任意格式的原始資料。
                        放入檔案後 watcher 自動觸發蒸餾。

  wiki/                 結構化 Markdown 知識層（LLM 維護）。
    source-notes/       每篇原始資料的蒸餾摘要，FTS 檢索主體。
    concepts/           跨文件綜合：概念解釋 / 方法論。
    comparisons/        跨文件綜合：多方案橫向對比。
    decisions/          跨文件綜合：技術決策 / 選型結論。
    _draft/             來源不足（< 2篇）的綜合頁，暫不索引。

  schema/               知識庫本地撰寫規範和頁面範本。
                        編輯此目錄可自訂蒸餾頁面格式。

  index/                生成產物（SQLite FTS 索引、查詢日誌）。
                        自動管理，無需手動修改。
```

## Agent 如何使用 WikiLoop

Agent 透過兩個 MCP 工具與 WikiLoop 互動：

**`kb_search(query, limit?)`** — 用關鍵詞或自然語言搜尋。每次返回最多 5 篇 source-note 和 3 篇 concept/comparison/decision 頁面。每條結果包含 `related` 欄位，列出關聯文件供繼續導覽。用不同關鍵詞多次搜尋，從多個角度涵蓋話題。

**`kb_page(ids, full?)`** — 按 ID（來自 `kb_search` 結果）獲取一篇或多篇頁面的完整內容。一次最多傳 5 個 ID，或對單篇使用 `full=true` 獲取不截斷的全文。

推薦的 Agent 工作流程：

```text
kb_search("關鍵詞 A")              → 發現相關文件
kb_search("關鍵詞 B")              → 換角度再搜一次
kb_page(["id1", "id2", "id3"])   → 深讀最相關的幾篇
Agent 從找到的材料中自己綜合結論
```

Agent 應主動迭代搜尋、跟隨 `related` 連結、交叉驗證、自行得出結論。WikiLoop 不生成答案。

## WikiLoop vs RAG

傳統 RAG 檢索上下文後交給 LLM 來回答，然後返回給 Agent。WikiLoop 把原始材料交給 Agent，讓 Agent 自己推理。

```text
RAG:       用戶提問 → 檢索上下文 → LLM 生成答案
WikiLoop:  Agent 搜尋 → Agent 深讀 → Agent 自己綜合
```

| | RAG | WikiLoop |
|---|---|---|
| 知識形式 | 隱式（向量或切塊） | 顯式（Markdown，可稽核） |
| Agent 角色 | 被動接收上下文 | 主動搜尋和閱讀 |
| 答案來源 | 系統生成 | Agent 自主彙總 |
| 可稽核 | 否 | 是 — git diff、lint、衝突連結 |
| 多跳推理 | 依賴 LLM | `related` 欄位的圖展開 |
| 需要 Embedding | 是 | 否（純 FTS） |

WikiLoop wiki bundle 遵循 [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) 規範。

## 知識流水線

原始文件經過蒸餾流水線處理後才能被 Agent 搜尋到：

**第一步 — 蒸餾（自動）**

把任意 Markdown 檔案放入 `raw/`，`wikiloop serve` 的 watcher 會自動完成蒸餾 + 建索引。LLM 把原始資料提取為結構化 source-note，寫入 `wiki/source-notes/`，包含：
- `key_claims`：內嵌同義詞和中英文變體（ALIAS RULE）——確保 FTS 能命中所有查詢變體
- `【實體|類型】` 格式的命名實體標註
- `related_to`、`supports`、`contradicts` 關係連結——驅動搜尋結果中的 `related` 欄位
- `authority`（1–5 權威度）和 `doc_type` 元資料

**第二步 — 綜合（按需）**

```bash
wikiloop synthesize --topic "RAG"
```

當某話題積累了足夠的 source-note 後，生成 concept / comparison / decision 綜合頁。來源不足 2 篇的新頁面會進入 `wiki/<type>/_draft/`，積累到閾值後自動晉級到正式目錄並加入索引。

**第三步 — 搜尋**

Agent 透過 MCP 使用 `kb_search` + `kb_page`。搜尋基於純 FTS（SQLite FTS5 + BM25 評分），不需要向量模型。

## 安裝

從 Releases 頁面下載對應平台的預編譯套件：

| 平台 | 檔案 |
|---|---|
| macOS Apple Silicon（ARM64）| `WikiLoop-<version>-macos-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `wikiloop-<version>-windows-amd64.zip` |

> **macOS Intel（x86_64）：** 暫無預編譯套件。GitHub Actions 的 Intel macOS runner 已於 2025 年 4 月停用。Intel Mac 用戶請在本機從原始碼建置：`CGO_ENABLED=1 go build -tags fts5 -o wikiloop ./cmd/wikiloop/`

**macOS：** 開啟 DMG，將 WikiLoop 拖入 Applications。App 以 menubar 圖示形式運行。

**Linux：**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

**Windows：** 解壓 zip，執行 `wikiloop.exe serve`（或 `wikiloop.exe stdio` 用於 MCP）。將目錄加入 `PATH`。無需 CGO，純 Go 二進位。

**鴻蒙 PC（社群實驗性支援）：** WikiLoop 暫無鴻蒙 PC 官方發行套件。但由於核心二進位無需 CGO（純 Go + SQLite），可透過社群 [Harmonybrew](https://harmonybrew.dev) 套件管理器在鴻蒙 PC 上原生建置。環境搭建方法參考 [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo)。

```bash
# 在鴻蒙 PC 上（透過 Harmonybrew 安裝 Go 後）
CGO_ENABLED=0 go build -tags fts5 -o wikiloop ./cmd/wikiloop/
wikiloop serve
```

## 從原始碼建置

需要 Go 1.25+，無需 CGO。

```bash
# macOS / Linux
go build -tags fts5 -o wikiloop ./cmd/wikiloop/

# Windows
go build -tags fts5 -o wikiloop.exe ./cmd/wikiloop/
```

或使用多平台建置腳本：

```bash
./scripts/build.sh [version] [target...]
```

| Target | 輸出 | 平台 |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/wikiloop-<version>-windows-amd64.zip` | Windows x86_64 |

## 儲存庫結構

```text
wikiloop/
  cmd/wikiloop/        # 主入口
  internal/
    kb/                # FTS 索引、搜尋、圖展開、頁面獲取
    mcp/               # MCP server（stdio + HTTP）
    watcher/           # 檔案監控，自動觸發蒸餾 + 建索引
    distill/           # LLM 蒸餾流水線
    synthesize/        # concept/comparison/decision 頁面生成
    convert/           # 原始檔案轉換
    service/           # 系統服務管理（launchd / systemd）
    webui/             # Web UI
    tray/              # macOS 系統托盤（僅 darwin）
    config/            # KB 設定（config.yaml）
  scripts/
    build.sh           # 多平台建置腳本
```

## Schema 與範本

`wikiloop init` 會把內建的撰寫規範和頁面範本複製到 KB 的 `schema/` 目錄：

- `schema/templates/`：source-note / concept / comparison / decision 頁面的 Markdown 範本。
- `schema/references/`：撰寫規範——頁面類型、引用規則、衝突規則、目錄結構。

distill/synthesize 的 prompt 會讀取這些範本，編輯它們即可按 KB 自訂生成的 wiki 格式。

## 快速開始

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop init           # 初始化 KB 目錄並複製 schema/範本
wikiloop serve          # 啟動服務：MCP + Web UI + 檔案監控
wikiloop index          # 建置/更新 FTS 索引
wikiloop status         # 索引統計
wikiloop lint           # 健康檢查 wiki 頁面
```

## 命令參考

所有命令都接受全域 `--kb <path>` 參數（預設取 `$WIKILOOP_KB`，再退回 `~/wikiloop-kb`）。

| 命令 | 說明 |
|---|---|
| `wikiloop init [--force]` | 初始化 KB 目錄並複製內建 schema/範本。 |
| `wikiloop serve` | 啟動常駐服務：HTTP MCP（`/mcp`）+ Web UI + 檔案監控。無子命令時的預設行為。 |
| `wikiloop index` | 從 `wiki/` 和 `raw/` 的 markdown 建置/更新 FTS 索引。 |
| `wikiloop search <query>` | FTS 關鍵詞搜尋，輸出帶路徑和摘要的排序結果。 |
| `wikiloop synthesize [--topic X] [--full]` | 從 source-notes 生成 concept/comparison/decision 頁面。 |
| `wikiloop synthesize --gaps --topic X` | 對某主題做知識缺口分析。 |
| `wikiloop import-lark <URL>` | 匯入飛書/Lark Wiki 頁面及內嵌多維表格到 `raw/lark/`。需要已登入的 `lark-cli`。 |
| `wikiloop lint` | 健康檢查 wiki 頁面：缺失 frontmatter 欄位、斷裂的來源連結。 |
| `wikiloop status` | 列印索引統計（文件數、索引大小）。 |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | 管理系統服務（launchd / systemd）。 |

**LLM 設定**（KB 根目錄的 `config.yaml` 的 `distill` 段）是 `distill` 和 `synthesize` 的必要條件。

## MCP Server

WikiLoop 透過 MCP 協議對外暴露 KB 工具。

**可用 tools：** `kb_search`、`kb_page`

管理操作（狀態、重建索引、健康檢查）透過 Web UI 或 CLI 執行：`wikiloop status`、`wikiloop index`、`wikiloop lint`。

---

### 場景一：本機多 Agent 共享

推薦使用 HTTP 方式：一個 WikiLoop 行程，所有 Agent 共用——Claude Code、Cursor、VS Code（Copilot）、Windsurf、Trae、Codex、Hermes、OpenClaw 等均可接入。

**第一步：啟動 WikiLoop**

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
wikiloop serve
```

> macOS 也可直接雙擊 WikiLoop.app 啟動（menubar 圖示）。

**第二步：各 Agent 設定 HTTP MCP**

在 `~/.claude.json` 的 `mcpServers` 中新增：

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

`x-api-key` 對應 `config.yaml` 中 `server.api_key`，未設定時可省略 headers。

---

### 場景二：託管 Agent 環境（Hermes / OpenClaw 等）

託管環境中，將 WikiLoop 安裝在持久卷上，透過 **stdio** 呼叫——WikiLoop 作為 Agent 宿主的子行程啟動，watcher 在背景自動運行。

以 NAS 掛載的 OpenClaw/Hermes 為例（掛載點 `/root/.openclaw`）：

**1. 安裝到持久卷（一次性）：**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop
```

**2. 安裝 markitdown（推薦）：**

markitdown 支援將 PDF、Word、Excel、PPT、HTML 檔案轉換為 Markdown 後再蒸餾。未安裝時，僅 `.md` 和 `.txt` 檔案會被蒸餾；二進位檔案僅按檔名建索引。

```bash
pip install markitdown
# 驗證
markitdown --version
```

> 已在 OpenClaw/Hermes 上驗證可用（路徑：`/root/.openclaw/workspace/bin/markitdown`）。將 `workspace/bin` 加入 PATH，或在環境中設定完整路徑。

如果 markitdown 不可用，Agent 可自行提取文字（使用 LLM 視覺或其他工具），直接將結果寫入 `$WIKILOOP_KB/raw/converted/<slug>.md`——watcher 會自動拾取。

**3. MCP 設定：**

Hermes（agent config 中的 `mcp_servers`）：

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

KB 目錄在首次啟動時自動建立，無需手動執行 `init`。

**4. 向知識庫新增內容：**

有 `write_file` 權限的 Agent 可直接寫入 KB——watcher 偵測到變更後自動觸發建索引和蒸餾。

| 內容類型 | 寫入路徑 |
|---|---|
| 文章、筆記、參考資料（Markdown/文字） | `$WIKILOOP_KB/raw/<你的分類>/<slug>.md` |
| Agent 轉換的 PDF / Word / Excel / EPUB 內容 | `$WIKILOOP_KB/raw/converted/<slug>.md` |

`raw/converted/` 中的檔案被視為已轉換，直接進入蒸餾，跳過 markitdown 步驟。`raw/` 下其他路徑均經過完整流水線處理（轉換 → 建索引 → 蒸餾）。

`raw/` 下的子目錄組織方式不限——WikiLoop 不強制規定固定結構。

## 系統服務（可選）

`wikiloop serve` 啟動後內建 watcher 會自動監控 KB 目錄變化、觸發蒸餾和建索引，無需額外設定。

如果需要讓 WikiLoop **開機自啟、背景常駐**，可以安裝為系統服務（macOS launchd / Linux systemd）：

```bash
wikiloop service install --kb /path/to/your-kb
wikiloop service status
wikiloop service uninstall
```

日誌：`{WIKILOOP_KB}/index/watcher.log`
