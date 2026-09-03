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

// TestEveryDeliverVerbReachesItsCommand. The eight verbs of the deliver grid
// are eight call sites of one pattern, and the one that is never exercised
// is the one that still hands a command the workspace.
func TestEveryDeliverVerbReachesItsCommand(t *testing.T) {
	for name, verb := range map[string]func(Model) (tea.Model, tea.Cmd){
		"create PR":        Model.deliverPR,
		"update PR":        Model.updatePRBranch,
		"merge PR":         Model.mergePR,
		"close PR":         Model.closePR,
		"fix checks":       Model.fixChecks,
		"more tests":       Model.addMoreTests,
		"resolve comments": Model.resolveComments,
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
