//go:build fts5

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/jasen215/wikiloop/internal/config"
	"github.com/jasen215/wikiloop/internal/convert"
	"github.com/jasen215/wikiloop/internal/distill"
	"github.com/jasen215/wikiloop/internal/kb"
	"github.com/jasen215/wikiloop/internal/kbinit"
	"github.com/jasen215/wikiloop/internal/larkimport"
	"github.com/jasen215/wikiloop/internal/mcp"
	"github.com/jasen215/wikiloop/internal/service"
	"github.com/jasen215/wikiloop/internal/synthesize"
	"github.com/jasen215/wikiloop/internal/tray"
	"github.com/jasen215/wikiloop/internal/watcher"
	"github.com/jasen215/wikiloop/internal/webui"
)

func main() {
	// Mirror startup logs to a file so GUI double-click launches (where stderr
	// is invisible) can be diagnosed; terminal launches still see stderr.
	if f, err := os.OpenFile(startupLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Printf("=== WikiLoop starting (pid %d) ===", os.Getpid())
	}
	if err := run(); err != nil {
		log.Printf("fatal: %v", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// startupLogPath returns the standard macOS app log location
// (~/Library/Logs/WikiLoop.log), falling back to /tmp if home is unavailable.
func startupLogPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(home, "Library", "Logs")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return filepath.Join(dir, "WikiLoop.log")
		}
	}
	return filepath.Join(os.TempDir(), "wikiloop-startup.log")
}

func run() error {
	// For the serve subcommand (including the default when no args given),
	// load WIKILOOP_* from shell rc files before flag defaults are evaluated.
	// This makes macOS GUI double-click behave identically to CLI invocation.
	// Must happen before flag.String() so envOr("WIKILOOP_KB",...) sees the value.
	if isServeInvocation() {
		loadShellEnv()
	}

	// Global flags — must be parsed before subcommand.
	kbRoot := flag.String("kb", envOr("WIKILOOP_KB", defaultKBRoot()), "knowledge-base root directory")
	flag.Parse()

	args := flag.Args()
	sub := "serve"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "serve":
		return runServe(*kbRoot)
	case "status":
		return runStatus(*kbRoot)
	case "search":
		return runSearch(*kbRoot, args[1:])
	case "context":
		return runContext(*kbRoot, args[1:])
	case "index":
		return runIndex(*kbRoot)
	case "lint":
		return runLint(*kbRoot, args[1:])
	case "service":
		return runService(*kbRoot, args[1:])
	case "init":
		return runInit(*kbRoot, args[1:])
	case "synthesize":
		return runSynthesize(*kbRoot, args[1:])
	case "import-lark":
		return runImportLark(*kbRoot, args[1:])
	case "stdio":
		return runStdio(*kbRoot)
	default:
		return fmt.Errorf("unknown subcommand: %s", sub)
	}
}

func runImportLark(kbRoot string, args []string) error {
	fs := flag.NewFlagSet("import-lark", flag.ContinueOnError)
	name := fs.String("name", "", "optional stable directory name for the imported document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: wikiloop import-lark [--name slug] <Lark Wiki URL>")
	}

	result, err := larkimport.Import(context.Background(), kbRoot, fs.Arg(0), *name, nil)
	if err != nil {
		return fmt.Errorf("import Lark document: %w", err)
	}
	fmt.Printf("imported document: %s\n", result.DocumentPath)
	for i, path := range result.TablePaths {
		fmt.Printf("saved snapshot:   %s (%d rows)\n", path, result.TableRows[i])
	}
	if result.DatasetPath != "" {
		fmt.Printf("indexed dataset:  %s (%d unique, %d duplicates removed)\n",
			result.DatasetPath, result.UniqueRows, result.DuplicatesRemoved)
	}
	return nil
}

const (
	runningFile       = "index/.running"
	lastShutdownFile  = "index/.last_shutdown"
	heartbeatInterval = time.Hour
)

// writeTimestamp writes the current Unix timestamp (seconds) to path.
func writeTimestamp(path string) {
	_ = os.WriteFile(path, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0o644)
}

