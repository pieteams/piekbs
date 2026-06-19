package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const launchdTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        {{range .Args}}<string>{{.}}</string>
        {{end}}
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>WIKILOOP_KB</key>
        <string>{{.KBRoot}}</string>
        <key>PATH</key>
        <string>{{.Path}}</string>
    </dict>
    <key>RunAtLoad</key>
    {{if .RunAtLoad}}<true/>{{else}}<false/>{{end}}
    <key>KeepAlive</key>
    {{if .KeepAlive}}<true/>{{else}}<false/>{{end}}
    {{if .WatchPath}}<key>WatchPaths</key>
    <array><string>{{.WatchPath}}</string></array>{{end}}
    <key>StandardOutPath</key>
    <string>/tmp/{{.Label}}.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/{{.Label}}.log</string>
</dict>
</plist>`

type launchdConfig struct {
	Label     string
	Args      []string
	KBRoot    string
	Path      string
	RunAtLoad bool
	KeepAlive bool
	WatchPath string
}

// macLabels are the launchd labels managed by WikiLoop.
var macLabels = []string{"com.wikiloop.mcp", "com.wikiloop.indexer"}

func installMacOS(kbRoot string) error {
	bin, err := findBinary()
	if err != nil {
		return err
	}

	agentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	// MCP server
	mcpCfg := launchdConfig{
		Label:     "com.wikiloop.mcp",
		Args:      []string{bin, "serve"},
		KBRoot:    kbRoot,
		Path:      servicePath(),
		RunAtLoad: true,
		KeepAlive: true,
	}
	if err := writePlist(agentsDir, mcpCfg); err != nil {
		return err
	}
	loadService(filepath.Join(agentsDir, "com.wikiloop.mcp.plist"))
	fmt.Println("  Installed and started: com.wikiloop.mcp")

	// Indexer
	idxCfg := launchdConfig{
		Label:     "com.wikiloop.indexer",
		Args:      []string{bin, "watch"},
		KBRoot:    kbRoot,
		Path:      servicePath(),
		RunAtLoad: false,
		KeepAlive: false,
		WatchPath: filepath.Join(kbRoot, "raw"),
	}
	if err := writePlist(agentsDir, idxCfg); err != nil {
		return err
	}
	loadService(filepath.Join(agentsDir, "com.wikiloop.indexer.plist"))
	fmt.Println("  Installed and started: com.wikiloop.indexer")

	fmt.Printf("\nMCP HTTP server: http://127.0.0.1:8766/mcp\n")
	fmt.Printf("Logs: /tmp/com.wikiloop.mcp.log, /tmp/com.wikiloop.indexer.log\n")
	return nil
}

func writePlist(dir string, cfg launchdConfig) error {
	path := filepath.Join(dir, cfg.Label+".plist")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	t := template.Must(template.New("plist").Parse(launchdTemplate))
	return t.Execute(f, cfg)
}

func loadService(path string) {
	exec.Command("launchctl", "unload", "-w", path).Run()
	if err := exec.Command("launchctl", "load", "-w", path).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: launchctl load failed for %s: %v\n", path, err)
	}
}

func uninstallMacOS() error {
	agentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	for _, label := range macLabels {
		path := filepath.Join(agentsDir, label+".plist")
		exec.Command("launchctl", "unload", "-w", path).Run()
		os.Remove(path)
		fmt.Printf("  Uninstalled: %s\n", label)
	}
	return nil
}

func statusMacOS() error {
	for _, label := range macLabels {
		out, err := exec.Command("launchctl", "list", label).Output()
		if err != nil {
			fmt.Printf("  %s: not loaded\n", label)
		} else {
			fmt.Printf("  %s:\n%s\n", label, string(out))
		}
	}
	return nil
}

// startMacOS loads each plist (-w persists across logins).
func startMacOS() error {
	agentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	for _, label := range macLabels {
		path := filepath.Join(agentsDir, label+".plist")
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("  %s: not installed (run 'service install' first)\n", label)
			continue
		}
		if err := exec.Command("launchctl", "load", "-w", path).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: start %s failed: %v\n", label, err)
		} else {
			fmt.Printf("  Started: %s\n", label)
		}
	}
	return nil
}

// stopMacOS unloads each plist (-w prevents auto-restart at login).
func stopMacOS() error {
	agentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	for _, label := range macLabels {
		path := filepath.Join(agentsDir, label+".plist")
		if err := exec.Command("launchctl", "unload", "-w", path).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: stop %s failed: %v\n", label, err)
		} else {
			fmt.Printf("  Stopped: %s\n", label)
		}
	}
	return nil
}

func restartMacOS() error {
	if err := stopMacOS(); err != nil {
		return err
	}
	return startMacOS()
}

// logsMacOS prints the tail of each service's log file.
func logsMacOS() error {
	for _, label := range macLabels {
		logPath := filepath.Join("/tmp", label+".log")
		fmt.Printf("── %s (%s) ──\n", label, logPath)
		out, err := exec.Command("tail", "-n", "40", logPath).Output()
		if err != nil {
			fmt.Printf("  (no log yet)\n")
		} else {
			fmt.Print(string(out))
		}
		fmt.Println()
	}
	return nil
}
