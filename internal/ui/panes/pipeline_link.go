package panes

// What a delivery verb shows beyond where it got to: the steps its carrier
// is taking while it is out, and the pull request it came back with.

import (
	"regexp"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// linkIn finds links in what a verb answered. Anything clickable counts,
// for the reason internal/task/promise.go gives: a forge's path is its own.
var linkIn = regexp.MustCompile(`https?://[^\s<>()\[\]"'` + "`" + `]+`)

// prLink is the link a pull request verb answered with, and empty for any
// other verb or an answer with none.
//
// The one that looks like a pull request when there are several, because
// an answer that names the CI run and then the pull request is about the
// pull request.
func prLink(verb, said string) string {
	if v := strings.ToUpper(strings.TrimSpace(verb)); v != "CREATE PR" && v != "UPDATE PR" {
		return ""
	}

	links := linkIn.FindAllString(said, -1)
	for _, l := range links {
		if strings.Contains(l, "/pull") || strings.Contains(l, "/merge_requests/") {
			return strings.TrimRight(l, ".,;:")
		}
	}

	if len(links) == 0 {
		return ""
	}

	return strings.TrimRight(links[0], ".,;:")
}

// shownSteps is how many of a verb's steps its node lists: enough to see
// what it is doing and what it did just before, not its whole transcript,
// which is on the timeline.
const shownSteps = 5

// withStep puts one step onto the last open verb it belongs to. A step for
// a verb nothing asked for is dropped: there is no node to hang it on.
func withStep(steps []handStep, entry view.Entry) []handStep {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].verb == entry.Verb && !steps[i].done {
			steps[i].doing = append(steps[i].doing, view.ToolLine(entry.Tool, entry.Text))

			break
		}
	}

	return steps
}

// stepItems is the tail of what a verb still out has done, the newest last
// and marked as the one in hand. Once it is back the answer says what it
// did, and the steps are left to the timeline.
func stepItems(st handStep) []subItem {
	if st.done || len(st.doing) == 0 {
		return nil
	}

	tail := st.doing[max(len(st.doing)-shownSteps, 0):]
	items := make([]subItem, 0, len(tail))

	for i, step := range tail {
		mark, role := "·", theme.Dim
		if i == len(tail)-1 {
			mark, role = SpinMark, theme.Live
		}

		items = append(items, subItem{text: theme.Paint(role).Render(mark + " " + cells.Fit(step, 90))})
	}

	return items
}

// last is the newest of a list of steps, and empty for none.
func last(doing []string) string {
	if len(doing) == 0 {
		return ""
	}

	return doing[len(doing)-1]
}
