package fact

// How a fact is named, which two screens depend on saying the same way.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestAScopeIsNamedTheSameWayWhereverItIsDrawn. The column down the side of
// the supervisor and the knowledge screen draw the same store: a rule that
// reaches "everywhere" in one and "general" in the other reads as two rules.
func TestAScopeIsNamedTheSameWayWhereverItIsDrawn(t *testing.T) {
	for _, c := range []struct {
		scope knowledge.Scope
		want  string
	}{
		{knowledge.Scope{Kind: knowledge.General}, "everywhere"},
		{knowledge.Scope{Kind: knowledge.Language, Lang: "go"}, "go"},
		{knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, "orbit"},
		{
			knowledge.Scope{Kind: knowledge.Symbol, Path: "internal/task/run.go", Symbol: "Start"},
			"internal/task/run.go#Start",
		},
		// A file, and everything else that reaches part of a checkout, is
		// named by the path itself.
		{knowledge.Scope{Kind: knowledge.File, Path: "internal/task/run.go"}, "internal/task/run.go"},
	} {
		if got := Where(c.scope); got != c.want {
			t.Errorf("Where(%+v) = %q, want %q", c.scope, got, c.want)
		}
	}
}

// TestARepositoryIsCalledByItsLastSegment, which is what a reader calls it —
// and a trailing slash is not a name of its own.
func TestARepositoryIsCalledByItsLastSegment(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"/Users/elio/work/orbit", "orbit"},
		{"/Users/elio/work/orbit/", "orbit"},
		{"orbit", "orbit"},
		{"", ""},
	} {
		if got := Repo(c.path); got != c.want {
			t.Errorf("Repo(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
