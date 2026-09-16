package supervisor

// The supervisor says which engine it is, to everything it starts.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
)

// TestTheSupervisorNamesItselfToWhatItStarts. The engine spawns Orbit's mcp
// server and runs orbit commands of its own, and both inherit this: a retry
// asked for from inside a supervisor session runs on the supervisor's
// engine rather than on the one the task was written against.
func TestTheSupervisorNamesItselfToWhatItStarts(t *testing.T) {
	s := fixture(t)
	eng := &engine.Fake{Output: "done"}

	if _, err := SuperviseIn(context.Background(), s, eng, "", "c-1", "look at task 2900"); err != nil {
		t.Fatalf("supervise: %v", err)
	}

	if len(eng.Calls) != 1 {
		t.Fatalf("the engine was asked %d times, want once", len(eng.Calls))
	}

	want := engine.SupervisorVar + "=" + eng.Name()
	for _, kv := range eng.Calls[0].Env {
		if kv == want {
			return
		}
	}

	t.Errorf("the request carries %q, want %q in it", eng.Calls[0].Env, want)
}

// TestTheNameIsTheEngineThatIsRunning and not whatever was dialled at some
// earlier moment: the thread records its answers by the same name, so the
// two can never disagree about who did the work.
func TestTheNameIsTheEngineThatIsRunning(t *testing.T) {
	s := fixture(t)
	eng := &engine.Fake{Output: "done"}

	if _, err := SuperviseIn(context.Background(), s, eng, "", "c-1", "anything"); err != nil {
		t.Fatalf("supervise: %v", err)
	}

	events, err := Events(s)
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	var said string

	for _, e := range events {
		if e.Data["by"] != "" {
			said = e.Data["by"]
		}
	}

	name := strings.TrimPrefix(mustEnv(t, eng.Calls[0].Env), engine.SupervisorVar+"=")
	if said != name {
		t.Errorf("the thread says %q answered and the environment says %q", said, name)
	}
}

// mustEnv is the one variable this package puts in a request.
func mustEnv(t *testing.T, env []string) string {
	t.Helper()

	for _, kv := range env {
		if strings.HasPrefix(kv, engine.SupervisorVar+"=") {
			return kv
		}
	}

	t.Fatalf("no %s in %q", engine.SupervisorVar, env)

	return ""
}

// TestTheModelReachesTheEngine. `orbit settings model` was a knob with no
// cable: it was written, read back by the settings screen, and reached no
// run anywhere. The supervisor is the first place it does something.
func TestTheModelReachesTheEngine(t *testing.T) {
	s := fixture(t)
	eng := &engine.Fake{Output: "done"}

	if _, err := SuperviseIn(context.Background(), s, eng, "opus", "c-1", "anything"); err != nil {
		t.Fatalf("SuperviseIn: %v", err)
	}

	if len(eng.Calls) != 1 {
		t.Fatalf("the engine ran %d times, want one", len(eng.Calls))
	}

	if eng.Calls[0].Model != "opus" {
		t.Errorf("it was asked for model %q, want opus", eng.Calls[0].Model)
	}

	// And nothing named is still the engine's own default rather than a
	// model called "".
	if _, err := SuperviseIn(context.Background(), s, eng, "", "c-1", "anything"); err != nil {
		t.Fatalf("SuperviseIn: %v", err)
	}

	if eng.Calls[1].Model != "" {
		t.Errorf("with no model named it asked for %q", eng.Calls[1].Model)
	}
}
