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

// Origin says where the flow a name resolves to came from.
//
// It is a classification and not a sentence, and that is the whole of why
// it exists. This package holds no English: a flow is data, and the words a
// reader sees are written at the call site that draws them, where
// internal/words can check them against the catalogue. A mark spliced in
// here as a Go constant was a user-facing sentence no translation test
// could see — so the fact travels as a value, and `orbit flows` and the
// window's start dialog each say it through the same catalogue key.
//
// A shadow is a case of its own because a flow that stopped behaving as
// documented is then one command away from explaining itself.
type Origin int

const (
	// OriginUnknown is a name nothing answers to: the zero value, and what
	// a task written against a flow somebody has since deleted has. List
	// never returns it.
	OriginUnknown Origin = iota
	// OriginBuiltin is a flow shipped inside the binary.
	OriginBuiltin
	// OriginUser is a file in the user's flow directory.
	OriginUser
	// OriginShadow is a file of the user's hiding a built-in of the same
	// name.
	OriginShadow
)

// Listed is one flow name and where the flow that name resolves to came
// from.
type Listed struct {
	Name   string
	Origin Origin
}

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

// List is every flow that could be resolved, sorted, each with where the
// one that would win came from.
//
// *This is the only place that decision is made.* Two screens show it —
// the `orbit flows` listing and the window's start dialog — and both take
// the classification from here rather than working it out again from
// BuiltinNames and a directory listing. Two implementations of one rule are
// two rules the day somebody edits one of them, and the drift is invisible:
// both keep printing a mark, and the marks disagree.
//
// It is a listing and not a load: a file in the directory is named here
// whether or not it parses, because "there is a file called that" is the
// thing a reader is asking, and Resolve is what says whether it is a flow.
func List(src Source) []Listed {
	origins := make(map[string]Origin)

	names := BuiltinNames()
	for _, n := range names {
		origins[n] = OriginBuiltin
	}

	for _, n := range userNames(src) {
		if _, shadows := origins[n]; shadows {
			origins[n] = OriginShadow
		} else {
			origins[n] = OriginUser
			names = append(names, n)
		}
	}

	sort.Strings(names)

	listed := make([]Listed, 0, len(names))
	for _, n := range names {
		listed = append(listed, Listed{Name: n, Origin: origins[n]})
	}

	return listed
}

// Names is that same listing as bare names.
//
// It is what a sentence uses when the reader is being told what they could
// have typed: the mark answers "where did this come from", which is not the
// question somebody who just misspelled a flow name is asking.
func Names(src Source) []string {
	listed := List(src)

	names := make([]string, 0, len(listed))
	for _, l := range listed {
		names = append(names, l.Name)
	}

	return names
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

// userNames lists the flows a user has written, by file name.
//
// A directory that is not there is a user who has written none, which is
// the ordinary case and not a fault: nothing creates $ORBIT_HOME/flows, and
// a reader that failed because a folder was missing would make the whole
// command unusable until somebody made one.
//
// It is unexported, and it went back to being so when List started
// answering with a classification. It was exported for one caller — the
// window, which draws the flow cycle and marks a flow of the reader's own
// the way `orbit flows` marks it — and that caller was then deciding, from
// these bare names and BuiltinNames, a thing List had already decided. The
// window takes List now, so nothing outside this package needs the halves.
func userNames(src Source) []string {
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
