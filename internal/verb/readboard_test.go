package verb

// The listing keeps itself a table: the reason wins the last cell, and the
// text never breaks the columns.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestDetailPrefersTheReasonOverTheOutput pins the entries that have no
// reason: they go on printing what the engine said.
func TestDetailPrefersTheReasonOverTheOutput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry view.Entry
		want  string
	}{
		{"a failure says why", view.Entry{Text: "stdout", Cause: "exit 1"}, "exit 1"},
		{"no cause at all", view.Entry{Text: "stdout"}, "stdout"},
		{"an empty cause is no cause", view.Entry{Text: "stdout", Cause: ""}, "stdout"},
		{"a join names its repository", view.Entry{Kind: "repo.joined", Repo: "payments"}, "payments"},
	} {
		if got := detailOf(tc.entry); got != tc.want {
			t.Errorf("%s: detail is %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestFirstLineKeepsTheTableATable pins what the listing does to text it
// did not write. The engine's output is arbitrary and the table is
// tab-delimited.
func TestFirstLineKeepsTheTableATable(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"plain", "plain"},
		{"first\nsecond", "first …"},
		// A break at the very front is still a break: the row would open
		// with a newline and every column below it would drift by one.
		{"\nsecond", " …"},
		{"before\tafter", "before after"},
		{"progress\rdone", "progress done"},
		{"", ""},
	} {
		if got := firstLine(tc.in); got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestARowNamesTheCheckoutItWasWorkedIn, and a task joined to none names
// none rather than falling over: a task written before any repository was
// named is the state every task passes through.
func TestARowNamesTheCheckoutItWasWorkedIn(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-1", r.Path, "pay the thing")
	w.wrote(t, "ACME-2", "", "decide what to build")

	joined, there, err := rowAnywhere(w, "ACME-1")
	if err != nil || !there {
		t.Fatalf("the row of a task in a checkout: there %v, %v", there, err)
	}

	if joined.RepoPath != r.Path || joined.Repo != "acme" {
		t.Errorf("it names %q, %q, want %q and acme", joined.RepoPath, joined.Repo, r.Path)
	}

	alone, there, err := rowAnywhere(w, "ACME-2")
	if err != nil || !there {
		t.Fatalf("the row of a task in no checkout: there %v, %v", there, err)
	}

	if alone.RepoPath != "" || alone.Repo != "" {
		t.Errorf("a task in no checkout names %q, %q", alone.RepoPath, alone.Repo)
	}
}
