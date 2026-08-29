package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userFlow drops one flow file into $ORBIT_HOME/flows, which is the whole
// of the extension mechanism: a file, and no code.
func userFlow(t *testing.T, orbitHome, name string) {
	t.Helper()

	dir := filepath.Join(orbitHome, "flows")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	body := `{"name":"` + name + `","phases":[{"name":"implement","engine":"claude","model":"sonnet"}]}`
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// listed is the flows one line each, with the blank tail dropped.
func listed(t *testing.T, out string) []string {
	t.Helper()

	var lines []string

	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}

	return lines
}

func TestFlowsListsTheBuiltinsOnACleanStateRoot(t *testing.T) {
	workspace(t)

	code, out, errOut := run(t, "flows")
	if code != 0 {
		t.Fatalf("flows exited %d: %s", code, errOut)
	}

	lines := listed(t, out)
	if len(lines) != 4 {
		t.Fatalf("flows listed %d flows, want the four builtins:\n%s", len(lines), out)
	}

	for _, name := range []string{"careful", "quick", "task", "tdd-fuzz-pr"} {
		if !strings.Contains(out, name+" (built in)") {
			t.Errorf("flows does not offer %q as a built-in:\n%s", name, out)
		}
	}
}

func TestAFlowFileIsListedBesideTheBuiltins(t *testing.T) {
	_, orbitHome := workspace(t)
	userFlow(t, orbitHome, "mine")

	code, out, errOut := run(t, "flows")
	if code != 0 {
		t.Fatalf("flows exited %d: %s", code, errOut)
	}

	if lines := listed(t, out); len(lines) != 5 {
		t.Fatalf("flows listed %d flows, want five:\n%s", len(lines), out)
	}

	if !strings.Contains(out, "mine (yours)") {
		t.Errorf("flows does not say where mine came from:\n%s", out)
	}
}

// A file that shadows a built-in name shadows it deliberately, and a flow
// that stopped behaving as documented is then one command away from
// explaining itself.
func TestAFileThatShadowsABuiltinSaysSo(t *testing.T) {
	_, orbitHome := workspace(t)
	userFlow(t, orbitHome, "task")

	code, out, errOut := run(t, "flows")
	if code != 0 {
		t.Fatalf("flows exited %d: %s", code, errOut)
	}

	if lines := listed(t, out); len(lines) != 4 {
		t.Fatalf("a shadowed built-in was listed twice:\n%s", out)
	}

	if !strings.Contains(out, "task (yours, shadowing the built-in)") {
		t.Errorf("flows does not mark the shadow:\n%s", out)
	}
}

// The mark is a sentence, and every sentence Orbit shows a person goes
// through the catalogue. This is the test that says so for this one: the
// same three facts are drawn by the window's start dialog through the same
// three keys. English written inside internal/flow, spliced into a Sprintf
// as data, cannot be translated and cannot be seen by the tests that check
// translations exist.
//
// The language is the saved setting. Nothing here reads $LANG, so this test
// says the same thing on a machine running in Spanish.
func TestTheListingIsInTheReadersOwnLanguage(t *testing.T) {
	_, orbitHome := workspace(t)
	userFlow(t, orbitHome, "mine")
	userFlow(t, orbitHome, "task")

	if code, _, errOut := run(t, "set", "language", "es"); code != 0 {
		t.Fatalf("set language exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "flows")
	if code != 0 {
		t.Fatalf("flows exited %d: %s", code, errOut)
	}

	for _, want := range []string{"careful (predeterminado)", "mine (tuyo)", "task (tuyo, reemplaza el predeterminado)"} {
		if !strings.Contains(out, want) {
			t.Errorf("the listing does not say %q:\n%s", want, out)
		}
	}
}

func TestNewSaysWhichFlowTheTaskWasWrittenAgainst(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, out, errOut := run(t, "new", "-repo", repoDir, "-flow", "careful", "-id", "ACME-1", "retry the webhook on 5xx")
	if code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "careful") {
		t.Errorf("new does not say which flow the task will walk:\n%s", out)
	}
}

// The flow is validated where it is walked, not where the task is written:
// a task written against a flow that is later deleted is still a task, and
// the sentence somebody typed is the one thing nobody can regenerate.
func TestNewAgainstAFlowThatDoesNotExistStillWritesTheTaskDown(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-flow", "nonesuch", "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "list", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("list exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("the task was not written down:\n%s", out)
	}
}
