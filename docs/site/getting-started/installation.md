# Installation

## Download Pre-built Binary

| Platform | File |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

Download from [GitHub Releases](https://github.com/pieteams/piekbs/releases).

> **macOS Intel (x86_64):** No pre-built release. Build from source: `CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

## macOS

Open the DMG and drag PieKBS to Applications. The app runs as a menubar icon.

## Linux

```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

## Windows

Extract the zip and run `piekbs.exe serve` (or `piekbs.exe stdio` for MCP). Add the directory to `PATH` for convenience.

## Build from Source

Requires Go 1.25+.

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

Or use the multi-platform build script:

```bash
./scripts/build.sh [version] [target...]
```
