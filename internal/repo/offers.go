package repo

// What a checkout can offer somebody filling in a form about it: the folders
// it actually has, and the commands it already runs on itself.
//
// Offered rather than remembered. Filing a rule under a folder used to mean
// knowing by heart which folders the project has and spelling one right, and
// giving a rule a command to refuse work with meant the same about the
// Makefile. Both are read off the checkout now, so the answer is picked.
//
// Neither is exhaustive and neither pretends to be. What is offered is what
// somebody would have typed nine times out of ten, and typing is still there
// for the tenth.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// atMostOffered is how many of each are handed back. A row of pills wider
// than the terminal is a row nobody reads to the end, and a project with
// forty top folders has no useful answer to offer anyway.
const atMostOffered = 8

// Folders are the top-level directories of a checkout, in the order a reader
// would look for them.
//
// One level and not the tree. Two levels of a real project is hundreds of
// entries, and the folder a rule is about is almost always one somebody
// could name out loud — internal, cmd, docs. Anything deeper is typed.
func Folders(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var out []string

	for _, one := range entries {
		name := one.Name()
		if !one.IsDir() || strings.HasPrefix(name, ".") || isIgnoredDir(name) {
			continue
		}

		out = append(out, name)
	}

	sort.Strings(out)

	if len(out) > atMostOffered {
		return out[:atMostOffered]
	}

	return out
}

// makeTarget is a line of a Makefile that declares one: a name at the head of
// the line, a colon, and no equals sign — which is what tells a target from a
// variable holding a colon.
var makeTarget = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_./-]*):[^=]*$`)

// Checks are the commands this checkout already runs on itself, read off its
// Makefile as `make <target>`.
//
// The Makefile and not a guess at the language's usual command. A project's
// own targets are the commands its people actually type, and a rule given
// one of those is a rule whose gate the team already trusts — where `go test
// ./...` invented from the file extension may be wrong about tags, about
// which packages, and about what has to be built first.
func Checks(root string) []string {
	f, err := os.Open(filepath.Join(root, "Makefile"))
	if err != nil {
		return nil
	}

	defer func() { _ = f.Close() }() //nolint:errcheck // nothing was written

	var (
		out  []string
		seen = map[string]bool{}
	)

	lines := bufio.NewScanner(f)
	for lines.Scan() {
		m := makeTarget.FindStringSubmatch(lines.Text())
		if m == nil || strings.Contains(m[1], "%") || seen[m[1]] {
			continue
		}

		seen[m[1]] = true

		out = append(out, "make "+m[1])
		if len(out) == atMostOffered {
			break
		}
	}

	return out
}
