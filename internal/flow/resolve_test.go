package flow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// homeDir is a Source rooted at a temporary directory. The interface is one
// method wide precisely so a test can satisfy it in three words rather than
// building a store — which is also why it is declared where it is consumed.
type homeDir string

func (d homeDir) FlowDir() string { return string(d) }

// flowsIn makes an empty flow directory and answers a Source for it.
func flowsIn(t *testing.T) homeDir {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "flows")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return homeDir(dir)
}

// writeFlow puts one flow file in a flow directory.
func writeFlow(t *testing.T, src homeDir, file, body string) string {
	t.Helper()
	path := filepath.Join(src.FlowDir(), file+".json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// onePhase is the smallest thing that is a flow at all.
func onePhase(name string) string {
	return `{"name":"` + name + `","phases":[{"name":"implement","engine":"claude"}]}`
}

func TestBuiltinFlowsShip(t *testing.T) {
	want := []string{"careful", "quick", "task", "tdd-fuzz-pr"}
	got := BuiltinNames()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("BuiltinNames() = %v, want %v", got, want)
	}
}

func TestEveryBuiltinIsAFlowThatCouldRun(t *testing.T) {
	for _, name := range BuiltinNames() {
		f, err := Builtin(name)
		if err != nil {
			t.Errorf("Builtin(%q): %v", name, err)
			continue
		}
		if f.Name != name {
			t.Errorf("%s.json calls itself %q — the file name is what a task records", name, f.Name)
		}
		if err := f.Validate(); err != nil {
			t.Errorf("the built-in %q does not validate: %v", name, err)
		}
	}
}

// The three shapes a change has. quick is a one-line fix nobody needs to
// read; task is written by one model and read by a better one before you see
// it; careful is the change you would not merge unread.
func TestTheBuiltinsAreTheThreeShapesAChangeHas(t *testing.T) {
	for _, tc := range []struct {
		flow   string
		phases []string
		waits  string // the phase that stops for a human, or "" for none
	}{
		{"quick", []string{"implement"}, ""},
		{"task", []string{"implement", "review"}, "review"},
		{"careful", []string{"implement", "review", "fix"}, "review"},
	} {
		t.Run(tc.flow, func(t *testing.T) {
			f, err := Builtin(tc.flow)
			if err != nil {
				t.Fatalf("Builtin: %v", err)
			}
			var names []string
			waits := ""
			for _, p := range f.Phases {
				names = append(names, p.Name)
				if p.Wait {
					waits += p.Name
				}
				if p.Engine == "" || p.Model == "" {
					t.Errorf("phase %q names engine %q and model %q; a phase with no model runs on whatever the engine defaults to", p.Name, p.Engine, p.Model)
				}
			}
			if strings.Join(names, ",") != strings.Join(tc.phases, ",") {
				t.Errorf("phases = %v, want %v", names, tc.phases)
			}
			if waits != tc.waits {
				t.Errorf("the phases that wait are %q, want %q", waits, tc.waits)
			}
		})
	}
}

func TestTheDefaultIsTheTaskFlow(t *testing.T) {
	if Default != "task" {
		t.Errorf("Default = %q, want task — every task already recorded was written against that name", Default)
	}
	if _, err := Builtin(Default); err != nil {
		t.Errorf("the default flow does not ship: %v", err)
	}
}

func TestResolveFindsABuiltin(t *testing.T) {
	f, err := Resolve(flowsIn(t), "careful")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if f.Name != "careful" || len(f.Phases) != 3 {
		t.Errorf("resolved %+v, want the built-in careful", f)
	}
}

func TestResolveWithoutASourceIsTheBuiltins(t *testing.T) {
	f, err := Resolve(nil, "quick")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if f.Name != "quick" {
		t.Errorf("resolved %+v, want the built-in quick", f)
	}
}

// A refusal that does not say what would have worked leaves the reader
// guessing at a name — which is the one thing they cannot look up from here.
func TestResolveSaysWhatThereIsWhenTheNameIsUnknown(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "mine", onePhase("mine"))

	_, err := Resolve(src, "nonesuch")
	if err == nil {
		t.Fatal("Resolve found a flow that does not exist")
	}
	if !strings.Contains(err.Error(), "nonesuch") {
		t.Errorf("the refusal does not name what was asked for: %v", err)
	}
	for _, name := range append(BuiltinNames(), "mine") {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal does not offer %q: %v", name, err)
		}
	}
}

