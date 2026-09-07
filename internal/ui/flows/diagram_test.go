package flows

// The diagram of a flow, fuzzed: whatever a phase is called and whichever
// toggles it carries, the drawing comes back with rows in it and never
// panics.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

func FuzzRenderFlowDiagram(f *testing.F) {
	f.Add("phase-1", "claude", "opus", true, false, 80)
	f.Add("p1", "codex", "sonnet", false, true, 40)
	f.Add("long-phase-name-test", "opencode", "default", true, true, 120)

	f.Fuzz(func(t *testing.T, name, engine, model string, feed, wait bool, width int) {
		if width < 10 || width > 300 {
			return
		}

		phases := []flow.Phase{
			{Name: name, Engine: engine, Model: model, FeedOutput: feed, Wait: wait},
			{Name: "2-" + name, Engine: engine, Model: model, FeedOutput: !feed, Wait: !wait},
		}

		lines := renderFlowDiagram(phases, width)
		if len(lines) == 0 {
			t.Errorf("expected diagram lines, got 0")
		}
	})
}