// readTimestamp reads a Unix timestamp from path. Returns zero time on error.
func readTimestamp(path string) time.Time {
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	var ts int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &ts); err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// lastKnownTime returns the best timestamp to use for incremental scan:
// prefers .last_shutdown (normal exit), falls back to .running (crash recovery).
func lastKnownTime(kbRoot string) time.Time {
	if t := readTimestamp(filepath.Join(kbRoot, lastShutdownFile)); !t.IsZero() {
		return t
	}
	return readTimestamp(filepath.Join(kbRoot, runningFile))
}

// ── serve ──────────────────────────────────────────────────────────────────────

// preflightCheck inspects the runtime environment for common misconfigurations
// (KB path writability, ONNX/sqlite-vec presence, port availability) and logs
// actionable warnings. Non-fatal issues (missing ONNX/sqlite-vec → FTS-only)
// let the app continue; a fatal issue (port already in use) returns an error
// so the caller aborts startup. On GUI launch, all messages are also shown as
// an HTML page in the browser since stderr/log is invisible there.
func preflightCheck(kbRoot string, cfg *config.Config) error {
	var warns []string
	var fatal string

	// 1. WIKILOOP_KB / KB path: must be an absolute, writable directory.
	if v, ok := os.LookupEnv("WIKILOOP_KB"); ok && strings.HasPrefix(v, "~") {
		warns = append(warns, fmt.Sprintf(
			"WIKILOOP_KB=%q contains an unexpanded '~'. Use an absolute path or $HOME.", v))
	}
	if !filepath.IsAbs(kbRoot) {
		warns = append(warns, fmt.Sprintf(
			"KB path %q is relative; resolved against the launch directory, which differs between CLI and GUI. Prefer an absolute path.", kbRoot))
	}
	if probe := filepath.Join(kbRoot, ".wikiloop-write-test"); true {
		if err := os.MkdirAll(kbRoot, 0o755); err != nil {
			warns = append(warns, fmt.Sprintf("KB path %q is not writable: %v", kbRoot, err))
		} else if f, err := os.Create(probe); err != nil {
			warns = append(warns, fmt.Sprintf("KB path %q is not writable: %v", kbRoot, err))
		} else {
			f.Close()
			_ = os.Remove(probe)
		}
	}

	// 2. Port availability — probe by dialing; a successful connection means
	// something is already listening. Read-only, does not claim the port.
	// This is fatal: HTTP MCP + Web UI cannot start on a taken port, so the
	// app would be useless. Abort instead of leaving a half-dead menubar.
	probeAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Server.Port)
	if conn, err := net.DialTimeout("tcp", probeAddr, 300*time.Millisecond); err == nil {
		_ = conn.Close()
		fatal = fmt.Sprintf(
			"Port %d is already in use — another WikiLoop instance may be running, or set server.port in config.yaml.\n"+
				"      Find the process: lsof -i :%d", cfg.Server.Port, cfg.Server.Port)
	}

	for _, w := range warns {
		log.Printf("preflight: %s", w)
	}
	if fatal != "" {
		log.Printf("preflight FATAL: %s", fatal)
	}

	// GUI launches can't see stderr/log; render messages to an HTML page and
	// open it in the default browser. Skip only in an interactive terminal
	// (TERM set), where the user can already see the log/stderr.
	if isGUI := runtime.GOOS == "darwin" && os.Getenv("TERM") == ""; isGUI {
		all := warns
		if fatal != "" {
			all = append([]string{fatal}, warns...)
		}
		if len(all) > 0 {
			showWarningPage(all)
		}
	}

	if fatal != "" {
		return fmt.Errorf("%s", strings.SplitN(fatal, "\n", 2)[0])
	}
	return nil
}