func TestAFileInTheFlowDirectoryResolves(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "mine", onePhase("mine"))

	f, err := Resolve(src, "mine")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if f.Name != "mine" || len(f.Phases) != 1 {
		t.Errorf("resolved %+v, want the flow the file holds", f)
	}
}

func TestAUserFlowBeatsTheBuiltinItShadows(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "task", `{"name":"task","phases":[{"name":"mine","engine":"claude"}]}`)

	f, err := Resolve(src, "task")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(f.Phases) != 1 || f.Phases[0].Name != "mine" {
		t.Errorf("resolved %+v, want the file rather than the built-in it shadows", f)
	}
}

// A shadow that fails open is worse than one that fails: the run would walk
// something other than what the file says, and nothing would say so.
func TestAMalformedUserFlowFailsRatherThanFallingBackToTheBuiltin(t *testing.T) {
	src := flowsIn(t)
	path := writeFlow(t, src, "task", `{"name":"task","phases":[`)

	f, err := Resolve(src, "task")
	if err == nil {
		t.Fatalf("a shadow that will not parse resolved to %+v", f)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal does not say which file will not parse: %v", err)
	}
}

// A file whose name and whose contents disagree would put one word in
// `orbit flows` and another in the record of every run that walked it.
func TestAUserFlowThatCallsItselfSomethingElseIsRefused(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "mine", onePhase("other"))

	_, err := Resolve(src, "mine")
	if err == nil {
		t.Fatal("a flow whose file name and whose own name disagree resolved")
	}
	if !strings.Contains(err.Error(), "mine") || !strings.Contains(err.Error(), "other") {
		t.Errorf("the refusal does not say which two names disagree: %v", err)
	}
}

// The name arrives from a command line and is joined onto a path, so this
// package is what stands between it and the filesystem.
func TestResolveRefusesANameThatWouldLeaveTheFlowDirectory(t *testing.T) {
	src := flowsIn(t)
	for _, name := range []string{"", ".", "..", "../task", "sub/task", "a" + string(os.PathSeparator) + "b"} {
		if _, err := Resolve(src, name); err == nil {
			t.Errorf("Resolve accepted %q as the name of a flow", name)
		}
	}
}

// The classification is this package's whole answer about where a flow came
// from, and it is the answer both screens draw. There is no English in it:
// what "yours, shadowing the built-in" says in a given language is decided
// where it is printed, by internal/cli and internal/ui, through one key each.
func TestListSaysWhereEachFlowCameFromAndSortsThem(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "mine", onePhase("mine"))
	writeFlow(t, src, "task", onePhase("task"))
	// Not a flow, and not listed: the directory holds JSON files and
	// whatever else a user happens to leave in it.
	if err := os.WriteFile(filepath.Join(src.FlowDir(), "notes.txt"), []byte("mine is the good one\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := List(src)
	want := []Listed{
		{Name: "careful", Origin: OriginBuiltin},
		{Name: "mine", Origin: OriginUser},
		{Name: "quick", Origin: OriginBuiltin},
		{Name: "task", Origin: OriginShadow},
		{Name: "tdd-fuzz-pr", Origin: OriginBuiltin},
	}
	if !slices.Equal(got, want) {
		t.Errorf("List() =\n%v\nwant\n%v", got, want)
	}
}

func TestListOnAStateRootWithNoFlowsOfItsOwn(t *testing.T) {
	want := []Listed{
		{Name: "careful", Origin: OriginBuiltin},
		{Name: "quick", Origin: OriginBuiltin},
		{Name: "task", Origin: OriginBuiltin},
		{Name: "tdd-fuzz-pr", Origin: OriginBuiltin},
	}
	for _, src := range []Source{flowsIn(t), homeDir(filepath.Join(t.TempDir(), "never-made")), nil} {
		if got := List(src); !slices.Equal(got, want) {
			t.Errorf("List() =\n%v\nwant\n%v", got, want)
		}
	}
}

// Names is bare because of who reads it: somebody who just typed a flow name
// wrong, and whose next move is to type one right. A mark in that sentence
// answers a question they did not ask, and "careful (built in)" is not a
// thing they can type.
func TestNamesAreBareNamesAndTheRefusalOffersThem(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "mine", onePhase("mine"))

	got := strings.Join(Names(src), ", ")
	want := "careful, mine, quick, task, tdd-fuzz-pr"
	if got != want {
		t.Fatalf("Names() = %q, want %q", got, want)
	}
	_, err := Resolve(src, "nope")
	if err == nil {
		t.Fatal("a name nothing answers to resolved")
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal does not offer the names there are: %v", err)
	}
}
