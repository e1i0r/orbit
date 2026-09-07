package compose

// The three buttons under the compose form, and where they actually are.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/words"
)

// TestEveryButtonAnswersWhereItIsDrawn. The zones counted from zero and
// folded the row's indent into the first button, so every one of them sat
// three cells left of what it pointed at: the right-hand end of Save & Run
// answered cancel.
func TestEveryButtonAnswersWhereItIsDrawn(t *testing.T) {
	p := words.For("en")

	at := composeIndent

	for _, want := range []struct {
		text string
		key  string
	}{
		{"[ " + p.T("compose.save_btn", "↵ Save") + " ]", "save"},
		{"[ " + p.T("compose.save_run_btn", "^R Save & Run") + " ]", "save_and_run"},
		{"[ " + p.T("compose.cancel_btn", "esc Cancel") + " ]", "cancel"},
	} {
		wide := lipgloss.Width(want.text)

		// Both ends of the button, and the middle.
		for _, x := range []int{at, at + wide/2, at + wide - 1} {
			if got := hitComposeActions(p, x); got.Key != want.key {
				t.Errorf("x=%d over %q answered %q, want %q", x, strings.TrimSpace(want.text), got.Key, want.key)
			}
		}

		// The gap before the next one is nobody's.
		if got := hitComposeActions(p, at+wide); got.Key == want.key {
			t.Errorf("the gap after %q still answers it", strings.TrimSpace(want.text))
		}

		at += wide + composeGap
	}

	// Past the last button there is nothing, rather than cancel reaching to
	// the right edge of the screen.
	if got := hitComposeActions(p, at+40); got.Kind != point.None {
		t.Errorf("empty space past the buttons answered %+v", got)
	}
}
