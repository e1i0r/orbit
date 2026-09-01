package supervisor

// What happens when the record itself is against you: out of reach on the
// way in, refusing what is handed to it on the way out.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// TestARetractionOnAThreadThatWillNotReadSaysSo. Retract has to find the turn
// before it can take it back, and a thread it cannot read is not a thread with
// nothing in it — answering "nothing was written at that time" would tell the
// operator their timestamp was wrong when the truth is that the log is broken.
func TestARetractionOnAThreadThatWillNotReadSaysSo(t *testing.T) {
	s := fixture(t)
	breakRecord(t, s)

	err := Retract(s, time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("Retract answered nil over a thread it could not read")
	}

	if strings.Contains(err.Error(), "nothing in the supervisor thread") {
		t.Errorf("a broken log was reported as a timestamp nobody wrote at: %v", err)
	}
}

// TestAnAnswerThatCannotBeWrittenDownIsStillReturned.
//
// The answer is the expensive part — a model ran to produce it — and the
// caller is a cockpit that draws it. Losing it because the log would not take
// it would mean paying for a supervisor turn and showing nothing, so the text
// comes back beside the error rather than instead of it.
func TestAnAnswerThatCannotBeWrittenDownIsStillReturned(t *testing.T) {
	s := fixture(t)

	// An answer too long to be one turn. The thread reads back perfectly and
	// the model runs; it is the writing down of what it said that fails,
	// which is the only shape this branch has now that reading and writing
	// go to the same record.
	answer := strings.Repeat("x", record.MaxLine+1)

	ans, err := Supervise(context.Background(), s, &engine.Fake{Output: answer}, "how is it going?")
	if err == nil {
		t.Fatal("Supervise answered nil over a thread it could not append to")
	}

	if ans != answer {
		t.Errorf("the answer was dropped along with the write that failed: %d bytes came back", len(ans))
	}
}
