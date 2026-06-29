# Config Reference

PieKBS is configured via `config.yaml` in the KB root directory.

## Example

```yaml
server:
  port: 8766
  api_key: "your-secret-key"   # omit to disable auth

distill:
  provider: openai              # openai | anthropic | ollama
  model: gpt-4o-mini
  api_key: "${OPENAI_API_KEY}"
  base_url: ""                  # optional, for custom endpoints

watcher:
  debounce_ms: 500              # file change debounce delay
```

## server

| Key | Default | Description |
|---|---|---|
| `port` | `8766` | HTTP server port |
| `api_key` | — | API key for MCP auth. Omit to disable. |

## distill

Required for `piekbs distill` and `piekbs synthesize`.

| Key | Default | Description |
|---|---|---|
| `provider` | — | LLM provider: `openai`, `anthropic`, `ollama` |
| `model` | — | Model name |
| `api_key` | — | API key (supports env var substitution) |
| `base_url` | — | Custom API endpoint (optional) |

## watcher

| Key | Default | Description |
|---|---|---|
| `debounce_ms` | `500` | Milliseconds to wait after last file change before triggering distill |
