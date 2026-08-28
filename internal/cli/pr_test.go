package cli

// What `orbit pr` writes into a commit subject and a pull request title.
// Neither can be exercised end to end here — the command pushes a branch and
// shells out to gh — so what is asked is the one decision this package makes
// about them: where the text is cut.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestASubjectIsCutAtCharactersAndNotAtBytes. len counts bytes, and the
// commit subject used to be cut with it: git records the half character, and
// the pull request shows U+FFFD where a word was.
//
// The title is Japanese because that is the case with nothing to catch it. A
// title with spaces in it usually survives by accident — the cut backs off to
// the last space, which throws the broken tail away with the word it was in —
// and a language that does not write spaces has no such accident: the whole
// title is one word, the fallback is the raw cut, and the raw cut is the
// defect.
func TestASubjectIsCutAtCharactersAndNotAtBytes(t *testing.T) {
	subject := "feat(ACME-1): 請求書の月次締め処理で金額が二重に計上される不具合を直して合計が合うようにする。再発しないように回帰テストも一緒に追加すること"
	// The fixture has to be a title a byte cut actually damages, or this test
	// would pass over any implementation at all.
	if utf8.ValidString(subject[:72]) {
		t.Fatalf("this title survives a byte cut, so it proves nothing: %q", subject[:72])
	}

	got := clipWords(subject, 72)
	if !utf8.ValidString(got) {
		t.Errorf("the subject was cut through a character: %q", got)
	}

	if n := utf8.RuneCountInString(got); n > 72 {
		t.Errorf("the subject is %d characters long, want at most 72: %q", n, got)
	}

	if !strings.HasPrefix(subject, got) {
		t.Errorf("the subject was rewritten rather than cut: %q", got)
	}
}

func TestASubjectShortEnoughIsLeftAlone(t *testing.T) {
	subject := "feat(ACME-1): corregir la validación"
	if got := clipWords(subject, 72); got != subject {
		t.Errorf("a subject under the limit came back as %q", got)
	}
}

// TestASubjectEndsOnAWholeWordUnlessThereIsNone. The space is preferred, and
// a space in the first few characters is not: a first word longer than the
// limit would otherwise leave a subject of almost nothing.
func TestASubjectEndsOnAWholeWordUnlessThereIsNone(t *testing.T) {
	if got := clipWords("feat(ACME-1): make the numbers add up everywhere", 33); got != "feat(ACME-1): make the numbers" {
		t.Errorf("cut at %q, want it to end on a whole word", got)
	}

	long := "x " + strings.Repeat("y", 100)
	if got := clipWords(long, 40); got != long[:40] {
		t.Errorf("cut at %q, want the hard cut: the only space is at the very start", got)
	}
}
