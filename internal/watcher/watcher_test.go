package watcher

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestDebounce verifies that multiple rapid Trigger calls result in exactly
// one callback invocation — the debouncer should coalesce all calls within
// the window into a single fire.
func TestDebounce(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(80*time.Millisecond, func() {
		count.Add(1)
	})

	// Fire 3 times rapidly (well within debounce window)
	d.Trigger()
	d.Trigger()
	d.Trigger()

	// Wait longer than debounce duration for the callback to fire
	time.Sleep(200 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected callback to fire exactly 1 time, got %d", got)
	}
}

// TestDebounce_ResetOnNewTrigger verifies that a second Trigger before the
// debounce window expires resets the timer, so the callback fires once
// measured from the LAST trigger, not the first.
func TestDebounce_ResetOnNewTrigger(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(80*time.Millisecond, func() {
		count.Add(1)
	})

	// First trigger
	d.Trigger()
	// Wait 50ms — timer not yet expired (80ms window)
	time.Sleep(50 * time.Millisecond)
	// Second trigger resets the window
	d.Trigger()
	// Wait 120ms — past the 80ms window from the second trigger
	time.Sleep(120 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected callback to fire exactly 1 time after reset, got %d", got)
	}
}
