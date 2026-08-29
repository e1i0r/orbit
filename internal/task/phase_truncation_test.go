package task

// phase.go's own truncation-reporting branches, which only fire when the
// text handed in is actually over maxOutput, and runGates' branches for a
// passing gate's own output being oversized and for its GatePassed write
// itself failing.

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

func TestPhaseThoughtAndRefusedReportTruncation(t *testing.T) {
	big := strings.Repeat("T", maxOutput+100)

	th := phaseThought("plan", 1, big)
	if th.Data["bytes"] != strconv.Itoa(len(big)) {
		t.Errorf("phaseThought bytes = %q, want %d", th.Data["bytes"], len(big))
	}

	ref := phaseRefused("plan", 1, engine.StreamRefusal{Tool: "rm", Input: big})
	if ref.Data["bytes"] != strconv.Itoa(len(big)) {
		t.Errorf("phaseRefused bytes = %q, want %d", ref.Data["bytes"], len(big))
	}
}

// TestRunGatesPassingGateWithOversizedOutputReportsBytes covers the
// GatePassed branch where the gate's own combined output is over maxOutput.
func TestRunGatesPassingGateWithOversizedOutputReportsBytes(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "GATES-BIG-OUT-1", "gates big output test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	wt := t.TempDir()

	p := flow.Phase{
		Name: "test",
		Gates: []flow.Gate{
			// A little over 1 MiB of 'A', with exit 0.
			{Name: "big", Command: "dd if=/dev/zero bs=1048576 count=2 2>/dev/null | tr '\\0' 'A'"},
		},
	}
	if err := runGates(context.Background(), s, tk, p, 1, wt, engine.Result{}); err != nil {
		t.Fatalf("runGates: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	var found bool

	for _, e := range events {
		if e.Data["gate"] == "big" {
			found = true

			if e.Data["bytes"] == "" {
				t.Error("GatePassed for an oversized gate output did not report bytes")
			}
		}
	}

	if !found {
		t.Error("no GatePassed event recorded for the big gate")
	}
}

// TestRunGatesPassedEmitFailure covers runGates' own GatePassed write
// failing: a bad task id, with a worktree directory that genuinely exists so
// the command itself still runs.
func TestRunGatesPassedEmitFailure(t *testing.T) {
	s, r := fixture(t)
	bad := Task{ID: "has/slash", Repo: r}
	wt := t.TempDir()

	p := flow.Phase{
		Name:  "test",
		Gates: []flow.Gate{{Name: "ok", Command: "true"}},
	}
	if err := runGates(context.Background(), s, bad, p, 1, wt, engine.Result{}); err == nil {
		t.Error("runGates should have failed to record GatePassed for a bad task id")
	}
}

// TestAToolCallCarriesItsArgumentsOnce is the size of the log, not a detail
// of where a field sits.
//
// phase.tool_call is the kind a phase writes hundreds of, and it used to put
// the same captured arguments in Text and in Data["args"] — so the events
// that dominate the record weighed twice what they had to, and a call with
// large arguments could take its own line past record.MaxLine, where Append
// refuses it and the run fails on an emit that is not best-effort.
func TestAToolCallCarriesItsArgumentsOnce(t *testing.T) {
	args := `{"command":"go test ./..."}`

	e := phaseToolCall("implement", 1, engine.StreamToolCall{Name: "Bash", Args: args})
	if e.Text != args {
		t.Errorf("Text = %q, want the arguments %q", e.Text, args)
	}

	for key, value := range e.Data {
		if strings.Contains(value, args) {
			t.Errorf("Data[%q] repeats the arguments the event already carries in Text", key)
		}
	}

	if e.Data["tool"] != "Bash" {
		t.Errorf("Data[\"tool\"] = %q, want Bash — the name stays beside the payload", e.Data["tool"])
	}
}

// TestATruncatedToolCallSaysSo. The one thing Data still holds about the
// arguments is how many bytes there were before they were cut, which is the
// honesty rule captured follows everywhere else.
func TestATruncatedToolCallSaysSo(t *testing.T) {
	big := strings.Repeat("A", maxOutput+100)

	e := phaseToolCall("implement", 1, engine.StreamToolCall{Name: "Bash", Args: big})
	if e.Data["bytes"] != strconv.Itoa(len(big)) {
		t.Errorf("bytes = %q, want %d", e.Data["bytes"], len(big))
	}

	if len(e.Text) > maxOutput+64 {
		t.Errorf("Text is %d bytes, which is not a cut at maxOutput", len(e.Text))
	}
}
