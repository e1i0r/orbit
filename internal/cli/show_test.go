package cli

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShowPrintsTheDayAsWellAsTheClock(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}
	// Without the date a task that ran last week reads as this morning's.
	if day := time.Now().Format("2006-01-02"); !strings.Contains(out, day) {
		t.Errorf("show does not print the day (%s):\n%s", day, out)
	}
}

// TestShowSaysNothingRatherThanTheYearOne is the reading end of a record
// with something in it that would not parse: the reader hands back a
// placeholder event, and a placeholder has no time. Printed as a date it
// would say 0001-01-01, which is a date, and a wrong one.
//
// The line is planted where an older Orbit's log was and carried across by
// the migration, which is the one thing that appends a record.unreadable —
// the damage a log took is a fact about the record and has to survive the
// move. It arrives with no time and keeps none, which is what this pins.
func TestShowSaysNothingRatherThanTheYearOne(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	plant(t, repoDir, "ACME-1", "{not json")

	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "record.unreadable") {
		t.Errorf("show hides the line it could not read:\n%s", out)
	}

	if strings.Contains(out, "0001-01-01") {
		t.Errorf("show dates an event that has no time:\n%s", out)
	}

	if !strings.Contains(out, "—") {
		t.Errorf("show leaves the time column blank instead of saying it is unknown:\n%s", out)
	}
}

// TestShowSaysWhyAPhaseFailed pins the last cell of a phase.failed row on
// the reason rather than on whatever the engine had printed. The reason
// moved into Data["error"] when Text became the engine's output, and a row
// that quotes stdout under the word "failed" reads as though stdout were the
// failure.
func TestShowSaysWhyAPhaseFailed(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	plant(t, repoDir, "ACME-1", `{"at":"2026-08-23T09:14:02Z","kind":"phase.failed","phase":"implement",`+
		`"text":"reading the webhook handler","data":{"error":"claude exited 1: no such model"}}`)

	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}

	row := rowContaining(t, out, "phase.failed")
	if !strings.Contains(row, "claude exited 1: no such model") {
		t.Errorf("the failed row does not say why it failed:\n%s", row)
	}

	if strings.Contains(row, "reading the webhook handler") {
		t.Errorf("the failed row quotes the engine's stdout in place of the reason:\n%s", row)
	}
}

// TestDetailPrefersTheReasonOverTheOutput is the same rule without a
// repository around it, and it pins the events that have no reason: they go
// on printing what the engine said.
func TestDetailPrefersTheReasonOverTheOutput(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		data map[string]string
		want string
	}{
		{"a failure says why", "stdout", map[string]string{"error": "exit 1"}, "exit 1"},
		{"no data at all", "stdout", nil, "stdout"},
		{"an empty reason is no reason", "stdout", map[string]string{"error": ""}, "stdout"},
		{"other data is not the reason", "stdout", map[string]string{"cost": "0.4"}, "stdout"},
	} {
		if got := detail(tc.text, tc.data); got != tc.want {
			t.Errorf("%s: detail(%q, %v) = %q, want %q", tc.name, tc.text, tc.data, got, tc.want)
		}
	}
}

// rowContaining is the one printed row that mentions something, because an
// assertion against the whole table cannot tell which row carried the text.
func rowContaining(t *testing.T, out, want string) string {
	t.Helper()

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, want) {
			return line
		}
	}

	t.Fatalf("no row mentioning %q:\n%s", want, out)

	return ""
}

// TestFirstLineKeepsTheTableATable pins what show does to text it did not
// write. The engine's output is arbitrary and the table is tab-delimited.
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

func TestShowDoesNotLetATabInTheTextAddAColumn(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "before\tafter"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "before after") {
		t.Errorf("a tab in the text opened a column of its own:\n%q", out)
	}
}
