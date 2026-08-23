package flow

// Which flow a run walks is a name, and this file is everything that turns
// a name into a flow. Three flows ship inside the binary; a file in
// $ORBIT_HOME/flows wins over the built-in of the same name, and that is
// the whole extension mechanism — no plugin API, no registry, one directory.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Default is the flow a task walks when nothing chooses one.
//
// It is the name of a built-in on purpose, and that name is "task" because
// every record already written says "task": renaming the default would make
// an old log claim its run walked something this binary would resolve
// differently.
const Default = "task"

// Source is the directory a user's own flows live in.
//
// It is one method wide and declared here rather than in internal/store
// because this is the package that consumes it: internal/flow imports
// nothing of Orbit's, which is what keeps a flow a piece of data rather
// than a thing that knows about state roots. *store.Store satisfies it.
type Source interface {
	FlowDir() string
}

// The words Names puts beside a flow, saying where the one that would be
// resolved came from. A shadow is marked as such because a flow that
// stopped behaving as documented is then one command away from explaining
// itself.
const (
	markBuiltin = "built in"
	markUser    = "yours"
	markShadow  = "yours, shadowing the built-in"
)

// Resolve turns a name into the flow that name means, looking in the user's
// flow directory before the flows shipped inside the binary.
//
// A file that shadows a built-in and will not parse is a failure and not a
// fallback. Falling back would walk something other than what the file
// says, silently, which is worse than refusing: a shadow that fails open is
// worse than one that fails.
//
// A nil Source is the builtins and nothing else, which is what a caller
// with no state root gets.
func Resolve(src Source, name string) (Flow, error) {
	if err := ValidName(name); err != nil {
		return Flow{}, err
	}
	if dir := dirOf(src); dir != "" {
		path := filepath.Join(dir, name+".json")
		f, err := Load(path)
		switch {
		case err == nil:
			return named(f, name, path)
		case !errors.Is(err, os.ErrNotExist):
			// Anything other than "there is no such file" — a directory
			// nobody may read, a file that will not parse — is the user's
			// flow failing, and it is theirs to hear about.
			return Flow{}, err
		}
	}
	if !slices.Contains(BuiltinNames(), name) {
		return Flow{}, fmt.Errorf("no flow called %q; there is %s", name, strings.Join(Names(src), ", "))
	}
	f, err := Builtin(name)
	if err != nil {
		return Flow{}, err
	}
	return named(f, name, name+".json")
}

// named refuses a flow whose file and whose own name disagree.
//
// The file name is what `orbit flows` lists and what a task records; the
// name inside is what a run writes into its log. Two of them is one flow
// with two names, and a reader comparing the window to the record would
// find them disagreeing with nothing to say why.
func named(f Flow, want, source string) (Flow, error) {
	if f.Name != want {
		return Flow{}, fmt.Errorf("the flow in %s is called %q, not %q; a flow is named by its file", source, f.Name, want)
	}
	return f, nil
}

// Names is every flow that could be resolved, sorted, each marked with
// where the one that would win came from.
//
// It is a listing and not a load: a file in the directory is named here
// whether or not it parses, because "there is a file called that" is the
// thing a reader is asking, and Resolve is what says whether it is a flow.
func Names(src Source) []string {
	marks := make(map[string]string)
	names := BuiltinNames()
	for _, n := range names {
		marks[n] = markBuiltin
	}
	for _, n := range UserNames(src) {
		if _, shadows := marks[n]; shadows {
			marks[n] = markShadow
		} else {
			marks[n] = markUser
			names = append(names, n)
		}
	}
	sort.Strings(names)
	listed := make([]string, 0, len(names))
	for _, n := range names {
		listed = append(listed, fmt.Sprintf("%s (%s)", n, marks[n]))
	}
	return listed
}

// ValidName rejects a name that could not be the file a flow lives in.
//
// The name arrives from a command line and is joined onto a path, so this
// function is what stands between a typed word and the filesystem — the
// same guard internal/store puts in front of a task id, for the same
// reason.
func ValidName(name string) error {
	switch {
	case name == "":
		return errors.New("a flow needs a name")
	case strings.ContainsRune(name, '/'), strings.ContainsRune(name, os.PathSeparator):
		return fmt.Errorf("the flow name %q contains a path separator", name)
	case name == ".", strings.Contains(name, ".."):
		return fmt.Errorf("the flow name %q is not a name", name)
	}
	return nil
}

// dirOf is where this Source keeps its flows, and the empty string for a
// caller that has none.
func dirOf(src Source) string {
	if src == nil {
		return ""
	}
	return src.FlowDir()
}

// UserNames lists the flows a user has written, by file name.
//
// A directory that is not there is a user who has written none, which is
// the ordinary case and not a fault: nothing creates $ORBIT_HOME/flows, and
// a reader that failed because a folder was missing would make the whole
// command unusable until somebody made one.
//
// It is exported for the window, which draws the flow cycle in the start
// dialog and has to mark a user's flow the way `orbit flows` marks it. Names
// is no use there: it returns each flow already written out as "name (built
// in)", and reading a name back out of a formatted line is the kind of
// parsing this repository refuses. So the two callers take what they need —
// this one the bare names, Names the marked line — and the marking rule
// stays here, in one function, above both.
func UserNames(src Source) []string {
	dir := dirOf(src)
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if ValidName(name) != nil {
			continue
		}
		names = append(names, name)
	}
	return names
}