// showWarningPage writes warns to an HTML file and opens it in the default
// browser. Best-effort: failures are ignored (the log has the full detail).
func showWarningPage(warns []string) {
	var items strings.Builder
	for _, w := range warns {
		items.WriteString("<li>" + html.EscapeString(w) + "</li>\n")
	}
	html := `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<title>WikiLoop Startup Check</title>
<style>
body{font-family:-apple-system,sans-serif;max-width:680px;margin:60px auto;padding:0 24px;color:#1d1d1f;line-height:1.6}
h1{font-size:22px}
.warn{background:#fff8e1;border-left:4px solid #f5a623;border-radius:6px;padding:16px 20px;margin:20px 0}
ul{padding-left:20px}
li{margin:12px 0;white-space:pre-wrap}
code{background:#f0f0f2;padding:2px 6px;border-radius:4px;font-size:13px}
.foot{color:#86868b;font-size:13px;margin-top:32px}
</style></head><body>
<h1>⚠️ WikiLoop Startup Check</h1>
<p>The following issues were detected (port conflict prevents startup; missing ONNX or model files disables vector search, FTS still works):</p>
<div class="warn"><ul>
` + items.String() + `</ul></div>
<p class="foot">Full log: <code>~/Library/Logs/WikiLoop.log</code></p>
</body></html>`

	tmp := filepath.Join(os.TempDir(), "wikiloop-preflight.html")
	if err := os.WriteFile(tmp, []byte(html), 0o644); err != nil {
		return
	}
	// Run synchronously: the `open` command hands the file to the browser and
	// returns immediately. A goroutine would be killed if the caller exits
	// right after (the fatal path does os.Exit before the goroutine runs).
	_ = exec.Command("open", tmp).Run()
}

