package task

// The order a relay reaches for engines in, and the two readings it makes
// through ports that may not be there.
//
// Small pieces, and each one a place a run could have panicked or picked
// differently on two runs of the same task. A relay nobody can reason about
// is a relay nobody trusts with a task they left running.

import (
	"errors"
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
)

// TestTheReadersOwnEngineIsReachedForFirst. It is the only preference Orbit
// has been told: the settings name one, and it is the one a relay should try
// before any other.
func TestTheReadersOwnEngineIsReachedForFirst(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the test is over

	engines := map[string]engine.Engine{
		"claude": under{engine.NewFake(""), "claude"},
		"codex":  under{engine.NewFake(""), "codex"},
		"zeta":   under{engine.NewFake(""), "zeta"},
	}

	// With nothing set, the order is the one names sort in: nobody's
	// preference, and the same on every run of the same task.
	if got := inOrder(s, engines); !slices.Equal(got, []string{"claude", "codex", "zeta"}) {
		t.Errorf("with nothing set the order is %v, want them sorted", got)
	}

	if err := s.SaveSettings(store.Settings{Engine: "zeta"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	if got := inOrder(s, engines); !slices.Equal(got, []string{"zeta", "claude", "codex"}) {
		t.Errorf("the order is %v, want the reader's own first", got)
	}
}

// TestAnEngineSetButNotInThisBuildIsNotReachedFor. A settings file written
// by a newer orbit names engines this one does not have, and putting the
// name at the front of the list would be a relay handing a task to nothing.
func TestAnEngineSetButNotInThisBuildIsNotReachedFor(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the test is over

	if err := s.SaveSettings(store.Settings{Engine: "gpt"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	engines := map[string]engine.Engine{
		"claude": under{engine.NewFake(""), "claude"},
		"codex":  under{engine.NewFake(""), "codex"},
	}

	got := inOrder(s, engines)
	if slices.Contains(got, "gpt") {
		t.Errorf("the order is %v, and this build has no gpt", got)
	}

	if !slices.Equal(got, []string{"claude", "codex"}) {
		t.Errorf("the order is %v, want the ones this build has, sorted", got)
	}
}

// TestReadingTheNameOfAnEngineThatIsNotThereIsNotAPanic. A phase with no
// engine cannot have run out, so the empty name never reaches a relay — this
// is here so that reading the name is not a place a run can die.
func TestReadingTheNameOfAnEngineThatIsNotThereIsNotAPanic(t *testing.T) {
	if got := callsItself(nil); got != "" {
		t.Errorf("an engine that is not there calls itself %q, want nothing", got)
	}

	if got := callsItself(under{engine.NewFake(""), "zeta"}); got != "zeta" {
		t.Errorf("it calls itself %q, want zeta", got)
	}
}

// TestABuildWithNoAllowanceReadingSaysEveryEngineIsFree. That is what every
// run had before relays existed, and a build without the port must not
// behave as though every engine were empty.
func TestABuildWithNoAllowanceReadingSaysEveryEngineIsFree(t *testing.T) {
	if got := spare(nil, "claude"); !got.Free {
		t.Errorf("with no reading behind it, claude reads as %+v, want free", got)
	}

	left := onlyFree("codex", 0)

	if got := spare(left, "codex"); !got.Free {
		t.Errorf("codex reads as %+v, want free", got)
	}

	if got := spare(left, "claude"); got.Free {
		t.Errorf("claude reads as %+v, want nothing left", got)
	}
}

// TestARanOutFailureCarriesTheOneItWrapped. It is an error and not a result
// because the phase did not do the work — and its own kind of error because
// it is the one failure another engine can answer. Both halves have to
// survive: the sentence a reader sees, and the error underneath for whatever
// asks what really happened.
func TestARanOutFailureCarriesTheOneItWrapped(t *testing.T) {
	underneath := errors.New("Claude AI usage limit reached")

	err := error(&noneLeft{engine: "claude", err: underneath})

	if err.Error() != underneath.Error() {
		t.Errorf("it reads as %q, want the sentence underneath", err)
	}

	if !errors.Is(err, underneath) {
		t.Error("the error it wrapped cannot be reached through it")
	}

	none := ranDry(err)
	if none == nil {
		t.Fatal("a ran-out failure does not read as one")
	}

	if none.engine != "claude" {
		t.Errorf("it says %q ran out, want claude", none.engine)
	}

	// And everything else a phase can die of is not one: the next engine
	// would die of it too, so a relay must not answer it.
	if ranDry(errors.New("exit status 1")) != nil {
		t.Error("an ordinary failure reads as an engine that ran out")
	}

	if ranDry(nil) != nil {
		t.Error("no failure at all reads as an engine that ran out")
	}
}
