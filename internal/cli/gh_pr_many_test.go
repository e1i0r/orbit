package cli

// Delivering a task that was worked in more than one repository, against a
// gh that is a shell script.
//
// A repository joins a task by being worked in, so what arrives at delivery
// is not one repository and one branch but two or three of each. What these
// tests read is the two things a reviewer depends on: that each repository
// with work in it got a pull request of its own, and that every one of those
// bodies says where the rest of the task is.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// twoRepositories is a workspace where one task has been worked in both. It
// was written against payments, and ledger joined it the way a phase joins
// one: by asking for a checkout of it halfway through.
func twoRepositories(t *testing.T, text string) (pay, led string) {
	t.Helper()

	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("ORBIT_HOME", filepath.Join(t.TempDir(), "orbit"))

	root := t.TempDir()
	t.Setenv(repo.WorkspaceEnv, root)

	pay = withRemote(t, root, "payments")
	led = withRemote(t, root, "ledger")

	if code, _, errOut := run(t, "new", "-repo", pay, "-id", "PAY-1", text); code != 0 {
		t.Fatalf("orbit new exited %d: %s", code, errOut)
	}

	plantWorktree(t, pay, text)

	if code, _, errOut := run(t, "join", "-repo", pay, "-task", "PAY-1", "ledger"); code != 0 {
		t.Fatalf("orbit join exited %d: %s", code, errOut)
	}

	return pay, led
}

// changeIn leaves work in the task's checkout of a repository, which is what
// a phase that reached into it would have left behind.
func changeIn(t *testing.T, dir, text string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(worktreeOf(t, dir), "done.txt"), []byte(text+"\n"), 0o600); err != nil {
		t.Fatalf("write the work: %v", err)
	}
}

// everyCall is a fake gh that appends the arguments of every call to one
// file, with a line of dashes between calls, and answers each pull request
// it is asked to open with a URL of its own.
//
// The fake the other delivery tests use overwrites what it wrote, which is
// enough while a delivery shells out once. This one has to be read call by
// call: what is being tested is that gh was asked several times, and what it
// was asked the second time.
func everyCall(t *testing.T) (log string) {
	t.Helper()

	log = filepath.Join(t.TempDir(), "calls")
	fakeGh(t, "printf '%s\\n' \"$@\" >> "+log+"\n"+
		"echo ---- >> "+log+"\n"+
		"if [ \"$1 $2\" = 'pr create' ]; then echo https://github.test/pr/$(grep -c '^----$' "+log+"); fi")

	return log
}

// calls is what gh was given, one element per call, in the order it was
// asked.
func calls(t *testing.T, log string) [][]string {
	t.Helper()

	written, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read what gh was given: %v", err)
	}

	var made [][]string

	for _, one := range strings.Split(string(written), "----\n") {
		if strings.TrimSpace(one) == "" {
			continue
		}

		made = append(made, strings.Split(strings.TrimSuffix(one, "\n"), "\n"))
	}

	return made
}

// asked is every call gh was given for one of its commands, in order.
func asked(made [][]string, command string) [][]string {
	var found [][]string

	for _, one := range made {
		if len(one) > 1 && one[0]+" "+one[1] == command {
			found = append(found, one)
		}
	}

	return found
}

// TestATaskWorkedInTwoRepositoriesIsDeliveredToBoth. One pull request cannot
// carry the work of two repositories: they are two branches on two remotes,
// and there is nowhere for the second one to go.
func TestATaskWorkedInTwoRepositoriesIsDeliveredToBoth(t *testing.T) {
	pay, led := twoRepositories(t, "retry the webhook on 5xx")
	changeIn(t, led, "the ledger side")

	log := everyCall(t)

	code, out, errOut := run(t, "pr", "-repo", pay, "PAY-1")
	if code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	for _, want := range []string{"payments", "ledger", "https://github.test/pr/1", "https://github.test/pr/2"} {
		if !strings.Contains(out, want) {
			t.Errorf("the delivery does not say %q:\n%s", want, out)
		}
	}

	if made := asked(calls(t, log), "pr create"); len(made) != 2 {
		t.Errorf("gh opened %d pull requests, want one per repository", len(made))
	}
}

