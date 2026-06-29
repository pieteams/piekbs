# 安装

## 下载预编译二进制

| 平台 | 文件 |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

从 [GitHub Releases](https://github.com/pieteams/piekbs/releases) 下载。

> **macOS Intel (x86_64)：** 没有预编译包，请从源码构建：`CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

## macOS

打开 DMG 文件，将 PieKBS 拖入 Applications。应用以菜单栏图标形式运行。

## Linux

```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

## Windows

解压 zip 文件，运行 `piekbs.exe serve`（MCP 模式用 `piekbs.exe stdio`）。将目录加入 `PATH` 方便使用。

## 从源码构建

需要 Go 1.25+。

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

或使用多平台构建脚本：

```bash
./scripts/build.sh [version] [target...]
```
