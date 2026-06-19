package embed

import (
	"sync"
	"time"
)

// ModelState represents the current loading state of the embedding model.
type ModelState int

const (
	StateUnloaded ModelState = iota
	StateLoading
	StateLoaded
)

// LifecycleManager tracks the load state of an embedding model and evicts it
// from memory after a configurable idle period.
type LifecycleManager struct {
	modelPath   string
	dim         int
	idleTimeout time.Duration
	mu          sync.Mutex
	state       ModelState
	lastUsed    time.Time
	timer       *time.Timer
}

// NewLifecycleManager creates a LifecycleManager in the unloaded state.
func NewLifecycleManager(modelPath string, dim int, idleTimeout time.Duration) *LifecycleManager {
	return &LifecycleManager{
		modelPath:   modelPath,
		dim:         dim,
		idleTimeout: idleTimeout,
		state:       StateUnloaded,
	}
}

// State returns the current ModelState (caller must not hold lm.mu).
func (lm *LifecycleManager) State() ModelState {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.state
}

// IsLoaded reports whether the model is currently in StateLoaded.
func (lm *LifecycleManager) IsLoaded() bool {
	return lm.State() == StateLoaded
}

// setState sets the state directly; used internally and in tests.
func (lm *LifecycleManager) setState(s ModelState) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.state = s
}

// resetIdleTimer restarts the idle timeout clock from now.
// Must be called while NOT holding lm.mu (it acquires the lock internally).
func (lm *LifecycleManager) resetIdleTimer() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.timer != nil {
		lm.timer.Stop()
	}
	lm.lastUsed = time.Now()
	lm.timer = time.AfterFunc(lm.idleTimeout, func() {
		// Timer fires; actual unload decision is left to ShouldUnload.
	})
}

// ShouldUnload reports whether the model has been idle long enough to be evicted.
// Returns true only when loaded and the idle timeout has elapsed.
func (lm *LifecycleManager) ShouldUnload() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.state != StateLoaded {
		return false
	}
	return time.Since(lm.lastUsed) >= lm.idleTimeout
}

// Unload transitions the model to StateUnloaded and stops any pending timer.
func (lm *LifecycleManager) Unload() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.timer != nil {
		lm.timer.Stop()
		lm.timer = nil
	}
	lm.state = StateUnloaded
}

// Touch records the current time as the last-used timestamp and resets the
// idle timer, preventing premature eviction while inference is active.
func (lm *LifecycleManager) Touch() {
	lm.resetIdleTimer()
}
