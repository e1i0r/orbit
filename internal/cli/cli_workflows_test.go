package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/view"
)

func TestCancelTaskExecution(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@orbit.local"},
		{"config", "user.name", "Orbit Tester"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)

		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	r, err := repo.Open(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	tk, err := task.Create(s, r, "TASK-CANC", "Cancel CLI test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Start a background process and plant its run marker
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }() //nolint:errcheck

	runPath, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(runPath), 0o700); err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf("pid: %d\nstarted: 2026-08-24T12:00:00Z\n", cmd.Process.Pid)
	if err := os.WriteFile(runPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	// Test cancel graceful
	_ = Run([]string{"cancel", "-repo", repoPath, tk.ID}, &out, &errOut)

	// Plant marker again for -now test
	_ = os.WriteFile(runPath, []byte(body), 0o600) //nolint:errcheck

	// Test cancel -now
	out.Reset()
	errOut.Reset()
	_ = Run([]string{"cancel", "-now", "-repo", repoPath, tk.ID}, &out, &errOut)
}

func TestFullScreenAndTakePortWithReader(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	rdr := board.NewReader(s, root)
	engines := map[string]engine.Engine{
		"claude": engine.NewClaude(),
		"fake":   engine.NewFake("output"),
	}

	// 1. takePort with fake engine that cannot resume
	tView := view.Task{
		ID:       "TASK-1",
		RepoPath: root,
		Engine:   "fake",
	}
	tp := takePort(rdr, engines)

	_, err = tp(tView)
	if err == nil || !strings.Contains(err.Error(), "cannot resume") {
		t.Errorf("expected cannot resume error for fake engine, got %v", err)
	}

	// 2. takePort with claude engine (no session -> nil cmd, nil err)
	tViewClaude := view.Task{
		ID:       "TASK-1",
		RepoPath: root,
		Engine:   "claude",
	}

	cmd, err := tp(tViewClaude)
	if err != nil || cmd != nil {
		t.Errorf("takePort claude with empty session = (%v, %v), want (nil, nil)", cmd, err)
	}

	// 3. fullScreen wrapper Update & View
	opts := ui.Options{
		Width:  100,
		Height: 30,
		Reader: rdr,
	}
	baseUI := ui.New(opts)
	fs := fullScreen{baseUI}

	next, _ := fs.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if next == nil {
		t.Error("expected non-nil model from fullScreen.Update")
	}

	v := fs.View()
	if !v.AltScreen {
		t.Error("expected AltScreen to be true in fullScreen.View")
	}
}

func TestNotePauseResumeReadShowCommands(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@orbit.local"},
		{"config", "user.name", "Orbit Tester"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)

		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	r, err := repo.Open(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	tk, err := task.Create(s, r, "TASK-CLI-FULL", "Full CLI test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer

	// 1. orbit note
	if code := Run([]string{"note", "-repo", repoPath, tk.ID, "A", "helpful", "operator", "note"}, &out, &errOut); code != 0 {
		t.Errorf("orbit note failed: %d: %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"note", "-repo", repoPath, tk.ID}, &out, &errOut); code == 0 {
		t.Error("expected error on note with empty text")
	}

	// 2. orbit pause & resume
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"pause", "-repo", repoPath, tk.ID}, &out, &errOut); code != 0 {
		t.Errorf("orbit pause failed: %d: %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"resume", "-repo", repoPath, tk.ID}, &out, &errOut); code != 0 {
		t.Errorf("orbit resume failed: %d: %s", code, errOut.String())
	}

	// 3. orbit read
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"read", "-repo", repoPath, tk.ID}, &out, &errOut); code != 0 {
		t.Errorf("orbit read failed: %d: %s", code, errOut.String())
	}

	// 4. orbit show
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"show", "-repo", repoPath, tk.ID}, &out, &errOut); code != 0 {
		t.Errorf("orbit show failed: %d: %s", code, errOut.String())
	}

	if !strings.Contains(out.String(), tk.ID) && !strings.Contains(out.String(), "task.created") {
		t.Errorf("expected show output to contain events, got %s", out.String())
	}

	// 5. stamp helper
	if stamp(time.Time{}) != "—" {
		t.Errorf("stamp(zero) = %q, want —", stamp(time.Time{}))
	}
}

func TestSetCommandComprehensive(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	var out, errOut bytes.Buffer

	// 1. orbit set with no args (prints table)
	if code := Run([]string{"set"}, &out, &errOut); code != 0 {
		t.Errorf("orbit set failed: %d: %s", code, errOut.String())
	}

	if !strings.Contains(out.String(), "language") || !strings.Contains(out.String(), "autopilot") {
		t.Errorf("expected settings table in output: %s", out.String())
	}

	// 2. orbit set with 1 arg (error)
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"set", "autopilot"}, &out, &errOut); code == 0 {
		t.Error("expected error on set with missing value")
	}

	// 3. orbit set with unknown key
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"set", "unknown_key_99", "value"}, &out, &errOut); code == 0 {
		t.Error("expected error on set with unknown key")
	}

	// 4. orbit set all settings
	for _, pair := range [][]string{
		{"autopilot", "on"},
		{"autopilot", "off"},
		{"language", "es"},
		{"unread-cap", "12"},
		{"engine", "claude"},
		{"model", "claude-3-7-sonnet-latest"},
		{"flow", "quick"},
		{"theme", "frauddi"},
	} {
		out.Reset()
		errOut.Reset()

		if code := Run([]string{"set", pair[0], pair[1]}, &out, &errOut); code != 0 {
			t.Errorf("orbit set %s %s failed: %d: %s", pair[0], pair[1], code, errOut.String())
		}
	}

	// 5. orbit run execution path
	out.Reset()
	errOut.Reset()
	_ = Run([]string{"run", "-timeout", "10s", "-repo", root, "NONEXISTENT-1"}, &out, &errOut)
}
