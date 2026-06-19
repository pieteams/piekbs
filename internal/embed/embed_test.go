package embed

import (
	"testing"
	"time"
)

// TestLifecycle_InitialState verifies that a freshly created LifecycleManager
// starts in StateUnloaded and IsLoaded returns false.
func TestLifecycle_InitialState(t *testing.T) {
	lm := NewLifecycleManager("model.onnx", 384, 5*time.Minute)
	if lm.State() != StateUnloaded {
		t.Fatalf("expected StateUnloaded, got %v", lm.State())
	}
	if lm.IsLoaded() {
		t.Fatal("expected IsLoaded=false for new manager")
	}
}

// TestLifecycle_UnloadModel verifies that calling Unload on a loaded model
// transitions state back to StateUnloaded, which is the correct cleanup behaviour.
func TestLifecycle_UnloadModel(t *testing.T) {
	lm := NewLifecycleManager("model.onnx", 384, 5*time.Minute)
	lm.setState(StateLoaded)
	if !lm.IsLoaded() {
		t.Fatal("expected IsLoaded=true after setState(StateLoaded)")
	}
	lm.Unload()
	if lm.State() != StateUnloaded {
		t.Fatalf("expected StateUnloaded after Unload, got %v", lm.State())
	}
}

// TestLifecycle_IdleTimeout verifies that ShouldUnload returns true once the
// idle timer fires, ensuring the model is evicted after inactivity.
func TestLifecycle_IdleTimeout(t *testing.T) {
	lm := NewLifecycleManager("model.onnx", 384, 100*time.Millisecond)
	lm.setState(StateLoaded)
	lm.resetIdleTimer()
	time.Sleep(150 * time.Millisecond)
	if !lm.ShouldUnload() {
		t.Fatal("expected ShouldUnload=true after idle timeout")
	}
}