// runStdio starts WikiLoop in stdio MCP mode for hosted agent environments
// (Hermes, OpenClaw, etc.). It starts the watcher and catch-up scan in
// background goroutines, then serves MCP over stdin/stdout. No HTTP server
// or system tray is started, so multiple instances can coexist without port
// conflicts.
func runStdio(kbRoot string) error {
	if err := ensureKBDirs(kbRoot); err != nil {
		return fmt.Errorf("ensure KB dirs: %w", err)
	}

	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	since := lastKnownTime(kbRoot)
	writeTimestamp(filepath.Join(kbRoot, runningFile))
	_ = os.Remove(filepath.Join(kbRoot, lastShutdownFile))

	go func() {
		t := time.NewTicker(heartbeatInterval)
		defer t.Stop()
		for range t.C {
			writeTimestamp(filepath.Join(kbRoot, runningFile))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		writeTimestamp(filepath.Join(kbRoot, lastShutdownFile))
		_ = os.Remove(filepath.Join(kbRoot, runningFile))
		os.Exit(0)
	}()

	if !since.IsZero() {
		go func() { catchUpFn(kbRoot, since, cfg) }()
	}

	go func() {
		if err := watcher.Watch(kbRoot, reindexFn); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()

	// Recover stale processing tasks from previous crash.
	if db, err := kb.OpenDB(kbRoot); err == nil {
		_ = distill.RecoverStale(db)
		db.Close()
	}

	// Start distill worker pool.
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	if cfg.Distill.IsConfigured() {
		distillCfg := distill.Config{
			BaseURL: cfg.Distill.BaseURL,
			Token:   cfg.Distill.Token,
			Model:   cfg.Distill.Model,
			APIType: cfg.Distill.APIType,
		}
		n := cfg.Distill.Workers
		if n <= 0 {
			n = 3
		}
		go distill.RunWorkers(workerCtx, distillCfg, kbRoot, n)
		log.Printf("distill: started %d worker(s)", n)
	}

	log.Printf("WikiLoop stdio MCP starting (kb: %s)", kbRoot)
	return mcp.ServeStdio(kbRoot, cfg.Server.APIKey)
}

func runServe(kbRoot string) error {
	if err := ensureKBDirs(kbRoot); err != nil {
		return fmt.Errorf("ensure KB dirs: %w", err)
	}

	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Pre-flight environment check: logs warnings (and on GUI launch opens an
	// HTML report). A fatal issue (e.g. port in use) aborts startup.
	if err := preflightCheck(kbRoot, cfg); err != nil {
		return err
	}

	// Determine if we need a catch-up scan (new files added while down).
	since := lastKnownTime(kbRoot)

	// Write .running marker immediately.
	writeTimestamp(filepath.Join(kbRoot, runningFile))
	// Remove stale .last_shutdown so next crash uses .running.
	_ = os.Remove(filepath.Join(kbRoot, lastShutdownFile))

	// Heartbeat: update .running timestamp every hour.
	go func() {
		t := time.NewTicker(heartbeatInterval)
		defer t.Stop()
		for range t.C {
			writeTimestamp(filepath.Join(kbRoot, runningFile))
		}
	}()

	// Graceful shutdown: write .last_shutdown and remove .running.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		writeTimestamp(filepath.Join(kbRoot, lastShutdownFile))
		_ = os.Remove(filepath.Join(kbRoot, runningFile))
		os.Exit(0)
	}()

	// Catch-up: index + distill only files newer than last known time.
	if !since.IsZero() {
		go func() {
			catchUpFn(kbRoot, since, cfg)
		}()
	}

	// File watcher in background.
	go func() {
		if err := watcher.Watch(kbRoot, reindexFn); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()

	// Recover stale processing tasks from previous crash.
	if db, err := kb.OpenDB(kbRoot); err == nil {
		_ = distill.RecoverStale(db)
		db.Close()
	}

	// Start distill worker pool.
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	if cfg.Distill.IsConfigured() {
		distillCfg := distill.Config{
			BaseURL: cfg.Distill.BaseURL,
			Token:   cfg.Distill.Token,
			Model:   cfg.Distill.Model,
			APIType: cfg.Distill.APIType,
		}
		n := cfg.Distill.Workers
		if n <= 0 {
			n = 3
		}
		go distill.RunWorkers(workerCtx, distillCfg, kbRoot, n)
		log.Printf("distill: started %d worker(s)", n)
	}

	// Combine MCP and Web UI on the same mux.
	mux := http.NewServeMux()
	mcp.RegisterRoutes(mux, kbRoot, cfg.Server.APIKey)

	webSrv := webui.NewServer(kbRoot)
	mux.Handle("/", webSrv.Handler())

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// HTTP server in background goroutine.
	go func() {
		log.Printf("WikiLoop listening on %s (kb: %s)", addr, kbRoot)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("http: %v", err)
		}
	}()

	// System tray on main goroutine (required by macOS).
	// Fall back to headless mode when no display is available (e.g. Linux server).
	if hasDisplay() {
		actionCh := make(chan tray.Action, 1)
		go func() {
			for action := range actionCh {
				if action == tray.ActionQuit {
					writeTimestamp(filepath.Join(kbRoot, lastShutdownFile))
					_ = os.Remove(filepath.Join(kbRoot, runningFile))
					os.Exit(0)
				}
			}
		}()
		tray.Run(kbRoot, cfg.Server.Port, cfg.UI.Language, actionCh)
	} else {
		log.Printf("no display detected — running headless (Ctrl-C to stop)")
		select {} // block forever; SIGTERM handler above handles shutdown
	}
	return nil
}

// hasDisplay reports whether a graphical display environment is available.
func hasDisplay() bool {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	// On macOS the OS always provides a window server.
	return runtime.GOOS == "darwin"
}

// catchUpFn indexes and distills files in raw/ newer than since.
func catchUpFn(kbRoot string, since time.Time, cfg *config.Config) {
	rawDir := filepath.Join(kbRoot, "raw")
	var newFiles []string
	_ = filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().After(since) {
			newFiles = append(newFiles, path)
		}
		return nil
	})

	if len(newFiles) == 0 {
		return
	}
	log.Printf("catch-up: %d new file(s) since %s", len(newFiles), since.Format(time.RFC3339))

	// Convert any non-md files first.
	if n, _ := convert.Run(kbRoot); n > 0 {
		log.Printf("catch-up convert: %d file(s) converted", n)
	}

	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		log.Printf("catch-up: open db: %v", err)
		return
	}
	defer db.Close()

	n, err := kb.IndexFiles(db, kbRoot)
	if err != nil {
		log.Printf("catch-up: reindex: %v", err)
		return
	}
	log.Printf("catch-up: %d files indexed", n)

	if cfg.Distill.BaseURL != "" && cfg.Distill.Token != "" && cfg.Distill.Model != "" {
		db2, err := kb.OpenDB(kbRoot)
		if err == nil {
			if n, err := distill.Enqueue(db2, kbRoot); err != nil {
				log.Printf("catch-up enqueue: %v", err)
			} else if n > 0 {
				log.Printf("catch-up enqueue: %d file(s) queued for distillation", n)
			}
			db2.Close()
		}
	}
}

