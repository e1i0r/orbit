package settings

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// TestEachGroupIsDrawnUnderItsHeading, once, and a click on a heading is
// not a click on a setting. Eighteen settings in one list read as eighteen
// unrelated switches; under a heading, the two that mean something only
// together are read together.
func TestEachGroupIsDrawnUnderItsHeading(t *testing.T) {
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

	s := unfolded(Open(e), e)

	drawn := ansi.Strip(strings.Join(s.View(40, 100, e), "\n"))
	for _, heading := range []string{"Queue", "Appearance"} {
		if strings.Count(drawn, heading) != 1 {
			t.Errorf("%q is drawn %d times, want once:\n%s",
				heading, strings.Count(drawn, heading), drawn)
		}
	}

	for i := range s.Rows(e) {
		line, shown := s.LineOf(i, e)
		if !shown {
			t.Fatalf("row %d is not on the screen", i)
		}

		if at, ok := s.RowAt(line, e); !ok || at != i {
			t.Errorf("row %d is drawn on line %d, and a click there finds row %d (%v)",
				i, line, at, ok)
		}

		// The line above the first row of a group is its heading.
		if i == 0 || i == 2 {
			if _, ok := s.RowAt(line-1, e); ok {
				t.Errorf("a click on the heading above row %d found a setting", i)
			}
		}
	}
}
