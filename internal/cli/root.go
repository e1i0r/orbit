package cli

// The root the window is opened over, written the way a person says it.
//
// It is here rather than in internal/ui because it is a fact about the
// environment this process runs in, and the window is handed facts rather
// than reading them. root_test.go is its partner.

import (
	"os"
	"strings"
)

// underHome writes a path that starts at the home directory the way a person
// says it, with a tilde. The window puts the root in its header on every
// frame, and an absolute path there is both longer than the header can afford
// and longer than anybody reads: the part that identifies a directory is the
// end of it, and the home prefix is the part every path on the machine shares.
//
// An empty home, or a root outside it, comes back unchanged — there is
// nothing to abbreviate and a bare "~" would name the wrong place.
func underHome(dir, home string) string {
	if home == "" || dir == home {
		return dir
	}
	if rest, ok := strings.CutPrefix(dir, home+string(os.PathSeparator)); ok {
		return "~" + string(os.PathSeparator) + rest
	}
	return dir
}
