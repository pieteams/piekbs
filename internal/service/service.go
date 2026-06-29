package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Action is a service management action.
type Action string

const (
	ActionInstall   Action = "install"
	ActionUninstall Action = "uninstall"
	ActionStart     Action = "start"
	ActionStop      Action = "stop"
	ActionRestart   Action = "restart"
	ActionStatus    Action = "status"
	ActionLogs      Action = "logs"
)

// Run executes a service management action.
func Run(action Action, kbRoot string) error {
	switch runtime.GOOS {
	case "darwin":
		return runMacOS(action, kbRoot)
	case "linux":
		return runLinux(action, kbRoot)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func runMacOS(action Action, kbRoot string) error {
	switch action {
	case ActionInstall:
		return installMacOS(kbRoot)
	case ActionUninstall:
		return uninstallMacOS()
	case ActionStart:
		return startMacOS()
	case ActionStop:
		return stopMacOS()
	case ActionRestart:
		return restartMacOS()
	case ActionStatus:
		return statusMacOS()
	case ActionLogs:
		return logsMacOS()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func runLinux(action Action, kbRoot string) error {
	switch action {
	case ActionInstall:
		return installLinux(kbRoot)
	case ActionUninstall:
		return uninstallLinux()
	case ActionStart:
		return startLinux()
	case ActionStop:
		return stopLinux()
	case ActionRestart:
		return restartLinux()
	case ActionStatus:
		return statusLinux()
	case ActionLogs:
		return logsLinux()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func findBinary() (string, error) {
	path, err := exec.LookPath("piekbs")
	if err != nil {
		// Try the current executable
		path, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("piekbs binary not found")
		}
	}
	return filepath.Abs(path)
}

// servicePath returns a PATH for the installed daemon. launchd/systemd start
// with a minimal PATH, so external converters (markitdown, pandoc) installed in
// Homebrew or user dirs would not be found. We propagate the current PATH and
// append common install locations as a fallback.
func servicePath() string {
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		add(d)
	}
	for _, d := range []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		add(d)
	}
	if home := os.Getenv("HOME"); home != "" {
		add(filepath.Join(home, ".local", "bin"))
	}
	return strings.Join(dirs, string(os.PathListSeparator))
}
