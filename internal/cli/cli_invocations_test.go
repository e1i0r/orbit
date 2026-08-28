package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllCliCommandsInvocations(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	repoPath := filepath.Join(t.TempDir(), "payments")
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

	var out, errOut bytes.Buffer

	// 1. orbit repos
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"repos", t.TempDir()}, &out, &errOut); code != 0 {
		t.Errorf("orbit repos failed: %d: %s", code, errOut.String())
	}
	// repos too many args
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"repos", "/a", "/b"}, &out, &errOut); code == 0 {
		t.Error("expected error on repos with too many args")
	}

	// 2. orbit flows
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"flows"}, &out, &errOut); code != 0 {
		t.Errorf("orbit flows failed: %d: %s", code, errOut.String())
	}

	// 3. orbit new
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"new"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit new")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"new", "-repo", repoPath, "-id", "PAY-1"}, &out, &errOut); code == 0 {
		t.Error("expected error on orbit new without text")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"new", "-repo", repoPath, "-id", "PAY-1", "Fix stripe webhooks"}, &out, &errOut); code != 0 {
		t.Errorf("orbit new failed: %d: %s", code, errOut.String())
	}

	// 4. orbit list
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"list", "-repo", "/nonexistent/repo/dir"}, &out, &errOut); code == 0 {
		t.Error("expected error on orbit list with nonexistent -repo")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"list", "-repo", repoPath}, &out, &errOut); code != 0 {
		t.Errorf("orbit list failed: %d: %s", code, errOut.String())
	}

	// 5. orbit show
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"show"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit show")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"show", "-repo", repoPath, "PAY-1"}, &out, &errOut); code != 0 {
		t.Errorf("orbit show failed: %d: %s", code, errOut.String())
	}

	// 6. orbit note
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"note"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit note")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"note", "-repo", repoPath, "PAY-1", "Operator note test"}, &out, &errOut); code != 0 {
		t.Errorf("orbit note failed: %d: %s", code, errOut.String())
	}

	// 7. orbit pause & resume
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"pause"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit pause")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"pause", "-repo", repoPath, "PAY-1"}, &out, &errOut); code != 0 {
		t.Errorf("orbit pause failed: %d: %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"resume", "-repo", repoPath, "PAY-1"}, &out, &errOut); code != 0 {
		t.Errorf("orbit resume failed: %d: %s", code, errOut.String())
	}

	// 8. orbit read
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"read"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit read")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"read", "-repo", repoPath, "PAY-1"}, &out, &errOut); code != 0 {
		t.Errorf("orbit read failed: %d: %s", code, errOut.String())
	}

	// 9. orbit cancel
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"cancel"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit cancel")
	}

	out.Reset()
	errOut.Reset()
	// Cancel on non-running task returns error
	if code := Run([]string{"cancel", "-now", "-repo", repoPath, "PAY-1"}, &out, &errOut); code == 0 {
		t.Error("expected error cancelling non-running task")
	}

	// 10. orbit reconcile
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"reconcile", "-repo", "/nonexistent/repo"}, &out, &errOut); code == 0 {
		t.Error("expected error on orbit reconcile with invalid -repo")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"reconcile", "-repo", repoPath}, &out, &errOut); code != 0 {
		t.Errorf("orbit reconcile -repo failed: %d: %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"reconcile", "-repo", repoPath, "PAY-1"}, &out, &errOut); code != 0 {
		t.Errorf("orbit reconcile -repo task failed: %d: %s", code, errOut.String())
	}

	// 11. orbit run validation errors
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"run"}, &out, &errOut); code == 0 {
		t.Error("expected error on empty orbit run")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"run", "-repo", repoPath, "NONEXISTENT-99"}, &out, &errOut); code == 0 {
		t.Error("expected error running nonexistent task")
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"run", "-repo", repoPath, "-flow", "nonexistent-flow", "PAY-1"}, &out, &errOut); code == 0 {
		t.Error("expected error running task with nonexistent flow")
	}

	// 12. orbit top -once
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"top", "-once", repoPath}, &out, &errOut); code != 0 {
		t.Errorf("orbit top -once failed: %d: %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := Run([]string{"top", "-once", "-lang", "es", repoPath}, &out, &errOut); code != 0 {
		t.Errorf("orbit top -once -lang es failed: %d: %s", code, errOut.String())
	}
}

func TestCliHelpAndVersion(t *testing.T) {
	var out, errOut bytes.Buffer

	// Help flags
	if code := Run([]string{"--help"}, &out, &errOut); code != 0 {
		t.Errorf("expected 0 for --help, got %d", code)
	}

	if !strings.Contains(out.String(), "Usage:") && !strings.Contains(out.String(), "orbit") {
		t.Errorf("unexpected help output: %s", out.String())
	}

	// Unknown command
	out.Reset()
	errOut.Reset()

	if code := Run([]string{"nonexistent-subcommand"}, &out, &errOut); code == 0 {
		t.Error("expected non-zero exit for unknown subcommand")
	}
}
