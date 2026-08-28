package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
)

func TestPromptAndCapturedHelpers(t *testing.T) {
	tk := Task{
		ID:   "PROMPT-1",
		Text: "Main task body",
		Repo: repoTaskRepo(t),
	}
	p := flow.Phase{
		Name:   "build",
		Prompt: "Extra phase prompt",
	}

	// 1. Full prompt with notes, instructions, previous output
	notes := []string{"First note", "Second note"}

	fullPrompt := prompt(tk, p, notes, "Previous output text")
	if !strings.Contains(fullPrompt, "Main task body") ||
		!strings.Contains(fullPrompt, "Extra phase prompt") ||
		!strings.Contains(fullPrompt, "Previous output text") ||
		!strings.Contains(fullPrompt, "First note") {
		t.Errorf("prompt missing components: %s", fullPrompt)
	}

	// 2. captured with short string
	short, full := captured("hello world")
	if short != "hello world" || full != 0 {
		t.Errorf("captured short = (%q, %d), want (\"hello world\", 0)", short, full)
	}

	// 3. captured with string > 1MB
	bigStr := strings.Repeat("A", maxOutput+100)

	truncated, fullBig := captured(bigStr)
	if fullBig != len(bigStr) || !strings.Contains(truncated, "truncated") {
		t.Errorf("captured big string failed to truncate: full=%d", fullBig)
	}

	// 4. captured cutting exactly on a multi-byte rune must back up rather
	// than split it: "é" is two bytes, placed so the cut point at maxOutput
	// lands on its second, continuation byte.
	withRune := strings.Repeat("A", maxOutput-1) + "é" + "tail"

	cut, fullRune := captured(withRune)
	if fullRune != len(withRune) {
		t.Errorf("captured rune-boundary full = %d, want %d", fullRune, len(withRune))
	}

	kept := strings.SplitN(cut, "\n…", 2)[0]
	if !utf8.ValidString(kept) {
		t.Errorf("captured split a rune in half: %q", kept)
	}

	if strings.Contains(kept, "é") {
		t.Error("captured kept a rune that straddled the cut point rather than backing up before it")
	}
}

func TestRunMultiPhaseFeedOutputAndThoughts(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RUN-FEED-1", "Feed output test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	testFlow := flow.Flow{
		Name: "multi-phase-flow",
		Phases: []flow.Phase{
			{Name: "phase-1", Engine: "mock-engine", FeedOutput: false},
			{Name: "phase-2", Engine: "mock-engine", FeedOutput: true},
		},
	}

	mockEng := &streamMockEngine{
		results: []engine.Result{
			{
				Output:    "Step 1 completed",
				Thoughts:  []string{"Thinking about step 1"},
				Refusals:  []engine.StreamRefusal{{Tool: "danger_tool", Input: "arg"}},
				SessionID: "sess-1",
				Cost:      0.01,
			},
			{
				Output:    "Step 2 completed",
				SessionID: "sess-2",
				Cost:      0.02,
			},
		},
	}

	engines := map[string]engine.Engine{"mock-engine": mockEng}
	if err := Run(context.Background(), s, tk, testFlow, engines, nil); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events failed: %v", err)
	}

	var hasThought, hasRefusal, hasFinished bool

	for _, e := range events {
		if e.Kind == record.PhaseThought {
			hasThought = true
		}

		if e.Kind == record.PhaseRefused {
			hasRefusal = true
		}

		if e.Kind == record.TaskFinished {
			hasFinished = true
		}
	}

	if !hasThought || !hasRefusal || !hasFinished {
		t.Errorf("missing expected events: thought=%v, refusal=%v, finished=%v",
			hasThought, hasRefusal, hasFinished)
	}
}

func TestStoppedHelper(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "STOP-1", "Stopped test", "")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Deadline exceeded
	err = stopped(s, tk, "phase-1", engine.Result{}, context.DeadlineExceeded)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("stopped on deadline = %v", err)
	}

	// 2. Canceled context
	err = stopped(s, tk, "phase-1", engine.Result{}, context.Canceled)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("stopped on cancel = %v", err)
	}
}

