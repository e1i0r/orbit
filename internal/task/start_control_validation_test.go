package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

func TestStartCapAndRunCommand(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Unread Cap enforcement
	if err := s.SaveSettings(store.Settings{UnreadCap: 3}); err != nil {
		t.Fatal(err)
	}

	tk := Task{
		ID:   "TASK-1",
		Repo: repo.Repo{Path: "/path/to/repo", Name: "repo"},
	}

	// unread=3 meets cap of 3 -> rejected
	_, err = Start(s, tk, "quick", 3)
	if err == nil || !strings.Contains(err.Error(), "was not started") {
		t.Errorf("expected cap rejection error, got %v", err)
	}

	// unread=0 below cap -> executes binary
	tkValidDir := Task{
		ID:   "TASK-1",
		Repo: repo.Repo{Path: t.TempDir(), Name: "repo"},
	}

	pid, startErr := Start(s, tkValidDir, "quick", 0)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	if pid <= 0 {
		t.Errorf("expected positive PID from Start, got %d", pid)
	}

	// 2. runCommand construction
	cmd := runCommand("/bin/orbit", root, tk, "custom-flow")
	if cmd.Path != "/bin/orbit" {
		t.Errorf("cmd.Path = %q, want /bin/orbit", cmd.Path)
	}

	if cmd.Dir != "/path/to/repo" {
		t.Errorf("cmd.Dir = %q, want /path/to/repo", cmd.Dir)
	}

	expectedArgs := []string{"/bin/orbit", "run", "-repo", "/path/to/repo", "-flow", "custom-flow", "TASK-1"}
	for i, arg := range expectedArgs {
		if i >= len(cmd.Args) || cmd.Args[i] != arg {
			t.Errorf("cmd.Args[%d] = %q, want %q", i, cmd.Args[i], arg)
		}
	}

	foundOrbitHome := false

	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "ORBIT_HOME="+root) {
			foundOrbitHome = true
			break
		}
	}

	if !foundOrbitHome {
		t.Error("cmd.Env missing ORBIT_HOME")
	}
}

func TestControlInvalidWord(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	tk := Task{ID: "TASK-1", Repo: repo.Repo{Path: "/tmp/repo"}}

	err = Control(s, tk, "invalid-word-123")
	if err == nil || !strings.Contains(err.Error(), "not something a run understands") {
		t.Errorf("expected invalid word error, got %v", err)
	}
}

func TestAliveMarkerParsingEdgeCases(t *testing.T) {
	// parsePid edge cases
	if _, err := parsePid("not_a_number"); err == nil {
		t.Error("expected error on missing pid line")
	}

	if _, err := parsePid("pid: abc"); err == nil {
		t.Error("expected error on non-numeric pid")
	}

	if _, err := parsePid("pid: -5"); err == nil {
		t.Error("expected error on negative pid")
	}

	if pid, err := parsePid("pid: 12345"); err != nil || pid != 12345 {
		t.Errorf("parsePid(pid: 12345) = (%d, %v), want (12345, nil)", pid, err)
	}

	// parseStarted edge cases
	if parsed := parseStarted("not-a-timestamp"); !parsed.IsZero() {
		t.Errorf("expected zero time on missing started line, got %v", parsed)
	}

	if parsed := parseStarted("started: bad-format"); !parsed.IsZero() {
		t.Errorf("expected zero time on invalid timestamp, got %v", parsed)
	}

	nowStr := "started: " + time.Now().UTC().Format(time.RFC3339)
	if parsed := parseStarted(nowStr); parsed.IsZero() {
		t.Errorf("parseStarted(%q) returned zero time", nowStr)
	}
}

func TestTaskCreateValidation(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	// Empty ID
	if _, err := Create(s, r, "", "some text", "quick"); err == nil {
		t.Error("expected error on empty ID")
	}

	// Empty text
	if _, err := Create(s, r, "TASK-1", "", "quick"); err == nil {
		t.Error("expected error on empty text")
	}

	// Successful creation
	created, err := Create(s, r, "TASK-1", "some text", "quick")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID != "TASK-1" {
		t.Errorf("created.ID = %q, want TASK-1", created.ID)
	}

	// Duplicate creation returns ErrExists
	if _, err := Create(s, r, "TASK-1", "duplicate text", "quick"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected ErrExists on duplicate task, got %v", err)
	}
}

