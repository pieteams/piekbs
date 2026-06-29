// Package watcher watches the raw/, wiki/, and schema/ directories for file
// changes and triggers a reindex after a debounce period.
package watcher

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 3 * time.Second

// Debouncer coalesces rapid trigger calls into a single callback invocation
// fired after a quiet period of at least duration.
type Debouncer struct {
	mu       sync.Mutex
	duration time.Duration
	callback func()
	timer    *time.Timer
	running  bool
	pending  bool
}

// NewDebouncer creates a Debouncer that waits duration after the last Trigger
// call before invoking callback.
func NewDebouncer(duration time.Duration, callback func()) *Debouncer {
	return &Debouncer{
		duration: duration,
		callback: callback,
	}
}

// Trigger starts (or resets) the debounce timer. The callback will fire
// duration after the most recent Trigger call.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		d.pending = true
		return
	}
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.duration, d.fire)
}

// fire serializes callback execution. Events arriving during a long callback
// are coalesced into one follow-up run after the callback completes.
func (d *Debouncer) fire() {
	d.mu.Lock()
	if d.running {
		d.pending = true
		d.mu.Unlock()
		return
	}
	d.running = true
	d.timer = nil
	d.mu.Unlock()

	d.callback()

	d.mu.Lock()
	d.running = false
	if d.pending {
		d.pending = false
		d.timer = time.AfterFunc(d.duration, d.fire)
	}
	d.mu.Unlock()
}

// ReindexFunc is called by Run/Watch when file changes are detected.
// kbRoot is the knowledge-base root directory.
type ReindexFunc func(kbRoot string)

// Watch monitors the raw/ subdirectory under kbRoot for file-system events and
// calls reindexFn (debounced) whenever files are created, written, removed, or
// renamed. It blocks until the watcher encounters an unrecoverable error.
func Watch(kbRoot string, reindexFn ReindexFunc) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	rawDir := filepath.Join(kbRoot, "raw")
	wikiDir := filepath.Join(kbRoot, "wiki")
	indexDir := filepath.Join(kbRoot, "index")

	// Watch raw/, wiki/, and schema/; ignore non-existent dirs silently.
	for _, dir := range []string{rawDir, filepath.Join(kbRoot, "wiki"), filepath.Join(kbRoot, "schema")} {
		_ = addDirRecursive(w, dir)
	}

	debouncer := NewDebouncer(defaultDebounce, func() {
		reindexFn(kbRoot)
	})

	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) ||
				event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// If a new directory was created, watch it too.
				if event.Has(fsnotify.Create) {
					_ = addDirRecursive(w, event.Name)
				}
				// Ignore index/ (generated artifacts) and raw/converted/.
				if isUnderDir(event.Name, indexDir) ||
					isConvertedPath(event.Name, rawDir) ||
					isGeneratedWikiPath(event.Name, wikiDir) {
					continue
				}
				debouncer.Trigger()
			}

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return err
		}
	}
}

// Run performs a cold-start reindex and then enters the Watch loop.
func Run(kbRoot string, reindexFn ReindexFunc) error {
	reindexFn(kbRoot)
	return Watch(kbRoot, reindexFn)
}

// isUnderDir reports whether path is inside dir.
func isUnderDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && (len(rel) < 2 || rel[:2] != "..")
}

// isConvertedPath reports whether path is inside raw/converted/.
func isConvertedPath(path, rawDir string) bool {
	convertedDir := filepath.Join(rawDir, "converted")
	rel, err := filepath.Rel(convertedDir, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && len(rel) > 0 && rel[:2] != ".."
}

// isGeneratedWikiPath reports whether path is maintained by PieKBS itself.
// These writes are already indexed by the active pipeline and must not trigger
// another watcher cycle.
func isGeneratedWikiPath(path, wikiDir string) bool {
	rel, err := filepath.Rel(wikiDir, path)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	base := filepath.Base(rel)
	return base == "index.md" || rel == "log.md"
}

// addDirRecursive adds dir and every subdirectory beneath it to the watcher.
// Non-existent paths are silently skipped.
func addDirRecursive(w *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Directory may not exist yet; ignore.
			return nil
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}
