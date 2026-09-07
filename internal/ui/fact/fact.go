// Package fact names a piece of knowledge in the window.
//
// Two screens draw the same facts — the column down the side of the
// supervisor and the knowledge screen — and both have to call a scope the
// same thing. A reader who is told a rule reaches "everywhere" in one place
// and "general" in the other has to work out that they are the same rule.
package fact

import (
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// Where is how far a fact reaches, in as few words as a column has.
func Where(s knowledge.Scope) string {
	switch s.Kind {
	case knowledge.General:
		return "everywhere"
	case knowledge.Language:
		return s.Lang
	case knowledge.Repo:
		return Repo(s.Repo)
	case knowledge.Symbol:
		return s.Path + "#" + s.Symbol
	default:
		return s.Path
	}
}

// Repo is a repository by its last segment, which is what a reader calls it.
func Repo(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")

	return parts[len(parts)-1]
}