// TestEachPullRequestOfATaskNamesTheOthers. Merging one half of a pair
// without the other is how a broken pair ships, and the body in front of a
// reviewer is the only place they will see that the other half exists.
func TestEachPullRequestOfATaskNamesTheOthers(t *testing.T) {
	pay, led := twoRepositories(t, "retry the webhook on 5xx")
	changeIn(t, led, "the ledger side")

	log := everyCall(t)
	if code, _, errOut := run(t, "pr", "-repo", pay, "PAY-1"); code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	edits := asked(calls(t, log), "pr edit")
	if len(edits) != 2 {
		t.Fatalf("gh was asked to rewrite %d bodies, want one per pull request", len(edits))
	}

	// A body names the others and not itself, which is what tells the two
	// rewrites apart: each one points at the pull request it is not.
	payments, ledger := strings.Join(edits[0], "\n"), strings.Join(edits[1], "\n")
	if !strings.Contains(payments, "ledger") || !strings.Contains(payments, "https://github.test/pr/2") {
		t.Errorf("the pull request of payments does not name the one of ledger:\n%s", payments)
	}

	if !strings.Contains(ledger, "payments") || !strings.Contains(ledger, "https://github.test/pr/1") {
		t.Errorf("the pull request of ledger does not name the one of payments:\n%s", ledger)
	}
}

// TestARepositoryTheWorkLeftAloneGetsNoPullRequest. A task may open a
// checkout of a repository, read it and find it needs nothing. It joined,
// and the record says so; what it does not get is a pull request asking
// somebody to review no change at all.
func TestARepositoryTheWorkLeftAloneGetsNoPullRequest(t *testing.T) {
	pay, _ := twoRepositories(t, "retry the webhook on 5xx")

	log := everyCall(t)

	code, out, errOut := run(t, "pr", "-repo", pay, "PAY-1")
	if code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	if made := asked(calls(t, log), "pr create"); len(made) != 1 {
		t.Fatalf("gh opened %d pull requests, want one: ledger has nothing to show", len(made))
	}

	if !strings.Contains(out, "ledger") || !strings.Contains(out, "nothing changed") {
		t.Errorf("the delivery does not say ledger was left as it was:\n%s", out)
	}

	// The pull request that was opened still names it. Whether the task
	// looked at ledger at all is a question its reviewer would otherwise have
	// to go and ask somebody.
	edits := asked(calls(t, log), "pr edit")
	if len(edits) != 1 || !strings.Contains(strings.Join(edits[0], "\n"), "ledger") {
		t.Errorf("the pull request of payments says nothing about ledger: %v", edits)
	}
}

// TestPullRequestsThatCouldNotBeLinkedSaySoRatherThanReadAsAFailedDelivery.
// The bodies are rewritten after every pull request is open, so a gh that
// refuses at that point has not stopped the delivery — it has left a set of
// pull requests that do not name each other, which is a different thing to
// go and fix.
func TestPullRequestsThatCouldNotBeLinkedSaySoRatherThanReadAsAFailedDelivery(t *testing.T) {
	pay, led := twoRepositories(t, "retry the webhook on 5xx")
	changeIn(t, led, "the ledger side")

	fakeGh(t, "if [ \"$1 $2\" = 'pr edit' ]; then echo 'could not update pull request: HTTP 422' >&2; exit 1; fi\n"+
		"echo https://github.test/pr/1")

	code, out, errOut := run(t, "pr", "-repo", pay, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 against a gh that would not rewrite a body")
	}

	if !strings.Contains(errOut, "HTTP 422") {
		t.Errorf("the refusal does not carry what gh said: %q", errOut)
	}

	// And the reader is still told where the pull requests are: they are
	// open, and this is the only place their URLs are printed.
	if !strings.Contains(out, "https://github.test/pr/1") {
		t.Errorf("the pull requests were opened and not named:\n%s", out)
	}
}

// TestARepositoryOfTheTaskThatIsGoneStopsTheDelivery. A repository joined
// the task and its checkout is no longer on the disk. Delivering the rest
// and saying nothing would read as a task that was delivered whole.
func TestARepositoryOfTheTaskThatIsGoneStopsTheDelivery(t *testing.T) {
	pay, led := twoRepositories(t, "retry the webhook on 5xx")
	changeIn(t, led, "the ledger side")

	if err := os.RemoveAll(led); err != nil {
		t.Fatalf("take the repository away: %v", err)
	}

	log := everyCall(t)

	code, _, errOut := run(t, "pr", "-repo", pay, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 with a repository of the task missing")
	}

	if !strings.Contains(errOut, "ledger") {
		t.Errorf("the refusal does not name the repository that is gone: %q", errOut)
	}

	if _, err := os.Stat(log); err == nil {
		t.Error("gh was asked for a pull request before the delivery was known to be possible")
	}
}
