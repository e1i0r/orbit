package known

// What a rule is made of, as a form: the rows, the groups they sit in, and
// the options each row offers.
//
// Offered and not remembered. The place a rule applies to used to be an
// empty line you typed a path into, so filing one meant knowing by heart
// which folders the checkout has and spelling one right; the check was an
// empty line too, and a rule that refuses work is worth nothing without one.
// Both are picked from what is actually there now, and typing is what you do
// when none of them is it.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/fact"
)

// The rows of the form, in the order tab walks them.
const (
	rowPhrase = iota
	rowWhere
	rowDoes
	rowCheck
	rowSave
	rowCancel
)

// aRow is one line of the form.
type aRow struct {
	// which row this is, and the typed field behind it when it has one.
	which, typed int
	label, hint  string
	// options is what this row offers, and empty for a row that is only
	// typed into. The first is what the row means when it holds nothing.
	options []option
	// button says the row is done rather than filled in.
	button string
}

// option is one pill: what it puts in the field, and what it reads as.
type option struct{ value, label string }

// rows is the form, which is not the same form for every gesture.
func (s State) rows2(e Env) []aRow {
	p := e.Words

	if s.pausing {
		return []aRow{
			{
				which: rowPhrase, typed: factPhrase,
				label: p.T("knowledge.field_why", "What for"),
				hint: p.T("knowledge.hint_why", "a pause with no reason is a switch under "+
					"another name; this is what you will read when you come back"),
			},
			{which: rowSave, button: p.T("knowledge.btn_pause", "✔ Pause it")},
			{which: rowCancel, button: p.T("knowledge.btn_cancel", "✖ Leave it as it was")},
		}
	}

	out := []aRow{
		{
			which: rowPhrase, typed: factPhrase,
			label: p.T("knowledge.field_phrase", "What it says"),
			hint:  p.T("knowledge.hint_phrase", "the sentence every run is told before it starts work"),
		},
		{
			which: rowWhere, typed: factWhere,
			label:   p.T("knowledge.field_where", "Where it applies"),
			hint:    s.whereHint(e),
			options: s.places(e),
		},
		{
			which: rowDoes,
			label: p.T("knowledge.field_does", "What it does"),
			hint: p.T("knowledge.hint_does", "every rule is put in front of the agent before "+
				"it works. This one also runs a command at the gate, and the work is sent back "+
				"when that command fails."),
			options: s.doings(e),
		},
	}

	out = append(out, aRow{
		which: rowCheck, typed: factCheck,
		label:   p.T("knowledge.field_check", "The check"),
		hint:    p.T("knowledge.hint_check", "the work is sent back when this command does not exit zero. It is yours and never a model's: it runs on every future phase in this repository."),
		options: s.commands(e),
	})

	return append(out,
		aRow{which: rowSave, button: p.T("knowledge.btn_save", "✔ Save the rule")},
		aRow{which: rowCancel, button: p.T("knowledge.btn_cancel", "✖ Leave it as it was")})
}

// places is everywhere a rule can be filed, read off the checkout it is
// about: the whole of it, each of its top folders, and whatever somebody
// typed that is none of those.
func (s State) places(e Env) []option {
	p := e.Words

	repo := s.repoOf(e)
	if repo == "" {
		return []option{{label: p.T("knowledge.place_all", "everywhere")}}
	}

	out := []option{{label: p.T("knowledge.place_repo", "all of {repo}",
		about("repo", fact.Repo(repo)))}}

	if e.Places != nil {
		for _, dir := range e.Places(repo) {
			out = append(out, option{value: dir, label: dir + "/"})
		}
	}

	return out
}

// whereHint says what the place under the cursor actually means, because
// "internal/db" on its own does not say whether that is where the rule
// applies or where it was written.
func (s State) whereHint(e Env) string {
	p := e.Words

	where := strings.TrimSpace(s.in[factWhere].Val)
	if where == "" {
		return p.T("knowledge.hint_where_all",
			"every run against this checkout is told it, whatever it is touching")
	}

	return p.T("knowledge.hint_where_in",
		"it is only told when the work is inside {path} — type another path to move it",
		about("path", where))
}

// doings is the two things a rule can do.
//
// It is a shortcut over the check and not a switch of its own. What decides
// whether a rule refuses work is whether it has a command that answers yes
// or no, so a switch beside the command could disagree with it — and a check
// typed under a switch left off is a gate somebody wrote and Orbit threw
// away without saying so.
func (s State) doings(e Env) []option {
	p := e.Words

	return []option{
		{label: p.T("knowledge.does_pill_says", "says it before the work")},
		{value: "stops", label: p.T("knowledge.does_pill_stops", "and refuses the work")},
	}
}

// commands is what this repository already runs on itself, so a check is
// picked rather than remembered.
func (s State) commands(e Env) []option {
	out := []option{{label: e.Words.T("knowledge.check_none", "(none yet)")}}

	repo := s.repoOf(e)
	if e.Commands == nil || repo == "" {
		return out
	}

	for _, one := range e.Commands(repo) {
		out = append(out, option{value: one, label: one})
	}

	return out
}

// repoOf is the checkout the rule being written is about.
func (s State) repoOf(e Env) string {
	if f, ok := s.onFact(); ok && f.Scope.Repo != "" {
		return f.Scope.Repo
	}

	return e.Repo
}

// held is what a row is holding right now: the typed value for a row with a
// field behind it, and the switch for the one that has none.
func (s State) held(r aRow) string {
	if r.which == rowDoes {
		if strings.TrimSpace(s.in[factCheck].Val) != "" {
			return "stops"
		}

		return ""
	}

	return strings.TrimSpace(s.in[r.typed].Val)
}

// walk moves what a row is holding one step along its options, which is what
// ←→ do. A value none of them offers counts as past the end, so the first
// press lands on the first option rather than nowhere.
func (s State) walk(r aRow, d int, e Env) State {
	if len(r.options) == 0 {
		return s
	}

	at := -1

	for i, o := range r.options {
		if o.value == s.held(r) {
			at = i
		}
	}

	return s.walkTo(r, r.options[(at+d+2*len(r.options))%len(r.options)])
}

// walkTo puts one option into the row.
func (s State) walkTo(r aRow, next option) State {
	// Asking for a gate is asking for the command that is one, so it puts
	// the cursor where that is typed rather than setting a switch nothing
	// is behind.
	if r.which == rowDoes {
		if next.value == "" {
			s.in[factCheck] = oneLine("")
			return s
		}

		s.field = rowCheck

		return s
	}

	s.in[r.typed] = oneLine(next.value)

	return s
}
