<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>面向 Agent 的本地优先 LLM Wiki 知识库</p>
  <p><a href="../../README.md">English</a></p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="Platform">
  </p>
</div>

WikiLoop 是一个面向 Agent 的 local-first LLM Wiki 知识库。它帮助 Agent 和人类基于本地原始资料，维护有来源、可 review、可版本化的 Markdown 知识层。

## 核心思路

```text
wikiloop-kb/
  raw/      权威来源；保存原始资料和近全文转换稿
  wiki/     Agent 与人共同维护的结构化 Markdown 知识层
  schema/   知识库本地规则、模板、引用规则和工作流
  index/    生成的搜索/索引产物
```

WikiLoop 不是长期记忆系统。用户偏好、项目习惯、跨会话记忆应交给外部 memory provider。WikiLoop 只管理外部知识资料，以及从这些资料派生出的 Wiki 知识层。

## WikiLoop vs RAG

传统 RAG 将原始文档直接送入向量存储——知识是隐式的，藏在 embedding 里，人无法审阅。WikiLoop 在中间增加了一个显式的蒸馏步骤：

```text
RAG:       原始文档 → embed → 向量存储 → LLM 检索
WikiLoop:  原始文档 → wiki（人/Agent 维护）→ FTS + 图 + embed（可选）
```

| | RAG | WikiLoop |
|---|---|---|
| 知识形式 | 隐式（向量） | 显式（Markdown） |
| 可审计 | 否 | 是 — git diff、lint、冲突链接 |
| 冷启动成本 | 低 | 较高（需要 wiki 撰写） |
| 知识衰减 | 难以发现 | 容易 — lint + 引用检查 |
| 多跳推理 | 依赖 LLM | 图展开（显式） |
| Embedding | 必须 | 可选增强 |

WikiLoop wiki bundle 遵循 [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) 规范。

## 安装

从 Releases 页面下载对应平台的预编译包：

| 平台 | 文件 |
|---|---|
| macOS Apple Silicon | `WikiLoop-<version>-darwin-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |

> **Windows** 暂无预编译包。构建依赖 [libtokenizers](https://github.com/daulet/tokenizers)，目前 Rust + CGO 工具链兼容性问题尚未解决。Windows 用户可安装 MinGW-w64 和 Rust 后从源码自行构建。

**macOS：** 打开 DMG，将 WikiLoop 拖入 Applications。App 以 menubar 图标形式运行。

**Linux：**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
# 二进制：/path/to/install/wikiloop
# 模型：  /path/to/install/models/
```

## 从源码构建

需要 Go 1.25+ 并启用 CGO。

```bash
./scripts/build.sh [version] [target...]
```

| Target | 输出 | 平台 |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-darwin-arm64.dmg` | macOS Apple Silicon |
| `darwin-amd64` | `dist/WikiLoop-<version>-darwin-amd64.dmg` | macOS Intel |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |

```bash
./scripts/build.sh 1.2.0               # 所有平台
./scripts/build.sh 1.2.0 linux-amd64   # 单个平台
```

**依赖：**

- Linux 目标：`brew install FiloSottile/musl-cross/musl-cross`
- macOS DMG：`brew install create-dmg`（可选，缺失时跳过）

每个 tar.gz 包含二进制和内置的 embedding 模型。DMG 是拖拽安装的 macOS app bundle，带系统托盘支持。

## 仓库结构

```text
wikiloop/
  cmd/wikiloop/        # 主入口
  internal/
    kb/                # 索引、FTS、向量搜索
    mcp/               # MCP server（stdio + HTTP）
    embed/             # ONNX embedding（bge-small-zh）
    watcher/           # 文件监控，自动触发重建索引
    distill/           # LLM 蒸馏流水线
    convert/           # 原始文件转换
    service/           # 系统服务管理（launchd / systemd）
    webui/             # Web UI
    tray/              # macOS 系统托盘（仅 darwin）
    config/            # KB 配置（config.yaml）
  scripts/
    build.sh           # 多平台构建脚本
    Info.plist         # macOS app bundle 元数据
    wikiloop.icns      # app 图标
  docs/
```

## Schema 与模板

`wikiloop init` 会把内置的撰写规范和页面模板（来自 `internal/kbinit/schema/`）复制到 KB 的 `schema/` 目录：

- `schema/templates/`：source-note / concept / comparison / decision 页面的 Markdown 模板。
- `schema/references/`：撰写规范——页面类型、引用规则、冲突规则、目录结构、维护循环。

distill/synthesize 的 prompt 会读取这些模板，因此编辑它们即可按 KB 定制生成的 wiki 格式。

## 知识库实例

默认知识库目录名为：

```text
wikiloop-kb/
```

这只是约定，用户可以放在任意路径并自定义名称。

最小结构：

```text
wikiloop-kb/
  raw/
  wiki/
  schema/
