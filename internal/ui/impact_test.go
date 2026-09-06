package ui

// The impact pane: what this change reaches beyond the files it touched.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// reading is a window with a reading already in it.
func reading(t *testing.T, got repo.Impact) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &fakeReader{}
	m.impact, m.impactKnown = got, true

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

	m.impactAsking = true
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "reading the history") {
		t.Errorf("while reading the pane says %q", got)
	}

	m.impactAsking, m.impactKnown, m.impactErr = false, true, errors.New("not a git repository")
	if got := strings.Join(m.impactRows(), " "); !strings.Contains(got, "not a git repository") {
		t.Errorf("a refused reading says %q", got)
	}

	m.impactErr = nil
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
	if got := m.tookImpact(late); got.impactKnown {
		t.Error("a reading about another task landed on this one")
	}
}
