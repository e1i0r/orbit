package verb

// The line that names one habit.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// TestAHabitsHeadingSaysWhereItWasSaid.
//
// The checkout and the phase, because the same words said in the plan phase
// and in the test phase are two rules. A habit read out of a record with no
// phase in it has a checkout and nothing else, and a heading that printed
// the separator anyway would read as a phase whose name went missing.
func TestAHabitsHeadingSaysWhereItWasSaid(t *testing.T) {
	var both strings.Builder

	heading(&both, words.For(""), learn.Habit{
		Repo: "/w/acme", Phase: "test", Words: []string{"fuzz", "testing"},
		Said: make([]learn.Directive, 3),
	})

	line := both.String()
	for _, want := range []string{"3", "acme", "·", "test", "fuzz, testing"} {
		if !strings.Contains(line, want) {
			t.Errorf("the heading reads %q, want it to carry %q", line, want)
		}
	}

	// The checkout by the name a person calls it, not the path they would
	// have to read to the end of.
	if strings.Contains(line, "/w/acme") {
		t.Errorf("the heading names the whole path: %q", line)
	}

	var alone strings.Builder

	heading(&alone, words.For(""), learn.Habit{
		Repo: "/w/acme", Words: []string{"fuzz"}, Said: make([]learn.Directive, 3),
	})

	if strings.Contains(alone.String(), "·") {
		t.Errorf("a habit said in no phase reads %q, with a separator and nothing after it", alone.String())
	}
}
