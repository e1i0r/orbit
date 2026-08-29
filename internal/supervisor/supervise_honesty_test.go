package supervisor

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// halfSpokenEngine answers with something and fails anyway, which is what
// every real engine does when it is stopped or falls over mid-answer: the
// text it had already printed comes back alongside the error that ended it
// (claude.go, and spec.report for all three adapters).
//
// It is written here because engine.Fake cannot do it — Fake.Err returns an
// empty Result — and this is precisely the case the code below is about.
type halfSpokenEngine struct {
	*engine.Fake
	said string
	err  error
}

// The embedded Fake answers everything else about the engine; only Run is
// its own, because only Run is what this is here to do differently.
func (e halfSpokenEngine) Run(_ context.Context, _ engine.Request) (engine.Result, error) {
	return engine.Result{Output: e.said}, e.err
}

// TestTheSupervisorReportsAnEngineThatStoppedMidAnswer.
//
// The error used to be dropped whenever the engine had printed anything at
// all, so half an answer was recorded in the thread and handed back as
// though the supervisor had finished speaking. Both halves are the point:
// what it said is kept, and the caller is told it did not finish.
func TestTheSupervisorReportsAnEngineThatStoppedMidAnswer(t *testing.T) {
	s := fixture(t)

	broke := errors.New("the model stopped answering")
	eng := halfSpokenEngine{&engine.Fake{}, "ORB-10 looks stuck, I was about to", broke}

	ans, err := Supervise(context.Background(), s, eng, "how is the board doing?")
	if !errors.Is(err, broke) {
		t.Fatalf("Supervise returned %v, want the error the engine stopped with", err)
	}

	if ans != "ORB-10 looks stuck, I was about to" {
		t.Errorf("Supervise answered %q, want what the supervisor managed to say", ans)
	}

	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 1 || events[0].Text != "ORB-10 looks stuck, I was about to" {
		t.Errorf("the thread holds %+v, want the half answer written down", events)
	}
}

// TestTheSupervisorRefusesAThreadItCannotRead.
//
// Reading the thread failed silently and the model was handed an empty
// history, which is the one thing history() takes trouble to prevent: it
// says out loud how many turns it is not showing, because a model that
// cannot see it is missing context speaks as though it has all of it. This
// supervisor also acts — it directs, retries and cancels tasks — so one that
// cannot remember is one that does it twice.
func TestTheSupervisorRefusesAThreadItCannotRead(t *testing.T) {
	s := fixture(t)

	if err := Record(s, record.SupervisorMessage, "operator", "tui", "", "", "ORB-10 is yours"); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// A line longer than the record can read back. Append refuses to write
	// one, so this is damage from outside — which is the case: a thread that
	// will not parse is a thread nobody can vouch for.
	damaged := strings.Repeat("x", record.MaxLine+1) + "\n"
	if err := os.WriteFile(s.SupervisorLogPath(), []byte(damaged), 0o600); err != nil {
		t.Fatalf("damage the thread: %v", err)
	}

	fake := &engine.Fake{Output: "all systems nominal"}

	if _, err := Supervise(context.Background(), s, fake, "how is the board doing?"); err == nil {
		t.Fatal("Supervise answered over a thread it could not read, want a refusal")
	}

	if len(fake.Calls) != 0 {
		t.Errorf("the engine was asked %d times over a thread nobody could read, want 0", len(fake.Calls))
	}
}
