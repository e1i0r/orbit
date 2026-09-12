package verb

// The first family, asked for the way every way in asks for it.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// onAStore is a world with somewhere to write and nothing else. The rules
// family reaches the record and the reader's language and no further, so a
// world that answered more would be saying this test covers more than it
// does.
type onAStore struct {
	World

	store *store.Store
}

func (o onAStore) Store() *store.Store   { return o.store }
func (o onAStore) Words() *words.Printer { return words.For("") }

// trayOf is a world whose tray holds the sentences given, oldest first.
func trayOf(t *testing.T, said ...string) onAStore {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	at := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	for i, one := range said {
		when := at.Add(time.Duration(i) * time.Minute)
		if err := learn.Propose(s, learn.Said{At: when, Text: one}); err != nil {
			t.Fatalf("propose %q: %v", one, err)
		}
	}

	return onAStore{store: s}
}

// asked runs one verb by the whole of what it is called.
func asked(t *testing.T, w onAStore, name string, args map[string]string) (Out, error) {
	t.Helper()

	return Run(context.Background(), w, name, In{Args: args, By: "operator"})
}

// facts is what Orbit knows.
func facts(t *testing.T, w onAStore) []knowledge.Fact {
	t.Helper()

	got, err := knowledge.NewStore(w.store.Root()).Load("")
	if err != nil {
		t.Fatalf("read what Orbit knows: %v", err)
	}

	return got
}

// TestAChildIsAskedForByBothItsWords.
//
// The whole point of a family: `orbit rules keep` and never `orbit keep`,
// which says nothing about what. Every way in keys on the path, so a child
// answering to its own word alone would be a fifth vocabulary.
func TestAChildIsAskedForByBothItsWords(t *testing.T) {
	if _, found := One("rules keep"); !found {
		t.Error("rules keep is not findable by what it is called")
	}

	if _, found := One("keep"); found {
		t.Error("a child answers to its own word alone")
	}
}

// TestAFamilyIsOneLevelDeep. Two levels is a tree, a tree needs a way to be
// walked, and nothing Orbit does has been hard to say in two words.
func TestAFamilyIsOneLevelDeep(t *testing.T) {
	for _, v := range Every() {
		if v.Under == "" {
			continue
		}

		parent, found := One(v.Under)
		if !found {
			t.Errorf("%q belongs to %q, and there is no such verb", v.Name, v.Under)
			continue
		}

		if parent.Under != "" {
			t.Errorf("%q is two levels down, under %q", v.Name, parent.Path())
		}
	}
}

// TestTheTrayIsNumberedInTheOrderItWasSaid, which is the order keep and drop
// read it back in.
func TestTheTrayIsNumberedInTheOrderItWasSaid(t *testing.T) {
	w := trayOf(t, "never push without the tests passing", "always wrap errors")

	out, err := asked(t, w, "rules", nil)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}

	lines := strings.Split(out.Said, "\n")
	if len(lines) != 2 {
		t.Fatalf("the tray printed %d lines:\n%s", len(lines), out.Said)
	}

	if !strings.HasSuffix(lines[0], "never push without the tests passing") {
		t.Errorf("the first line is %q", lines[0])
	}

	if _, ok := out.Saw.([]learn.Said); !ok {
		t.Errorf("a surface that draws structures was handed %T", out.Saw)
	}
}

// TestAnEmptyTraySaysSoInWords, rather than printing nothing at a reader who
// then has to work out whether it ran.
func TestAnEmptyTraySaysSoInWords(t *testing.T) {
	out, err := asked(t, trayOf(t), "rules", nil)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}

	if !strings.Contains(out.Said, "waiting") {
		t.Errorf("an empty tray said %q", out.Said)
	}
}

// TestKeepingOneIsHowOrbitLearnsIt, in the words it was said in when nobody
// typed better ones.
func TestKeepingOneIsHowOrbitLearnsIt(t *testing.T) {
	w := trayOf(t, "never push without the tests passing", "always wrap errors")

	if _, err := asked(t, w, "rules keep", map[string]string{"n": "2"}); err != nil {
		t.Fatalf("rules keep: %v", err)
	}

	got := facts(t, w)
	if len(got) != 1 || got[0].Phrase != "always wrap errors" {
		t.Fatalf("Orbit learned %v, want the second sentence", got)
	}

	if got[0].Source != knowledge.Human {
		t.Error("a sentence you kept is not written down as yours")
	}

	out, err := asked(t, w, "rules", nil)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}

	if strings.Contains(out.Said, "always wrap errors") {
		t.Error("a sentence already kept is still in the tray")
	}
}

// TestKeepingItInBetterWordsWritesTheBetterOnes.
//
// Correcting is how most of these are accepted: what you meant is what you
// typed the second time, and a check turns advice into something that can
// refuse work.
func TestKeepingItInBetterWordsWritesTheBetterOnes(t *testing.T) {
	w := trayOf(t, "never push without tests")

	better := "never open a pull request until make check is green"

	_, err := asked(t, w, "rules keep", map[string]string{
		"n": "1", "text": better, "check": "make check",
	})
	if err != nil {
		t.Fatalf("rules keep: %v", err)
	}

	got := facts(t, w)
	if len(got) != 1 || got[0].Phrase != better {
		t.Fatalf("Orbit learned %v, want it as it was corrected", got)
	}

	if got[0].Action() != knowledge.Stops {
		t.Error("a rule given a command that answers yes or no does not refuse work")
	}
}

// TestDroppingOneLeavesItWhereItWasSaid.
func TestDroppingOneLeavesItWhereItWasSaid(t *testing.T) {
	w := trayOf(t, "never mind, that was not a rule")

	if _, err := asked(t, w, "rules drop", map[string]string{"n": "1"}); err != nil {
		t.Fatalf("rules drop: %v", err)
	}

	if got := facts(t, w); len(got) != 0 {
		t.Errorf("Orbit learned %d things from a sentence you said was not a rule", len(got))
	}
}

// TestANumberNothingIsWaitingUnderIsRefused, and says what there is instead.
// The number comes off a listing, and a listing read a minute ago is a
// listing something may have left.
func TestANumberNothingIsWaitingUnderIsRefused(t *testing.T) {
	w := trayOf(t, "always wrap errors")

	for _, n := range []string{"0", "2", "seven", ""} {
		_, err := asked(t, w, "rules keep", map[string]string{"n": n})
		if err == nil {
			t.Errorf("rule %q was kept and there is no such rule", n)
		}
	}

	if got := facts(t, w); len(got) != 0 {
		t.Errorf("Orbit learned %d things from numbers that named nothing", len(got))
	}
}
