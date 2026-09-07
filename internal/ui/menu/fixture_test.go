package menu

// The world every test in this package is built on: one task on the board,
// the twelve panes of it, a verb that can be done and one that cannot, and
// a command table with both kinds of command in it.

import (
	"testing"

	"charm.land/bubbles/v2/key"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// theTask is the id every fixture is about; gone is one that has left the
// board while its menu was up.
const (
	theTask = "ACME-7"
	gone    = "ACME-NOWHERE"
)

// panes is the twelve, keyed as the detail screen keys them.
func panes() []Pane {
	keys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "i", "w"}

	out := make([]Pane, 0, len(keys))
	for _, k := range keys {
		out = append(out, Pane{Key: k, Title: "pane " + k, Detail: "what is in pane " + k})
	}

	return out
}

// table is the commands: one about no task, one refused, and the nine a
// task's menu carries.
func table() []Command {
	out := []Command{
		{Name: "reconcile", About: "read the board again"},
		{Name: "top", Refused: true, Because: "you are already in it"},
		{Name: "export", About: "write the record out"},
	}

	for _, c := range taskCommands {
		out = append(out, Command{Name: c.name, About: "what " + c.name + " does", AboutATask: true})
	}

	return out
}

// verbs is what can be done to the task: one offered and one refused with
// its reason, which is the pair every drawing rule here is about.
func verbs() []keymap.Affordance {
	return []keymap.Affordance{
		{Key: key.NewBinding(key.WithHelp("p", "pause")), OK: true},
		{
			Key:    key.NewBinding(key.WithHelp("c", "cancel")),
			WhyNot: words.Arg{Name: keymap.WhyMarkerUnreadable},
		},
	}
}

// world is the Env on the board: no panes, because there is no task being
// read, and the whole table.
func world(t *testing.T) Env {
	t.Helper()

	frame, err := layout.Fit(100, 30)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	p := words.For("en")

	return Env{
		Words:    p,
		Keys:     keymap.New(p),
		Frame:    frame,
		Commands: table(),
		Verbs: func(id string) ([]keymap.Affordance, bool) {
			if id != theTask {
				return nil, false
			}

			return verbs(), true
		},
		Args: func(id string) []string { return []string{"-repo", "/checkouts/acme", id} },
	}
}

// inside is the same world with the task open, where the panes are moves
// too.
func inside(t *testing.T) Env {
	t.Helper()

	e := world(t)
	e.Detail = true
	e.Panes = panes()

	return e
}
