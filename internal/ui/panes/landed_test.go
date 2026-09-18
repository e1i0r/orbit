package panes

// The verb that has just come back.
//
// The other half of the verb still out, and it read zero. The band said who
// was working and then went quiet, and quiet is the same picture as a key
// that did nothing: a reader could not tell a pull request that had been
// opened from one that never was.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// TestAVerbThatJustCameBackIsSaidAndAnOldOneIsNot. Just is the window a
// reader is still looking at the screen for; past it, what a verb did is
// history and the flow tree is where history is read.
func TestAVerbThatJustCameBackIsSaidAndAnOldOneIsNot(t *testing.T) {
	just := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", By: "operator", At: ago(4 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened #12", At: ago(time.Minute)},
	})

	st, landed := Landed(just)
	if !landed {
		t.Fatal("a verb that came back a minute ago says nothing")
	}

	if st.Verb != "PR" || st.Said != "opened #12" {
		t.Errorf("it came back as %+v, want the verb and what it said", st)
	}

	old := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", By: "operator", At: ago(time.Hour)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened #12", At: ago(30 * time.Minute)},
	})

	if _, still := Landed(old); still {
		t.Error("a verb that came back half an hour ago is still being announced")
	}
}

// TestAVerbStillOutHasNotLanded. The two are one question asked from both
// ends, and a band saying a verb came back while it is still out is the
// worst of the three answers.
func TestAVerbStillOutHasNotLanded(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", By: "operator", At: ago(time.Minute)},
	})

	if _, landed := Landed(e); landed {
		t.Error("a verb that is still out reads as one that came back")
	}
}

// TestTheNewestAnswerIsTheOneAnnounced. Two verbs can be out at once, and
// the one a reader is asking about is the one that just came back.
func TestTheNewestAnswerIsTheOneAnnounced(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", At: ago(3 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened #12", At: ago(2 * time.Minute)},
		{Kind: "deliver.asked", Verb: "FIX CHECKS", At: ago(2 * time.Minute)},
		{Kind: "deliver.answered", Verb: "FIX CHECKS", Text: "green", At: ago(time.Minute)},
	})

	st, landed := Landed(e)
	if !landed || st.Verb != "FIX CHECKS" {
		t.Errorf("it announced %+v, want the one that came back last", st)
	}
}

// TestWhatAVerbSaidIsItsOwnSentence. A verb that opened a pull request
// answered with the number; a reader who wanted to know whether it finished
// is told what finished, in the engine's words rather than in ones written
// here.
func TestWhatAVerbSaidIsItsOwnSentence(t *testing.T) {
	p := words.For("en")

	cases := []struct {
		name  string
		st    Step
		wants []string
		not   string
	}{
		{
			name:  "it said what it did",
			st:    Step{Verb: "PR", Said: "opened #12", Ended: ago(time.Minute)},
			wants: []string{"PR", "opened #12", "ago"},
		},
		{
			name:  "it broke, and why is what matters",
			st:    Step{Verb: "PR", Said: "opened #12", Cause: "gh: not logged in", Ended: ago(time.Minute)},
			wants: []string{"PR", "broken", "gh: not logged in"},
			// The cause wins over what it said: a verb that broke did not
			// do the thing its own sentence claims.
			not: "opened #12",
		},
		{
			name:  "it came back with nothing to say",
			st:    Step{Verb: "PR", Ended: ago(time.Minute)},
			wants: []string{"PR", "came back"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			said := CameBack(p, c.st, now)

			for _, want := range c.wants {
				if !strings.Contains(said, want) {
					t.Errorf("it said %q, want %q in it", said, want)
				}
			}

			if c.not != "" && strings.Contains(said, c.not) {
				t.Errorf("it said %q, and %q has no business in it", said, c.not)
			}
		})
	}
}

// TestOnlyTheFirstLineOfAnAnswerReachesTheBand. The band has one line and
// the reader is mid-task, so a stack trace pasted into it is the board gone.
func TestOnlyTheFirstLineOfAnAnswerReachesTheBand(t *testing.T) {
	p := words.For("en")

	st := Step{
		Verb:  "FIX CHECKS",
		Cause: "  the build failed  \ngo: cannot find module\nexit status 1",
		Ended: ago(time.Minute),
	}

	said := CameBack(p, st, now)

	if strings.Contains(said, "\n") {
		t.Errorf("the band was handed %d lines:\n%s", strings.Count(said, "\n")+1, said)
	}

	if !strings.Contains(said, "the build failed") {
		t.Errorf("it said %q, want the first line of what broke", said)
	}

	if strings.Contains(said, "exit status 1") {
		t.Errorf("it said %q, want only the first line", said)
	}
}

// TestAVerbWithNoClockOnItsAnswerIsNotAnnounced. An answer the record has no
// instant for cannot be placed in the window a reader is still watching, and
// announcing it would put an hour-old pull request in front of them as news.
func TestAVerbWithNoClockOnItsAnswerIsNotAnnounced(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", At: ago(2 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened #12"},
	})

	if _, landed := Landed(e); landed {
		t.Error("an answer with no instant on it was announced as news")
	}
}
