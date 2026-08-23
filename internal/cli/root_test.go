package cli

import (
	"os"
	"testing"
)

// The header carries the root on every frame, so it is written the way a
// person says it. Every case here is one the goldens never reach: they are
// handed "~/work" already, so nothing in internal/ui ever proves that form is
// produced rather than assumed.
func TestTheRootIsWrittenTheWayAPersonSaysIt(t *testing.T) {
	sep := string(os.PathSeparator)
	cases := []struct {
		name, dir, home, want string
	}{
		{"under home", "/Users/x/work/repos", "/Users/x", "~" + sep + "work/repos"},
		{"the home itself", "/Users/x", "/Users/x", "/Users/x"},
		{"outside home", "/srv/repos", "/Users/x", "/srv/repos"},
		{"no home to speak of", "/srv/repos", "", "/srv/repos"},
		// A sibling whose name merely starts with the home path is not
		// under it: abbreviating "/Users/xavier" against "/Users/x" would
		// name a directory that does not exist.
		{"a sibling with a longer name", "/Users/xavier/work", "/Users/x", "/Users/xavier/work"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := underHome(c.dir, c.home); got != c.want {
				t.Errorf("underHome(%q, %q) = %q, want %q", c.dir, c.home, got, c.want)
			}
		})
	}
}
