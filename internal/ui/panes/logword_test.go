package panes

// What a line of the record says happened, and the one fact worth putting
// beside it.
//
// Two switches over the whole vocabulary, and most of their cases read zero.
// A kind added to internal/view without a word here is a row that says
// nothing on the screen a reader opens to find out what happened — and there
// is no example test that would catch it, because the case that is missing
// is the one nobody thought to write.
//
// So the test walks the vocabulary itself rather than a list written out
// here: a list would go stale the same afternoon.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// everyKind is the vocabulary, walked from the first to the last.
//
// The bounds are the enum's own ends, so a kind added between them is
// walked without this file being touched.
func everyKind() []view.EntryKind {
	var out []view.EntryKind

	for k := view.EntryUnknown; k <= view.EntryUnreadable; k++ {
		out = append(out, k)
	}

	return out
}

// TestEveryKindTheWindowReadsHasAWordForIt. A row with no word on it is a
// reader looking at the one screen that exists to say what happened and
// being told nothing.
func TestEveryKindTheWindowReadsHasAWordForIt(t *testing.T) {
	e := world(t, nil)

	for _, k := range everyKind() {
		if k == view.EntryUnknown {
			continue
		}

		said, _ := e.logWord(view.Entry{Kind: kindSpelling(k)})
		if strings.TrimSpace(said) == "" {
			t.Errorf("kind %d has no word for it", k)
		}
	}
}

// TestAKindThisBuildDoesNotKnowIsDrawnAsTheRecordSpelledIt.
//
// The one string on this screen that is deliberately not translated: it is
// not a word, it is a key out of somebody else's log, and inventing a
// sentence for it would be inventing the meaning too. A newer orbit writes
// kinds this one has never heard of, and a row that drew nothing for them
// would be this window hiding what it does not understand.
func TestAKindThisBuildDoesNotKnowIsDrawnAsTheRecordSpelledIt(t *testing.T) {
	e := world(t, nil)

	said, role := e.logWord(view.Entry{Kind: "task.teleported"})

	if said != "task.teleported" {
		t.Errorf("an unknown kind is drawn as %q, want the record's own spelling", said)
	}

	if role != theme.Dim {
		t.Errorf("an unknown kind is painted %v, want it left quiet", role)
	}
}

// TestTheFactBesideTheWordIsAlwaysSomethingTheRecordSaid, and never
// something this package composed: what a row offers to open is the thing
// itself, so a detail invented here is a detail nobody can go and check.
func TestTheFactBesideTheWordIsAlwaysSomethingTheRecordSaid(t *testing.T) {
	cases := []struct {
		name  string
		entry view.Entry
		want  string
	}{
		{
			"a phase says which engine ran it",
			view.Entry{Kind: "phase.started", Engine: "claude", Model: "opus"},
			"claude opus",
		},
		{
			"a phase with no model named says only the engine",
			view.Entry{Kind: "phase.started", Engine: "claude"},
			"claude",
		},
		{
			"a gate says which gate",
			view.Entry{Kind: "gate.failed", Gate: "coverage"},
			"coverage",
		},
		{
			"a failure says what broke",
			view.Entry{Kind: "phase.failed", Cause: "exit status 1"},
			"exit status 1",
		},
		{
			"a repository joining says which",
			view.Entry{Kind: "repo.joined", Repo: "acme"},
			"acme",
		},
		{
			"a delivery verb says who pressed it",
			view.Entry{Kind: "deliver.asked", Verb: "PR", By: "operator"},
			"PR · operator",
		},
		{
			"a tool call with nothing said is the tool alone",
			view.Entry{Kind: "phase.tool_call", Tool: "Bash"},
			"Bash",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := logDetail(c.entry); got != c.want {
				t.Errorf("the fact beside it is %q, want %q", got, c.want)
			}
		})
	}
}

// TestTheWholeOfWhatWasRefusedIsKept. The row is wrapped and folded like
// every other row here, so a reason written over three lines is three rows
// under an arrow — cutting it at the first break put the rest of it on no
// row at all, and left the row with nothing to open.
func TestTheWholeOfWhatWasRefusedIsKept(t *testing.T) {
	refused := view.Entry{
		Kind: "phase.refused",
		Tool: "Bash",
		Text: "the command needs a permission\nthis phase was not given\nand nobody was asked",
	}

	got := logDetail(refused)

	for _, line := range strings.Split(refused.Text, "\n") {
		if !strings.Contains(got, line) {
			t.Errorf("the fact beside it lost %q: %q", line, got)
		}
	}

	if !strings.HasPrefix(got, "Bash") {
		t.Errorf("it reads %q, want the tool first", got)
	}
}

// TestAnEntryTheRecordSaidNothingElseAboutHasNoFactBesideIt, rather than an
// empty separator or a word this package made up.
func TestAnEntryTheRecordSaidNothingElseAboutHasNoFactBesideIt(t *testing.T) {
	quiet := []view.Entry{
		{Kind: "gate.passed"},
		{Kind: "phase.failed"},
		{Kind: "task.read"},
		{Kind: "deliver.asked", Verb: "PR"},
	}

	for _, one := range quiet {
		got := logDetail(one)
		if strings.Contains(got, "·") && one.By == "" {
			t.Errorf("%s drew a separator with nothing after it: %q", one.Kind, got)
		}
	}
}

// kindSpelling is the record's own word for one of this package's kinds.
//
// internal/ui/panes may not reach internal/record, so the spellings are
// written here — and the test that they are the right ones is that
// view.Entry.What() reads each of them back as the kind it was asked for.
func kindSpelling(k view.EntryKind) string {
	for _, spelled := range recordKinds {
		if (view.Entry{Kind: spelled}).What() == k {
			return spelled
		}
	}

	return ""
}

// recordKinds is every kind internal/record writes, as it spells them.
var recordKinds = []string{
	"task.created", "task.started", "task.finished", "task.failed", "task.cancelled",
	"task.requeued", "task.timedout", "task.abandoned", "task.read", "task.noted",
	"task.dialogue", "task.stuck", "task.over_budget", "task.over_diff",
	"task.new_dependency", "task.contradicts", "task.relayed", "task.needs_engine",
	"task.no_engine",
	"phase.started", "phase.finished", "phase.failed", "phase.cancelled", "phase.waiting",
	"phase.resumed", "phase.retried", "phase.thought", "phase.tool_call", "phase.refused",
	"phase.asked", "phase.denied",
	"gate.passed", "gate.failed", "loop.checked",
	"decision.made", "decision.superseded", "dependency.approved",
	"repo.joined", "deliver.asked", "deliver.answered", "deliver.step", "record.unreadable",
}

// TestTheSpellingsThisTestUsesAreTheRecordsOwn. A spelling that has gone
// stale would make the walk above skip a kind silently, which is the one
// failure a test like this can have.
func TestTheSpellingsThisTestUsesAreTheRecordsOwn(t *testing.T) {
	for _, spelled := range recordKinds {
		if (view.Entry{Kind: spelled}).What() == view.EntryUnknown {
			t.Errorf("%q is not a kind the window reads any more", spelled)
		}
	}
}