func TestReconcileAllCases(t *testing.T) {
	s, r := fixture(t)

	// 1. Task in flight with dead PID marker -> Reconcile abandons it
	tk1, err := Create(s, r, "REC-INFLIGHT", "In flight test", "")
	if err != nil {
		t.Fatal(err)
	}

	_ = emit(s, tk1, record.Event{Kind: record.TaskStarted}) //nolint:errcheck
	_, _ = mark(s, tk1, 9999999)                             //nolint:errcheck

	changed, err := Reconcile(s, tk1)
	if err != nil || !changed {
		t.Errorf("Reconcile inFlight = (%v, %v), want (true, nil)", changed, err)
	}

	// 2. Task already finished with stale marker -> Reconcile sweeps marker, returns false
	tk2, err := Create(s, r, "REC-FINISHED", "Finished test", "")
	if err != nil {
		t.Fatal(err)
	}

	_ = emit(s, tk2, record.Event{Kind: record.TaskStarted})  //nolint:errcheck
	_ = emit(s, tk2, record.Event{Kind: record.TaskFinished}) //nolint:errcheck
	_, _ = mark(s, tk2, 9999999)                              //nolint:errcheck

	changed, err = Reconcile(s, tk2)
	if err != nil || changed {
		t.Errorf("Reconcile finished with stale marker = (%v, %v), want (false, nil)", changed, err)
	}

	// 3. inFlight helper invariant checks
	for _, endKind := range []string{
		record.TaskFinished,
		record.TaskFailed,
		record.TaskCancelled,
		record.TaskTimedOut,
		record.TaskAbandoned,
	} {
		events := []record.Event{
			{Kind: record.TaskStarted},
			{Kind: endKind},
		}
		if inFlight(events) {
			t.Errorf("inFlight should be false after %s", endKind)
		}
	}
}

type streamMockEngine struct {
	results []engine.Result
	callIdx int
}

func (m *streamMockEngine) Name() string { return "mock-engine" }
func (m *streamMockEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	if m.callIdx < len(m.results) {
		res := m.results[m.callIdx]
		m.callIdx++

		return res, nil
	}

	return engine.Result{Output: "default"}, nil
}
func (m *streamMockEngine) Models() []engine.Choice  { return nil }
func (m *streamMockEngine) Efforts() []engine.Choice { return nil }
func (m *streamMockEngine) CanThink() bool           { return true }
func (m *streamMockEngine) Locate() (string, error)  { return "mock", nil }
func (m *streamMockEngine) CanResume() bool          { return true }

func repoTaskRepo(t *testing.T) repo.Repo {
	t.Helper()
	return repo.Repo{Path: "/tmp/repo", Name: "repo"}
}

// TestPrepareErrorPaths covers prepare's two error returns: the worktree
// parent cannot be resolved, and the worktree itself cannot be added.
func TestPrepareErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. CreateWorktreeParent fails: a bad id.
	bad := Task{ID: "has/slash", Repo: r}
	if _, err := prepare(s, bad); err == nil {
		t.Error("prepare should have failed when the worktree parent cannot be resolved")
	}

	// 2. AddWorktree fails: the repository is not on any branch, which
	// AddWorktree refuses before it ever runs git.
	detached := r
	detached.Base = ""

	tk := Task{ID: "PREPARE-ERR-1", Repo: detached}
	if _, err := prepare(s, tk); err == nil {
		t.Error("prepare should have failed against a repository with no base branch")
	}
}

// TestPrepareReusesAnExistingWorktree covers the branch prepare exists for:
// a second call finds the worktree already on disk and hands it back rather
// than asking git to add it again.
func TestPrepareReusesAnExistingWorktree(t *testing.T) {
	s, r := fixture(t)
	tk := Task{ID: "PREPARE-REUSE-1", Repo: r}

	first, err := prepare(s, tk)
	if err != nil {
		t.Fatalf("prepare (first): %v", err)
	}

	second, err := prepare(s, tk)
	if err != nil {
		t.Fatalf("prepare (second): %v", err)
	}

	if first != second {
		t.Errorf("prepare returned %q then %q for the same task", first, second)
	}
}
