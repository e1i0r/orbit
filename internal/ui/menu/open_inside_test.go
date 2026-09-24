package menu

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/keymap"
)

// TestInsideATaskTheVerbsDoNotOfferToOpenIt. "⏎ open" was on the verbs
// of the menu inside a task, and choosing it sent ⏎, which the task's
// screen reads as "to the end of the pane": the reader asked to open what
// was open and the pane moved. On the board it still opens the task.
func TestInsideATaskTheVerbsDoNotOfferToOpenIt(t *testing.T) {
	withOpen := func(e Env) Env {
		e.Verbs = func(string) ([]keymap.Affordance, bool) {
			return append([]keymap.Affordance{{Key: e.Keys.Open, OK: true}}, verbs()...), true
		}

		return e
	}

	for name, c := range map[string]struct {
		e    Env
		want bool
	}{
		"inside a task": {withOpen(inside(t)), false},
		"on the board":  {withOpen(world(t)), true},
	} {
		s := State{open: true, task: theTask, sub: "task"}

		offered := false

		for _, entry := range s.verbEntries(c.e) {
			if entry.Glyph == c.e.Keys.Open.Help().Key {
				offered = true
			}
		}

		if offered != c.want {
			t.Errorf("%s the verbs offer open: %v, want %v", name, offered, c.want)
		}
	}
}