// reindexFn is the callback used by the file watcher.
// Order: convert → index → distill → post-distill reindex.
func reindexFn(kbRoot string) {
	// 1. Convert non-md files (docx, pdf, etc.) to markdown.
	if n, _ := convert.Run(kbRoot); n > 0 {
		log.Printf("watcher convert: %d file(s) converted", n)
	}

	// 2. Index all text files into FTS.
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		log.Printf("watcher reindex: open db: %v", err)
		return
	}
	defer db.Close()

	n, err := kb.IndexFiles(db, kbRoot)
	if err != nil {
		log.Printf("watcher reindex: %v", err)
	} else {
		log.Printf("watcher reindex: %d files updated", n)
	}

	// Load config; use a zero-value config on failure.
	cfg, err := config.Load(kbRoot)
	if err != nil {
		log.Printf("watcher: load config: %v", err)
		cfg = &config.Config{}
	}
	// 3. Enqueue new raw files for distillation (workers process asynchronously).
	if cfg.Distill.IsConfigured() {
		db2, err2 := kb.OpenDB(kbRoot)
		if err2 == nil {
			if n, err := distill.Enqueue(db2, kbRoot); err != nil {
				log.Printf("watcher enqueue: %v", err)
			} else if n > 0 {
				log.Printf("watcher enqueue: %d file(s) queued", n)
			}
			db2.Close()
		}
	}
}

// ── status ─────────────────────────────────────────────────────────────────────

func runStatus(kbRoot string) error {
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT layer, COUNT(*) FROM documents GROUP BY layer")
	if err != nil {
		return fmt.Errorf("query layers: %w", err)
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var layer string
		var count int
		if err := rows.Scan(&layer, &count); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}
		fmt.Printf("  %-8s %d\n", layer, count)
		total += count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}
	fmt.Printf("  %-8s %d\n", "total", total)
	return nil
}

// ── search ─────────────────────────────────────────────────────────────────────

func runSearch(kbRoot string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: wikiloop search <query>")
	}
	query := strings.Join(args, " ")

	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	results, err := kb.FTSSearch(db, query, nil, 10)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("no results")
		return nil
	}

	for i, r := range results {
		fmt.Printf("[%d] [%s] %s\n", i+1, r.Layer, r.Title)
		fmt.Printf("    path: %s\n", r.Path)
		if r.Snippet != "" {
			fmt.Printf("    %s\n", r.Snippet)
		}
		fmt.Println()
	}
	return nil
}

// ── context ────────────────────────────────────────────────────────────────────

func runContext(kbRoot string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: wikiloop context <question>")
	}
	question := strings.Join(args, " ")

	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	bundle := kb.BuildContext(db, kbRoot, question, nil, 5) //nolint — CLI path uses FTS-only (no embedder)

	fmt.Printf("Question: %s\n\n", bundle.Question)

	if len(bundle.WikiPages) == 0 {
		fmt.Println("No wiki pages found.")
		return nil
	}

	fmt.Printf("Wiki pages (%d):\n", len(bundle.WikiPages))
	for _, wp := range bundle.WikiPages {
		fmt.Printf("  - [%s] %s\n", wp.Layer, wp.Title)
		if wp.Description != "" {
			fmt.Printf("    %s\n", wp.Description)
		}
	}

	if len(bundle.RawSources) > 0 {
		fmt.Printf("\nRaw sources (%d):\n", len(bundle.RawSources))
		for _, rs := range bundle.RawSources {
			fmt.Printf("  - %s\n", rs.Title)
		}
	}

	return nil
}

// ── index ──────────────────────────────────────────────────────────────────────

