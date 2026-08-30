package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/quota"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

func TestSettingsAdapterComprehensive(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	adapter, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings failed: %v", err)
	}

	// Autopilot
	if err := adapter.SetAutopilot(true); err != nil {
		t.Errorf("SetAutopilot: %v", err)
	}

	if !adapter.Autopilot() {
		t.Error("Autopilot() = false, want true")
	}

	// Language
	if err := adapter.SetLanguage("es"); err != nil {
		t.Errorf("SetLanguage: %v", err)
	}

	if adapter.Language() != "es" {
		t.Errorf("Language() = %q, want es", adapter.Language())
	}

	// UnreadCap
	if err := adapter.SetUnreadCap(42); err != nil {
		t.Errorf("SetUnreadCap: %v", err)
	}

	if adapter.UnreadCap() != 42 {
		t.Errorf("UnreadCap() = %d, want 42", adapter.UnreadCap())
	}

	// Engine
	if err := adapter.SetEngine("codex"); err != nil {
		t.Errorf("SetEngine: %v", err)
	}

	if adapter.Engine() != "codex" {
		t.Errorf("Engine() = %q, want codex", adapter.Engine())
	}

	// Model
	if err := adapter.SetModel("gpt-4o"); err != nil {
		t.Errorf("SetModel: %v", err)
	}

	if adapter.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want gpt-4o", adapter.Model())
	}

	// Flow
	if err := adapter.SetFlow("careful"); err != nil {
		t.Errorf("SetFlow: %v", err)
	}

	if adapter.Flow() != "careful" {
		t.Errorf("Flow() = %q, want careful", adapter.Flow())
	}

	// Theme
	if err := adapter.SetTheme("nord"); err != nil {
		t.Errorf("SetTheme: %v", err)
	}

	if adapter.Theme() != "nord" {
		t.Errorf("Theme() = %q, want nord", adapter.Theme())
	}
}

func TestSetCommandInvocations(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	// 1. Print all settings (0 args)
	var out, errOut bytes.Buffer

	code := Run([]string{"set"}, &out, &errOut)
	if code != 0 {
		t.Errorf("expected exit code 0 for `orbit set`, got %d: %s", code, errOut.String())
	}

	if !strings.Contains(out.String(), "language") || !strings.Contains(out.String(), "theme") {
		t.Errorf("output missing setting keys:\n%s", out.String())
	}

	// 2. Set with 1 arg (error)
	out.Reset()
	errOut.Reset()

	code = Run([]string{"set", "theme"}, &out, &errOut)
	if code == 0 {
		t.Error("expected error for `orbit set theme` without value")
	}

	// 3. Set with invalid key
	out.Reset()
	errOut.Reset()

	code = Run([]string{"set", "nonexistent-key", "val"}, &out, &errOut)
	if code == 0 {
		t.Error("expected error for unknown setting key")
	}

	// 4. Set valid keys
	for _, pair := range [][]string{
		{"language", "es"},
		{"autopilot", "on"},
		{"unread-cap", "7"},
		{"engine", "claude"},
		{"model", "claude-3-7-sonnet"},
		{"flow", "quick"},
		{"theme", "frauddi"},
	} {
		out.Reset()
		errOut.Reset()

		code = Run([]string{"set", pair[0], pair[1]}, &out, &errOut)
		if code != 0 {
			t.Errorf("failed setting %s to %s: %s", pair[0], pair[1], errOut.String())
		}
	}
}

func TestPortsHelpers(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(t.TempDir(), "repo")
	tView := view.Task{
		ID:       "TASK-1",
		Repo:     "repo",
		RepoPath: repoPath,
		Engine:   "fake",
	}

	// 1. subject conversion
	subj := subject(tView)
	if subj.ID != "TASK-1" || subj.Repo.Name != "repo" || subj.Repo.Path != repoPath {
		t.Errorf("subject(%+v) = %+v", tView, subj)
	}

	// 2. unknownEngineError string
	uErr := &unknownEngineError{Name: "custom", ID: "TASK-1"}
	if !strings.Contains(uErr.Error(), "custom") || !strings.Contains(uErr.Error(), "TASK-1") {
		t.Errorf("unexpected unknownEngineError text: %s", uErr.Error())
	}

	// 3. controlPort
	cPort := controlPort(s)
	// Task not created yet -> returns error
	if err := cPort(tView, "pause"); err == nil {
		t.Error("expected error controlling uncreated task")
	}

	// 4. markReadPort
	mPort := markReadPort(s)
	if err := mPort(tView); err != nil {
		t.Errorf("markReadPort failed: %v", err)
	}

	// 5. enginesPort
	engFn := enginesPort(map[string]engine.Engine{
		"claude":   engine.NewClaude(),
		"codex":    engine.NewCodex(),
		"opencode": engine.NewOpenCode(),
	})

	engList := engFn()
	if len(engList) == 0 {
		t.Error("enginesPort returned empty engines list")
	}

	// 6. takePort with empty engine
	takeP := takePort(nil, map[string]engine.Engine{"fake": engine.NewFake("ok")})

	cmd, err := takeP(view.Task{ID: "TASK-1"})
	if err != nil || cmd != nil {
		t.Errorf("takePort on empty engine = (%v, %v), want (nil, nil)", cmd, err)
	}

	// 7. takePort with unknown engine
	_, err = takeP(view.Task{ID: "TASK-1", Engine: "missing"})
	if err == nil {
		t.Error("expected unknownEngineError for missing engine")
	}

	// 8. startPort
	stPort := startPort(s)
	_, _ = stPort(tView, "quick", 5) //nolint:errcheck

	// 9. reconcileAll
	if err := reconcileAll(s); err != nil {
		t.Errorf("reconcileAll failed: %v", err)
	}
}

func TestDoPortAndTopHelpers(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	adapter, err := newSettings(s)
	if err != nil {
		t.Fatal(err)
	}

	dp := doPort(adapter)

	var buf bytes.Buffer

	// 1. Unknown command
	if err := dp("nonexistent_command_123", nil, &buf); err == nil {
		t.Error("expected error for unknown command in doPort")
	}

	// 2. Command refused in window (top)
	if err := dp("top", nil, &buf); err == nil {
		t.Error("expected error for top in doPort")
	}

	// 3. Command opening screen (set)
	buf.Reset()

	if err := dp("set", []string{"language", "en"}, &buf); err == nil || !strings.Contains(err.Error(), "opens a screen") {
		t.Errorf("expected WindowOpens error for set in doPort, got %v", err)
	}

	// 4. Command executing in window (reconcile)
	buf.Reset()
	_ = dp("reconcile", []string{"-repo", t.TempDir()}, &buf) //nolint:errcheck

	// 4. mustBeDirectory
	if err := mustBeDirectory(words.For("en"), "/nonexistent/directory/path"); err == nil {
		t.Error("expected error on nonexistent directory")
	}

	fileBlocker := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(fileBlocker, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := mustBeDirectory(words.For("en"), fileBlocker); err == nil {
		t.Error("expected error on regular file passed to mustBeDirectory")
	}

	// 5. interactive helper
	if interactive(&buf) {
		t.Error("expected interactive(buffer) to be false")
	}

	// 6. quotaPort
	if qp := quotaPort(nil, false); qp != nil {
		t.Error("expected quotaPort(nil) to be nil")
	}

	qc := quota.FromEnv()
	if qp := quotaPort(qc, false); qc != nil && qp == nil {
		t.Error("expected quotaPort(client) to be non-nil")
	}
}
