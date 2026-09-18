package db

// Sentences waiting to be told whether they were rules.

import (
	"strings"
	"testing"
	"time"
)

// said is a proposal stamped at the nth second of the morning the rest of
// this package's tests happen on, so a test can say which came first without
// sleeping. The instant is the row's own name, so no two share one.
func said(n int, text string) Proposal {
	return Proposal{
		SaidAt: time.Date(2026, 9, 17, 9, 0, n, 0, time.UTC),
		Said:   text,
		By:     "operator",
	}
}

// TestAProposalComesBackWithEverythingItWasSaidWith. Whoever is about to
// decide reads where it came from first — a sentence cannot be agreed with
// until it can be placed — so every field of the row is part of the answer.
func TestAProposalComesBackWithEverythingItWasSaidWith(t *testing.T) {
	d := open(t)

	p := said(1, "run the tests before you push")
	p.About = "ACME-1"
	p.Repo = "/src/acme"
	p.Path = "internal/db"
	p.Topic = "tests"
	p.Habit = "asks-for-the-tests"
	p.Gate = "make check"

	if err := d.Propose(p); err != nil {
		t.Fatalf("propose: %v", err)
	}

	waiting, err := d.Waiting()
	if err != nil {
		t.Fatalf("read what is waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Fatalf("%d proposals are waiting, want one", len(waiting))
	}

	got := waiting[0]
	if !got.SaidAt.Equal(p.SaidAt) {
		t.Errorf("it is stamped %s, want %s", got.SaidAt, p.SaidAt)
	}

	got.SaidAt = p.SaidAt
	p.State = Waiting

	if got != p {
		t.Errorf("it came back as %+v, want %+v", got, p)
	}
}

// TestProposingTheSameLineTwiceMakesOneRow. The caller is a screen opening:
// it reads the whole thread every time and most of what it finds it has
// found before. A second row would offer the same sentence twice.
func TestProposingTheSameLineTwiceMakesOneRow(t *testing.T) {
	d := open(t)

	p := said(1, "run the tests before you push")

	for range 2 {
		if err := d.Propose(p); err != nil {
			t.Fatalf("propose: %v", err)
		}
	}

	waiting, err := d.Waiting()
	if err != nil {
		t.Fatalf("read what is waiting: %v", err)
	}

	if len(waiting) != 1 {
		t.Errorf("proposing one line twice left %d rows, want one", len(waiting))
	}
}

// TestWhatIsWaitingIsOldestFirst. The tray reads them in the order they were
// said, because a later sentence is often the same rule said better and the
// reader wants to have met the first one.
func TestWhatIsWaitingIsOldestFirst(t *testing.T) {
	d := open(t)

	for _, p := range []Proposal{said(3, "third"), said(1, "first"), said(2, "second")} {
		if err := d.Propose(p); err != nil {
			t.Fatalf("propose %q: %v", p.Said, err)
		}
	}

	waiting, err := d.Waiting()
	if err != nil {
		t.Fatalf("read what is waiting: %v", err)
	}

	var order []string
	for _, p := range waiting {
		order = append(order, p.Said)
	}

	if strings.Join(order, ",") != "first,second,third" {
		t.Errorf("they are waiting in the order %v, want first, second, third", order)
	}
}

// TestADecidedProposalStopsWaiting. A row moves out of waiting once: the
// tray is what nobody has answered, and an answered sentence asked again is
// a question already settled.
func TestADecidedProposalStopsWaiting(t *testing.T) {
	for _, state := range []string{Kept, Dropped} {
		t.Run(state, func(t *testing.T) {
			d := open(t)

			p := said(1, "run the tests before you push")
			if err := d.Propose(p); err != nil {
				t.Fatalf("propose: %v", err)
			}

			if err := d.Decide(p.SaidAt, state); err != nil {
				t.Fatalf("decide %s: %v", state, err)
			}

			waiting, err := d.Waiting()
			if err != nil {
				t.Fatalf("read what is waiting: %v", err)
			}

			if len(waiting) != 0 {
				t.Errorf("%d proposals are still waiting after being %s, want none",
					len(waiting), state)
			}
		})
	}
}

// TestDecidingTwiceIsRefused. Two answers to one question is not a thing a
// screen should be able to produce by being clicked twice, and the refusal
// says which sentence rather than that something went wrong.
func TestDecidingTwiceIsRefused(t *testing.T) {
	d := open(t)

	p := said(1, "run the tests before you push")
	if err := d.Propose(p); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := d.Decide(p.SaidAt, Kept); err != nil {
		t.Fatalf("decide: %v", err)
	}

	err := d.Decide(p.SaidAt, Dropped)
	if err == nil {
		t.Fatal("deciding twice was allowed")
	}

	if !strings.Contains(err.Error(), "already been decided") {
		t.Errorf("the refusal is %q, want it to say the sentence was already decided", err)
	}
}

// TestDecidingWhatNobodySaidIsRefused. The row is named by the instant, and
// an instant with no row is the same answer as one already decided: nothing
// moved, and saying so is better than reporting a success that wrote nothing.
func TestDecidingWhatNobodySaidIsRefused(t *testing.T) {
	d := open(t)

	if err := d.Decide(said(1, "").SaidAt, Kept); err == nil {
		t.Error("deciding about a sentence nobody said was allowed")
	}
}

// TestAHabitAskedAboutIsNotAskedAgain. Dropping a rule is an answer, so a
// habit is answered whatever was said about it — otherwise the tray offers
// the same rule every time the same sentences are read.
func TestAHabitAskedAboutIsNotAskedAgain(t *testing.T) {
	cases := []struct {
		name  string
		state string
	}{
		{"kept", Kept},
		{"dropped", Dropped},
		{"still waiting", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := open(t)

			p := said(1, "run the tests before you push")
			p.Habit = "asks-for-the-tests"

			if err := d.Propose(p); err != nil {
				t.Fatalf("propose: %v", err)
			}

			if c.state != "" {
				if err := d.Decide(p.SaidAt, c.state); err != nil {
					t.Fatalf("decide %s: %v", c.state, err)
				}
			}

			answered, err := d.Answered(p.Habit)
			if err != nil {
				t.Fatalf("ask whether the habit was offered: %v", err)
			}

			if !answered {
				t.Error("a habit already offered reads as never offered")
			}
		})
	}
}

// TestAHabitNobodyDrewFromIsUnanswered. A sentence somebody said outright
// carries no habit, and the empty name is not a habit every one of them
// shares: answering true there would silence every habit at once.
func TestAHabitNobodyDrewFromIsUnanswered(t *testing.T) {
	d := open(t)

	p := said(1, "run the tests before you push")
	if err := d.Propose(p); err != nil {
		t.Fatalf("propose: %v", err)
	}

	for _, habit := range []string{"", "never-offered"} {
		answered, err := d.Answered(habit)
		if err != nil {
			t.Fatalf("ask about %q: %v", habit, err)
		}

		if answered {
			t.Errorf("the habit %q reads as already offered", habit)
		}
	}
}
