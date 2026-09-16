package chat

// What this service is handed.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestNoBackslashSurvivesFromTheMarkdownAttempt. MarkdownV2 was tried, and
// its escaper outlived the switch back inside the HTML one: every hyphen
// arrived on the phone as a visible backslash.
func TestNoBackslashSurvivesFromTheMarkdownAttempt(t *testing.T) {
	// Wide enough to have given up being a table, which is where the
	// emphasis is put — a narrow listing stays in a block and needs none.
	said := "unread-cap  5  how many finished tasks may sit unread before nothing starts\n" +
		"chat-id  9  the one account it answers over a service"

	got := asHTML(Reply(verb.Out{Said: said}, en()))
	if strings.Contains(got, `\`) {
		t.Errorf("the markdown escaping survived: %q", got)
	}

	if !strings.Contains(got, "<b>unread-cap</b>") {
		t.Errorf("the name is not emphasised: %q", got)
	}
}

// TestOnlyWhatHTMLCannotTakeLiterallyIsEscaped — three characters, which is
// the whole argument for this markup over the other one.
func TestOnlyWhatHTMLCannotTakeLiterallyIsEscaped(t *testing.T) {
	got := asHTML(Reply(verb.Out{Said: `a < b & c > d. (e) [f] -g_h`}, en()))

	for _, want := range []string{"&lt;", "&amp;", "&gt;"} {
		if !strings.Contains(got, want) {
			t.Errorf("%s was not escaped: %q", want, got)
		}
	}

	for _, plain := range []string{".", "(", ")", "[", "]", "-", "_"} {
		if !strings.Contains(got, plain) {
			t.Errorf("%q was mangled: %q", plain, got)
		}
	}
}

// TestWhatSomebodyElseFormattedIsLeftAlone. The supervisor answers in HTML
// because it was asked to, and escaping that would be deleting the
// formatting rather than protecting it.
func TestWhatSomebodyElseFormattedIsLeftAlone(t *testing.T) {
	got := asHTML(Prose("<b>done</b> — nothing moved."))
	if got != "<b>done</b> — nothing moved." {
		t.Errorf("prose was escaped: %q", got)
	}
}
