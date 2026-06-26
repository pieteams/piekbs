#!/bin/bash
# Build wikiloop for all platforms.
#
# Usage:
#   ./scripts/build.sh [version] [target...]
#
# Targets:
#   darwin-arm64   → WikiLoop.app + .dmg  (requires macOS Apple Silicon)
#   darwin-amd64   → WikiLoop.app + .dmg  (requires macOS Intel)
#   linux-amd64    → tar.gz with binary
#   linux-arm64    → tar.gz with binary
#   windows-amd64  → zip with binary (pure Go, no CGO)
#   all            → all of the above (default)
#
# Examples:
#   ./scripts/build.sh 1.2.0
#   ./scripts/build.sh 1.2.0 linux-amd64
#
# Dependencies:
#   Linux targets: brew install FiloSottile/musl-cross/musl-cross
#   macOS dmg:     brew install create-dmg  (optional, skipped if absent)
set -e

VERSION=${1:-0.1.0}
shift || true
REQUESTED=("$@")
[ ${#REQUESTED[@]} -eq 0 ] && REQUESTED=("all")

OUTDIR="dist"

mkdir -p "$OUTDIR"
# Clean up create-dmg temp files from previous runs
find "$OUTDIR" -maxdepth 1 -name "rw.*.dmg" -delete 2>/dev/null || true

# ── helpers ──────────────────────────────────────────────────────────────────

want() {
    local t=$1
    for r in "${REQUESTED[@]}"; do
        [ "$r" = "all" ] && return 0
        [ "$r" = "$t" ] && return 0
    done
    return 1
}

# ── macOS .app + dmg ─────────────────────────────────────────────────────────

build_darwin_arm64() {
    echo "→ building darwin-arm64 (.app + dmg) ..."

    local app_dir="$OUTDIR/WikiLoop.app"
    mkdir -p "$app_dir/Contents/MacOS"

    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$app_dir/Contents/MacOS/wikiloop" \
        ./cmd/wikiloop/

    mkdir -p "$app_dir/Contents"
    sed "s/1.0.0/${VERSION}/g" scripts/Info.plist > "$app_dir/Contents/Info.plist"

    mkdir -p "$app_dir/Contents/Resources/web"
    cp -r internal/webui/static/* "$app_dir/Contents/Resources/web/"

    [ -f "scripts/wikiloop.icns" ] && cp scripts/wikiloop.icns "$app_dir/Contents/Resources/wikiloop.icns"

    # Ad-hoc sign so macOS Gatekeeper accepts the app without a developer cert.
    codesign --force --deep --sign - "$app_dir" >/dev/null 2>&1 || true
    xattr -cr "$app_dir" 2>/dev/null || true

    local app_size
    app_size=$(du -sh "$app_dir" | cut -f1)
    echo "  ✓ $app_dir ($app_size)"

    if command -v create-dmg &>/dev/null; then
        local dmg="$OUTDIR/WikiLoop-${VERSION}-darwin-arm64.dmg"
        create-dmg \
            --volname "WikiLoop ${VERSION}" \
            --volicon "scripts/wikiloop.icns" \
            --background "scripts/dmg-background.png" \
            --window-pos 200 100 \
            --window-size 660 380 \
            --icon-size 100 \
            --icon "WikiLoop.app" 495 140 \
            --app-drop-link 165 140 \
            "$dmg" "$app_dir" >/dev/null 2>&1 || true
        if [ -f "$dmg" ]; then
            echo "  ✓ $dmg ($(du -sh "$dmg" | cut -f1))"
        else
            echo "  ✗ dmg creation failed"
        fi
    else
        echo "  ℹ  skipping dmg (install: brew install create-dmg)"
    fi
}

# ── macOS Intel .app + dmg ───────────────────────────────────────────────────

build_darwin_amd64() {
    echo "→ building darwin-amd64 (.app + dmg) ..."

    local app_dir="$OUTDIR/WikiLoop-amd64.app"
    mkdir -p "$app_dir/Contents/MacOS"

    CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$app_dir/Contents/MacOS/wikiloop" \
        ./cmd/wikiloop/

    mkdir -p "$app_dir/Contents"
    sed "s/1.0.0/${VERSION}/g" scripts/Info.plist > "$app_dir/Contents/Info.plist"

    mkdir -p "$app_dir/Contents/Resources/web"
    cp -r internal/webui/static/* "$app_dir/Contents/Resources/web/"

    [ -f "scripts/wikiloop.icns" ] && cp scripts/wikiloop.icns "$app_dir/Contents/Resources/wikiloop.icns"

    codesign --force --deep --sign - "$app_dir" >/dev/null 2>&1 || true
    xattr -cr "$app_dir" 2>/dev/null || true

    local app_size
    app_size=$(du -sh "$app_dir" | cut -f1)
    echo "  ✓ $app_dir ($app_size)"

    if command -v create-dmg &>/dev/null; then
        local dmg="$OUTDIR/WikiLoop-${VERSION}-darwin-amd64.dmg"
        create-dmg \
            --volname "WikiLoop ${VERSION}" \
            --volicon "scripts/wikiloop.icns" \
            --background "scripts/dmg-background.png" \
            --window-pos 200 100 \
            --window-size 660 380 \
            --icon-size 100 \
            --icon "WikiLoop-amd64.app" 495 140 \
            --app-drop-link 165 140 \
            "$dmg" "$app_dir" >/dev/null 2>&1 || true
        if [ -f "$dmg" ]; then
            echo "  ✓ $dmg ($(du -sh "$dmg" | cut -f1))"
        else
            echo "  ✗ dmg creation failed"
        fi
    else
        echo "  ℹ  skipping dmg (install: brew install create-dmg)"
    fi
}

# ── Linux tar.gz ──────────────────────────────────────────────────────────────

build_linux() {
    local goarch=$1 cc=$2 suffix=$3
    echo "→ building $suffix (tar.gz) ..."

    if ! command -v "$cc" &>/dev/null; then
        echo "  ✗ $cc not found — skipping $suffix"
        echo "    install: brew install FiloSottile/musl-cross/musl-cross"
        return
    fi

    local bin="$OUTDIR/wikiloop-${suffix}"

    CGO_ENABLED=1 GOOS=linux GOARCH=$goarch \
        CC="$cc" \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$bin" ./cmd/wikiloop/

    local staging="$OUTDIR/.pkg-${suffix}"
    local tarball="$OUTDIR/wikiloop-${VERSION}-${suffix}.tar.gz"
    mkdir -p "$staging"
    cp "$bin" "$staging/wikiloop"
    tar -czf "$tarball" -C "$staging" .
    rm -r "$staging" "$bin"

    echo "  ✓ $tarball ($(du -sh "$tarball" | cut -f1))"
}

# ── Windows zip ───────────────────────────────────────────────────────────────

build_windows_amd64() {
    echo "→ building windows-amd64 (zip) ..."

    local bin="$OUTDIR/wikiloop.exe"
    # Pure Go build — modernc.org/sqlite works without CGO.
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$bin" ./cmd/wikiloop/

    local staging="$OUTDIR/.pkg-windows-amd64"
    local zipfile="$OUTDIR/wikiloop-${VERSION}-windows-amd64.zip"
    mkdir -p "$staging"
    cp "$bin" "$staging/wikiloop.exe"

    cd "$OUTDIR/.pkg-windows-amd64" && zip -q "../wikiloop-${VERSION}-windows-amd64.zip" * && cd - >/dev/null
    rm -r "$staging" "$bin"

    echo "  ✓ $zipfile ($(du -sh "$zipfile" | cut -f1))"
}

# ── dispatch ──────────────────────────────────────────────────────────────────

echo "Building wikiloop v${VERSION}"
echo

want "darwin-arm64"  && build_darwin_arm64
want "darwin-amd64"  && build_darwin_amd64
want "linux-amd64"   && build_linux amd64 x86_64-linux-musl-gcc  linux-amd64
want "linux-arm64"   && build_linux arm64 aarch64-linux-musl-gcc linux-arm64
want "windows-amd64" && build_windows_amd64

echo
echo "Done. Artifacts in $OUTDIR/"
ls -lh "$OUTDIR"/ | grep -v "^total\|\.app$" || true
