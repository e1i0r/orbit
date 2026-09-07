package flows

// The builder names the build's own engines and nothing else: a flow written
// on a machine with one engine must not offer the models of three others.

import (
	"strings"
	"testing"
)

// TestNoScreenNamesAnEngineThisBuildDoesNotHave. Eleven places in this
// package answered "claude" whenever nothing named an engine — the knob
// chip, the supervisor thread, the overview pane, the flow builder, the
// phase a new flow is born with. On a build without claude every one of
// them named something that was not going to run, and the reader had no
// way to tell that from a choice they had made.
func TestNoScreenNamesAnEngineThisBuildDoesNotHave(t *testing.T) {
	s, e := designer(t)
	e.Engines = zeta

	// The saved engine is the reader's own word and outranks the roster,
	// so it is cleared here: what is under test is what the window says
	// when nobody has said anything.
	if got := e.Engine; got != "zeta" {
		t.Fatalf("with nothing chosen the window names %q, want the only engine there is", got)
	}

	s = s.startCreateFlow(e).OnFields()
	if got := s.cur().Engine; got != "zeta" {
		t.Errorf("a new flow's first phase is born on %q, want zeta", got)
	}

	frame := strings.Join(linesOf(s.builderView(30, 100, e)), "\n")
	for _, invented := range []string{"claude", "codex", "opencode", "sonnet", "haiku"} {
		if strings.Contains(frame, invented) {
			t.Errorf("the flow builder names %q, which this build does not have:\n%s", invented, frame)
		}
	}

	if !strings.Contains(frame, "zeta") {
		t.Errorf("the flow builder names no engine at all:\n%s", frame)
	}
}
