package task

// A correction typed at a run is half the time a rule, said here because
// here is where somebody was standing.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/learn"
)

// TestARuleTypedAtARunReachesTheTray, carrying the task and the checkout it
// was said about, so that whoever answers it knows where it came from.
func TestARuleTypedAtARunReachesTheTray(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-9", "fix the totals", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "operator", "never merge a PR without the tests passing"); err != nil {
		t.Fatalf("Direct: %v", err)
	}

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 1 {
		t.Fatalf("%d sentences reached the tray, want the rule just typed", len(waiting))
	}

	one := waiting[0]
	if one.Text != "never merge a PR without the tests passing" {
		t.Errorf("the tray holds %q", one.Text)
	}

	if one.By != "operator" || one.About != tk.ID || one.Repo != r.Path {
		t.Errorf("the tray does not say where it came from: %+v", one)
	}
}

// TestACorrectionAboutThisRunIsNotARule, which is most of them. A tray that
// filled with every directive is a tray nobody opens.
func TestACorrectionAboutThisRunIsNotARule(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-10", "fix the totals", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "operator", "the endpoint should reject negative amounts"); err != nil {
		t.Fatalf("Direct: %v", err)
	}

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("a correction about this run was offered as a rule: %+v", waiting)
	}
}

// TestOrbitDirectingARunIsNotYouSayingSomething.
//
// Its own loop passes on what you already said to it, and the thread caught
// that the first time. Read as yours, the tray would fill with Orbit quoting
// you back at yourself.
func TestOrbitDirectingARunIsNotYouSayingSomething(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-11", "fix the totals", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "supervisor", "never merge a PR without the tests passing"); err != nil {
		t.Fatalf("Direct: %v", err)
	}

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("Orbit's own directive was offered back as a rule: %+v", waiting)
	}
}

// TestTheCorrectionStandsWhateverTheTrayDoes. The directive is what matters
// at that moment; noticing it was also a rule is a question asked about it.
func TestTheCorrectionStandsWhateverTheTrayDoes(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-12", "fix the totals", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "operator", "always run make check first"); err != nil {
		t.Fatalf("Direct: %v", err)
	}

	notes, err := unconsumedNotes(s, tk)
	if err != nil {
		t.Fatalf("read the notes: %v", err)
	}

	if len(notes) != 1 || notes[0] != "[operator] always run make check first" {
		t.Errorf("the correction reached the record as %v", notes)
	}
}