func runSynthesize(kbRoot string, args []string) error {
	var topic string
	var full bool
	var gaps bool
	var incrementalAll bool
	for i, a := range args {
		switch {
		case a == "--topic" && i+1 < len(args):
			topic = args[i+1]
		case strings.HasPrefix(a, "--topic="):
			topic = strings.TrimPrefix(a, "--topic=")
		case a == "--full":
			full = true
		case a == "--gaps":
			gaps = true
		case a == "--incremental-all":
			incrementalAll = true
		}
	}

	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	synCfg := synthesize.Config{
		BaseURL: cfg.Distill.BaseURL,
		Token:   cfg.Distill.Token,
		Model:   cfg.Distill.Model,
		APIType: cfg.Distill.APIType,
	}

	if gaps {
		return runSynthesizeGaps(kbRoot, synCfg, topic)
	}

	// --incremental-all: process each source-note individually via RunIncremental.
	// This avoids the context-window limit of bulk Plan() and ensures every
	// note triggers its own tag-based search + AppendOrCreate cycle.
	if incrementalAll {
		// Pre-load all notes once to avoid repeated disk scans (O(N²) → O(N)).
		allNotes, err := synthesize.LoadSourceNotes(kbRoot)
		if err != nil {
			return fmt.Errorf("load source notes: %w", err)
		}
		total := len(allNotes)
		fmt.Printf("synthesize --incremental-all: processing %d source-notes\n", total)
		for i, note := range allNotes {
			fmt.Printf("[%d/%d] %s\n", i+1, total, note.Title)
			if err := synthesize.RunIncrementalWithNotes(synCfg, kbRoot, note.Path, allNotes); err != nil {
				if strings.Contains(err.Error(), "429") {
					fmt.Printf("  rate limited, waiting 30s...\n")
					time.Sleep(30 * time.Second)
					if err2 := synthesize.RunIncrementalWithNotes(synCfg, kbRoot, note.Path, allNotes); err2 != nil {
						fmt.Printf("  warning: %v\n", err2)
					}
				} else {
					fmt.Printf("  warning: %v\n", err)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println("synthesize --incremental-all: done")
		return nil
	}

	if topic != "" {
		fmt.Printf("synthesize: topic=%q full=%v\n", topic, full)
	}
	n, err := synthesize.Run(kbRoot, synCfg, topic, full)
	if err != nil {
		return err
	}
	if n == 0 {
		fmt.Println("synthesize: nothing new to generate")
	} else {
		fmt.Printf("synthesize: %d page(s) generated\n", n)
	}
	return nil
}

func runSynthesizeGaps(kbRoot string, cfg synthesize.Config, topic string) error {
	if topic == "" {
		return fmt.Errorf("--gaps requires --topic, e.g.: wikiloop synthesize --gaps --topic \"chip industry\"")
	}
	fmt.Printf("synthesize --gaps: analyzing topic %q\n", topic)
	if err := synthesize.RunGaps(kbRoot, cfg, topic); err != nil {
		return err
	}
	slug := synthesize.TopicSlug(topic)
	fmt.Printf("synthesize --gaps: report written to index/gaps/%s.md\n", slug)
	return nil
}

func runInit(kbRoot string, args []string) error {
	force := len(args) > 0 && args[0] == "--force"
	if err := kbinit.Init(kbRoot, force); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	fmt.Printf("Initialized KB at %s\n", kbRoot)
	fmt.Println("  raw/        — place source documents here")
	fmt.Println("  wiki/       — LLM-maintained knowledge layer")
	fmt.Println("  schema/     — authoring rules and templates")
	return nil
}

func runIndex(kbRoot string) error {
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	n, err := kb.IndexFiles(db, kbRoot)
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}
	fmt.Printf("indexed %d files\n", n)
	return nil
}

// ── lint ───────────────────────────────────────────────────────────────────────

func runLint(kbRoot string, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "exit with code 1 if any warnings are found")
	if err := fs.Parse(args); err != nil {
		return err
	}

	warnings, err := kb.Lint(kbRoot)
	if err != nil {
		return fmt.Errorf("lint: %w", err)
	}

	if len(warnings) == 0 {
		fmt.Println("lint: ok")
		return nil
	}

	fmt.Printf("lint: %d warning(s)\n", len(warnings))
	for _, w := range warnings {
		switch w.Kind {
		case "missing_field":
			fmt.Printf("  missing '%s': %s\n", w.Detail, w.Path)
		case "broken_source":
			fmt.Printf("  broken source link '%s': %s\n", w.Detail, w.Path)
		default:
			fmt.Printf("  %s (%s): %s\n", w.Kind, w.Detail, w.Path)
		}
	}

	if *strict {
		return fmt.Errorf("lint found %d warning(s)", len(warnings))
	}
	return nil
}

// ── service ────────────────────────────────────────────────────────────────────

func runService(kbRoot string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: wikiloop service <install|uninstall|start|stop|status|logs>")
	}
	action := service.Action(args[0])
	return service.Run(action, kbRoot)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// ensureKBDirs creates the standard KB directory structure and initializes
// schema/ with bundled authoring rules if it is empty (first run).
func ensureKBDirs(kbRoot string) error {
	schemaDir := filepath.Join(kbRoot, "schema")
	isNew := false
	if _, err := os.Stat(schemaDir); os.IsNotExist(err) {
		isNew = true
	} else {
		// schema/ exists but may be empty
		entries, _ := os.ReadDir(schemaDir)
		isNew = len(entries) == 0
	}

	if err := kbinit.Init(kbRoot, false); err != nil {
		return fmt.Errorf("init KB: %w", err)
	}

	if isNew {
		log.Printf("KB initialized at %s", kbRoot)
	}
	return nil
}

// envOr returns the environment variable value or the fallback.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// isServeInvocation returns true when the process was invoked as "serve"
// (explicitly or implicitly — serve is the default subcommand).
func isServeInvocation() bool {
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			return arg == "serve"
		}
	}
	return true // no non-flag args → default subcommand is serve
}

