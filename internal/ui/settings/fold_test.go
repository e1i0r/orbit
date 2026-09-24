package settings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// grouped is a screen of three settings in two groups, with room for all
// of them.
func grouped(t *testing.T) Env {
	t.Helper()

	e := env(t, newFile())

	frame, err := layout.Fit(100, 40)
	if err != nil {
		t.Fatalf("layout.Fit: %v", err)
	}

	e.Frame = frame
	e.Kept = func() []Kept {
		return []Kept{
			{Name: "max-running", Value: "3", Group: "Queue"},
			{Name: "unread-cap", Value: "3", Group: "Queue"},
			{Name: "language", Value: "en", Group: "Appearance"},
		}
	}

	return e
}

func stripped(s State, e Env) []string {
	out := s.View(40, 100, e)
	for i := range out {
		out[i] = ansi.Strip(out[i])
	}

	return out
}

// line is the first line drawn that holds text.
func line(t *testing.T, lines []string, text string) string {
	t.Helper()

	for _, l := range lines {
		if strings.Contains(l, text) {
			return l
		}
	}

	t.Fatalf("nothing drawn holds %q:\n%s", text, strings.Join(lines, "\n"))

	return ""
}

// TestAGroupHasALineDownItsSide, from the heading's sign to its last
// setting, so which settings are one group is seen rather than counted.
func TestAGroupHasALineDownItsSide(t *testing.T) {
	e := grouped(t)
	s := Open(e).Point(2, e)
	lines := stripped(s, e)

	if got := line(t, lines, "Queue"); !strings.HasPrefix(got, "  ▾ Queue") {
		t.Errorf("the heading is %q, want it open with ▾", got)
	}

	if got := line(t, lines, "unread-cap"); !strings.HasPrefix(got, "  │ unread-cap") {
		t.Errorf("a row of the group is %q, want the line down its side", got)
	}

	// The line runs through the gap between two rows of one group, and
	// stops under the group's last.
	at := 0

	for i, l := range lines {
		if strings.Contains(l, "max-running") {
			at = i
		}
	}

	if lines[at+2] != "  │" {
		t.Errorf("the gap inside the group is %q, want the line", lines[at+2])
	}

	if got := line(t, lines, "language"); !strings.HasPrefix(got, "  ▸ language") {
		t.Errorf("the chosen row is %q, want the cursor on the line", got)
	}
}

// TestAHeadingFoldsItsGroup: the cursor stands on it, enter hides the
// group's settings and says how many, and enter again shows them.
func TestAHeadingFoldsItsGroup(t *testing.T) {
	e := grouped(t)
	s := Open(e)

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyUp}, e)
	if s.OnHeading() != "Queue" {
		t.Fatalf("up from the first row stood on %q, want the Queue heading", s.OnHeading())
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	lines := strings.Join(stripped(s, e), "\n")

	if !strings.Contains(lines, "▸ Queue (2)") || strings.Contains(lines, "max-running") {
		t.Errorf("folded, the screen draws:\n%s\nwant the heading with its count and no rows", lines)
	}

	// Down from a folded heading skips the rows it hides.
	next, _ := s.Key(down(), e)
	if next.OnHeading() != "Appearance" {
		t.Errorf("down from a folded group stood on %q, want the next heading", next.OnHeading())
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if !strings.Contains(strings.Join(stripped(s, e), "\n"), "max-running") {
		t.Error("enter again left the group folded")
	}
}

// TestAClickOnAHeadingFindsTheGroup, and a folded group's rows are not
// under any line of the screen.
func TestAClickOnAHeadingFindsTheGroup(t *testing.T) {
	e := grouped(t)
	s := Open(e)

	first, _ := s.LineOf(0, e)

	group, on := s.HeadAt(first-1, e)
	if !on || group != "Queue" {
		t.Fatalf("the line above the first row is %q (%v), want the Queue heading", group, on)
	}

	s = s.Fold(group, e)
	if _, shown := s.LineOf(0, e); shown {
		t.Error("a folded group's row is still on a line of the screen")
	}

	if at, on := s.RowAt(first, e); on && at != 2 {
		t.Errorf("under the folded heading a click finds row %d, want the next group's or nothing", at)
	}
}
