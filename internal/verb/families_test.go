package verb

// A family and its children, and the two listings a number is read off.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/supervisor"
)

// TestAFamilyKnowsItsChildren, which is what every way in builds its menu,
// its routes and its tool names out of. A parent that answered with nothing
// would be a family that exists in the declaration and nowhere a reader can
// reach it.
func TestAFamilyKnowsItsChildren(t *testing.T) {
	parent, found := One("rules")
	if !found {
		t.Fatal("rules is not a verb")
	}

	kids := parent.Children()
	if len(kids) == 0 {
		t.Fatal("rules has no children, and keep and drop belong to it")
	}

	for _, kid := range kids {
		if kid.Under != "rules" {
			t.Errorf("%q is among the children of rules and belongs to %q", kid.Name, kid.Under)
		}

		if kid.Path() != "rules "+kid.Name {
			t.Errorf("a child of rules is called %q", kid.Path())
		}
	}

	// A verb that belongs to nobody has nobody belonging to it either.
	alone, found := One("quota")
	if !found {
		t.Fatal("quota is not a verb")
	}

	if got := alone.Children(); len(got) != 0 {
		t.Errorf("quota has %d children", len(got))
	}
}

// TestANumberOffAListingIsReadBackTheSameWay.
//
// The thread and the tray are both numbered by position, which is the whole
// of what a number means here: there is no id to point at. So the two ways
// of getting it wrong have to answer in words — a number that is not one,
// and a number nothing is waiting under.
func TestANumberOffAListingIsReadBackTheSameWay(t *testing.T) {
	w := worldOf(t)

	for _, one := range []struct{ verb, arg, typed string }{
		{"supervisor retract", "line", "seven"},
		{"rules drop", "n", "seven"},
		{"rules keep", "n", "seven"},
	} {
		t.Run(one.verb+" not a number", func(t *testing.T) {
			err := refuseErr(t, w, one.verb,
				In{Args: map[string]string{one.arg: one.typed}, By: "operator"})
			if !strings.Contains(err.Error(), "seven") {
				t.Errorf("refused with %q, which does not say what was typed", err)
			}
		})
	}

	for _, one := range []struct{ verb, arg string }{
		{"supervisor retract", "line"},
		{"rules drop", "n"},
	} {
		t.Run(one.verb+" nothing there", func(t *testing.T) {
			if err := refuseErr(t, w, one.verb,
				In{Args: map[string]string{one.arg: "9"}, By: "operator"}); err == nil {
				t.Error("a number nothing is waiting under was accepted")
			}
		})
	}
}

// TestASentenceInTheTrayIsKeptOrDropped, which is the whole of what the tray
// is for: it is read by number, and answered once.
func TestASentenceInTheTrayIsKeptOrDropped(t *testing.T) {
	w := worldOf(t)

	at := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	for i, text := range []string{"never force-push", "always wrap errors"} {
		said := learn.Said{At: at.Add(time.Duration(i) * time.Minute), Text: text}
		if err := learn.Propose(w.store, said); err != nil {
			t.Fatalf("propose %q: %v", text, err)
		}
	}

	waiting := mustAsk(t, w, "rules", In{By: "operator"})
	if !strings.Contains(waiting.Said, "never force-push") {
		t.Fatalf("the tray reads %q", waiting.Said)
	}

	kept := mustAsk(t, w, "rules keep", In{Args: map[string]string{"n": "1"}, By: "operator"})
	if !strings.Contains(kept.Said, "never force-push") {
		t.Errorf("keeping one answered %q", kept.Said)
	}

	dropped := mustAsk(t, w, "rules drop", In{Args: map[string]string{"n": "1"}, By: "operator"})
	if !strings.Contains(dropped.Said, "always wrap errors") {
		t.Errorf("dropping one answered %q", dropped.Said)
	}

	empty := mustAsk(t, w, "rules", In{By: "operator"})
	if strings.Contains(empty.Said, "always wrap errors") {
		t.Errorf("an answered sentence is still in the tray: %q", empty.Said)
	}
}

// TestALineOfTheThreadSaysWhoSaidItAndWhereAbout.
//
// A thread read back is a conversation between several: a person at the
// cockpit, a model through a tool call, the supervisor answering itself. A
// line that did not say which, about a task that is not named, is a line
// nobody can place afterwards — and placing it is the whole reason the
// thread is kept.
func TestALineOfTheThreadSaysWhoSaidItAndWhereAbout(t *testing.T) {
	w := worldOf(t)

	if err := supervisor.Record(w.store, "", "operator", "cli", "ACME-40", "", "use redis"); err != nil {
		t.Fatalf("record: %v", err)
	}

	out := mustAsk(t, w, "supervisor thread", In{By: "operator"})

	for _, want := range []string{"operator", "cli", "ACME-40", "use redis"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("the line does not say %q:\n%s", want, out.Said)
		}
	}

	mustAsk(t, w, "supervisor retract", In{Args: map[string]string{"line": "1"}, By: "operator"})

	// Taken back and still listed, marked: it was said, and hiding it would
	// leave the rest of the thread answering something that is not there.
	back := mustAsk(t, w, "supervisor thread", In{By: "operator"})
	if !strings.Contains(back.Said, "retracted") {
		t.Errorf("a line taken back is not marked as one:\n%s", back.Said)
	}
}