func defaultKBRoot() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "wikiloop-kb")
	}
	return "./wikiloop-kb"
}

// expandHome expands a leading ~ or $HOME in a path read from a shell rc file.
// Shell rc files store these unexpanded; we read the literal string with no
// shell, so without this a value like ~/.hermes/wikiloop-kb stays verbatim.
func expandHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	case strings.HasPrefix(p, "$HOME/"):
		return filepath.Join(home, p[len("$HOME/"):])
	case p == "$HOME":
		return home
	}
	return p
}

// loadShellEnv reads WIKILOOP_* variables from shell rc files and sets them
// in the process environment if not already set. This allows macOS GUI apps
// (which don't inherit shell env) to behave identically to CLI invocations.
// Sources tried in order: ~/.zshenv, ~/.zshrc, ~/.bash_profile, ~/.bashrc, ~/.profile.
// NOTE: must be called before flag.String() reads WIKILOOP_KB via envOr.
func loadShellEnv() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, name := range []string{".zshenv", ".zshrc", ".bash_profile", ".bashrc", ".profile"} {
		parseEnvFile(filepath.Join(home, name))
	}
}

// parseEnvFile scans a shell rc file for lines of the form:
//
//	export WIKILOOP_FOO=bar
//	export WIKILOOP_FOO="bar"
//	WIKILOOP_FOO=bar
//
// and sets them in the process environment if not already present.
// Inline comments (# ...) and surrounding quotes are stripped.
func parseEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Accept both "export KEY=val" and bare "KEY=val".
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || !strings.HasPrefix(k, "WIKILOOP_") {
			continue
		}
		// Strip inline comments (unquoted only).
		if len(v) == 0 || (v[0] != '"' && v[0] != '\'') {
			if idx := strings.Index(v, " #"); idx >= 0 {
				v = strings.TrimSpace(v[:idx])
			}
		}
		// Strip surrounding quotes.
		if len(v) >= 2 && v[0] == v[len(v)-1] && (v[0] == '"' || v[0] == '\'') {
			v = v[1 : len(v)-1]
		}
		// Expand leading ~ and $HOME — shell rc files use these unexpanded,
		// but we read the literal string (no shell). Without this, a value
		// like ~/.hermes/wikiloop-kb would be used verbatim and mkdir fails.
		v = expandHome(v)
		if _, set := os.LookupEnv(k); !set {
			_ = os.Setenv(k, v)
		}
	}
}
