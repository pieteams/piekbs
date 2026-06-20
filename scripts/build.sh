#!/bin/bash
# Build wikiloop for all platforms.
#
# Usage:
#   ./scripts/build.sh [version] [target...]
#
# Targets:
#   darwin-arm64   → WikiLoop.app + .dmg  (requires macOS Apple Silicon)
#   darwin-amd64   → WikiLoop.app + .dmg  (requires macOS Intel)
#   linux-amd64    → tar.gz with binary only (models downloaded separately)
#   linux-arm64    → tar.gz with binary only (models downloaded separately)
#   windows-amd64  → zip with binary + onnxruntime.dll (FTS only; vector requires model)
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
LIBDIR="lib"
TOKENIZERS_VERSION="v1.27.0"
TOKENIZERS_BASE="https://github.com/daulet/tokenizers/releases/download/${TOKENIZERS_VERSION}"
ORT_VERSION="1.26.0"
ORT_BASE="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}"

mkdir -p "$OUTDIR" "$LIBDIR"
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

ensure_lib() {
    local artifact=$1 suffix=$2
    local libfile="$LIBDIR/$suffix/libtokenizers.a"
    [ -f "$libfile" ] && return 0
    echo "  ↓ downloading ${artifact}.tar.gz ..."
    mkdir -p "$LIBDIR/$suffix"
    if ! curl -fsSL "${TOKENIZERS_BASE}/${artifact}.tar.gz" | tar -xz -C "$LIBDIR/$suffix"; then
        echo "  ✗ failed to download ${TOKENIZERS_BASE}/${artifact}.tar.gz"
        return 1
    fi
    echo "  ✓ $libfile"
}

# ensure_ort downloads libonnxruntime for the given platform into lib/ort/<platform>/
ensure_ort() {
    local platform=$1  # e.g. "osx-arm64", "linux-x64", "linux-aarch64"
    local libfile
    case "$platform" in
        osx-*)   libfile="libonnxruntime.dylib" ;;
        linux-*) libfile="libonnxruntime.so" ;;
    esac
    local outdir="$LIBDIR/ort-${platform}"
    local dest="$outdir/$libfile"
    [ -f "$dest" ] && return 0
    echo "  ↓ downloading libonnxruntime ${ORT_VERSION} (${platform}) ..."
    mkdir -p "$outdir"
    local url="${ORT_BASE}/onnxruntime-${platform}-${ORT_VERSION}.tgz"
    if ! curl -fsSL "$url" | tar -xz -C "$outdir" --strip-components=3 \
        "./onnxruntime-${platform}-${ORT_VERSION}/lib/${libfile}"; then
        echo "  ✗ failed to download $url"
        return 1
    fi
    echo "  ✓ $dest"
}

# ── macOS .app + dmg ─────────────────────────────────────────────────────────

