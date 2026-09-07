package panes

// The impact pane: three sections that are three different kinds of claim,
// and the states each of them can be in before there is one.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// weighed is the pane with a reading behind it: two files changed, one that
// usually follows and did not, and what its tests say they hold.
func weighed(t *testing.T) Env {
	t.Helper()

	e := world(t, nil)
	e.Reach = Reach{
		Read: true,
		History: repo.Impact{
			Changed:   []string{"pricing.py"},
			Commits:   500,
			Coupled:   []repo.Coupled{{File: "invoice.py", With: "pricing.py", Times: 34, Of: 39}},
			Contracts: []repo.Contract{{File: "invoice_test.py", Says: "rejects negative amounts"}},
		},
	}

	return e
}

// TestTheImpactPaneSaysHowFarItGot. Nothing found and nothing read are
// different facts, and a pane that folded them into one would tell a reader
// a repository has no coupling when what happened is that git timed out.
func TestTheImpactPaneSaysHowFarItGot(t *testing.T) {
	for _, c := range []struct {
		name string
		of   func(Env) Env
		want string
	}{
		{"a build with no port", func(e Env) Env { e.Reach = Reach{NoHistory: true}; return e }, "cannot read the history"},
		{"a reading still out", func(e Env) Env { e.Reach = Reach{Asking: true}; return e }, "reading the history…"},
		{"nothing asked yet", func(e Env) Env { e.Reach = Reach{}; return e }, "nothing read yet"},
		{
			"a reading that broke",
			func(e Env) Env { e.Reach = Reach{Read: true, Failed: "git did not answer in time"}; return e },
			"did not answer in time",
		},
		{"a task that changed nothing", func(e Env) Env { e.Reach = Reach{Read: true}; return e }, "nothing to weigh"},
	} {
		if got := text(Impact(c.of(weighed(t)))); !strings.Contains(got, c.want) {
			t.Errorf("%s drew a pane that does not say %q:\n%s", c.name, c.want, got)
		}
	}
}

// TestWhatUsuallyComesAlongIsCountedAndSourced. The coupling is the
// history's word and not a proof: two files that always moved together may
// have been split apart on purpose today, and the pane says how it knows.
func TestWhatUsuallyComesAlongIsCountedAndSourced(t *testing.T) {
	got := text(Impact(weighed(t)))

	for _, want := range []string{
		"WHAT USUALLY COMES ALONG",
		"pricing.py", "changed here",
		"invoice.py", "87% of the time (34/39)", "not touched",
		"last 500 commits",
		"WHAT THOSE TESTS SAY THEY HOLD", "invoice_test.py", "rejects negative amounts",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the pane does not carry %q:\n%s", want, got)
		}
	}
}

// TestARepositoryWhereNothingFollowsSaysSo, rather than drawing a heading
// over an empty list.
func TestARepositoryWhereNothingFollowsSaysSo(t *testing.T) {
	e := weighed(t)
	e.Reach.History.Coupled = nil

	if got := text(Impact(e)); !strings.Contains(got, "nothing else follows these files") {
		t.Errorf("a reading that found no coupling drew:\n%s", got)
	}
}

// TestTheAgentsOwnAccountIsLastAndMarkedAsUnverified. The two sections above
// it are the repository's history; this one is a claim by the thing whose
// work is being weighed, and nothing verified it.
func TestTheAgentsOwnAccountIsLastAndMarkedAsUnverified(t *testing.T) {
	e := weighed(t)
	e.Entries = []view.Entry{{Kind: "task.delta", Delta: &view.Delta{
		Needs:      []string{"pass the currency in"},
		Guarantees: []string{"rounds half to even"},
		Assumes:    []string{"amounts are in minor units"},
		Instead:    []string{"a decimal type, which the store cannot hold"},
	}}}

	got := text(Impact(e))
	for _, want := range []string{
		"WHAT THE AGENT SAYS IT DID",
		"callers must now", "pass the currency in",
		"it now holds", "rounds half to even",
		"it took for granted", "amounts are in minor units",
		"considered and not taken", "a decimal type",
		"Nobody verified it",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the delta section does not carry %q:\n%s", want, got)
		}
	}

	// The claim comes after the history, which comes after the exit codes.
	if at, was := strings.Index(got, "WHAT THE AGENT SAYS"), strings.Index(got, "WHAT USUALLY COMES"); at < was {
		t.Errorf("the agent's own account is drawn above the history it should follow:\n%s", got)
	}
}

