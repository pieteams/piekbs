# CLI 参考

所有命令支持全局 `--kb <path>` 参数（默认依次使用 `$PIEKBS_KB`、`~/piekbs-kb`）。

## 命令列表

| 命令 | 描述 |
|---|---|
| `piekbs init [--force]` | 初始化 KB 目录结构，复制内置 schema/templates |
| `piekbs serve` | 启动长运行服务器：HTTP MCP + Web UI + 文件监听。无子命令时的默认行为。 |
| `piekbs index` | 从 `wiki/` 和 `raw/` 中的 Markdown 构建/更新 FTS 索引 |
| `piekbs search <query>` | FTS 关键词搜索，打印排序结果、路径和摘要 |
| `piekbs synthesize [--topic X] [--full]` | 从 source-note 生成 concept/comparison/decision 页面 |
| `piekbs synthesize --gaps --topic X` | 主题知识空白分析 |
| `piekbs import-lark <URL>` | 导入飞书 Wiki 页面到 `raw/lark/` |
| `piekbs lint` | 检查 wiki 页面健康状况：缺失 frontmatter、断开的 source 链接 |
| `piekbs status` | 打印索引统计（文档数、索引大小） |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | 管理系统服务（launchd / systemd） |

## 系统服务

让 PieKBS 开机启动并在后台运行：

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

日志：`{PIEKBS_KB}/index/watcher.log`
