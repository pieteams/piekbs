# AGENTS.md

## 通用开发规范

永远不要提交 Co-Authored-By
所有 git commit 必须等用户确认后才能执行，不得主动提交
设计文档和对应的实现代码一起提交，不单独提交设计文档
打 tag 时只执行 `git tag <tagname>`，不执行 `git push origin <tagname>`

## 项目概览

PieKBS 是面向 AI Agent 的本地优先知识库引擎。Go 单二进制，无外部依赖。

```
cmd/piekbs/         CLI 入口（serve / index / distill / lint / init 等子命令）
internal/
  kb/               核心检索（FTS5 + RRF + graph boost + 时间衰减）
  mcp/              MCP 服务（stdio + HTTP 双模式，工具：kb_search / kb_page / kb_add）
  distill/          蒸馏队列（SQLite 持久化 + 3 worker + 指数退避）
  synthesize/       综合页生成（concept / comparison / decision）
  watcher/          文件变更监听，自动触发 distill
  webui/            内置 Web UI
  config/           配置（config.yaml，支持 PIEKBS_* 环境变量覆盖）
scripts/            构建脚本（build.sh，产物：PieKBS-{ver}-macos-arm64.dmg 等）
eval/               RAGAS 评测脚本（eval_piekbs.py，eval/baseline_result.json 为基准）
docs/superpowers/   设计文档和探索文档（gitignore，仅本地）
```

## 技术栈

- Go 1.25+，构建 tag：`fts5`
- SQLite（modernc.org/sqlite，纯 Go，无 CGO）
- FTS5 tokenize='trigram'
- MCP 协议（mark3labs/mcp-go）
- 环境变量：`PIEKBS_KB`、`PIEKBS_API_KEY`、`PIEKBS_HOST`、`PIEKBS_PORT`

## 开发常用命令

```bash
go build -tags fts5 ./cmd/piekbs/          # 构建
go test -tags fts5 ./...                   # 测试（136 个，< 20s）
./scripts/build.sh 0.4.7 darwin-arm64      # 打包 macOS DMG
PIEKBS_KB=~/.hermes/piekbs-kb piekbs serve # 启动服务
python3 eval/eval_piekbs.py                # 运行评测（需要 LLM 环境）
```

## 本地部署测试

修改代码后，需构建并替换 `/Applications/PieKBS.app` 验证。步骤：

```bash
# 1. 测试
go test -tags fts5 ./...

# 2. 构建（产物在 dist/）
./scripts/build.sh <version> darwin-arm64

# 3. 杀进程 → 替换 app → 重启
pkill -x piekbs
rm -rf /Applications/PieKBS.app
cp -R dist/PieKBS.app /Applications/
open /Applications/PieKBS.app
```

一步执行（仅构建已完成时）：

```bash
pkill -x piekbs; rm -rf /Applications/PieKBS.app; cp -R dist/PieKBS.app /Applications/; open /Applications/PieKBS.app
```

## 检索架构要点

- AND-first → OR fallback → LIKE fallback
- 多种类并行 FTS + RRF（k=60）融合
- 排序：synthesized 1.3× boost + authority 加权 + 时间衰减（指数 λ）+ graph_boost
- 分层返回：source-note pool + synth pool（concept/comparison/decision）

## 评测基准

`eval/baseline_result.json`，v3 题集（12 道 source-note 为 expected_page）：
AR=1.000，CP=0.536，CR=0.567，Hit Rate=0.583，MRR=0.165

改动 `internal/kb/search.go`、`internal/kb/index.go`、`internal/distill/` 或 `internal/synthesize/` 前后需本地跑一次 eval 并对比基准。

## 发布流程

运行 /release 查看完整发布步骤。