build_darwin_arm64() {
    echo "→ building darwin-arm64 (.app + dmg) ..."

    local app_dir="$OUTDIR/WikiLoop.app"
    local lib_suffix="darwin-arm64"
    ensure_lib "libtokenizers.darwin-arm64" "$lib_suffix"
    local libpath
    libpath="$(pwd)/$LIBDIR/$lib_suffix"

    # Binary
    mkdir -p "$app_dir/Contents/MacOS"
    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
        CGO_LDFLAGS="-L${libpath}" \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$app_dir/Contents/MacOS/wikiloop" \
        ./cmd/wikiloop/ 2>/dev/null

    # Info.plist
    mkdir -p "$app_dir/Contents"
    sed "s/1.0.0/${VERSION}/g" scripts/Info.plist > "$app_dir/Contents/Info.plist"

    # Web UI static files
    mkdir -p "$app_dir/Contents/Resources/web"
    cp -r internal/webui/static/* "$app_dir/Contents/Resources/web/"

    # Models are NOT bundled — users download them separately and place in
    # their KB directory (<WIKILOOP_KB>/models/bge-small-zh/).
    # See: https://github.com/jasen215/wikiloop/releases (models asset)

    # Bundle libonnxruntime into Contents/Frameworks/ so the app is self-contained.
    # FindOrtLib searches ../Frameworks/ relative to the binary.
    ensure_ort "osx-arm64"
    local ort_dylib="$LIBDIR/ort-osx-arm64/libonnxruntime.dylib"
    if [ -f "$ort_dylib" ]; then
        mkdir -p "$app_dir/Contents/Frameworks"
        cp "$ort_dylib" "$app_dir/Contents/Frameworks/libonnxruntime.dylib"
        # Fix the dylib's own install name so it loads correctly from Frameworks/.
        install_name_tool -id "@rpath/libonnxruntime.dylib" \
            "$app_dir/Contents/Frameworks/libonnxruntime.dylib" 2>/dev/null || true
        # Add rpath pointing to Frameworks/ so the binary finds it.
        install_name_tool -add_rpath "@executable_path/../Frameworks" \
            "$app_dir/Contents/MacOS/wikiloop" 2>/dev/null || true
        echo "  ✓ bundled libonnxruntime.dylib → Contents/Frameworks/"
    else
        echo "  ⚠ libonnxruntime not found — ONNX will require brew install onnxruntime"
    fi

    # Icon
    [ -f "scripts/wikiloop.icns" ] && cp scripts/wikiloop.icns "$app_dir/Contents/Resources/wikiloop.icns"

    # Ad-hoc sign so macOS Gatekeeper accepts the app without a developer cert.
    # Without this, residual signature metadata causes silent launch rejection.
    codesign --force --deep --sign - "$app_dir" >/dev/null 2>&1 || true
    xattr -cr "$app_dir" 2>/dev/null || true

    local app_size
    app_size=$(du -sh "$app_dir" | cut -f1)
    echo "  ✓ $app_dir ($app_size)"

    # DMG (optional)
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
    local lib_suffix="darwin-amd64"
    ensure_lib "libtokenizers.darwin-x86_64" "$lib_suffix"
    local libpath
    libpath="$(pwd)/$LIBDIR/$lib_suffix"

    mkdir -p "$app_dir/Contents/MacOS"
    CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
        CGO_LDFLAGS="-L${libpath}" \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$app_dir/Contents/MacOS/wikiloop" \
        ./cmd/wikiloop/ 2>/dev/null

    mkdir -p "$app_dir/Contents"
    sed "s/1.0.0/${VERSION}/g" scripts/Info.plist > "$app_dir/Contents/Info.plist"

    mkdir -p "$app_dir/Contents/Resources/web"
    cp -r internal/webui/static/* "$app_dir/Contents/Resources/web/"

    ensure_ort "osx-x86_64"
    local ort_dylib="$LIBDIR/ort-osx-x86_64/libonnxruntime.dylib"
    if [ -f "$ort_dylib" ]; then
        mkdir -p "$app_dir/Contents/Frameworks"
        cp "$ort_dylib" "$app_dir/Contents/Frameworks/libonnxruntime.dylib"
        install_name_tool -id "@rpath/libonnxruntime.dylib" \
            "$app_dir/Contents/Frameworks/libonnxruntime.dylib" 2>/dev/null || true
        install_name_tool -add_rpath "@executable_path/../Frameworks" \
            "$app_dir/Contents/MacOS/wikiloop" 2>/dev/null || true
        echo "  ✓ bundled libonnxruntime.dylib → Contents/Frameworks/"
    else
        echo "  ⚠ libonnxruntime not found — ONNX will require brew install onnxruntime"
    fi

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
    local goarch=$1 cc=$2 lib_artifact=$3 suffix=$4
    echo "→ building $suffix (tar.gz) ..."

    if ! command -v "$cc" &>/dev/null; then
        echo "  ✗ $cc not found — skipping $suffix"
        echo "    install: brew install FiloSottile/musl-cross/musl-cross"
        return
    fi

    ensure_lib "$lib_artifact" "$suffix"
    local libpath
    libpath="$(pwd)/$LIBDIR/$suffix"
    local bin="$OUTDIR/wikiloop-${suffix}"

    CGO_ENABLED=1 GOOS=linux GOARCH=$goarch \
        CC="$cc" \
        CGO_LDFLAGS="-L${libpath}" \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$bin" ./cmd/wikiloop/

    # Bundle libonnxruntime.so alongside the binary so the package is self-contained.
    # FindOrtLib searches the binary's own directory first (rpath $ORIGIN).
    local ort_platform
    [ "$goarch" = "amd64" ] && ort_platform="linux-x64" || ort_platform="linux-aarch64"
    ensure_ort "$ort_platform"
    local ort_so="$LIBDIR/ort-${ort_platform}/libonnxruntime.so"

    # Pack: binary + libonnxruntime.so (models downloaded separately by the user)
    local staging="$OUTDIR/.pkg-${suffix}"
    local tarball="$OUTDIR/wikiloop-${VERSION}-${suffix}.tar.gz"
    mkdir -p "$staging"
    cp "$bin" "$staging/wikiloop"
    if [ -f "$ort_so" ]; then
        cp "$ort_so" "$staging/libonnxruntime.so"
        echo "  ✓ bundled libonnxruntime.so"
    else
        echo "  ⚠ libonnxruntime.so not found — ONNX requires manual install"
    fi
    tar -czf "$tarball" -C "$staging" .
    rm -r "$staging" "$bin"

    echo "  ✓ $tarball ($(du -sh "$tarball" | cut -f1))"
}

# ── Windows zip ───────────────────────────────────────────────────────────────

build_windows_amd64() {
    echo "→ building windows-amd64 (zip) ..."

    local lib_suffix="windows-amd64"
    local libpath
    libpath="$(pwd)/$LIBDIR/$lib_suffix"

    if [ ! -f "$libpath/libtokenizers.a" ] && [ ! -f "$libpath/libtokenizers.lib" ]; then
        echo "  ✗ libtokenizers not found in $libpath — skipping windows-amd64"
        echo "    Build it with: cd /tmp/tokenizers && cargo build --release"
        return
    fi

    local bin="$OUTDIR/wikiloop.exe"
    # Use mingw gcc for CGO on Windows (supports GNU ABI used by libtokenizers.a)
    local cc="x86_64-w64-mingw32-gcc"
    if ! command -v "$cc" &>/dev/null; then
        cc="gcc"  # fallback on Windows runners where mingw is on PATH as gcc
    fi
    CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
        CC="$cc" \
        CGO_LDFLAGS="-L${libpath}" \
        go build -tags fts5 \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$bin" ./cmd/wikiloop/

    local staging="$OUTDIR/.pkg-windows-amd64"
    local zipfile="$OUTDIR/wikiloop-${VERSION}-windows-amd64.zip"
    mkdir -p "$staging"
    cp "$bin" "$staging/wikiloop.exe"

    # Bundle onnxruntime.dll if available
    local ort_dll="$LIBDIR/ort-windows-amd64/onnxruntime.dll"
    if [ -f "$ort_dll" ]; then
        cp "$ort_dll" "$staging/onnxruntime.dll"
        echo "  ✓ bundled onnxruntime.dll"
    fi

    cd "$OUTDIR/.pkg-windows-amd64" && zip -q "../wikiloop-${VERSION}-windows-amd64.zip" * && cd - >/dev/null
    rm -r "$staging" "$bin"

    echo "  ✓ $zipfile ($(du -sh "$zipfile" | cut -f1))"
}

# ── dispatch ──────────────────────────────────────────────────────────────────

echo "Building wikiloop v${VERSION}"
echo

want "darwin-arm64"  && build_darwin_arm64
want "darwin-amd64"  && build_darwin_amd64
want "linux-amd64"   && build_linux amd64 x86_64-linux-musl-gcc  libtokenizers.linux-musl-amd64 linux-amd64
want "linux-arm64"   && build_linux arm64 aarch64-linux-musl-gcc libtokenizers.linux-musl-arm64 linux-arm64
want "windows-amd64" && build_windows_amd64

echo
echo "Done. Artifacts in $OUTDIR/"
ls -lh "$OUTDIR"/ | grep -v "^total\|\.app$" || true
