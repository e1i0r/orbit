package arch

// The yes-or-no boxes the browser draws, against the fields the verbs
// declare them under.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestTheBrowsersBoxesCarryTheNamesTheVerbsDeclare.
//
// A verb's yes-or-no arrives by name: permit reads "yes", critical reads
// "on", direct reads "restart". The browser posts whatever key its own
// table says, and a key the verb does not read is a box the verb never
// sees — which reads as the answer nobody gave.
//
// It went wrong exactly there. Permit's box was posted as "no" and
// critical's as "off", so pressing Permit always refused and pressing Mark
// critical always unmarked. Both did the opposite of what the button said,
// and one of them is the gate in front of things that cannot be undone.
func TestTheBrowsersBoxesCarryTheNamesTheVerbsDeclare(t *testing.T) {
	page := readAll(t, "ui/src/task", ".tsx")

	// The reading itself is asserted, because a walk that matched nothing
	// would pass this test in silence — which is exactly what it did when
	// it was first written against read(), whose walk is Go files only.
	if !strings.Contains(page, "also: { name:") {
		t.Fatal("no yes-or-no boxes were read from the browser's verb table")
	}

	fields := map[string]map[string]bool{}

	for _, v := range verb.Every() {
		fields[v.Path()] = map[string]bool{}
		for _, f := range v.Takes {
			fields[v.Path()][f.Name] = true
		}
	}

	// `name: "x"` inside an `also:` is one box, and the verb above it is
	// the block it sits in — read by walking back to the nearest key.
	for _, one := range boxes(page) {
		if !fields[one.verb][one.field] {
			t.Errorf("the browser posts %s's box as %q, which %s does not take",
				one.verb, one.field, one.verb)
		}
	}
}

// box is one yes-or-no the browser offers, and the verb it belongs to.
type box struct{ verb, field string }

// boxes reads the verb table the browser draws from.
func boxes(page string) []box {
	var (
		out  []box
		verb string
	)

	for _, line := range strings.Split(page, "\n") {
		trimmed := strings.TrimSpace(line)

		// A verb opens a block: two spaces, its name, a colon and a brace.
		// A family is both its words in quotes — `"task direct": {` — and
		// the quotes are what tell it apart from any other keyed line.
		if name, rest, found := strings.Cut(trimmed, ":"); found && strings.TrimSpace(rest) == "{" {
			if unquoted, isFamily := strings.CutPrefix(name, `"`); isFamily {
				verb, _ = strings.CutSuffix(unquoted, `"`)
			} else if name != "" && !strings.Contains(name, " ") {
				verb = name
			}
		}

		at := strings.Index(trimmed, `also: { name: "`)
		if at < 0 || verb == "" {
			continue
		}

		field, _, _ := strings.Cut(trimmed[at+len(`also: { name: "`):], `"`)
		out = append(out, box{verb: verb, field: field})
	}

	return out
}
