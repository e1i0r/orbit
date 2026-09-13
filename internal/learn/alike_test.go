package learn

// Which sentences are the same sentence said again.

import (
	"slices"
	"testing"
)

// TestAWordSaidTwoWaysIsOneWord.
//
// Endings are where both of the languages this reads put their grammar, so
// what two forms of a word agree on is the front of it. Matching whole words
// would have missed the case this whole thing is for: somebody asks for
// "fuzz testing" once and "fuzz tests" the next time, and means the same
// thing both times.
func TestAWordSaidTwoWaysIsOneWord(t *testing.T) {
	for _, one := range []struct {
		a, b string
		stem string
		same bool
	}{
		{"tests", "testing", "test", true},
		{"parse", "parser", "parse", true},
		{"coverage", "coverage", "coverage", true},
		{"pruebas", "probar", "", false},
		{"rename", "reads", "", false},
		{"fuzz", "fuzzing", "fuzz", true},
	} {
		stem, same := sameWord(one.a, one.b)
		if same != one.same || stem != one.stem {
			t.Errorf("%q and %q read as (%q, %v), want (%q, %v)",
				one.a, one.b, stem, same, one.stem, one.same)
		}
	}
}

// TestTheWordsWorthComparingBy: long enough to carry something, and not one
// of the words every sentence has. Without both, every pair of sentences in
// the record has "this" and "with" in common and the whole list is one
// habit.
func TestTheWordsWorthComparingBy(t *testing.T) {
	got := meaningful("Please add more coverage to this, with fuzz tests").sorted()

	want := []string{"coverage", "fuzz", "more", "tests"}
	if !slices.Equal(got, want) {
		t.Errorf("the sentence reads as %v, want %v", got, want)
	}
}

// TestPunctuationIsNotPartOfAWord, so that "coverage," and "coverage" are
// not two things a sentence says.
func TestPunctuationIsNotPartOfAWord(t *testing.T) {
	if got := meaningful("coverage, coverage; (coverage)").sorted(); len(got) != 1 {
		t.Errorf("one word written three ways reads as %v", got)
	}
}
