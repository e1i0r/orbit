package task

// What a phase's own events say, and what they leave out.
//
// A key left out is how every ending event already treats a session id and a
// cost it does not have: an empty string, or a zero, reads as something
// somebody wrote down, and nobody did. It is the same honesty rule the
// truncation follows — a `bytes: 0` on every event in the log is a reader
// who cannot tell which of them was cut.

import (
	"strconv"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestAPhaseWritesDownOnlyWhatItWasGiven.
func TestAPhaseWritesDownOnlyWhatItWasGiven(t *testing.T) {
	bare := phaseStart(flow.Phase{Name: "implement", Engine: "claude"}, "claude", 1, nil)

	for _, key := range []string{"model", "effort", "thinking", "permissions", "notes"} {
		if got, there := bare.Data[key]; there {
			t.Errorf("a phase given no %s wrote it down as %q", key, got)
		}
	}

	// And the two it is always given, because a log that cannot say which
	// engine did the work is the one thing this cannot afford.
	if bare.Data["engine"] != "claude" || bare.Data["n"] != "1" {
		t.Errorf("the event says engine %q, phase %q", bare.Data["engine"], bare.Data["n"])
	}
}

// TestAPhaseWritesDownEverythingItWasGiven, which is the other half: the
// posture goes into the record rather than being left to the flow file,
// because the log is the only account of a run that outlives it and "the
// flow file said so at the time" is not an account.
func TestAPhaseWritesDownEverythingItWasGiven(t *testing.T) {
	full := phaseStart(flow.Phase{
		Name: "implement", Engine: "claude", Model: "opus",
		Effort: "high", Thinking: "deep", Permissions: []string{"read", "write"},
	}, "codex", 2, []string{"cents, not floats", "use redis"})

	for _, one := range []struct{ key, want string }{
		{"engine", "codex"},
		{"n", "2"},
		{"model", "opus"},
		{"effort", "high"},
		{"thinking", "deep"},
		{"permissions", "read,write"},
		{"notes", "2"},
	} {
		if got := full.Data[one.key]; got != one.want {
			t.Errorf("the event says %s = %q, want %q", one.key, got, one.want)
		}
	}
}

// TestWhatWasNotCutSaysNothingAboutBeingCut.
//
// The bytes are how many there were before they were cut. On an event
// nothing was cut from, the key is a number a reader compares against the
// text in front of them and finds they agree — which teaches them the number
// means nothing, and the one event that really was cut goes past unnoticed.
func TestWhatWasNotCutSaysNothingAboutBeingCut(t *testing.T) {
	short := phaseThought("plan", 1, "read the file first")
	if got, there := short.Data["bytes"]; there {
		t.Errorf("a thought nothing was cut from says %q bytes were", got)
	}

	if short.Text != "read the file first" {
		t.Errorf("a short thought came back as %q", short.Text)
	}

	call := phaseToolCall("implement", 1, engine.StreamToolCall{Name: "Bash", Args: "go test ./..."})
	if got, there := call.Data["bytes"]; there {
		t.Errorf("a tool call nothing was cut from says %q bytes were", got)
	}

	if call.Data["tool"] != "Bash" || call.Text != "go test ./..." {
		t.Errorf("the call reads %q with tool %q", call.Text, call.Data["tool"])
	}

	// A refusal the model was given carries the same silence, and so does
	// the prompt the phase was sent — which is the one event somebody opens
	// when a phase's engine never came back.
	refused := phaseRefused("implement", 1, engine.StreamRefusal{Tool: "WebFetch", Input: "https://example.test"})
	if got, there := refused.Data["bytes"]; there {
		t.Errorf("a refusal nothing was cut from says %q bytes were", got)
	}

	if refused.Data["tool"] != "WebFetch" || refused.Text != "https://example.test" {
		t.Errorf("the refusal reads %q for tool %q", refused.Text, refused.Data["tool"])
	}

	asked := phaseAsked("implement", "claude", "do the thing")
	if got, there := asked.Data["bytes"]; there {
		t.Errorf("a prompt nothing was cut from says %q bytes were", got)
	}

	if asked.Data["engine"] != "claude" || asked.Text != "do the thing" {
		t.Errorf("the prompt was written down as %q, sent to %q", asked.Text, asked.Data["engine"])
	}

	// One byte over, and the count is there: the boundary is where the
	// number starts meaning something.
	long := longEnoughToCut()
	for _, one := range []struct {
		what  string
		event record.Event
	}{
		{"thought", phaseThought("plan", 1, long)},
		{"refusal", phaseRefused("plan", 1, engine.StreamRefusal{Tool: "Bash", Input: long})},
		{"prompt", phaseAsked("plan", "claude", long)},
	} {
		if one.event.Data["bytes"] != strconv.Itoa(maxOutput+1) {
			t.Errorf("a %s one byte over says %q bytes", one.what, one.event.Data["bytes"])
		}
	}
}

// longEnoughToCut is a string of exactly one byte more than is kept.
func longEnoughToCut() string {
	b := make([]byte, maxOutput+1)
	for i := range b {
		b[i] = 'a'
	}

	return string(b)
}

// TestAPhaseThatHadNothingToAddCarriesNoDataAtAll.
//
// An empty map is not the same as no map: it is written into the record as
// `"data":{}`, and every reader that asks whether a phase reported anything
// is told yes and then handed nothing. The cheap engines — a fake, one that
// prints and exits — end exactly this way, so it is the ordinary case and
// not the odd one.
func TestAPhaseThatHadNothingToAddCarriesNoDataAtAll(t *testing.T) {
	bare := phaseEnd(record.PhaseFinished, "implement", engine.Result{Output: "done"}, nil)
	if bare.Data != nil {
		t.Errorf("a phase that reported nothing carries %v", bare.Data)
	}

	if bare.Text != "done" {
		t.Errorf("what the engine printed reads %q", bare.Text)
	}

	// And one with a single thing to say carries that and only that.
	paid := phaseEnd(record.PhaseFinished, "implement",
		engine.Result{Output: "done", Cost: 0.25}, nil)

	if len(paid.Data) != 1 || paid.Data["cost"] != "0.25" {
		t.Errorf("a phase that cost a quarter carries %v", paid.Data)
	}

	// A count the engine did not report is left out for the same reason:
	// zero written down reads as a phase that sent no prompt.
	used := phaseEnd(record.PhaseFinished, "implement",
		engine.Result{Output: "done", Usage: engine.Usage{Input: 10}}, nil)

	for _, key := range []string{"tokens_out", "cache_read", "cache_write", "session", "output_bytes"} {
		if got, there := used.Data[key]; there {
			t.Errorf("a phase the engine said nothing about wrote %s = %q", key, got)
		}
	}

	if used.Data["tokens_in"] != "10" {
		t.Errorf("the one count it did report reads %q", used.Data["tokens_in"])
	}
}
