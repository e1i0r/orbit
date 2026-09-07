package supervisor

// What the supervisor is told it already knows, and what it does about
// being told something it should have known.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// TestTheSupervisorIsToldWhatOrbitAlreadyKnows.
//
// It answers questions about the work and directs tasks, and it was doing
// both without the standing rules in front of it — so it could say something
// a gate would refuse an hour later, and could be told a rule it had already
// been given without noticing.
func TestTheSupervisorIsToldWhatOrbitAlreadyKnows(t *testing.T) {
	asked := buildSupervisorPrompt("", nil, "what should I look at?", []knowledge.Fact{
		{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "the PRs are written in English"},
	})

	if !strings.Contains(asked, "the PRs are written in English") {
		t.Errorf("the supervisor is not told what Orbit knows:\n%s", asked)
	}
}

// TestKnowingNothingDrawsNoSection, the rule every other part of this prompt
// follows: an empty heading is a question the model answers for itself.
func TestKnowingNothingDrawsNoSection(t *testing.T) {
	asked := buildSupervisorPrompt("", nil, "what should I look at?", nil)
	if strings.Contains(asked, "What Orbit already knows") {
		t.Errorf("a heading was drawn with nothing under it:\n%s", asked)
	}
}

// TestTheSupervisorIsAskedToOfferToRememberThings.
//
// The axis is not "you said it twice": it is being told something that should
// have been standing. A model reading the thread and the facts can see that,
// where matching text cannot — the same thing said in different words is not
// the same string.
func TestTheSupervisorIsAskedToOfferToRememberThings(t *testing.T) {
	asked := strings.ToLower(buildSupervisorPrompt("", nil, "the fuzz tests hang sometimes", nil))

	for _, want := range []string{"orbit_learn", "offer"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the supervisor is not asked to offer to write things down (%q missing):\n%s", want, asked)
		}
	}
}

// TestTheSupervisorIsToldNotToWriteWithoutBeingAsked. Nothing is promoted
// silently: a rule that appeared without somebody agreeing to it is a rule
// nobody put there, and the first one of those costs the whole feature its
// trust.
func TestTheSupervisorIsToldNotToWriteWithoutBeingAsked(t *testing.T) {
	asked := strings.ToLower(buildSupervisorPrompt("", nil, "anything", nil))
	if !strings.Contains(asked, "without") || !strings.Contains(asked, "agree") {
		t.Errorf("nothing tells the supervisor to ask first:\n%s", asked)
	}
}

// TestTheFactsComeOffDiskFromEveryRepository.
//
// The supervisor is global — it answers about tasks wherever they are — so
// the facts it is shown are every repository's, and nothing else proves the
// walk actually happens. Without this the feature can be wired end to end
// and still put an empty section in front of the model forever.
func TestTheFactsComeOffDiskFromEveryRepository(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	ledger, checkout := filepath.Join(t.TempDir(), "ledger"), filepath.Join(t.TempDir(), "checkout")

	for _, repo := range []string{ledger, checkout} {
		if _, err := s.CreateTaskDir(repo, "LED-1"); err != nil {
			t.Fatalf("CreateTaskDir(%q): %v", repo, err)
		}
	}

	ks := knowledge.NewStore(root)

	everywhere := knowledge.Scope{Kind: knowledge.General}
	inLedger := knowledge.Scope{Kind: knowledge.Repo, Repo: ledger}
	inCheckout := knowledge.Scope{Kind: knowledge.Repo, Repo: checkout}

	for _, f := range []knowledge.Fact{
		{Scope: everywhere, Source: knowledge.Human, Phrase: "the PRs are written in English"},
		{Scope: inLedger, Source: knowledge.Human, Phrase: "the ledger only appends"},
		{Scope: inCheckout, Source: knowledge.Human, Phrase: "the card brand comes from the token"},
		{Scope: inLedger, Source: knowledge.Human, Phrase: "the suite needs a Postgres", Off: true},
	} {
		if _, err := ks.Save(f); err != nil {
			t.Fatalf("save %q: %v", f.Phrase, err)
		}
	}

	var said []string
	for _, f := range standing(s) {
		said = append(said, f.Phrase)
	}

	for _, want := range []string{"the PRs are written in English", "the ledger only appends", "the card brand comes from the token"} {
		if !slices.Contains(said, want) {
			t.Errorf("the supervisor was not shown %q: %v", want, said)
		}
	}

	// The state root is read once per repository, so a fact that belongs to
	// no repository would be told twice — and a model weighs a rule it was
	// given twice more heavily than the one beside it.
	if n := slices.Index(said, "the PRs are written in English"); n >= 0 && slices.Contains(said[n+1:], "the PRs are written in English") {
		t.Errorf("a general fact was told once per repository: %v", said)
	}

	// A fact somebody turned off is a fact nothing is told, here as in a
	// phase's prompt: the file stays so the screen can turn it back on.
	if slices.Contains(said, "the suite needs a Postgres") {
		t.Errorf("a fact that was turned off was still told: %v", said)
	}
}

