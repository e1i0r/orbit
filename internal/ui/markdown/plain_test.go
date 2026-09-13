package markdown

import "testing"

// Plain strips the marks and leaves the words: bold, code and links go,
// the sentence stays.

// TestPlainKeepsTheWordsDropsTheMarks.
func TestPlainKeepsTheWordsDropsTheMarks(t *testing.T) {
	if got := Plain("run **make check** and `orbit top`"); got != "run make check and orbit top" {
		t.Errorf("plain reads %q", got)
	}

	if got := Plain("no marks at all"); got != "no marks at all" {
		t.Errorf("plain reads %q", got)
	}
}
