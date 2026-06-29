# Schema 与模板

`piekbs init` 会将内置写作规则和页面模板复制到 KB 的 `schema/` 目录。

## 目录结构

```text
schema/
  templates/     各页面类型的 Markdown 模板
  references/    写作规则 — 页面类型、引用规则、冲突处理、目录结构
```

## 页面类型

| 类型 | 位置 | 描述 |
|---|---|---|
| source-note | `wiki/source-notes/` | 每个原始文档对应一个提炼笔记 |
| concept | `wiki/concepts/` | 概念的跨文档综合 |
| comparison | `wiki/comparisons/` | 方案横向对比 |
| decision | `wiki/decisions/` | 带理由的技术决策记录 |

## 自定义

提炼/综合的 prompt 会读取这些模板，因此编辑它们可以自定义每个 KB 的生成格式。

编辑 `schema/templates/` 修改提炼页面的结构；编辑 `schema/references/` 修改写作规则，如引用要求、冲突处理和命名规范。

修改在下次提炼运行时生效。
