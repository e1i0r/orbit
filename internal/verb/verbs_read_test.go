package verb

// The readings, asked the way every door asks them.
//
// A reading answers twice — Saw for a surface that draws, Said for a
// terminal — so these assert the sentence and, where it matters, the
// shape. The thread and the settings go through the real store: what is
// read is what was written, and a fake would prove nothing.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/supervisor"
)

// TestTheThreadIsNumberedAndRetractable. Lines are numbered in the order
// they were said; taking one back leaves it listed, marked.
func TestTheThreadIsNumberedAndRetractable(t *testing.T) {
	w := worldOf(t)

	empty := mustAsk(t, w, "thread", In{By: "operator"})
	if !strings.Contains(empty.Said, "empty") {
		t.Errorf("an empty thread reads %q", empty.Said)
	}

	for _, text := range []string{"never force-push", "always wrap errors"} {
		if err := supervisor.Record(w.store, "", "operator", "cli", "", "", text); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	out := mustAsk(t, w, "thread", In{By: "operator"})
	if !strings.Contains(out.Said, "1") || !strings.Contains(out.Said, "never force-push") {
		t.Errorf("the thread reads:\n%s", out.Said)
	}

	mustRefuse(t, w, "retract", In{Args: map[string]string{"line": "many"}, By: "operator"})
	mustRefuse(t, w, "retract", In{Args: map[string]string{"line": "9"}, By: "operator"})

	took := mustAsk(t, w, "retract", In{Args: map[string]string{"line": "1"}, By: "operator"})
	if !strings.Contains(took.Said, "never force-push") {
		t.Errorf("retract answered %q", took.Said)
	}

	mustRefuse(t, w, "retract", In{Args: map[string]string{"line": "1"}, By: "operator"})
}

// TestSettingsReadAndWrite. The whole table reads; one key writes, and a
// key nothing knows is refused and named.
func TestSettingsReadAndWrite(t *testing.T) {
	w := worldOf(t)

	out := mustAsk(t, w, "settings", In{By: "operator"})
	if !strings.Contains(out.Said, "autopilot") {
		t.Errorf("settings reads:\n%s", out.Said)
	}

	changed := mustAsk(t, w, "set", In{
		Args: map[string]string{"key": "autopilot", "value": "on"}, By: "operator",
	})
	if !strings.Contains(changed.Said, "autopilot") {
		t.Errorf("set answered %q", changed.Said)
	}

	mustRefuse(t, w, "set", In{
		Args: map[string]string{"key": "colour", "value": "blue"}, By: "operator",
	})
}

// TestShapesAndCheckouts. The flows that ship and the checkouts in view,
// both listed rather than described.
func TestShapesAndCheckouts(t *testing.T) {
	w := worldOf(t)

	flows := mustAsk(t, w, "flows", In{By: "operator"})
	if flows.Said == "" {
		t.Error("flows said nothing at all")
	}

	empty := mustAsk(t, w, "repos", In{By: "operator"})
	if !strings.Contains(empty.Said, "no repositories") {
		t.Errorf("repos with nothing in view reads %q", empty.Said)
	}

	w.board = board.Board{RepoList: []board.RepoInfo{{Name: "acme", Path: "/src/acme"}}}

	full := mustAsk(t, w, "repos", In{By: "operator"})
	if !strings.Contains(full.Said, "acme") {
		t.Errorf("repos reads %q", full.Said)
	}
}

// TestExportWritesWhereItIsTold. A relative into is read against wherever
// the caller stands, which is what stops two ways in disagreeing about
// where "out" is.
func TestExportWritesWhereItIsTold(t *testing.T) {
	w := worldOf(t)

	rel := mustAsk(t, w, "export", In{Args: map[string]string{"into": "out"}, By: "operator"})
	if !strings.Contains(rel.Said, "out") {
		t.Errorf("export answered %q", rel.Said)
	}

	abs := mustAsk(t, w, "export", In{Args: map[string]string{"into": t.TempDir()}, By: "operator"})
	if abs.Said == "" {
		t.Error("export said nothing at all")
	}
}

// TestWhatOrbitKnows, in the columns a terminal draws: what it does, how
// far it reaches, and the sentence.
func TestWhatOrbitKnows(t *testing.T) {
	w := worldOf(t)

	bare := mustAsk(t, w, "knowledge", In{By: "operator"})
	_ = bare

	w.facts = []knowledge.Fact{
		{Phrase: "PRs in English", Scope: knowledge.Scope{Kind: knowledge.General}},
		{
			Phrase: "amounts are cents", Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/src/acme"},
			Stops: true, Check: "make check",
		},
		{Phrase: "wrap errors", Scope: knowledge.Scope{Kind: knowledge.Language, Lang: "go"}},
		{Phrase: "retry on 5xx", Scope: knowledge.Scope{Kind: knowledge.Dir, Path: "webhook"}},
		{Phrase: "never index", Scope: knowledge.Scope{Kind: knowledge.File, Path: "notes.md"}},
		{Phrase: "pure", Scope: knowledge.Scope{Kind: knowledge.Symbol, Path: "sum.go", Symbol: "Sum"}},
	}

	out := mustAsk(t, w, "knowledge", In{By: "operator"})
	for _, want := range []string{"stops", "warns", "acme", "go", "Sum"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("knowledge does not mention %q:\n%s", want, out.Said)
		}
	}
}

// TestEnginesAndQuota. The catalogue and the windows, read without spending
// anything: a reading changes nothing.
func TestEnginesAndQuota(t *testing.T) {
	w := worldOf(t)

	if out := mustAsk(t, w, "engines", In{By: "operator"}); out.Said == "" {
		t.Error("engines said nothing at all")
	}

	if out := mustAsk(t, w, "quota", In{By: "operator"}); out.Said == "" {
		t.Error("quota said nothing at all")
	}
}

// TestAWhileAgoInWords. Seconds read as now, minutes, and hours.
func TestAWhileAgoInWords(t *testing.T) {
	if got := awhile(0); got != "now" {
		t.Errorf("awhile(0) = %q, want now", got)
	}

	if got := awhile(-5); got != "now" {
		t.Errorf("awhile(-5) = %q, want now", got)
	}

	if got := awhile(90); got != "1m" {
		t.Errorf("awhile(90) = %q, want 1m", got)
	}

	if got := awhile(3700); got != "1h1m" {
		t.Errorf("awhile(3700) = %q, want 1h1m", got)
	}
}
