package flows

// A loop, in the screen that shows what a flow is made of.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/flow"
)

func showingCoverage(t *testing.T) string {
	t.Helper()

	f, err := flow.Builtin("coverage")
	if err != nil {
		t.Fatalf("Builtin(coverage): %v", err)
	}

	s, e := designer(t)
	s.listed = []flow.Listed{{Name: f.Name, Origin: flow.OriginBuiltin}}
	// The detail view draws the editor's own phases, which is what opening a
	// flow fills in.
	s.phases = f.Phases
	s.creating, s.showingDetail, s.sel = false, true, 0

	return ansi.Strip(strings.Join(s.View(36, 120, e), "\n"))
}

// TestALoopIsDrawnAsABlockAndNotAsAnEmptyPhase.
//
// A loop has no engine and no prompt of its own — what runs is inside it — so
// the phase row drew a name, two blanks and "runs automatically". Somebody
// reading that sees a step that does nothing, which is the opposite of what a
// loop is.
func TestALoopIsDrawnAsABlockAndNotAsAnEmptyPhase(t *testing.T) {
	drawn := showingCoverage(t)

	if !strings.Contains(strings.ToLower(drawn), "loop") {
		t.Errorf("nothing says the phase is a loop:\n%s", drawn)
	}

	if !strings.Contains(drawn, "3") {
		t.Errorf("the screen does not say how many turns the loop gets:\n%s", drawn)
	}
}

// TestTheLoopSaysWhatWouldLetItStop. The checks are the whole of why a loop
// is not a doom loop, so they are the thing a reader most needs to see.
func TestTheLoopSaysWhatWouldLetItStop(t *testing.T) {
	drawn := showingCoverage(t)

	for _, want := range []string{"tests", "coverage over 90%"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the screen does not name the check %q:\n%s", want, drawn)
		}
	}
}

// TestThePhasesInsideTheLoopAreShown, since they are what actually runs.
func TestThePhasesInsideTheLoopAreShown(t *testing.T) {
	if drawn := showingCoverage(t); !strings.Contains(drawn, "fix") {
		t.Errorf("the phases inside the loop are not shown:\n%s", drawn)
	}
}

// TestTheDiagramSaysHowManyTurnsRatherThanNoEngine. A loop runs nothing
// itself, so the box that names the engine and the model had nothing to put
// in it and printed "/def" — a phase that reads as never configured.
func TestTheDiagramSaysHowManyTurnsRatherThanNoEngine(t *testing.T) {
	_, e := designer(t)
	s := Preview("coverage", FromBoard, e)

	rows := strings.Join(s.flowDetailRows(e.Frame.Body.H, e.Frame.Body.W, e), "\n")
	if strings.Contains(rows, "/def") {
		t.Errorf("the diagram still names no engine:\n%s", rows)
	}

	if !strings.Contains(rows, "↻ ×3") {
		t.Errorf("the diagram does not say how many turns the loop takes:\n%s", rows)
	}
}
