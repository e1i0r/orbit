package verb

// What a verb answers when it worked.
//
// Every one of these sentences is read by somebody: printed at a terminal,
// drawn in the cockpit's bar, handed back to a model driving Orbit through a
// tool call. A verb that did the work and answered nothing reads to all
// three as a verb that did nothing — and to the model it reads as one that
// quietly succeeded, which is the reading nobody can argue with.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/supervisor"
)

// TestAVerbThatChangedSomethingSaysWhatItChanged.
func TestAVerbThatChangedSomethingSaysWhatItChanged(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-1", r.Path, "pay the thing")

	for _, one := range []struct {
		name string
		in   In
		want string
	}{
		{"task note", In{Args: map[string]string{"text": "cents, not floats"}}, "noted on ACME-1"},
		{"task direct", In{Args: map[string]string{"text": "use redis"}}, "ACME-1 redirected"},
		{"task critical", In{Args: map[string]string{"on": "true"}}, "ACME-1 is critical"},
		{"task critical", In{Args: map[string]string{"on": "false"}}, "ACME-1 is no longer critical"},
		{"task read", In{}, "ACME-1 marked as read"},
		{"task delete", In{}, "ACME-1 is gone"},
	} {
		one.in.Task, one.in.By = "ACME-1", "operator"

		out := mustAsk(t, w, one.name, one.in)
		if !strings.Contains(out.Said, one.want) {
			t.Errorf("%s answered %q, want it to say %q", one.name, out.Said, one.want)
		}
	}
}

// TestSayingSomethingInSomebodyElsesNameKeepsTheirName.
//
// A line arriving through a channel — Telegram, Slack, a tool call — knows
// who wrote it, and this surface does not. Filing it under whoever is at the
// controls puts a stranger's words in the operator's mouth, in the one
// thread the supervisor reads back to decide anything.
func TestSayingSomethingInSomebodyElsesNameKeepsTheirName(t *testing.T) {
	w := worldOf(t)

	mustAsk(t, w, "supervisor say", In{
		Args: map[string]string{"text": "the deploy is out", "by": "ana"},
		By:   "operator",
	})

	mustAsk(t, w, "supervisor say", In{
		Args: map[string]string{"text": "and the checks are green"},
		By:   "operator",
	})

	if len(w.said) != 2 {
		t.Fatalf("two lines were said and the thread holds %d", len(w.said))
	}

	if w.said[0].by != "ana" {
		t.Errorf("a line ana wrote is filed under %q", w.said[0].by)
	}

	if w.said[1].by != "operator" {
		t.Errorf("a line with nobody named is filed under %q", w.said[1].by)
	}
}

// TestAFactIsFiledAgainstTheCheckoutTheReaderNamed.
//
// A surface with no working directory — a browser tab, a tool call — has
// only what it was told. Filing the fact against whatever directory the
// process happens to sit in puts it on a repository nobody mentioned, where
// it reaches every run against that one and no run against the right one.
func TestAFactIsFiledAgainstTheCheckoutTheReaderNamed(t *testing.T) {
	w := worldOf(t)

	mustAsk(t, w, "knowledge learn", In{
		Repo: "/w/standing-here",
		Args: map[string]string{"text": "amounts are cents", "repo": "/w/asked-about"},
		By:   "operator",
	})

	mustAsk(t, w, "knowledge learn", In{
		Repo: "/w/standing-here",
		Args: map[string]string{"text": "the commits are written in English"},
		By:   "operator",
	})

	if len(w.learnt) != 2 {
		t.Fatalf("two facts were learnt and the world holds %d", len(w.learnt))
	}

	if w.learnt[0].Scope.Repo != "/w/asked-about" {
		t.Errorf("a fact about a named checkout was filed against %q", w.learnt[0].Scope.Repo)
	}

	if w.learnt[1].Scope.Repo != "/w/standing-here" {
		t.Errorf("a fact naming no checkout was filed against %q", w.learnt[1].Scope.Repo)
	}
}

// TestTheThreadIsNumberedFromOne, because retract reads the listing back the
// same way: the number a reader types is the line they are looking at, and
// an answer counted from anywhere else takes back a different line.
func TestTheThreadIsNumberedFromOne(t *testing.T) {
	w := worldOf(t)

	for _, text := range []string{"the deploy is out", "and the checks are green"} {
		if err := supervisor.Record(w.store, "", "operator", "cli", "", "", text); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	out := mustAsk(t, w, "supervisor", In{By: "operator"})

	lines := strings.Split(out.Said, "\n")
	if len(lines) != 2 {
		t.Fatalf("two lines were said and the listing has %d: %q", len(lines), out.Said)
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "1 ") {
		t.Errorf("the first line reads %q, want it numbered 1", lines[0])
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[1]), "2 ") {
		t.Errorf("the second line reads %q, want it numbered 2", lines[1])
	}
}

// TestStoppingAndRequeueingSayWhichTheyDid.
//
// They are two different things to have asked for — one stops the run where
// it stands, the other takes the task back — and both are read from a bar
// that shows one line. A pair of answers that could be each other's is a
// reader who cannot tell which of the two keys they pressed.
func TestStoppingAndRequeueingSayWhichTheyDid(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-60", r.Path, "pay the thing")

	back := mustAsk(t, w, "task requeue", In{
		Task: "ACME-60", Args: map[string]string{"why": "wrong brief"}, By: "operator",
	})
	if !strings.Contains(back.Said, "ACME-60") || !strings.Contains(back.Said, "back in the queue") {
		t.Errorf("requeueing answered %q", back.Said)
	}

	cmd := holdARun(t, w, "ACME-60")

	stop := mustAsk(t, w, "task cancel", In{Task: "ACME-60", By: "operator"})

	_ = cmd.Process.Kill() //nolint:errcheck // the cleanup kills it again; this only hurries it

	if !strings.Contains(stop.Said, "ACME-60") || !strings.Contains(stop.Said, "stop") {
		t.Errorf("cancelling answered %q", stop.Said)
	}
}
