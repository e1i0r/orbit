package known

// Walking a row's options without opening its list, and the two rows that
// only exist when there is something for them to hold.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestTheArrowsWalkARowsOptions. The list behind a row is for finding one
// answer among many; ← and → are for the reader who wants the next one, and
// a row that only opened a box would make that two keystrokes and a search.
func TestTheArrowsWalkARowsOptions(t *testing.T) {
	s, e := writing(t)

	r, _ := func() (aRow, bool) {
		open, _ := s.Key(press("enter"), e)

		return open.picked(e)
	}()

	if len(r.options) < 3 {
		t.Fatalf("the row offers %d options, want the checkout and its folders", len(r.options))
	}

	s, _ = s.Key(press("right"), e)
	if got := s.held(r); got != r.options[1].value {
		t.Errorf("→ left the row holding %q, want %q", got, r.options[1].value)
	}

	s, _ = s.Key(press("left"), e)
	if got := s.held(r); got != r.options[0].value {
		t.Errorf("← left the row holding %q, want %q", got, r.options[0].value)
	}

	// And it wraps rather than stopping: walking is for a short list, and a
	// reader one past the end wants the first one back.
	s, _ = s.Key(press("left"), e)
	if got := s.held(r); got != r.options[len(r.options)-1].value {
		t.Errorf("← from the first left the row holding %q, want the last", got)
	}
}

// TestAValueNoneOfThemOffersCountsAsPastTheEnd. Somebody typed a path of
// their own, then reached for the arrows: the first press has to land on the
// first option rather than nowhere.
func TestAValueNoneOfThemOffersCountsAsPastTheEnd(t *testing.T) {
	s, e := writing(t)
	s = typed(s, e, "cmd/orbit")

	r, _ := func() (aRow, bool) {
		open, _ := s.Key(press("enter"), e)

		return open.picked(e)
	}()

	s, _ = s.Key(press("right"), e)

	if got := s.held(r); got != r.options[0].value {
		t.Errorf("→ from a typed path left the row holding %q, want the first option", got)
	}
}

// TestTheRowThatPicksACheckoutIsOnlyThereWhenThereIsAChoice. One repository
// on the board is not a question, and a form that asked it anyway would be a
// row somebody tabs past forever.
func TestTheRowThatPicksACheckoutIsOnlyThereWhenThereIsAChoice(t *testing.T) {
	one, e := writing(t)

	for _, r := range one.rows2(e) {
		if r.which == rowRepo {
			t.Fatal("a board with one checkout asked which one")
		}
	}

	e.Repos = []string{"/w/orbit", "/w/acme"}

	var picks aRow

	for _, r := range one.rows2(e) {
		if r.which == rowRepo {
			picks = r
		}
	}

	// One per repository, and one more: a rule about every project on this
	// machine is a rule that travels to none of them, and it has to be
	// possible to say so from the row that asks.
	if len(picks.options) != len(e.Repos)+1 {
		t.Fatalf("the row offers %d checkouts, want one per repository and the none",
			len(picks.options))
	}

	if last := picks.options[len(picks.options)-1]; last.value != "" {
		t.Errorf("the last option holds %q, want the one that is about no checkout", last.value)
	}
}

// TestMovingToAnotherCheckoutTakesThePlaceWithIt. The folder beside it was
// read off the one being left, and a path that means nothing in the new
// checkout is worse than none.
func TestMovingToAnotherCheckoutTakesThePlaceWithIt(t *testing.T) {
	s, e := writing(t)
	e.Repos = []string{"/w/orbit", "/w/acme"}

	s = typed(s, e, "internal/db")

	if got := strings.TrimSpace(s.in[factWhere].Val); got != "internal/db" {
		t.Fatalf("the place holds %q, want the path that was typed", got)
	}

	var picks aRow

	for _, r := range s.rows2(e) {
		if r.which == rowRepo {
			picks = r
		}
	}

	s = s.walkTo(picks, picks.options[1])

	if s.repo != picks.options[1].value {
		t.Errorf("the rule is about %q, want the checkout that was moved to", s.repo)
	}

	if got := strings.TrimSpace(s.in[factWhere].Val); got != "" {
		t.Errorf("the place came along as %q, want it emptied", got)
	}
}