// TestAStateRootWithNothingInItSaysNothing. Every repository starts knowing
// nothing about itself, and the supervisor still has to answer.
func TestAStateRootWithNothingInItSaysNothing(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	if got := standing(s); len(got) != 0 {
		t.Errorf("facts came out of an empty root: %v", got)
	}
}

// TestARuleSaysWhichRepositoryItIsAbout.
//
// The supervisor answers about every repository at once, so a rule that
// arrives as a bare sentence is a rule it will apply to the checkout beside
// the one it was written for.
func TestARuleSaysWhichRepositoryItIsAbout(t *testing.T) {
	said := alreadyKnown([]knowledge.Fact{
		{Scope: knowledge.Scope{Kind: knowledge.General}, Phrase: "the PRs are written in English"},
		{Scope: knowledge.Scope{Kind: knowledge.Language, Lang: "go"}, Phrase: "never discard an error"},
		{Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/code/ledger"}, Phrase: "the ledger only appends"},
		{Scope: knowledge.Scope{Kind: knowledge.Dir, Repo: "/code/ledger", Path: "money"}, Phrase: "minor units only"},
		{
			Scope:  knowledge.Scope{Kind: knowledge.Symbol, Repo: "/code/ledger", Path: "ledger.go", Symbol: "Charge"},
			Phrase: "stays idempotent",
		},
	})

	for _, want := range []string{
		"- the PRs are written in English",
		"- (go) never discard an error",
		"- (ledger) the ledger only appends",
		"- (ledger/money) minor units only",
		"- (ledger/ledger.go#Charge) stays idempotent",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("%q is not in what the supervisor is shown:\n%s", want, said)
		}
	}
}

// TestOneDamagedRepositoryDoesNotCostTheRest.
//
// The supervisor is global, so one checkout with a fact nobody can read
// would otherwise take every other repository's rules down with it — and the
// answer that came back would look exactly like an answer given with the
// rules in hand.
func TestOneDamagedRepositoryDoesNotCostTheRest(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	broken, sound := filepath.Join(t.TempDir(), "broken"), filepath.Join(t.TempDir(), "sound")

	for _, repo := range []string{broken, sound} {
		if _, err := s.CreateTaskDir(repo, "LED-1"); err != nil {
			t.Fatalf("CreateTaskDir(%q): %v", repo, err)
		}
	}

	ks := knowledge.NewStore(root)

	saved := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.Repo, Repo: sound},
		Source: knowledge.Human,
		Phrase: "the ledger only appends",
	}
	if _, err := ks.Save(saved); err != nil {
		t.Fatalf("save: %v", err)
	}

	// A header naming a source that is not one: what a hand-written file or
	// a bad merge leaves behind, and what read refuses rather than guessing
	// at.
	damaged := filepath.Join(broken, ".orbit", "knowledge", "rubbish.md")
	if err := os.MkdirAll(filepath.Dir(damaged), 0o755); err != nil {
		t.Fatalf("make room: %v", err)
	}

	if err := os.WriteFile(damaged, []byte("---\nsource: nobody\n---\n\nsomething\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var said []string
	for _, f := range standing(s) {
		said = append(said, f.Phrase)
	}

	if !slices.Contains(said, "the ledger only appends") {
		t.Errorf("one unreadable repository cost the others their rules: %v", said)
	}
}