```

完整推荐结构见 `schema/references/kb-directory-structure.md`（由 `wikiloop init` 创建）。

## 快速开始

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop index          # 构建/更新索引
wikiloop embed          # 生成 embedding（可选）
wikiloop status         # 索引状态
wikiloop search "query"
wikiloop context "question"
wikiloop lint
```

完整选项：`wikiloop --help`。

## 命令参考

所有命令都接受全局 `--kb <path>` 参数（默认取 `$WIKILOOP_KB`，再退回 `~/wikiloop-kb`）。

| 命令 | 说明 |
|---|---|
| `wikiloop init [--force]` | 初始化 KB 目录并复制内置 schema/模板。`--force` 覆盖已有 schema 文件。 |
| `wikiloop serve` | 启动常驻服务：HTTP MCP（`/mcp`）+ Web UI + 文件监控。无子命令时的默认行为。 |
| `wikiloop index` | 从 `wiki/` 和 `raw/` 的 markdown 构建/更新 FTS 索引。 |
| `wikiloop embed [--full]` | 为文档生成向量嵌入。`--full` 删除并重建向量存储。需要 ONNX runtime。 |
| `wikiloop search <query>` | FTS 关键词搜索，输出带路径和摘要的排序结果。 |
| `wikiloop context <question>` | 为问题构建上下文包（相关页面 + 来源）。 |
| `wikiloop distill` | （在 `serve`/watcher 内运行）通过 LLM 把 `raw/` 新文件转为 `wiki/source-notes/`。非独立子命令，自动触发。 |
| `wikiloop synthesize [--topic X] [--full]` | 从 source-notes 生成 concept/comparison/decision 页面。默认增量；`--full` 全量重跑；`--topic` 限定标签/标题匹配。 |
| `wikiloop synthesize --gaps --topic X` | 对某主题做知识缺口分析，报告写入 `index/gaps/<slug>.md`。 |
| `wikiloop import-lark <URL>` | 导入飞书/Lark Wiki 文档，并将内嵌多维表格完整展开为本地可搜索数据集。需要已登录的 `lark-cli`。 |
| `wikiloop lint` | 健康检查 wiki 页面：缺失 frontmatter 字段、断裂的来源链接。 |
| `wikiloop status` | 打印索引统计（文档/嵌入数、索引大小）。 |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | 管理系统服务（launchd / systemd）。 |

**LLM 配置**（KB 根目录的 `config.yaml` 的 `distill` 段）是 `distill` 和 `synthesize` 的必要条件。格式见 MCP Server 章节；`api_type` 选择 `openai`（默认）或 `anthropic`。

DeepSeek 使用 OpenAI 兼容接口：

```yaml
distill:
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"
  api_type: "openai"
```

建议不要把 Token 写入 `config.yaml`，改用环境变量：

```bash
export WIKILOOP_DISTILL_TOKEN="your-api-key"
```

设置页也提供 **Use DeepSeek** 一键预设。Base URL 是否带结尾 `/v1` 均可。

### 导入飞书/Lark Wiki 页面

```bash
wikiloop import-lark "https://example.larkoffice.com/wiki/..."
```

内嵌多维表格会自动分页读取。原始表格以 `.snapshot.tsv` 保存用于审计；
WikiLoop 只索引 `records-deduplicated.txt` 去重合集。去重规则包括相同链接，
以及同一昵称重复提交的相同标题；不同昵称的同名作品会保留。

### synthesize 工作流：从原始资料到主题汇总

典型场景：Agent 收集或生成资料 → 自动蒸馏 → 按主题汇总。

原始资料来源不限，常见方式包括：
- Agent 从网页/公众号抓取的文章
- 用户手动放入的文档（PDF 转换稿、本地笔记）
- Agent 生成的调研报告或分析文档
- 任意 markdown 文件

**1. 资料进入 KB（自动）**

把 markdown 放入 `raw/`（按来源分目录，如 `raw/wechat-tech/`、`raw/papers/`、`raw/reports/`），`wikiloop serve` 的 watcher 会自动完成：
- `distill`：调 LLM 提取 key_claims、关系字段，生成 `wiki/source-notes/`
- `index` + `embed`：更新 FTS 和向量索引

