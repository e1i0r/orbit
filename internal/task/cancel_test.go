package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// blockingEngine holds a phase open until its context is done and then
// answers the way engine.Claude does when it is killed: with the text it had
// already printed, alongside the error that stopped it (claude.go:45-51). It
// follows resultEngine (run_test.go:196), and it exists because engine.Fake
// returns before there is anything to interrupt.
type blockingEngine struct {
	output  string
	once    sync.Once
	running chan struct{} // closed once Run is inside the phase
}

func newBlockingEngine(output string) *blockingEngine {
	return &blockingEngine{output: output, running: make(chan struct{})}
}

func (e *blockingEngine) Name() string { return "fake" }

func (e *blockingEngine) Run(ctx context.Context, _ engine.Request) (engine.Result, error) {
	e.once.Do(func() { close(e.running) })
	<-ctx.Done()
	return engine.Result{Output: e.output}, ctx.Err()
}

// eventsOf reads a task's log, or fails the test trying.
func eventsOf(t *testing.T, s *store.Store, tk Task) []record.Event {
	t.Helper()
	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	return events
}

// wantKinds asserts the whole sequence at once, because a run's record is
// read as a sequence: which events are there and in what order is the fact,
// and a mismatch is worth printing in full.
func wantKinds(t *testing.T, events []record.Event, want ...string) {
	t.Helper()
	var got []string
	for _, e := range events {
		got = append(got, e.Kind)
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestARunThatIsCancelledSaysSoAndKeepsWhatThePhasePrinted(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	eng := newBlockingEngine("wrote half of the retry")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, s, tk, oneFlow(), map[string]engine.Engine{"fake": eng}, nil) }()
	<-eng.running
	cancel()
	runErr := <-done

	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run returned %v, want an error carrying context.Canceled", runErr)
	}
	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted, record.PhaseStarted,
		record.PhaseCancelled, record.TaskCancelled)

	if got := find(t, events, record.PhaseCancelled).Text; got != "wrote half of the retry" {
		t.Errorf("phase.cancelled text = %q, want what the engine had printed before it was stopped", got)
	}
	for _, e := range events {
		if e.Kind == record.PhaseFailed || e.Kind == record.TaskFailed {
			t.Errorf("a run stopped on purpose was written down as %q: a cancellation and a failure are different facts", e.Kind)
		}
	}
}

func TestARunThatOutlivesItsDeadlineSaysItTimedOut(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	eng := newBlockingEngine("thinking about it")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	runErr := Run(ctx, s, tk, oneFlow(), map[string]engine.Engine{"fake": eng}, nil)

	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("Run returned %v, want an error carrying context.DeadlineExceeded", runErr)
	}
	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted, record.PhaseStarted,
		record.PhaseCancelled, record.TaskTimedOut)

	if got := find(t, events, record.PhaseCancelled).Text; got != "thinking about it" {
		t.Errorf("phase.cancelled text = %q, want what the engine had printed before the deadline", got)
	}
}

func TestARunHoldsAMarkerWhileItGoesAndTakesItOffAfterwards(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	eng := newBlockingEngine("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, s, tk, oneFlow(), map[string]engine.Engine{"fake": eng}, nil) }()
	<-eng.running

	pid, ok, err := Alive(s, tk)
	if err != nil {
		t.Fatalf("Alive: %v", err)
	}
	if !ok || pid != os.Getpid() {
		t.Errorf("while the phase was running Alive = (%d, %v), want (%d, true)", pid, ok, os.Getpid())
	}

	cancel()
	<-done

	pid, ok, err = Alive(s, tk)
	if err != nil {
		t.Fatalf("Alive: %v", err)
	}
	if ok || pid != 0 {
		t.Errorf("after the run ended Alive = (%d, %v), want (0, false) — a claim outlived the run that made it", pid, ok)
	}
}

// held starts a process that sits there until it is signalled, in a process
// group of its own, the way Start spawns a run. The script writes the pid of
// a grandchild so a test can check that the group went with it.
func held(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start a process to signal: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck // best effort: the test is over
		_ = cmd.Wait()                                      //nolint:errcheck // best effort: the test is over
	})
	return cmd
}

// signalOf waits for a process to end and says which signal ended it, or 0
// if it ended on its own.
func signalOf(t *testing.T, cmd *exec.Cmd) syscall.Signal {
	t.Helper()
	_ = cmd.Wait() //nolint:errcheck // the exit status is read from ProcessState below
	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		t.Fatalf("exit status of an unexpected shape: %T", cmd.ProcessState.Sys())
	}
	if !status.Signaled() {
		return 0
	}
	return status.Signal()
}

func TestCancelAsksTheProcessHoldingTheTaskToStop(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	cmd := held(t, "sleep 5")
	if _, err := mark(s, tk, cmd.Process.Pid); err != nil {
		t.Fatalf("mark: %v", err)
	}

	if err := Cancel(s, tk); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got := signalOf(t, cmd); got != syscall.SIGTERM {
		t.Errorf("the run was ended by signal %v, want SIGTERM — anything else gives it no chance to write down that it stopped", got)
	}
}

func TestCancelWillNotSignalATaskNobodyIsRunning(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Cancel(s, tk); err == nil {
		t.Error("Cancel reported success for a task no process holds")
	}
	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if err := Cancel(s, tk); err == nil {
		t.Error("Cancel reported success against a pid that is gone")
	}
}

func TestKillTakesTheWholeGroupWithIt(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// The engine a run spawns is the process actually holding the work, and
	// it is a child of the run. A shell with a background sleep is the same
	// shape: kill only the shell and the sleep carries on.
	pidFile := filepath.Join(t.TempDir(), "child")
	cmd := held(t, "sleep 5 & echo $! > "+pidFile+"; wait")
	if _, err := mark(s, tk, cmd.Process.Pid); err != nil {
		t.Fatalf("mark: %v", err)
	}
	child := waitForPid(t, pidFile)

	if err := Kill(s, tk); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if got := signalOf(t, cmd); got != syscall.SIGKILL {
		t.Errorf("the run was ended by signal %v, want SIGKILL", got)
	}
	if err := waitForExit(child); err != nil {
		t.Errorf("the engine the run had spawned is still there: %v", err)
	}
}

// waitForPid reads the pid the shell wrote, waiting for the file to appear.
func waitForPid(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		body, err := os.ReadFile(path)
		if err == nil {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(body)))
			if convErr == nil {
				return pid
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("the shell never wrote a pid to %s", path)
	return 0
}

// waitForExit waits for a process that is not ours to disappear. It is not
// instant: the kernel delivers the signal, and the parent that reaps it is
// init once the group leader is gone.
func waitForExit(pid int) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !running(pid) {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("process %d is still running", pid)
}
