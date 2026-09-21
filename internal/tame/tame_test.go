package tame

// What the stripper keeps, and what it takes out.

import "testing"

// TestTamingLeavesTheWordsAlone. Everything a reader came for survives:
// the words, the line breaks the panes split on, and the accents and the
// emoji a tracker's title is full of.
func TestTamingLeavesTheWordsAlone(t *testing.T) {
	for _, c := range []struct{ from, want string }{
		{"plain words", "plain words"},
		{"two\nlines", "two\nlines"},
		{"decisión 決済 🔥", "decisión 決済 🔥"},
		{"a\ttab", "a tab"},
		{"\x1b[32mgreen\x1b[0m words", "green words"},
		{"\x1b]0;title\x07after", "after"},
		{"cut off mid-\x1b[", "cut off mid-"},
	} {
		if got := Text(c.from); got != c.want {
			t.Errorf("Text(%q) = %q, want %q", c.from, got, c.want)
		}
	}
}
