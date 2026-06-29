# 配置参考

PieKBS 通过 KB 根目录下的 `config.yaml` 配置。

## 示例

```yaml
server:
  port: 8766
  api_key: "your-secret-key"   # 省略则禁用认证

distill:
  provider: openai              # openai | anthropic | ollama
  model: gpt-4o-mini
  api_key: "${OPENAI_API_KEY}"
  base_url: ""                  # 可选，用于自定义端点

watcher:
  debounce_ms: 500              # 文件变化防抖延迟
```

## server

| 键 | 默认值 | 描述 |
|---|---|---|
| `port` | `8766` | HTTP 服务器端口 |
| `api_key` | — | MCP 认证 API Key，省略则禁用认证。 |

## distill

`piekbs distill` 和 `piekbs synthesize` 需要此配置。

| 键 | 默认值 | 描述 |
|---|---|---|
| `provider` | — | LLM 提供商：`openai`、`anthropic`、`ollama` |
| `model` | — | 模型名称 |
| `api_key` | — | API Key（支持环境变量替换） |
| `base_url` | — | 自定义 API 端点（可选） |

## watcher

| 键 | 默认值 | 描述 |
|---|---|---|
| `debounce_ms` | `500` | 最后一次文件变化后等待多少毫秒再触发提炼 |