**2. 按主题生成汇总（手动，Agent 主动调用）**

```bash
# 对"芯片产业"主题下所有 source-notes 生成 concept/comparison/decision 页面
wikiloop synthesize --topic "芯片产业"

# 重新全量生成（忽略增量缓存）
wikiloop synthesize --topic "芯片产业" --full

# 不指定主题：扫描所有新增/变更的 source-notes 做跨主题合成
wikiloop synthesize

# 分析"芯片产业"的知识缺口（哪些方向还缺资料）
wikiloop synthesize --gaps --topic "芯片产业"
# 输出：index/gaps/zhi-pian-chan-ye.md
```

**`--topic` 匹配规则**：按 source-note 的 `title` 或 `tags` 字段做大小写不敏感的子串匹配。
例如 `--topic "芯片"` 会匹配 title 含"芯片"或 tags 含"芯片"的所有 source-notes。

**生成内容说明**：

| 类型 | 输出目录 | 触发条件 |
|---|---|---|
| concept | `wiki/concepts/` | ≥3 篇 source-notes 共同提到的概念/方法论 |
| comparison | `wiki/comparisons/` | ≥2 篇 source-notes 可横向对比的工具/方案 |
| decision | `wiki/decisions/` | ≥2 篇 source-notes 支撑的技术判断/选型结论 |

**Agent 调用示例**（MCP tool call）：

```
// 新文章进来后，对"AI Agent 记忆"主题做一次汇总
kb_search("AI Agent 记忆")          // 先确认有哪些 source-notes
// 然后触发 synthesize（通过 shell 或 wikiloop serve 的 /api/synthesize 接口）
wikiloop synthesize --topic "AI Agent 记忆"
```

## 自动索引服务

监控 KB 目录变化，自动触发 `index + embed`。支持 macOS (launchd) 和 Linux (systemd)。

```bash
# 安装并启动系统服务
wikiloop service install --kb /path/to/your-kb

# 查看状态 / 卸载
wikiloop service status
wikiloop service uninstall
```

日志输出到 `{WIKILOOP_KB}/index/watcher.log`。

## MCP Server

WikiLoop 通过 MCP 协议对外暴露 KB 工具，支持两种部署场景。

可用 tools：`kb_search`、`kb_context`、`kb_status`、`kb_reindex`、`kb_lint`

---

### 场景一：本机多 Agent 共享

适合 Claude Code、Cursor 等多个本地 Agent 共享同一 KB 实例。

多 Agent 场景推荐使用 **HTTP 方式**：一个 WikiLoop 进程，所有 Agent 共用，避免 stdio 模式下各 Agent 独立拉起进程导致的索引并发冲突。

**第一步：设置 KB 路径并启动 WikiLoop 服务**

程序默认使用 `~/wikiloop-kb` 作为 KB 目录，也可通过环境变量指定：

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
```

安装完成后，先创建命令行软链：

```bash
# macOS
sudo ln -sf /Applications/WikiLoop.app/Contents/MacOS/wikiloop /usr/local/bin/wikiloop

# Linux（解压后的二进制路径）
sudo ln -sf /path/to/wikiloop /usr/local/bin/wikiloop
```

启动服务：

```bash
wikiloop serve
```

> macOS 也可直接双击 WikiLoop.app 启动（menubar 图标）。app 启动时会自动读取
> `~/.zshenv`、`~/.bashrc` 等 shell 配置文件中的 `WIKILOOP_*` 环境变量，
> 与命令行启动行为一致。

`serve` 启动后同时提供 HTTP MCP（`/mcp`）和 Web UI，端口由 `config.yaml` 的 `server` 段配置（默认 8766）。

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

`x-api-key` 对应 `config.yaml` 中 `server.api_key`，未配置 api_key 时可省略 headers。

---

### 场景二：托管 Agent 环境（Hermes / OpenClaw 等）

托管环境的容器无法访问用户本地进程，需要将 WikiLoop 安装在 Agent 所在环境的**持久卷**中，通过 stdio 本地调用。

以阿里云 NAS 挂载的 OpenClaw/Hermes 为例（挂载点 `/root/.openclaw`）：

**1. 安装到持久卷（一次性）：**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop

# 将模型放到 KB 目录下
mkdir -p /root/.openclaw/wikiloop-kb/models
cp -r /root/.openclaw/wikiloop/models/* /root/.openclaw/wikiloop-kb/models/
```

容器重建后 NAS 重新挂载，二进制和 KB 数据均保留，无需重新安装。

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

OpenClaw 配置格式同 Hermes，参考各平台文档。
