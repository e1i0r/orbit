package learn

// Which of the things somebody said are the same thing said again.
//
// By the words they share and not by what they mean. "add fuzz testing" and
// "put some fuzz tests on this" are one instruction, and what says so is
// that both say fuzz and test — a comparison anybody can check, that runs on
// a machine with nothing installed, and that can be argued with when it
// groups two things that are not one.
//
// The alternative was asking a model which of these are alike. It would be
// better at it and it would also be a call, a key and a wait in the middle
// of reading a list — and the thing it would be deciding is one a person is
// about to read anyway.

import (
	"sort"
	"strings"
	"unicode"
)

// enoughTimes is how often something has to be said before it is a habit
// rather than a correction.
//
// Three, by hand, and meant to be argued with: twice is a coincidence often
// enough that it would fill this with noise, and waiting for four means
// somebody repeats themselves a fourth time before Orbit notices. Change it
// once it has been used and it is clear which way it annoys.
const enoughTimes = 3

// enoughWords is how much two sentences have to have in common before they
// are read as the same instruction.
//
// Two, for the same kind of reason: one word in common is "the tests" and
// "the test file", which are not one instruction, and three is more than
// most of these sentences have in them at all.
const enoughWords = 2

// alike groups the sentences that say the same thing, and answers with the
// groups big enough to be a habit.
//
// Greedy, and the first sentence of each group is the one the rest are
// measured against: it walks the list once, takes everything that matches
// what it is holding, and narrows what it is holding as it goes — so a group
// ends up keeping only the words all of its members share. A different order
// can group these differently, which is a known limit and not a hidden one:
// what comes out is read by somebody, and the words are printed beside it.
func alike(said []Directive) []Habit {
	var (
		out  []Habit
		done = make([]bool, len(said))
	)

	for i, one := range said {
		if done[i] {
			continue
		}

		done[i] = true
		group, shared := []Directive{one}, meaningful(one.Text)

		for j := i + 1; j < len(said); j++ {
			if done[j] {
				continue
			}

			both := shared.and(meaningful(said[j].Text))
			if len(both) < enoughWords {
				continue
			}

			done[j], shared = true, both
			group = append(group, said[j])
		}

		if len(group) < enoughTimes {
			continue
		}

		out = append(out, Habit{
			Repo: one.Repo, Phase: one.Phase, Words: shared.sorted(), Said: group,
		})
	}

	return out
}

// A wording is the set of words one sentence is worth comparing by.
type wording map[string]bool

// and is what two sentences both say, with a word said two ways counted
// once: "test", "tests" and "testing" are one instruction, and what is kept
// is the part of it they agree on.
func (w wording) and(other wording) wording {
	both := wording{}

	for mine := range w {
		for theirs := range other {
			if stem, same := sameWord(mine, theirs); same {
				both[stem] = true
			}
		}
	}

	return both
}

// sameWord says whether two words are one word said differently, and answers
// with the part of it they share.
//
// The shared opening, four runes of it, and not one word being the start of
// the other: "tests" and "testing" are the same instruction and neither
// begins the other. Endings are where a language puts its grammar, which is
// why cutting at the front works in both of the two this reads.
//
// It groups "commit" with "common", and that is the price. It is the kind of
// wrong somebody spots in a second, because the words are printed beside the
// sentences they came from — and the alternative was a list of every ending
// in two languages, which is a list somebody would have to keep.
func sameWord(a, b string) (string, bool) {
	x, y := []rune(a), []rune(b)

	n := 0
	for n < len(x) && n < len(y) && x[n] == y[n] {
		n++
	}

	if n < shortWord {
		return "", false
	}

	return string(x[:n]), true
}

// sorted is the words in one order, so that two readings of the same habit
// read the same.
func (w wording) sorted() []string {
	out := make([]string, 0, len(w))
	for word := range w {
		out = append(out, word)
	}

	sort.Strings(out)

	return out
}

// shortWord is how long a word has to be to be worth comparing by.
//
// Four runes. Under that almost everything is grammar — the, and, for, que,
// los, con — and a list of every short word in two languages is a list
// somebody has to keep, where a length is a rule that needs no maintenance.
const shortWord = 4

// filler is the longer words that are in every sentence and say nothing
// about what it asks for, in English first and Spanish after.
//
// Short and hand-written, like the openings a rule is noticed by, and meant
// to be argued with the same way: a word that keeps grouping two unrelated
// instructions goes on, and one that turns out to matter comes off.
var filler = map[string]bool{
	"this": true, "that": true, "with": true, "when": true, "then": true,
	"here": true, "there": true, "please": true, "should": true, "would": true,
	"esto": true, "este": true, "esta": true, "para": true, "como": true,
	"pero": true, "porque": true, "entonces": true, "favor": true,
}

// meaningful is the words of a sentence worth comparing it by: long enough
// to carry something, and not one of the words every sentence has.
func meaningful(text string) wording {
	out := wording{}

	for _, word := range strings.FieldsFunc(strings.ToLower(text), notAWord) {
		if len([]rune(word)) < shortWord || filler[word] {
			continue
		}

		out[word] = true
	}

	return out
}

// notAWord is where one word ends and the next begins: anything that is not
// a letter or a digit. Apostrophes and hyphens split too — "don't" is two
// words here and both are dropped as short, which is the answer either way.
func notAWord(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
