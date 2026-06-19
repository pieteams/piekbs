//go:build fts5

package embed

// The daulet/tokenizers package declares `-ltokenizers` but no library search
// path, so the linker cannot find libtokenizers.a on its own. The static
// library ships in the repo's lib/ directory; ${SRCDIR} expands to this file's
// directory at build time, making the path independent of the build CWD or any
// external CGO_LDFLAGS. (ONNX Runtime, by contrast, is loaded at runtime via
// ort.SetSharedLibraryPath and needs no link-time path.)

// #cgo darwin LDFLAGS: -L${SRCDIR}/../../lib
// #cgo linux LDFLAGS: -L${SRCDIR}/../../lib
import "C"
