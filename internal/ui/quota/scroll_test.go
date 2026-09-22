package quota

// The reading is longer than the screen, and this is the suite that says so.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/roster"
)

// sixEngines is a build that can run six of them, each with two windows:
// more rows than a short terminal has.
func sixEngines(t *testing.T) (Env, []string) {
	t.Helper()

	names := []string{"claude", "codex", "opencode", "gemini", "cursor", "aider"}

	e := world(t)
	e.Engines = func() []roster.Engine {
		out := make([]roster.Engine, 0, len(names))
		for _, n := range names {
			out = append(out, roster.Engine{Name: n})
		}

		return out
	}
	e.Read = func(engine string) roster.Reading {
		return roster.Reading{Engine: engine, Sourced: true, Windows: []roster.Window{
			{Label: "5h", Pct: 12, ResetsIn: 75 * time.Minute},
			{Label: "week", Pct: 84, ResetsIn: 3 * time.Hour},
		}}
	}

	return e, names
}

// TestTheLastEngineCanBeRead. Four engines with two windows each is more
// rows than a short terminal has, and the screen was drawn whole into a
// body that cut it — with nothing to press, the engines past the bottom
// were a reading nobody could reach.
func TestTheLastEngineCanBeRead(t *testing.T) {
	e, names := sixEngines(t)
	last := strings.ToUpper(names[len(names)-1])

	s := Open()
	if strings.Contains(drawn(s, e), last) {
		t.Fatal("the last engine is on a screen this test needs it to be off")
	}

	for range 30 {
		s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyDown}, e)
	}

	if !strings.Contains(drawn(s, e), last) {
		t.Errorf("the reading scrolled to its end does not say %q:\n%s", last, drawn(s, e))
	}

	// And it comes back: a reading that can only be walked one way is one
	// the reader has to close and open again.
	back := s
	for range 30 {
		back, _ = back.Key(tea.KeyPressMsg{Code: tea.KeyUp}, e)
	}

	if !strings.Contains(drawn(back, e), strings.ToUpper(names[0])) {
		t.Errorf("walked back to the top, the reading does not say %q", names[0])
	}
}

// TestTheTitleAndTheWayOutStayWhileTheReadingScrolls.
func TestTheTitleAndTheWayOutStayWhileTheReadingScrolls(t *testing.T) {
	e, _ := sixEngines(t)

	s := Open().Scroll(99, e)

	for _, want := range []string{"Quota", "back"} {
		if !strings.Contains(drawn(s, e), want) {
			t.Errorf("a scrolled reading does not say %q:\n%s", want, drawn(s, e))
		}
	}
}

// TestTheReadingStopsAtBothEnds. A page that scrolls past its own end is
// blank rows with nothing on the screen saying why.
func TestTheReadingStopsAtBothEnds(t *testing.T) {
	e, _ := sixEngines(t)

	rows := len(readingRows(100, e))
	view := room(15)

	for _, c := range []struct {
		name string
		s    State
		want int
	}{
		{"opened", Open(), 0},
		{"walked up from the top", Open().Scroll(-9, e), 0},
		{"walked past the end", Open().Scroll(999, e), rows - view},
		{"a notch down", Open().Scroll(1, e), 1},
	} {
		if got := c.s.Off(15, e); got != c.want {
			t.Errorf("%s, the reading starts at row %d, want %d", c.name, got, c.want)
		}
	}

	// A body taller than the reading does not scroll at all.
	if got := Open().Scroll(99, e).Off(200, e); got != 0 {
		t.Errorf("a body with room for the whole reading starts it at row %d", got)
	}
}

// TestHomeAndEndAreTheEndsOfTheReading, which is what a reader who opened
// this screen to check one engine reaches for.
func TestHomeAndEndAreTheEndsOfTheReading(t *testing.T) {
	e, names := sixEngines(t)

	end, _ := Open().Key(tea.KeyPressMsg{Code: tea.KeyEnd}, e)
	if !strings.Contains(drawn(end, e), strings.ToUpper(names[len(names)-1])) {
		t.Error("end does not reach the last engine")
	}

	home, _ := end.Key(tea.KeyPressMsg{Code: tea.KeyHome}, e)
	if got := home.Off(15, e); got != 0 {
		t.Errorf("home left the reading at row %d", got)
	}
}

// drawn is the screen as one string, at the height a short terminal has.
func drawn(s State, e Env) string {
	return ansi.Strip(strings.Join(s.View(15, 100, e), "\n"))
}
