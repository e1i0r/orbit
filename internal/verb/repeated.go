package verb

// What somebody keeps telling runs, listed.
//
// The tray above is the rules a person meant to lay down. This is the ones
// they never said: "add fuzz testing" typed at six tasks in a row is a rule
// nobody ever enunciated, and reading it on its own it is not one.
//
// It offers nothing and decides nothing. What it does is put a person in
// front of what they do without noticing, which is the whole of what they
// need to write the rule themselves.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// repeating is the listing: the habit said most often first, and under each
// one every time it was said.
//
// The sentences and not a count. The words somebody used are the rule, and
// the number is only the reason to go and look at them — a listing that gave
// the count alone would be asking them to remember what they said.
func repeating(w World) (Out, error) {
	habits, err := learn.Repeated(w.Store())
	if err != nil {
		return Out{}, err
	}

	if len(habits) == 0 {
		return Out{Said: w.Words().T("verb.rules.repeated.none",
			"nothing you have told a run has come up often enough to be a habit"), Saw: habits}, nil
	}

	var b strings.Builder

	for _, h := range habits {
		heading(&b, w.Words(), h)

		for _, one := range h.Said {
			fmt.Fprintf(&b, "     %-10s %s\n", one.Task, one.Text)
		}

		b.WriteString("\n")
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: habits}, nil
}

// heading is the line that names one habit: how often, where, and the words
// that hold it together.
//
// The words are on the heading because they are what somebody checks first.
// A habit grouped by "commit" and "common" is wrong, and the only way to see
// that at a glance is to be shown what it was grouped by.
func heading(b *strings.Builder, p *words.Printer, h learn.Habit) {
	where := filepath.Base(h.Repo)
	if h.Phase != "" {
		where += " · " + h.Phase
	}

	fmt.Fprintf(b, "%s  %s  %s\n",
		p.P("verb.rules.repeated.times", h.Times(), "said once", "said {n} times"),
		where, strings.Join(h.Words, ", "))
}
