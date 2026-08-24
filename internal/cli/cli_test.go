package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func workspace(t *testing.T) (root string, orbitHome string) {
	t.Helper()
	root = t.TempDir()
	orbitHome = filepath.Join(t.TempDir(), "orbit")
	initRepo(t, filepath.Join(root, "payments"))
	t.Setenv("ORBIT_HOME", orbitHome)
	return root, orbitHome
}

// initRepo makes one git repository at dir: a branch, a first commit and a
// remote, which is the least repo.Open needs to read a shape out of it.
//
// It is a function of its own rather than four lines inside workspace
// because a board is only a board over more than one repository — the frame
// `orbit top -once` draws gives the repository column up when every row
// would carry the same value in it, so top_test.go builds two.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	home := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"remote", "add", "acme", "https://example.invalid/x.git"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			// The developer's real ~/.gitconfig must never leak into the
			// test: a global commit.gpgsign or hooksPath would make this
			// suite pass or fail depending on whose machine runs it.
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"HOME="+home,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var o, e bytes.Buffer
	code = Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestNoArgumentsPrintsUsage(t *testing.T) {
	code, out, _ := run(t)
	if code == 0 {
		t.Error("exit code 0 with no command")
	}
	if !strings.Contains(out, "orbit") {
		t.Errorf("usage does not mention the command:\n%s", out)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, errOut := run(t, "fly")
	if code == 0 {
		t.Error("an unknown command exited 0")
	}
	if !strings.Contains(errOut, "fly") {
		t.Errorf("the error does not name the command:\n%s", errOut)
	}
}

func TestReposListsWhatIsUnderTheRoot(t *testing.T) {
	root, _ := workspace(t)
	code, out, errOut := run(t, "repos", root)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "payments") {
		t.Errorf("output does not list the repository:\n%s", out)
	}
	if !strings.Contains(out, "acme") {
		t.Errorf("output does not show the remote name:\n%s", out)
	}
}

func TestNewThenListThenShow(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "retry the webhook on 5xx"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "list", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("list exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "ACME-1") {
		t.Errorf("list does not show the task:\n%s", out)
	}

	code, out, errOut = run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "task.created") {
		t.Errorf("show does not print the record:\n%s", out)
	}
	if !strings.Contains(out, "retry the webhook on 5xx") {
		t.Errorf("show does not print what was asked for:\n%s", out)
	}
}

// TestRunNeedsAnID, TestRunFailsOnAnUnknownFlow and
// TestRunFailsOnATaskThatWasNeverCreated cover runTask's early exits — the
// guard rails on the one command that spends money. Each is built so it
// returns from runTask before engine.NewClaude() is ever constructed: no id
// fails at the `id == ""` check before openBoth is even called, and an id
// that was never created fails inside task.Load, which now comes before the
// flow is resolved — the run walks the task's own flow unless -flow says
// otherwise, so which flow it is cannot be known until the task is loaded.
// None of these tests may ever reach the real claude binary.
//
// That ordering is why the unknown-flow test below is also given an id that
// was never created: it is refused twice over, and the second refusal is
// what makes it safe. The refusal a reader actually meets — the name, and
// the list of names that would have worked — is asserted in
// internal/flow/resolve_test.go, where no engine exists to reach.

func TestRunNeedsAnID(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")
	code, _, errOut := run(t, "run", "-repo", repoDir)
	if code == 0 {
		t.Error("run with no id exited 0")
	}
	if errOut == "" {
		t.Error("run failed silently")
	}
}

func TestRunFailsOnAnUnknownFlow(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")
	code, _, errOut := run(t, "run", "-repo", repoDir, "-flow", "does-not-exist", "ACME-1")
	if code == 0 {
		t.Error("run with an unknown flow exited 0")
	}
	if errOut == "" {
		t.Error("run failed silently")
	}
}

func TestRunFailsOnATaskThatWasNeverCreated(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")
	code, _, errOut := run(t, "run", "-repo", repoDir, "ACME-404")
	if code == 0 {
		t.Error("run succeeded against a task that was never created")
	}
	if errOut == "" {
		t.Error("run failed silently")
	}
}

func TestNewRefusesADirectoryThatIsNotARepository(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	code, _, errOut := run(t, "new", "-repo", t.TempDir(), "-id", "ACME-1", "x")
	if code == 0 {
		t.Error("new succeeded outside a repository")
	}
	if errOut == "" {
		t.Error("new failed silently")
	}
}
