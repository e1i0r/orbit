package task

// The prompt a phase was given, written down.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// TestWhatAPhaseWasAskedIsWrittenDown is the whole of it: the one thing
// Orbit sends, kept where every other side of the run already is.
func TestWhatAPhaseWasAskedIsWrittenDown(t *testing.T) {
	s, tk := nowhere(t, "ACME-1", "make the build green")

	fake := engine.NewFake("wrote it")

	if err := Run(context.Background(), s, tk, oneEngineFlow("fake"), fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	one := lastOfKind(t, s, tk, record.PhaseAsked)
	if one.Kind == "" {
		t.Fatalf("no prompt was written down:\n%v", kindsFor(t, s, tk))
	}

	// It is the prompt the engine was handed, not a rendering of it.
	if one.Text != fake.Calls[0].Prompt {
		t.Errorf("the record kept something other than what was sent:\n%s", one.Text)
	}

	if one.Phase != "implement" || one.Data["engine"] != "fake" {
		t.Errorf("it says phase %q on %q, want implement on fake", one.Phase, one.Data["engine"])
	}
}

// TestAPhaseThatNeverAnsweredStillSaysWhatItWasAsked is why it is written
// before the engine is called rather than after: a phase whose engine never
// comes back is exactly the phase somebody needs the prompt of.
func TestAPhaseThatNeverAnsweredStillSaysWhatItWasAsked(t *testing.T) {
	s, tk := nowhere(t, "ACME-2", "make the build green")

	broken := engine.NewFake("")
	broken.Err = errRanOut

	err := Run(context.Background(), s, tk, oneEngineFlow("fake"), fakes(broken), nil)
	if err == nil {
		t.Fatal("Run: want an error when the engine broke")
	}

	one := lastOfKind(t, s, tk, record.PhaseAsked)
	if one.Kind == "" {
		t.Fatalf("a phase that never answered wrote down no prompt:\n%v", kindsFor(t, s, tk))
	}

	if !strings.Contains(one.Text, "ACME-2") {
		t.Errorf("the prompt does not name the task it was about:\n%s", one.Text)
	}
}