// TestThePlaceIsOnTheScreenWhileItIsTyped. A row whose value is typed
// rather than picked has to show what is in it as it goes in: a form that
// only redraws on save is a form somebody types into blind.
func TestThePlaceIsOnTheScreenWhileItIsTyped(t *testing.T) {
	s, e := writing(t)

	// The row opens holding the whole checkout, and says so in words rather
	// than as a blank line.
	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "all of orbit") {
		t.Errorf("the place row drew nothing for the checkout:\n%s", drawn)
	}

	s = typed(s, e, "internal/db")

	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "internal/db") {
		t.Errorf("what was typed is not on the screen:\n%s", drawn)
	}

	// And the hint under it changes with what the row holds, because
	// "internal/db" on its own does not say whether that is where the rule
	// applies or where it was written.
	if drawn := drawnKnowledge(t, s, e); !strings.Contains(drawn, "only told when the work is inside") {
		t.Errorf("the hint did not follow what was typed:\n%s", drawn)
	}
}

// TestARuleSentBackAppliesAgain. Both ways a rule ends up waiting — paused
// on purpose, or skipped once — have the same answer, and it says so rather
// than leaving the rule in a band nobody can get it out of.
func TestARuleSentBackAppliesAgain(t *testing.T) {
	paused := knowledge.Rule{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "the fuzz tests hang",
		State: knowledge.Paused, Why: "too noisy", Review: true,
	}

	s, e := onScreen(t, paused)

	var wrote knowledge.Rule

	e.Replace = func(_, now knowledge.Rule) error {
		wrote = now

		return nil
	}

	s, _ = s.Key(press("enter"), e)
	if !s.reading {
		t.Fatal("enter did not open the rule")
	}

	s, out := s.Key(press("u"), e)

	if wrote.State != knowledge.Active {
		t.Errorf("the rule was left at %v, want it applying again", wrote.State)
	}

	if wrote.Why != "" || wrote.Review {
		t.Errorf("it applies again as %+v, want the reason and the waiting cleared", wrote)
	}

	if !strings.Contains(out.Said, "applies again") {
		t.Errorf("it said %q, want it to say the rule applies again", out.Said)
	}

	if s.reading {
		t.Error("deciding left the rule open")
	}
}

// TestReadingWhatTheCheckoutRefusesWorkOverIsTheOneFreeKey. The other three
// readings ask a model, and a key that spends money is a key somebody
// presses by accident.
func TestReadingWhatTheCheckoutRefusesWorkOverIsTheOneFreeKey(t *testing.T) {
	cases := []struct {
		name  string
		found int
		want  string
	}{
		{"nothing unanswered", 0, "nothing this checkout refuses work over"},
		{"one waiting", 1, "1 rule is waiting"},
		{"several waiting", 3, "3 rules are waiting"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, e := onScreen(t, known("the fuzz tests hang", knowledge.Scope{Kind: knowledge.General}))
			e.Enforced = func(string) (int, error) { return c.found, nil }

			_, out := s.Key(press("g"), e)

			if !strings.Contains(out.Said, c.want) {
				t.Errorf("it said %q, want %q in it", out.Said, c.want)
			}
		})
	}
}

// TestAReadingThatFailedSaysWhat. Nothing is hidden: a reading that could
// not open the repository says what stopped it rather than answering as if
// the checkout refused work over nothing.
func TestAReadingThatFailedSaysWhat(t *testing.T) {
	s, e := onScreen(t, known("the fuzz tests hang", knowledge.Scope{Kind: knowledge.General}))
	e.Enforced = func(string) (int, error) { return 0, errNoCheckout }

	_, out := s.Key(press("g"), e)

	if !strings.Contains(out.Said, errNoCheckout.Error()) {
		t.Errorf("it said %q, want what stopped it", out.Said)
	}

	// And a window built without that port at all says nothing rather than
	// reaching for it.
	e.Enforced = nil

	if _, out := s.Key(press("g"), e); out.Said != "" {
		t.Errorf("a window with no reading behind the key said %q", out.Said)
	}
}

// errNoCheckout is what a reading of a checkout that is not there answers.
var errNoCheckout = errNoCheckoutType{}

type errNoCheckoutType struct{}

func (errNoCheckoutType) Error() string { return "/w/orbit is not a checkout" }
