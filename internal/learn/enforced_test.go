package learn

// The rules a checkout is already keeping.

import (
	"testing"
)

// gated is the two gates every test here starts from.
func gated() []Gate {
	return []Gate{
		{Command: "go vet ./...", Where: ".github/workflows/check.yml"},
		{Command: "make test", Where: ".github/workflows/check.yml"},
	}
}

// TestAGateArrivesWithItsCommand, which is the whole of why this source is
// worth reading on its own: every other one brings a sentence and leaves the
// gate to whoever keeps it.
func TestAGateArrivesWithItsCommand(t *testing.T) {
	s := root(t)

	said, err := Enforced(s, "/w/acme", gated())
	if err != nil {
		t.Fatal(err)
	}

	if len(said) != 2 {
		t.Fatalf("two gates offered %d rules: %+v", len(said), said)
	}

	if said[0].Gate != "go vet ./..." {
		t.Errorf("the rule arrived with gate %q", said[0].Gate)
	}

	if said[0].Text != "go vet ./... has to pass" {
		t.Errorf("the rule reads %q", said[0].Text)
	}

	// And it waits, like everything else. That a repository runs something
	// is not the same as wanting Orbit to send work back over it.
	waiting, err := Waiting(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(waiting) != 2 {
		t.Fatalf("%d are waiting, want the two that were offered", len(waiting))
	}

	if waiting[0].Gate != "go vet ./..." {
		t.Errorf("the tray lost the command: %q", waiting[0].Gate)
	}
}

// TestAGateAlreadyAnsweredIsNotAskedAgain.
//
// This runs from a key and a button, so it runs often. A reading that
// offered the same six commands every time would be a tray nobody opens,
// and they share it with three other sources.
func TestAGateAlreadyAnsweredIsNotAskedAgain(t *testing.T) {
	s := root(t)

	if _, err := Enforced(s, "/w/acme", gated()); err != nil {
		t.Fatal(err)
	}

	said, err := Enforced(s, "/w/acme", gated())
	if err != nil {
		t.Fatal(err)
	}

	if len(said) != 0 {
		t.Errorf("reading twice offered %+v a second time", said)
	}
}

// TestTheSameCommandInTwoCheckoutsIsTwoQuestions, because a rule is written
// inside the repository it is about: agreeing to `make test` in one project
// says nothing about another.
func TestTheSameCommandInTwoCheckoutsIsTwoQuestions(t *testing.T) {
	s := root(t)

	if _, err := Enforced(s, "/w/acme", gated()); err != nil {
		t.Fatal(err)
	}

	said, err := Enforced(s, "/w/ledger", gated())
	if err != nil {
		t.Fatal(err)
	}

	if len(said) != 2 {
		t.Errorf("the second checkout offered %d rules, want its own two", len(said))
	}
}

// TestAReadingWithNoCheckoutIsRefused, rather than filing rules against
// nowhere — which is every repository on this machine.
func TestAReadingWithNoCheckoutIsRefused(t *testing.T) {
	if _, err := Enforced(root(t), "", gated()); err == nil {
		t.Error("a reading of no checkout was allowed")
	}
}