// TestTheChecksSectionSaysWhatItWouldRunBeforeItRunsIt. It is a test suite
// twice on somebody's machine, so the cost is named before it is paid.
func TestTheChecksSectionSaysWhatItWouldRunBeforeItRunsIt(t *testing.T) {
	e := weighed(t)
	e.Reach.Offered = []repo.Check{{Name: "tests", Command: "go test ./..."}}

	got := text(Impact(e))
	for _, want := range []string{"WHAT THE CHECKS SAY, BOTH SIDES", "runs these on both sides", "go test ./..."} {
		if !strings.Contains(got, want) {
			t.Errorf("the checks section does not carry %q:\n%s", want, got)
		}
	}

	// A flow with no checks says so rather than offering a key that would
	// run nothing.
	e.Reach.Offered = nil
	if got := text(Impact(e)); !strings.Contains(got, "carries no checks") {
		t.Errorf("a flow with nothing to run drew:\n%s", got)
	}
}

// TestTheChecksSectionSaysWhileItIsOutAndWhenItBroke.
func TestTheChecksSectionSaysWhileItIsOutAndWhenItBroke(t *testing.T) {
	e := weighed(t)
	e.Spinner = "⠋ "
	e.Reach.Running = true
	e.Reach.For = 42 * time.Second
	e.Reach.Offered = []repo.Check{{Name: "tests", Command: "go test ./..."}}

	if got := text(Impact(e)); !strings.Contains(got, "42s") || !strings.Contains(got, "running 1 checks") {
		t.Errorf("a run that is out does not say what it is doing:\n%s", got)
	}

	e.Reach.Running = false
	e.Reach.ChecksFailed = "the base could not be checked out"

	if got := text(Impact(e)); !strings.Contains(got, "could not be checked out") {
		t.Errorf("a comparison that broke does not say why:\n%s", got)
	}

	e.Reach.ChecksFailed, e.Reach.Compared = "", true

	if got := text(Impact(e)); !strings.Contains(got, "no checks ran") {
		t.Errorf("a comparison that ran nothing drew:\n%s", got)
	}
}

// TestEachCheckSaysWhatBothSidesAnswered, and the end of what a failing one
// printed: a test runner says which test failed a few lines above the word
// FAIL, and a pane that shows only the last line shows the word.
func TestEachCheckSaysWhatBothSidesAnswered(t *testing.T) {
	e := weighed(t)
	e.Reach.Compared = true
	e.Reach.Checks = []repo.Divergence{
		{Check: repo.Check{Name: "build", Command: "go build ./..."}},
		{
			Check: repo.Check{Name: "tests", Command: "go test ./..."},
			Base:  repo.Ran{},
			Now:   repo.Ran{Exit: 1, Out: "--- FAIL: TestUpsertIsIdempotent\n    items_test.go:31: two rows\nFAIL"},
		},
		{
			Check: repo.Check{Name: "lint", Command: "golangci-lint run"},
			Base:  repo.Ran{Exit: 2},
			Now:   repo.Ran{},
		},
		{
			Check: repo.Check{Name: "vet", Command: "go vet ./..."},
			Base:  repo.Ran{Failed: errors.New("no such directory")},
			Now:   repo.Ran{Failed: errors.New("no such directory")},
		},
	}

	got := text(Impact(e))
	for _, want := range []string{
		"the same on both sides",
		"passed before, fails now", "exit 1", "TestUpsertIsIdempotent", "items_test.go:31",
		"failed before, passes now",
		"could not be run", "no such directory",
		"[r] runs them again",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the checks do not carry %q:\n%s", want, got)
		}
	}
}

// TestTheTabCarriesTheCountOfWhatWasFound, and nothing at all when the
// reading found nothing: a zero on a tab strip is a number nobody reads.
func TestTheTabCarriesTheCountOfWhatWasFound(t *testing.T) {
	if got := Mark(weighed(t)); !strings.Contains(got, "1") {
		t.Errorf("the tab mark is %q, want it to count the one file that follows", got)
	}

	for _, c := range []struct {
		name string
		of   func(Env) Env
	}{
		{"a reading that found nothing", func(e Env) Env { e.Reach.History.Coupled = nil; return e }},
		{"a reading that has not landed", func(e Env) Env { e.Reach.Read = false; return e }},
		{"a reading that broke", func(e Env) Env { e.Reach.Failed = "git broke"; return e }},
	} {
		if got := Mark(c.of(weighed(t))); got != "" {
			t.Errorf("%s put %q on the tab", c.name, got)
		}
	}
}
