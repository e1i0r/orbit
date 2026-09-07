package roster

// What a window says about itself.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/words"
)

// TestHowLongUntilItComesBack. The number a reader is deciding on is when
// they can run again, so the largest unit that still says something is the
// one drawn: "2h15m" and not "8100s".
func TestHowLongUntilItComesBack(t *testing.T) {
	for _, c := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{-5 * time.Second, "0s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
	} {
		if got := resetIn(c.in); got != c.want {
			t.Errorf("resetIn(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAnOverageIsDrawnAsAFullWindow. A proxy reporting more used than there
// was is reporting an overage, and 140% of a window drawn as a bar is a bar
// with nowhere to go.
func TestAnOverageIsDrawnAsAFullWindow(t *testing.T) {
	if got := Used(Window{Pct: 140}); got != 100 {
		t.Errorf("Used(140%%) = %v, want it capped at the whole window", got)
	}

	if got := Used(Window{Pct: 42}); got != 42 {
		t.Errorf("Used(42%%) = %v, want what was read", got)
	}
}

// TestOneWindowSaysWhatIsLeftAndWhenItComesBack, in one line: both halves
// are what a reader is deciding on, and a line with only one of them sends
// them to another screen for the other.
func TestOneWindowSaysWhatIsLeftAndWhenItComesBack(t *testing.T) {
	said := Says(words.For("en"), Window{Label: "5h", Pct: 60, ResetsIn: 90 * time.Minute})

	for _, want := range []string{"60%", "5h", "1h30m"} {
		if !strings.Contains(said, want) {
			t.Errorf("the line %q does not say %q", said, want)
		}
	}
}

// TestAnEngineWithNoWindowSaysWhichKindOfNothingItIs. Three ways to have no
// percentage, and one blank for all of them would be the silence a quota was
// read to end.
func TestAnEngineWithNoWindowSaysWhichKindOfNothingItIs(t *testing.T) {
	p := words.For("en")

	for _, c := range []struct {
		in   Reading
		want string
	}{
		{Reading{Engine: "codex", Money: true}, "billed per token"},
		{Reading{Engine: "claude", Sourced: true}, "source answered nothing"},
		{Reading{Engine: "bare"}, "no quota source for bare"},
	} {
		if got := Quiet(p, c.in); got != c.want {
			t.Errorf("Quiet(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestEveryWindowIsOnTheOneLine, with the word said once at the end: a
// percentage on a status bar is read as whichever of the two numbers the
// reader expects, and the two are opposites.
func TestEveryWindowIsOnTheOneLine(t *testing.T) {
	said := Spent(words.For("en"), Reading{Engine: "claude", Windows: []Window{
		{Label: "5h", Pct: 12}, {Label: "week", Pct: 84},
	}})

	for _, want := range []string{"12% 5h", "84% week", "used"} {
		if !strings.Contains(said, want) {
			t.Errorf("the line %q does not say %q", said, want)
		}
	}

	// A reading with no window draws nothing rather than a sentence about
	// absence: the one place that says a source is missing is the status
	// line, once.
	if got := Spent(words.For("en"), Reading{Engine: "bare"}); got != "" {
		t.Errorf("a reading with no windows drew %q", got)
	}
}
