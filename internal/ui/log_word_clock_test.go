package ui

// log_coverage_test.go is logWord's whole vocabulary — every kind the task
// view knows a word for, and the one it does not — plus clock's own two
// answers.
//
// Every Kind here is spelled as the record's own constant would render it,
// as a literal string: internal/ui may not import internal/record (see
// arch.layers), so reader_test.go and detail_test.go build their fixture
// entries the same way.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

func TestLogWordCoversEveryKnownKindAndTheUnknownOne(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	cases := []struct {
		kind string
		want string
	}{
		{"task.created", "written down"},
		{"task.started", "started"},
		{"phase.started", "started"},
		{"task.finished", "finished"},
		{"phase.finished", "finished"},
		{"task.failed", "failed"},
		{"phase.failed", "failed"},
		{"task.cancelled", "cancelled"},
		{"phase.cancelled", "cancelled"},
		{"task.timedout", "timed out"},
		{"task.abandoned", "abandoned"},
		{"task.read", "read"},
		{"phase.waiting", "waiting"},
		{"phase.resumed", "let go again"},
		{"gate.passed", "gate passed"},
		{"gate.failed", "gate failed"},
		{"phase.thought", "thought"},
		{"phase.tool_call", "tool call"},
		{"phase.refused", "refused"},
		{"record.unreadable", "could not be read"},
	}
	for _, c := range cases {
		word, _ := m.logWord(view.Entry{Kind: c.kind})
		if !strings.Contains(word, c.want) {
			t.Errorf("logWord(%q) = %q, want it to say %q", c.kind, word, c.want)
		}
	}

	// A kind this build has never heard of is drawn exactly as the record
	// spelled it, and not translated.
	word, role := m.logWord(view.Entry{Kind: "custom.unknown.kind"})
	if word != "custom.unknown.kind" {
		t.Errorf("logWord on an unrecognised kind = %q, want the kind verbatim", word)
	}
	if role != Dim {
		t.Errorf("logWord on an unrecognised kind painted %v, want Dim", role)
	}
}

func TestClockFormatsOrLeavesADamagedTimeBlank(t *testing.T) {
	if got := clock(time.Time{}); got != "" {
		t.Errorf("clock(zero time) = %q, want empty", got)
	}
	at := time.Date(2026, 8, 23, 9, 5, 3, 0, time.UTC)
	if got := clock(at); got != "09:05:03" {
		t.Errorf("clock(9:05:03) = %q, want \"09:05:03\"", got)
	}
}
