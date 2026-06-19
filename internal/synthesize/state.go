//go:build fts5

package synthesize

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateFile = "index/synth_state.json"

// SynthState tracks which source-notes have been processed by synthesize.
// Persisted to index/synth_state.json.
type SynthState struct {
	LastRun   time.Time         `json:"last_run"`
	Processed map[string]string `json:"processed"` // path → SHA256(path+title+description+tags+key_claims)
}

// LoadSynthState reads the state file. Returns an empty state if absent.
func LoadSynthState(kbRoot string) (*SynthState, error) {
	path := filepath.Join(kbRoot, stateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SynthState{Processed: map[string]string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read synth state: %w", err)
	}
	var s SynthState
	if err := json.Unmarshal(data, &s); err != nil {
		return &SynthState{Processed: map[string]string{}}, nil
	}
	if s.Processed == nil {
		s.Processed = map[string]string{}
	}
	return &s, nil
}

// Save writes the state to disk.
func (s *SynthState) Save(kbRoot string) error {
	s.LastRun = time.Now()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	path := filepath.Join(kbRoot, stateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// NewOrChanged returns only the notes whose content hash differs from the saved state.
func (s *SynthState) NewOrChanged(notes []SourceNote) []SourceNote {
	var result []SourceNote
	for _, n := range notes {
		h := hashNote(n)
		if s.Processed[n.Path] != h {
			result = append(result, n)
		}
	}
	return result
}

// Record updates the hash for all given notes (call after successful generation).
func (s *SynthState) Record(notes []SourceNote) {
	for _, n := range notes {
		s.Processed[n.Path] = hashNote(n)
	}
}

// hashNote returns a stable SHA256 digest of the note's synthesize-relevant fields.
func hashNote(n SourceNote) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00", n.Title, n.Description, n.Path)
	for _, t := range n.Tags {
		fmt.Fprintf(h, "%s\x00", t)
	}
	for _, c := range n.KeyClaims {
		fmt.Fprintf(h, "claim:%s\x00", c)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// filterViablePlans removes plans that don't meet minimum source thresholds:
//   - concept: requires ≥3 source-notes
//   - comparison, decision: require ≥2 source-notes
func filterViablePlans(plans []PagePlan) []PagePlan {
	var out []PagePlan
	for _, p := range plans {
		switch p.Type {
		case "concept":
			if len(p.Sources) >= 3 {
				out = append(out, p)
			}
		case "comparison", "decision":
			if len(p.Sources) >= 2 {
				out = append(out, p)
			}
		}
	}
	return out
}
