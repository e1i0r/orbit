package arch

// What the browser's own pages offer, as against what the API routes.
//
// The two are not the same claim and it took a year for anybody to notice.
// verbs_test.go checks that internal/web routes every verb by name, which it
// does by construction — and so the browser has been able to ask for `rules
// pause` since the day the route was written, while nothing on any page ever
// did. A reader with a browser open could not pause a rule, and no test went
// red about it.
//
// This one reads the pages. A verb a page names is a verb somebody can
// reach; one no page names is a verb that exists only for the command line
// and the tool call, and the reason has to be written down here.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// notDrawn is every verb the browser deliberately has no page for, and why.
//
// The reasons are of three kinds and none of them is "nobody got round to
// it": what a browser cannot do, what would be a second screen for something
// a page already does whole, and what spends money with no task behind it.
var notDrawn = map[string]string{
	"task take": "this hands a terminal to an engine, and a browser tab is not one",

	"board list": "the board is the page itself; a browser has no pipe to print a " +
		"listing into",
	"task show": "a task's page is what showing one is",
	"quota":     "the header draws what is left of every engine's allowance, on every page",
	"queue":     "the board's rows say a task is queued; the line itself is a reading for the terminal and tools",

	"rules history": "the rule's own page reads `rules review`, which is these same turns " +
		"written as sentences; the raw rows are what a terminal greps",
	"knowledge learn": "it offers a rule into your own tray, which is the tool a model " +
		"calls mid-task; a person with this page open keeps the rule instead of " +
		"proposing one to themselves",

	"rules read": "it spends money with no task behind it: a page with a button for it " +
		"would be a page that bills somebody for pressing it without meaning to",
	"rules draft": "the same, and the tray it writes into is drawn here — what is " +
		"missing is somewhere to say yes to the spending first",
	"rules repeated": "it is the reading behind `rules draft` and useful on its own only " +
		"from a terminal, where the output is grepped",

	"supervisor thread": "the same: the thread is the page",
	"task history":      "the task's timeline tab is this reading, drawn",
	"task flow":         "the task's flow tab is this reading, drawn",
	"task diff":         "the task's diff tab is this reading, drawn",
	"task prompt":       "the task's prompt tab is this reading, drawn",
	"task tree":         "the task's map tab is this reading, drawn",
	"task impact":       "the task's impact tab is this reading, drawn",
	"task compare":      "the task's diff tab draws what the checks say on both sides",
}

// TestEveryVerbTheBrowserCanReachIsDrawnSomewhere.
func TestEveryVerbTheBrowserCanReachIsDrawnSomewhere(t *testing.T) {
	page := readAll(t, "ui/src", ".tsx") + readAll(t, "ui/src", ".ts")

	for _, v := range verb.Every() {
		name := v.Path()
		why, excused := notDrawn[name]

		// A page names a verb the way it asks for it: the whole path in a
		// string, which is what api.did and api.read take.
		drawn := strings.Contains(page, `"`+name+`"`) || strings.Contains(page, "`"+name)

		switch {
		case drawn && excused:
			t.Errorf("a page offers %q, and notDrawn says it does not — drop the line", name)
		case !drawn && !excused:
			t.Errorf("no page offers %q; draw it, or write down why not in notDrawn", name)
		case !drawn && why == "":
			t.Errorf("%q is excused with no reason", name)
		}
	}
}
