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
		{"before\tafter", "before after"},
		{"progress\rdone", "progress done"},
		{"", ""},
	} {
		if got := firstLine(tc.in); got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
