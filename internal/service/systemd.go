package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installLinux(kbRoot string) error {
	bin, err := findBinary()
	if err != nil {
		return err
	}

	unitDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	units := []struct {
		name        string
		description string
		execStart   string
	}{
		{"wikiloop-mcp", "WikiLoop MCP HTTP server", bin + " serve"},
		{"wikiloop-indexer", "WikiLoop KB file watcher", bin + " watch"},
	}

	for _, u := range units {
		content := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
Environment=WIKILOOP_KB=%s
Environment=PATH=%s
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, u.description, kbRoot, servicePath(), u.execStart)

		path := filepath.Join(unitDir, u.name+".service")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write unit file %s: %w", path, err)
		}
		if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
			return fmt.Errorf("systemctl daemon-reload: %w", err)
		}
		if err := exec.Command("systemctl", "--user", "enable", "--now", u.name).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: enable %s failed: %v\n", u.name, err)
		}
		fmt.Printf("  Installed and started: %s\n", u.name)
	}

	fmt.Printf("\nMCP HTTP server: http://127.0.0.1:8766/mcp\n")
	return nil
}

// linuxUnits are the systemd user units managed by WikiLoop.
var linuxUnits = []string{"wikiloop-mcp", "wikiloop-indexer"}

func uninstallLinux() error {
	unitDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	for _, name := range linuxUnits {
		exec.Command("systemctl", "--user", "disable", "--now", name).Run()
		os.Remove(filepath.Join(unitDir, name+".service"))
		fmt.Printf("  Uninstalled: %s\n", name)
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

// systemctlEach runs `systemctl --user <verb> <unit>` for each managed unit.
func systemctlEach(verb string) error {
	for _, name := range linuxUnits {
		if err := exec.Command("systemctl", "--user", verb, name).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s %s failed: %v\n", verb, name, err)
		} else {
			fmt.Printf("  %s: %s\n", verb, name)
		}
	}
	return nil
}

func startLinux() error   { return systemctlEach("start") }
func stopLinux() error    { return systemctlEach("stop") }
func restartLinux() error { return systemctlEach("restart") }

func statusLinux() error {
	for _, name := range linuxUnits {
		out, _ := exec.Command("systemctl", "--user", "is-active", name).Output()
		state := strings.TrimSpace(string(out))
		if state == "" {
			state = "unknown"
		}
		fmt.Printf("  %s: %s\n", name, state)
	}
	return nil
}

func logsLinux() error {
	for _, name := range linuxUnits {
		fmt.Printf("── %s ──\n", name)
		out, err := exec.Command("journalctl", "--user", "-u", name, "-n", "40", "--no-pager").Output()
		if err != nil {
			fmt.Printf("  (no logs available: %v)\n", err)
		} else {
			fmt.Print(string(out))
		}
		fmt.Println()
	}
	return nil
}
