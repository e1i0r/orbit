package cli

// What the window says about a fact, turned into the scope it is filed
// under.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestAFactIsFiledWhereItWasSaidToBelong. An empty scope with no repository
// to fall back on becomes a general fact rather than a refusal: the operator
// said something true about their work, and the widest scope is the honest
// place for it when nothing narrower is known.
func TestAFactIsFiledWhereItWasSaidToBelong(t *testing.T) {
	for _, c := range []struct {
		what  string
		scope string
		repo  string
		want  knowledge.Scope
	}{
		{
			"said out loud", "general", "/w/orbit",
			knowledge.Scope{Kind: knowledge.General},
		},
		{
			"nothing said and nowhere to say it about", "", "",
			knowledge.Scope{Kind: knowledge.General},
		},
		{
			"a language", "go", "/w/orbit",
			knowledge.Scope{Kind: knowledge.Language, Lang: "go"},
		},
		{
			"nothing said, in a checkout", "", "/w/orbit",
			knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"},
		},
	} {
		if got := factScope(c.scope, c.repo); got != c.want {
			t.Errorf("%s: factScope(%q, %q) = %+v, want %+v", c.what, c.scope, c.repo, got, c.want)
		}
	}
}
