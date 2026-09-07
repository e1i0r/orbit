package flows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestEditingAFlowKeepsHowManyAttemptsItAllows is a form that rebuilds the
// whole flow from its fields: anything the fields do not hold is gone the
// moment somebody saves. The attempt cap is not on the form, so the form has
// to carry it.
func TestEditingAFlowKeepsHowManyAttemptsItAllows(t *testing.T) {
	dir := t.TempDir()

	s, e := designer(t)
	e.Flows = flowsTestDir(dir)

	saved := flow.Flow{
		Name:     "one-shot",
		Attempts: 1,
		Phases:   []flow.Phase{{Name: "implement", Engine: "claude"}},
	}
	if _, err := flow.Save(e.Flows, saved); err != nil {
		t.Fatalf("seed the flow: %v", err)
	}

	m2, _ := s.editFlow(saved.Name, e)

	m3, out1 := m2.saveCustomFlow(e)
	if out1.Said != "" && m3.creating {
		t.Fatalf("the save did not go through: %s", out1.Said)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "one-shot.json"))
	if err != nil {
		t.Fatalf("read the saved flow: %v", err)
	}

	back, err := flow.Decode(raw, "one-shot")
	if err != nil {
		t.Fatalf("decode the saved flow: %v", err)
	}

	if back.Attempts != 1 {
		t.Errorf("the saved flow allows %d attempts, want the 1 it was written with", back.Attempts)
	}
}