func TestRunEngineAndModelValidations(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	tk := Task{ID: "TASK-1", Repo: repo.Repo{Path: "/tmp/repo", Name: "repo"}}

	// 1. Missing engine in engines map
	fMissingEngine := flow.Flow{
		Name: "test-flow",
		Phases: []flow.Phase{
			{Name: "phase-1", Engine: "nonexistent"},
		},
	}

	err = Run(context.Background(), s, tk, fMissingEngine, map[string]engine.Engine{}, nil)
	if err == nil || !strings.Contains(err.Error(), "which is not configured") {
		t.Errorf("expected unconfigured engine error, got %v", err)
	}

	// Engine with models, efforts, thinking
	claudeEng := engine.NewClaude()
	engines := map[string]engine.Engine{"claude": claudeEng}

	// 2. Unsupported model
	fBadModel := flow.Flow{
		Name: "test-flow",
		Phases: []flow.Phase{
			{Name: "phase-1", Engine: "claude", Model: "nonexistent-model"},
		},
	}

	err = Run(context.Background(), s, tk, fBadModel, engines, nil)
	if err == nil || !strings.Contains(err.Error(), "does not offer") {
		t.Errorf("expected unsupported model error, got %v", err)
	}

	// 3. Unsupported effort
	fBadEffort := flow.Flow{
		Name: "test-flow",
		Phases: []flow.Phase{
			{Name: "phase-1", Engine: "claude", Effort: "ultra-high"},
		},
	}

	err = Run(context.Background(), s, tk, fBadEffort, engines, nil)
	if err == nil || !strings.Contains(err.Error(), "effort") {
		t.Errorf("expected unsupported effort error, got %v", err)
	}

	// 4. Fake engine (which does not support thinking mode)
	fakeEng := engine.NewFake("out")
	enginesFake := map[string]engine.Engine{"fake": fakeEng}
	fBadThinking := flow.Flow{
		Name: "test-flow",
		Phases: []flow.Phase{
			{Name: "phase-1", Engine: "fake", Thinking: "high"},
		},
	}

	err = Run(context.Background(), s, tk, fBadThinking, enginesFake, nil)
	if err == nil || !strings.Contains(err.Error(), "does not support thinking") {
		t.Errorf("expected unsupported thinking error, got %v", err)
	}
}

func TestTaskLoadAndFlowChoice(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	// 1. Load non-existent task
	if _, err := Load(s, r, "NONEXISTENT-1"); err == nil {
		t.Error("expected error loading nonexistent task")
	}

	// 2. Create task with custom flow and load it back
	created, err := Create(s, r, "TASK-CUSTOM", "test task", "custom-flow")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loaded, err := Load(s, r, "TASK-CUSTOM")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Flow != "custom-flow" || loaded.Text != "test task" {
		t.Errorf("loaded task = %+v, want flow=custom-flow", loaded)
	}

	if created.ID != loaded.ID {
		t.Errorf("created ID %q != loaded ID %q", created.ID, loaded.ID)
	}
}

// TestStartRefusesATaskWhoseMarkerWillNotRead.
//
// Start looks before it spawns, and a claim it cannot read is a claim it
// cannot rule out. Treating it as "nothing is running" would put a second
// `orbit run` on one worktree, one branch and one log -- which is the whole
// reason the look is there, and the run's own hold cannot help because the
// two runs would each be holding a marker the other could not parse.
func TestStartRefusesATaskWhoseMarkerWillNotRead(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "START-MARKER-1", "start over a broken marker", "quick")
	if err != nil {
		t.Fatal(err)
	}

	path, err := s.RunPath(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("pid: not a number\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	pid, err := Start(s, tk, "quick", 0)
	if err == nil {
		t.Fatal("Start spawned a run over a marker it could not read")
	}

	if pid != 0 {
		t.Errorf("Start returned pid %d alongside its refusal", pid)
	}
}
