package knowledge

// How a fact written by hand is read back: the kind its header names, and
// where the file sits when the header says nothing.

import "testing"

// TestAFactWithNoKindInItsHeaderIsPlacedByWhereItSits.
//
// The store is files in directories, and a file written by hand — or by an
// older Orbit — carries no kind. Where it is is the answer: under lang/ it
// is about a language, at the top of a repository's own directory it is
// about that repository, and deeper in one it is about that directory.
func TestAFactWithNoKindInItsHeaderIsPlacedByWhereItSits(t *testing.T) {
	for _, c := range []struct {
		what  string
		name  string
		where string
		repo  string
		want  Kind
	}{
		{"a kind said out loud", kindNames[Symbol], "anywhere", "orbit", Symbol},
		{"under lang", "", langDir + "/go/never-discard.md", "", Language},
		{"nowhere in particular", "", "prs-in-english.md", "", General},
		{"the top of a repository", "", "the-fuzz-tests-hang.md", "orbit", Repo},
		{"deeper in one", "", "internal/ui/measure-in-cells.md", "orbit", Dir},
	} {
		if got := kindNamed(c.name, c.where, c.repo); got != c.want {
			t.Errorf("%s: kindNamed(%q, %q, %q) = %v, want %v",
				c.what, c.name, c.where, c.repo, got, c.want)
		}
	}
}
