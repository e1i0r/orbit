package panes

// The pull request a delivery verb came back with, on its line of the tree.

import (
	"regexp"
	"strings"
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
