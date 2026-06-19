//go:build !darwin

package tray

type Action int

const (
	ActionOpenUI Action = iota
	ActionOpenKBDir
	ActionSettings
	ActionQuit
)

// Run is a no-op on non-Darwin platforms (no system tray support).
func Run(_ string, _ int, _ chan<- Action) {}
