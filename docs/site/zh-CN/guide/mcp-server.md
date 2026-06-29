# MCP 服务器

PieKBS 通过 MCP 协议暴露知识库工具，支持两种传输模式。

## HTTP 模式（推荐）

单进程共享给所有 Agent — Claude Code、Cursor、VS Code (Copilot)、Windsurf 等。

**启动 PieKBS：**

```bash
export WIKILOOP_KB=/path/to/piekbs-kb
piekbs serve
```

**配置各 Agent：**

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${WIKILOOP_API_KEY}"
      }
    }
  }
}
```

`x-api-key` 对应 `config.yaml` 中的 `server.api_key`，未设置时可省略 `headers`。

## stdio 模式

适用于托管环境（Hermes、OpenClaw 等），PieKBS 作为 Agent 主机的子进程运行。

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "stdio",
      "command": "/path/to/piekbs",
      "args": ["stdio"],
      "env": {
        "WIKILOOP_KB": "/path/to/your-kb"
      }
    }
  }
}
```

首次启动时 KB 目录自动创建，无需手动 `init`。

## 可用工具

| 工具 | 描述 |
|---|---|
| `kb_search` | FTS 关键词搜索，返回带 related 链接的排序结果 |
| `kb_page` | 通过 ID 获取完整页面内容 |
| `kb_add` | 向知识库添加文本文档，触发增量索引和异步提炼 |

管理操作（`status`、`reindex`、`lint`）可通过 Web UI 或 CLI 执行。
