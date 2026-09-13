package verb

// The small answers a listing is made of: who asked, when something
// happened, where a flow came from, and which checkout the reader meant.
//
// They are here as their own tests because each of them has a case that only
// turns up on a bad day — a record that could not be parsed, a flow origin
// nobody has added yet — and a bad day is exactly when a listing has to
// still print.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/words"
)

// TestWhoAskedIsThePersonUnlessSomethingElseSaysSo.
//
// The record says who acted, and "operator" is what a person at any of the
// controls is called. Empty is the way round that fails safe: a gesture with
// nobody's name on it reads as the person, not as a model.
func TestWhoAskedIsThePersonUnlessSomethingElseSaysSo(t *testing.T) {
	if got := (In{}).who(); got != "operator" {
		t.Errorf("a gesture with no name on it is recorded as %q", got)
	}

	if got := (In{By: "claude"}).who(); got != "claude" {
		t.Errorf("a model's own name came back as %q", got)
	}
}

// TestAMomentThatIsNotOneReadsAsADash.
//
// The zero time is the first instant of the Christian era, and it is the
// placeholder for a line the record could not parse. Printing 0001-01-01
// there would be a date, and a wrong one — which is worse than a dash,
// because somebody would try to work out what happened that day.
func TestAMomentThatIsNotOneReadsAsADash(t *testing.T) {
	if got := stamp(time.Time{}); got != "—" {
		t.Errorf("a moment that is not one reads %q", got)
	}

	at := time.Date(2026, 9, 12, 14, 30, 5, 0, time.UTC)
	if got := stamp(at); !strings.HasPrefix(got, "2026-09-12") {
		t.Errorf("a real moment reads %q", got)
	}
}

// TestWhereAFlowCameFromIsSaidOrNotSaidAtAll.
//
// The three a reader can act on are named. An origin nobody has added yet
// gets the empty string rather than a panic: a listing that died over a flow
// it did not recognise would take the flows a reader does recognise with it.
func TestWhereAFlowCameFromIsSaidOrNotSaidAtAll(t *testing.T) {
	p := words.For("")

	for _, o := range []flow.Origin{flow.OriginBuiltin, flow.OriginUser, flow.OriginShadow} {
		if FlowMark(p, o) == "" {
			t.Errorf("origin %v says nothing about itself", o)
		}
	}

	if got := FlowMark(p, flow.Origin(99)); got != "" {
		t.Errorf("an origin nobody has added yet reads %q", got)
	}
}

// TestTheCheckoutTheReaderMeant.
//
// What was typed, then where they are standing, then nowhere — and nowhere
// is an answer rather than a refusal, because a task against no repository
// is a task all the same.
func TestTheCheckoutTheReaderMeant(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	typed, err := openRepo(w, In{Args: map[string]string{"repo": r.Path}, Repo: "/nowhere"})
	if err != nil {
		t.Fatalf("the repository the reader typed: %v", err)
	}

	if typed.Path != r.Path {
		t.Errorf("opened %q, and the reader typed %q", typed.Path, r.Path)
	}

	standing, err := openRepo(w, In{Repo: r.Path})
	if err != nil {
		t.Fatalf("the repository the reader is standing in: %v", err)
	}

	if standing.Path != r.Path {
		t.Errorf("opened %q, and the reader is standing in %q", standing.Path, r.Path)
	}

	none, err := openRepo(w, In{})
	if err != nil {
		t.Errorf("naming no repository was refused: %v", err)
	}

	if none.Path != "" {
		t.Errorf("naming no repository opened %q", none.Path)
	}

	if _, err := openRepo(w, In{Args: map[string]string{"repo": t.TempDir()}}); err == nil {
		t.Error("a directory that is no checkout opened anyway")
	}
}

// TestATaskCanBeWrittenAgainstNoRepository, which is what somebody writing
// one down before they have decided where it happens means.
func TestATaskCanBeWrittenAgainstNoRepository(t *testing.T) {
	w := worldOf(t)

	out := mustAsk(t, w, "board new", In{
		Args: map[string]string{"id": "ACME-36", "text": "decide where this goes"},
		By:   "operator",
	})

	if !strings.Contains(out.Said, "ACME-36") {
		t.Errorf("writing it down answered %q", out.Said)
	}
}
