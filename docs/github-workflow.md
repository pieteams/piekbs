# GitHub 工作流指南

本文档说明 PieKBS 的 GitHub 配置、CI 流程、PR 提交规范以及本地代码同步方式。

---

## 目录结构

```
.github/
├── workflows/
│   ├── ci.yml            # 代码检查：push/PR 触发构建和测试
│   ├── release.yml       # 发布：tag 触发三平台构建
│   └── auto-label.yml    # 自动给 PR 打标签
├── ISSUE_TEMPLATE/
│   ├── bug_report.yml    # Bug 报告模板
│   ├── feature_request.yml # 功能请求模板
│   └── config.yml        # Issue 入口配置
├── pull_request_template.md  # PR 描述模板
└── release.yml           # Release notes 分类配置
```

---

## CI 工作流

### ci.yml — 代码检查

**触发条件**：向 `main` 分支 push，或向 `main` 分支提 PR。

**执行内容**：
1. 在 `macos-latest` 上安装 Go（版本来自 `go.mod`）
2. `go build -tags fts5 ./...` — 验证编译通过
3. `go test -tags fts5 ./...` — 跑全部单元测试

所有 PR 合并前必须通过这两步。

### release.yml — 发布构建

**触发条件**：push 以 `v` 开头的 tag（如 `v0.4.4`）。

**三个 job**：

| Job | Runner | 产物 |
|---|---|---|
| `build-macos` | `macos-latest` | macOS arm64 DMG + Linux amd64/arm64 tar.gz |
| `build-windows` | `ubuntu-latest` | Windows amd64 zip（`CGO_ENABLED=0`） |
| `release` | `ubuntu-latest` | 创建 GitHub Release，上传所有产物 |

Release notes 由 GitHub 自动从 PR 标题生成（`generate_release_notes: true`），按 label 分类展示（见下方 release.yml 配置）。

含 `alpha`、`beta`、`rc` 的 tag 自动标记为预发布（prerelease）。

### auto-label.yml — 自动打标签

**触发条件**：PR 创建、标题编辑、新 commit push。

**规则**：读取 PR 标题前缀，自动添加对应 label：

| 标题前缀 | Label |
|---|---|
| `feat:` / `feat(...)` | `feat` |
| `fix:` / `fix(...)` | `fix` |
| `docs:` | `docs` |
| `chore:` | `chore` |
| `ci:` | `ci` |
| `refactor:` | `refactor` |
| `perf:` | `perf` |

标题不符合格式则不打标签，进 release notes 的"Other Changes"分类。

---

## Release Notes 分类

`.github/release.yml` 控制 release notes 的分组展示：

| 分组 | Labels |
|---|---|
| 🚀 Features | `feat` |
| 🐛 Bug Fixes | `fix` |
| 📖 Documentation | `docs` |
| ⚙️ Maintenance | `chore`, `ci`, `refactor`, `perf` |
| 🔀 Other Changes | 其他 |

---

## 提交 PR

### 分支命名

按功能类型命名分支：

```bash
feat/add-multilang-support
fix/mcp-api-key-auth
docs/update-readme
chore/upgrade-go-version
```

### PR 标题规范（Conventional Commits）

标题格式：`<type>(<scope>): <description>`

- `feat(mcp): add kb_add tool for agent writes`
- `fix(webui): language setting not persisting`
- `docs: add GitHub workflow guide`
- `chore(deps): upgrade mcp-go to v0.6`

**scope 可选**，加了更清晰。标题决定 label，label 决定 release notes 分类。

### 提交流程

```bash
# 1. 从最新 main 创建分支
git fetch origin && git reset --hard origin/main
git checkout -b feat/your-feature

# 2. 开发、提交
git add <files>
git commit -m "feat(kb): add support for xlsx direct import"

# 3. 本地测试（必须通过再提交）
go build -tags fts5 ./...
go test -tags fts5 ./...

# 4. push 并创建 PR
git push -u origin feat/your-feature
gh pr create --title "feat(kb): add support for xlsx direct import" --body "..."
```

### PR 描述模板

创建 PR 时自动填入以下模板（`.github/pull_request_template.md`）：

```markdown
## Summary
<!-- What does this PR do? -->

## Changes
- 

## Testing
- [ ] `CGO_ENABLED=1 go build -tags fts5 ./...` passes
- [ ] `CGO_ENABLED=1 go test -tags fts5 ./...` passes
- [ ] Manually tested on macOS / Linux (if applicable)

## Related Issues
<!-- Closes #xxx -->
```

### 合并策略

- 统一使用 **Squash merge**，保持 main 分支历史整洁
- 合并前确保 CI 全部通过
- 合并后在 GitHub 上删除远端分支

---

## 合并后同步本地

PR 合并后，同步本地 main 并清理分支：

```bash
# 同步 main
git fetch origin && git reset --hard origin/main

# 删除已合并的本地分支（squash merge 需要强制删除）
git branch -D feat/your-feature

# 清理已删除的远端分支追踪
git remote prune origin
```

---

## 发布新版本

```bash
# 确保在最新 main
git fetch origin && git reset --hard origin/main

# 打 tag（语义化版本）
git tag v0.4.5
git push origin v0.4.5
```

CI 自动完成：构建三平台产物 → 创建 GitHub Release → 上传 DMG/tar.gz/zip。

---

## 提交 Issue

GitHub 上提 Issue 时有两个模板可选：

**Bug Report**：需填写 PieKBS 版本、平台（macOS/Linux）、复现步骤、相关日志。

**Feature Request**：需填写问题背景、期望的解决方案、已考虑的替代方案。

直接打开空白 Issue 已被禁用，必须通过模板提交。
