package engine

// What an engine leaves running when it goes.
//
// An engine spawns work of its own — a tool call is a process — and the
// run is waiting on a pipe those children inherited. Both of these are
// about what happens to them, and to the run, when the engine stops.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// answered is what one run said, carried off the goroutine that waited for
// it.
type answered struct {
	out Result
	err error
}

// TestAnEngineThatLeavesSomethingRunningDoesNotHoldTheRun. Wait waits for
// the output pipe to be drained, and a child that inherited stdout holds it
// open after the engine itself has exited: a tool call left in the
// background held the phase — and the task's slot — for as long as it lived.
func TestAnEngineThatLeavesSomethingRunningDoesNotHoldTheRun(t *testing.T) {
	s := toy(t, "sleep 30 & echo done")

	began := time.Now()
	done := make(chan answered, 1)

	go func() {
		out, err := s.run(context.Background(), Request{Dir: t.TempDir()})
		done <- answered{out: out, err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("run: %v", res.err)
		}

		if took := time.Since(began); took > 20*time.Second {
			t.Errorf("the run waited %s on an engine that had already exited", took)
		}

		// And the phase says so: the output stops where the pipe was
		// closed rather than where the engine did, and a text that read
		// like a whole one would be the record claiming what it cannot
		// know.
		if !strings.Contains(res.out.Output, "leaving something running") {
			t.Errorf("the engine left something running and the answer does not say so: %q", res.out.Output)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the run is still waiting on an engine that exited at once")
	}
}

// TestCancellingAPhaseKillsWhatTheEngineStarted. A tool call is a process
// of the engine's own, and killing the engine alone left it running in a
// worktree the run had finished with — a build, a test suite, a server.
func TestCancellingAPhaseKillsWhatTheEngineStarted(t *testing.T) {
	mark := filepath.Join(t.TempDir(), "still-here")
	s := toy(t, "(sleep 2; touch "+mark+") & sleep 20")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan answered, 1)

	go func() {
		out, err := s.run(ctx, Request{Dir: t.TempDir()})
		done <- answered{out: out, err: err}
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case res := <-done:
		// A cancelled run is an error by definition, and the one it
		// answers with names the engine and the worktree.
		if res.err == nil {
			t.Error("a cancelled run answered as if it had finished")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("the run did not stop when the phase was cancelled")
	}

	// Long enough for the child to have touched the file, if it lived.
	time.Sleep(3 * time.Second)

	if _, err := os.Stat(mark); err == nil {
		t.Error("the phase was cancelled and what the engine started carried on")
	}
}
