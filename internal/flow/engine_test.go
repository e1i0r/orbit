package flow

import "testing"

// TestAFlowCanBeWalkedByAnotherEngine, without the file on disk changing.
func TestAFlowCanBeWalkedByAnotherEngine(t *testing.T) {
	f := Flow{Name: "task", Phases: []Phase{
		{Name: "implement", Engine: "claude", Model: "opus"},
		{Name: "loop", Loop: &Loop{
			Max:    3,
			Until:  []Gate{{Name: "tests", Command: "go test ./..."}},
			Phases: []Phase{{Name: "fix", Engine: "claude", Model: "sonnet"}},
		}},
	}}

	got := WithEngine(f, "codex")

	for _, p := range got.Phases {
		if p.Engine != "codex" {
			t.Errorf("phase %q still runs on %q", p.Name, p.Engine)
		}

		if p.Model != "" {
			t.Errorf("phase %q carried %q across to another engine", p.Name, p.Model)
		}
	}

	inner := got.Phases[1].Loop.Phases[0]
	if inner.Engine != "codex" {
		t.Errorf("the phase inside the loop still runs on %q", inner.Engine)
	}

	// The original is what every other run of Orbit reads off disk.
	if f.Phases[0].Engine != "claude" || f.Phases[1].Loop.Phases[0].Engine != "claude" {
		t.Error("the flow itself was rewritten, not a copy of it")
	}
}

// TestNoEngineNamedLeavesTheFlowAlone.
func TestNoEngineNamedLeavesTheFlowAlone(t *testing.T) {
	f := Flow{Name: "task", Phases: []Phase{{Name: "implement", Engine: "claude", Model: "opus"}}}

	if got := WithEngine(f, ""); got.Phases[0].Engine != "claude" || got.Phases[0].Model != "opus" {
		t.Errorf("an empty override changed the flow: %+v", got.Phases[0])
	}
}
