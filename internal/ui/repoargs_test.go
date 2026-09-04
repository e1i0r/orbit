package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// TestATaskWithNoRepositoryIsNotToldToUseThisOne. Orbit is opened on the
// directory above the repositories, so "." is the workspace and not a
// checkout. Passing it as -repo makes the command answer that it is not a
// repository — a sentence about Orbit's own working directory, in front of a
// reader who asked something about a task.
func TestATaskWithNoRepositoryIsNotToldToUseThisOne(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard([]view.Task{{
		ID: "ACME-9", Title: "written against nothing", Band: view.NeedsYou, Since: ago(time.Minute),
	}}, 1)

	if got := m.taskRepoPath("ACME-9"); got != "" {
		t.Errorf("a task with no repository answers %q, want nothing at all", got)
	}

	args := repoArgs(m.taskRepoPath("ACME-9"), "ACME-9")
	if strings.Join(args, " ") != "ACME-9" {
		t.Errorf("the command would be given %v, want the task and nothing else", args)
	}
}

// TestATaskWithARepositoryStillNamesIt.
func TestATaskWithARepositoryStillNamesIt(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard([]view.Task{{
		ID: "ACME-8", Title: "written against payments", Band: view.NeedsYou,
		Repo: "payments", RepoPath: "/work/payments", Since: ago(time.Minute),
	}}, 1)

	args := repoArgs(m.taskRepoPath("ACME-8"), "ACME-8")
	if strings.Join(args, " ") != "-repo /work/payments ACME-8" {
		t.Errorf("the command would be given %v, want the repository's path", args)
	}
}

// TestARepositoryNameIsNotAPath. A task carries the repository's name for
// the row to draw, and handing that to -repo is handing a command a
// directory that is not there.
func TestARepositoryNameIsNotAPath(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard([]view.Task{{
		ID: "ACME-7", Title: "a name and no path", Band: view.NeedsYou,
		Repo: "payments", Since: ago(time.Minute),
	}}, 1)

	if got := m.taskRepoPath("ACME-7"); got != "" {
		t.Errorf("a task with a name and no path answers %q, want nothing at all", got)
	}

	if got := m.taskRepoPath("ACME-404"); got != "" {
		t.Errorf("a task the board does not hold answers %q, want nothing at all", got)
	}
}

// TestTheArgumentsCarryWhateverFollowsThem, so the note verbs keep their
// text and their separator.
func TestTheArgumentsCarryWhateverFollowsThem(t *testing.T) {
	got := repoArgs("/work/payments", "ACME-1", "--", "fix the tests")
	if strings.Join(got, " ") != "-repo /work/payments ACME-1 -- fix the tests" {
		t.Errorf("repoArgs = %v", got)
	}
}

// TestEveryDeliverVerbReachesItsCommand. Merge and close are the two verbs
// of the deliver grid that are still one command each, and a task written
// against no repository must reach them without the workspace being handed
// over as its checkout.
func TestEveryDeliverVerbReachesItsCommand(t *testing.T) {
	for name, verb := range map[string]func(Model) (tea.Model, tea.Cmd){
		"merge PR": Model.mergePR,
		"close PR": Model.closePR,
	} {
		m, _ := testModel(t, 120, 30)
		m.board = fixtureBoard([]view.Task{{
			ID: "ACME-9", Title: "no repository at all", Band: view.NeedsYou, Since: ago(time.Minute),
		}}, 1)
		m.detail = "ACME-9"

		next, cmd := verb(m)
		if cmd == nil {
			t.Errorf("%s ran nothing", name)
		}

		if asModel(t, next).message == "" {
			t.Errorf("%s said nothing about what it was doing", name)
		}
	}
}

// TestTheVerbsHandedToTheSupervisorCarryTheTaskAndItsCheckout.
//
// The five that are not one command are one sentence to the supervisor, and
// the sentence is the whole of what it is given: a prompt that does not name
// the checkout sends it to work in the state directory, and a prompt that
// does not name the task sends it to work on whatever it last heard about.
func TestTheVerbsHandedToTheSupervisorCarryTheTaskAndItsCheckout(t *testing.T) {
	for name, c := range map[string]struct {
		verb    func(Model) (tea.Model, tea.Cmd)
		caption string
	}{
		"create PR":        {Model.deliverPR, "CREATE PR"},
		"update PR":        {Model.updatePRBranch, "UPDATE PR"},
		"fix checks":       {Model.fixChecks, "FIX CHECKS"},
		"more tests":       {Model.addMoreTests, "MORE TESTS"},
		"resolve comments": {Model.resolveComments, "RESOLVE COMMENTS"},
		"deep review":      {Model.reviewPR, "DEEP REVIEW"},
	} {
		m, _ := testModel(t, 120, 30)
		m.board = fixtureBoard([]view.Task{{
			ID: "ACME-9", Title: "a task in a checkout", Band: view.NeedsYou,
			Repo: "payments", RepoPath: "/work/payments", Since: ago(time.Minute),
		}}, 1)
		m.detail = "ACME-9"

		var said string

		m.opts.RecordSupervisor = func(_, _, text string) error {
			said = text
			return nil
		}

		next, cmd := c.verb(m)
		if cmd == nil {
			t.Errorf("%s asked for nothing", name)
		}

		if asModel(t, next).message == "" {
			t.Errorf("%s said nothing about what it was doing", name)
		}

		for _, want := range []string{"ACME-9", "/work/payments", c.caption} {
			if !strings.Contains(said, want) {
				t.Errorf("what %s asked the supervisor does not carry %q:\n%s", name, want, said)
			}
		}
	}
}

// TestAVerbForTheSupervisorNeedsACheckout. A task written against no
// repository — which FRA-61 made a thing a reader can do — has nowhere for
// the supervisor to run git, and a run spent finding that out is a run spent
// for nothing.
func TestAVerbForTheSupervisorNeedsACheckout(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard([]view.Task{{
		ID: "ACME-9", Title: "no repository at all", Band: view.NeedsYou, Since: ago(time.Minute),
	}}, 1)
	m.detail = "ACME-9"

	next, cmd := m.fixChecks()
	if cmd != nil {
		t.Error("a task with no checkout still asked the supervisor to work in one")
	}

	if got := asModel(t, next).message; !strings.Contains(got, "checkout") {
		t.Errorf("it said %q, want it saying there is nowhere to work", got)
	}
}

// TestADeliverVerbWithNoTaskSaysSo, rather than running a command about
// whichever task the cursor last touched.
func TestADeliverVerbWithNoTaskSaysSo(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard(nil, 1)
	m.detail = ""

	next, cmd := m.deliverPR()
	if cmd != nil {
		t.Error("a deliver verb with no task in front of it ran a command")
	}

	if !strings.Contains(asModel(t, next).message, "select") {
		t.Errorf("it said %q, want it asking for a task", asModel(t, next).message)
	}
}
