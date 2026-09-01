package supervisor

import (
	"context"
	"errors"
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
// Dropping the error whenever the engine printed anything at all would
// record half an answer in the thread and hand it back as though the
// supervisor had finished speaking. Both halves are the point:
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

	// The record taken out from under it, which is damage from outside — as
	// it has to be: a turn this package wrote is a turn it can read back, and
	// a thread nobody can vouch for is one something else broke.
	breakRecord(t, s)

	fake := &engine.Fake{Output: "all systems nominal"}

	if _, err := Supervise(context.Background(), s, fake, "how is the board doing?"); err == nil {
		t.Fatal("Supervise answered over a thread it could not read, want a refusal")
	}

	if len(fake.Calls) != 0 {
		t.Errorf("the engine was asked %d times over a thread nobody could read, want 0", len(fake.Calls))
	}
}
