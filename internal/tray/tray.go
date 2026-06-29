//go:build darwin

package tray

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

// Action represents a tray menu action.
type Action int

const (
	ActionOpenUI Action = iota
	ActionOpenKBDir
	ActionSettings
	ActionQuit
)

type trayLabels struct {
	OpenDashboard string
	OpenKBDir     string
	Settings      string
	Quit          string
}

func labelsFor(lang string) trayLabels {
	if lang == "zh" {
		return trayLabels{
			OpenDashboard: "打开控制台",
			OpenKBDir:     "打开知识库目录",
			Settings:      "设置",
			Quit:          "退出",
		}
	}
	return trayLabels{
		OpenDashboard: "Open Dashboard",
		OpenKBDir:     "Open KB Directory",
		Settings:      "Settings",
		Quit:          "Quit",
	}
}

// Run starts the system tray icon and menu. Actions are sent to the action channel.
func Run(kbRoot string, port int, lang string, actionCh chan<- Action) {
	labels := labelsFor(lang)
	systray.Run(func() {
		systray.SetTemplateIcon(iconPNG, iconPNG)
		systray.SetTooltip("PieKBS Knowledge Base")

		mOpenUI := systray.AddMenuItem(labels.OpenDashboard, "")
		mOpenKB := systray.AddMenuItem(labels.OpenKBDir, "")
		systray.AddSeparator()
		mSettings := systray.AddMenuItem(labels.Settings, "")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem(labels.Quit, "")

		go func() {
			for {
				select {
				case <-mOpenUI.ClickedCh:
					url := fmt.Sprintf("http://localhost:%d", port)
					openBrowser(url)
					actionCh <- ActionOpenUI
				case <-mOpenKB.ClickedCh:
					openFileManager(kbRoot)
					actionCh <- ActionOpenKBDir
				case <-mSettings.ClickedCh:
					url := fmt.Sprintf("http://localhost:%d/settings", port)
					openBrowser(url)
					actionCh <- ActionSettings
				case <-mQuit.ClickedCh:
					systray.Quit()
					// Non-blocking send: receiver may not be ready.
					select {
					case actionCh <- ActionQuit:
					default:
					}
					return
				}
			}
		}()
	}, func() {
		// cleanup
	})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	}
	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Printf("tray: failed to open browser: %v", err)
		}
	}
}

func openFileManager(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	}
	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Printf("tray: failed to open file manager: %v", err)
		}
	}
}
