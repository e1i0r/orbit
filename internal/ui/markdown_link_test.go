package ui

// Links, in a window where nothing is clickable.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestALinkIsItsTextAndNotItsAddress.
//
// A terminal cannot follow a link, so the address is not an affordance — it
// is a hundred characters of noise in the middle of a sentence, and it wraps
// mid-path because a path has nowhere to break. What a reader needs is the
// words somebody wrote around it.
func TestALinkIsItsTextAndNotItsAddress(t *testing.T) {
	got := ansi.Strip(formatInlineMarkdown(
		"edits in [internal/ui/supervisor.go](file:///Users/e/repos/orbit/internal/ui/supervisor.go) and elsewhere"))

	if !strings.Contains(got, "internal/ui/supervisor.go") {
		t.Errorf("the link lost the words it was written with: %q", got)
	}

	if strings.Contains(got, "file:///") || strings.Contains(got, "](") {
		t.Errorf("the address is still on screen: %q", got)
	}

	if !strings.Contains(got, "edits in") || !strings.Contains(got, "and elsewhere") {
		t.Errorf("the sentence around the link was lost: %q", got)
	}
}

// TestALinkWithNoWordsShowsItsAddress. Dropping both would leave a sentence
// pointing at nothing.
func TestALinkWithNoWordsShowsItsAddress(t *testing.T) {
	got := ansi.Strip(formatInlineMarkdown("see []( https://example.com/x ) for more"))
	if !strings.Contains(got, "example.com") {
		t.Errorf("a link with no words showed nothing at all: %q", got)
	}
}

// TestSomethingThatIsNotALinkIsLeftAlone: brackets people type, and the
// arrays they write in prose.
func TestSomethingThatIsNotALinkIsLeftAlone(t *testing.T) {
	for _, said := range []string{
		"the list [a, b, c] came back empty",
		"see [1] below",
		"a bracket ] on its own",
	} {
		if got := ansi.Strip(formatInlineMarkdown(said)); got != said {
			t.Errorf("formatInlineMarkdown(%q) = %q, want it untouched", said, got)
		}
	}
}
