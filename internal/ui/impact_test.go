package ui

// The impact pane: what this change reaches beyond the files it touched.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// reading is a window with a reading already in it.
func reading(t *testing.T, got repo.Impact) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &fakeReader{}
	m.weigh.reach, m.weigh.reachKnown = got, true

	return m
}

// TestEverySectionSaysWhatItIsBeforeWhatItFound. This reading is not one a
// reader has met before: a list of filenames under a heading nobody
// understands is a list nobody acts on.
func TestEverySectionSaysWhatItIsBeforeWhatItFound(t *testing.T) {
	m := reading(t, repo.Impact{
		Changed:   []string{"pricing.py"},
		Commits:   500,
		Coupled:   []repo.Coupled{{File: "invoice.py", With: "pricing.py", Times: 34, Of: 39}},
		Contracts: []repo.Contract{{File: "test_pricing.py", Says: "rejects negative amounts"}},
	})

	rows := strings.Join(m.impactRows(), "\n")

	for _, want := range []string{
		"WHAT USUALLY COMES ALONG",
		"committed together",
		"read from the last 500 commits",
		"not a rule",
		"invoice.py",
		"87%",
		"(34/39)",
		"WHAT THOSE TESTS SAY THEY HOLD",
		"Nothing here was run",
		"rejects negative amounts",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("the pane does not say %q:\n%s", want, rows)
		}
	}
}

// TestTheMarkOnTheStripCountsWhatWasFound, and nothing is not a number.
func TestTheMarkOnTheStripCountsWhatWasFound(t *testing.T) {
	m := reading(t, repo.Impact{
		Changed: []string{"pricing.py"},
		Coupled: []repo.Coupled{
			{File: "invoice.py", With: "pricing.py", Times: 34, Of: 39},
			{File: "ledger.py", With: "pricing.py", Times: 30, Of: 39},
		},
	})

	if got := m.impactMark(); !strings.Contains(got, "2") {
		t.Errorf("the mark is %q", got)
	}

	quiet := reading(t, repo.Impact{Changed: []string{"pricing.py"}})
	if got := quiet.impactMark(); got != "" {
		t.Errorf("a reading that found nothing marked the strip with %q", got)
	}

	// And the strip carries it, so every place the tabs are drawn agrees.
	found := false

	for _, n := range m.tabNames() {
		if n.tab == tabImpact && strings.Contains(n.text, "2") {
			found = true
		}
	}

	if !found {
		t.Error("the tab strip does not carry the mark")
	}
}

// TestTheFourStatesAreToldApart: not read, reading, refused, and nothing to
// weigh are four different facts and the pane says which.
func TestTheFourStatesAreToldApart(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &fakeReader{}

	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "nothing read yet") {
		t.Errorf("before anything was read the pane says %q", got)
	}

	m.weigh.reachAsking = true
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "reading the history") {
		t.Errorf("while reading the pane says %q", got)
	}

	m.weigh.reachAsking, m.weigh.reachKnown, m.weigh.reachErr = false, true, errors.New("not a git repository")
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "not a git repository") {
		t.Errorf("a refused reading says %q", got)
	}

	m.weigh.reachErr = nil
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "changed no files") {
		t.Errorf("a task with no changes says %q", got)
	}
}

// TestAReadingForAnotherTaskIsDropped. The view moves while a git log is out,
// and what comes back is about the task the reader has left.
func TestAReadingForAnotherTaskIsDropped(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.detail = "ACME-2"

	late := impactMsg{id: "ACME-1", impact: repo.Impact{Changed: []string{"pricing.py"}}}
	if got := m.tookImpact(late); got.weigh.reachKnown {
		t.Error("a reading about another task landed on this one")
	}
}

