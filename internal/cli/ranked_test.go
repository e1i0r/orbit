package cli

// ranked is the digest's two lists, and the assertion worth making about them
// is what it does NOT print.
//
// A heading over no rows reads as a fact about the work — that nobody is ever
// stopped — when it is a fact about a record with nothing in it yet. The
// empty case is the one that would be wrong silently, so it is the one asked
// about first.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAnEmptyListPrintsNoHeading.
func TestAnEmptyListPrintsNoHeading(t *testing.T) {
	var out bytes.Buffer

	ranked(Context{Out: &out}, "phases that go round again", nil)

	if out.Len() != 0 {
		t.Errorf("an empty list printed %q, want nothing at all", out.String())
	}
}

// TestAListPrintsItsHeadingAndOneRowPerCount.
func TestAListPrintsItsHeadingAndOneRowPerCount(t *testing.T) {
	var out bytes.Buffer

	ranked(Context{Out: &out}, "phases that go round again", []view.Count{
		{Name: "spec", N: 3},
		{Name: "build", N: 1},
	})

	said := out.String()

	if !strings.Contains(said, "phases that go round again") {
		t.Errorf("the list printed without its heading:\n%s", said)
	}

	for _, want := range []string{"spec", "3", "build", "1"} {
		if !strings.Contains(said, want) {
			t.Errorf("the list printed without %q:\n%s", want, said)
		}
	}

	// One row per count and no more: a tabwriter that was never flushed
	// would hold the last row, and a digest missing its own last line is a
	// digest that says less than the record holds.
	if got := strings.Count(said, "\n"); got != 4 {
		t.Errorf("the list printed %d lines, want a blank, the heading and two rows:\n%q", got, said)
	}
}

// TestASingleCountStillPrints.
//
// One is not empty, and a digest that only ever named the phases that went
// round more than once would be a digest that cannot show a single loop.
func TestASingleCountStillPrints(t *testing.T) {
	var out bytes.Buffer

	ranked(Context{Out: &out}, "who stopped it", []view.Count{{Name: "permission", N: 1}})

	if !strings.Contains(out.String(), "permission") {
		t.Errorf("a list of one printed nothing:\n%q", out.String())
	}
}