// TestTheAgentsOwnAccountIsMarkedAsSuch. The two sections above it are the
// repository's history; this one is a claim by the thing whose work is being
// weighed, and nothing verified it.
func TestTheAgentsOwnAccountIsMarkedAsSuch(t *testing.T) {
	m := reading(t, repo.Impact{Changed: []string{"pricing.py"}, Commits: 120})
	m.entries = []view.Entry{
		{Kind: "task.delta", Delta: &view.Delta{
			Needs:   []string{"amount must be a Decimal"},
			Instead: []string{"rounding per currency, dropped for the configuration it would need"},
		}},
	}

	rows := strings.Join(m.impactRows(), "\n")

	for _, want := range []string{
		"WHAT THE AGENT SAYS IT DID",
		"Nobody verified it",
		"callers must now",
		"amount must be a Decimal",
		"considered and not taken",
		"rounding per currency",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("the pane does not say %q:\n%s", want, rows)
		}
	}

	// It does not count as a warning: nothing here was checked, and a mark
	// on the strip would read as one.
	if got := m.impactMark(); got != "" {
		t.Errorf("the agent's own account marked the strip with %q", got)
	}
}

// TestTheAttemptThatStandsIsTheOneDrawn: a task run three times said this
// three times, and the two before it are about work that was thrown away.
func TestTheAttemptThatStandsIsTheOneDrawn(t *testing.T) {
	m := reading(t, repo.Impact{Changed: []string{"pricing.py"}})
	m.entries = []view.Entry{
		{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"the first attempt said this"}}},
		{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"and the one that stands says this"}}},
	}

	if got := m.lastDelta(); got == nil || got.Needs[0] != "and the one that stands says this" {
		t.Errorf("the pane draws %+v", got)
	}
}

// TestTheVerifiedThingComesFirst. The checks have an exit code behind them,
// the history is a pattern, and the delta is a claim: a reader deciding what
// to do next should meet them in that order.
func TestTheVerifiedThingComesFirst(t *testing.T) {
	m := reading(t, repo.Impact{
		Changed: []string{"pricing.py"},
		Coupled: []repo.Coupled{{File: "invoice.py", With: "pricing.py", Times: 34, Of: 39}},
	})
	m.weigh.checksKnown = true
	m.weigh.checks = []repo.Divergence{{
		Check: repo.Check{Name: "tests", Command: "go test ./..."},
		Base:  repo.Ran{Exit: 0},
		Now:   repo.Ran{Exit: 1, Out: "3 tests failing"},
	}}
	m.entries = []view.Entry{{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"a Decimal now"}}}}

	rows := strings.Join(m.impactRows(), "\n")

	checks := strings.Index(rows, "WHAT THE CHECKS SAY")
	history := strings.Index(rows, "WHAT USUALLY COMES ALONG")
	claim := strings.Index(rows, "WHAT THE AGENT SAYS")

	if checks < 0 || history < 0 || claim < 0 {
		t.Fatalf("a section is missing:\n%s", rows)
	}

	if checks >= history || history >= claim {
		t.Errorf("the sections are ordered checks=%d history=%d claim=%d", checks, history, claim)
	}

	for _, want := range []string{"passed before, fails now", "base:     passed", "worktree: exit 1", "3 tests failing"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the comparison does not say %q:\n%s", want, rows)
		}
	}
}

// TestTheComparisonIsOfferedBeforeItIsRun, with what it would run and what
// it costs: a test suite twice on somebody's machine is not something to
// start on their behalf.
func TestTheComparisonIsOfferedBeforeItIsRun(t *testing.T) {
	m := reading(t, repo.Impact{Changed: []string{"pricing.py"}})
	m.detail = "ACME-1"
	m.board.Tasks = []view.Task{{ID: "ACME-1", Flow: "careful", RepoPath: "/w/api"}}
	m.opts.Flows = flowsTestDir(t.TempDir())

	rows := strings.Join(m.impactRows(), "\n")
	if !strings.Contains(rows, "no checks") && !strings.Contains(rows, "[r] runs these") {
		t.Errorf("the pane neither offers the run nor says why it cannot:\n%s", rows)
	}

	// While it is out, the pane says so and the band carries it.
	m.weigh.running, m.weigh.since = true, m.now
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "running…") {
		t.Errorf("a comparison that is out says %q", got)
	}

	if got := m.waitingLine(); !strings.Contains(got, "ACME-1") {
		t.Errorf("the band says %q while the checks run", got)
	}
}
