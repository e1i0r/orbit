// Import direction is enforced here, checked against the source with
// go/parser and go/ast rather than against a running build — a violation has
// to be visible to `go test ./...` before it is visible to a reviewer.
//
// The terminal-width rule was in this file too, until the two of them
// together reached the size ceiling and the map of layers had no room left
// to take a line. It is the same kind of rule read the same way and it is
// about a different thing entirely, so it is in width_test.go beside this
// one.
package arch

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// layers says which of Orbit's own packages each package may import.
// A package may import anything on its list, and nothing else of Orbit's.
//
// The load-bearing entries are the absences. internal/ui does not list
// internal/record, so the window cannot append an event; it does not list
// internal/store, so the window cannot build a path under the state root;
// it does not list internal/engine, so the window cannot start a model.
// "The window derives everything and holds no authority" is those three
// missing lines, and nothing else makes it true.
var layers = map[string][]string{
	"cmd/orbit":     {"internal/cli"},
	"internal/arch": {},
	// internal/task is on internal/board's list for one function: task.Alive,
	// which reads the run marker and asks the operating system whether the
	// pid it names is still there. It is a widening, and it was argued
	// rather than assumed. The alternative was a second implementation of
	// the marker's format and the liveness check living inside
	// internal/board, and two readers of one file drift — the very class of
	// defect the record exists to prevent, arriving through the back door.
	// The direction is safe: nothing in internal/task imports
	// internal/board, so there is no cycle, and internal/ui already lists
	// internal/task because that is how every gesture reaches the function
	// its subcommand calls. What is not widened is the line above:
	// internal/board still does not append anything itself, and internal/ui
	// still cannot reach internal/record, internal/store or internal/engine.
	"internal/board": {"internal/record", "internal/repo", "internal/store", "internal/task", "internal/view"},
	// internal/db is the record: it turns an event into rows and reads them
	// back. internal/record is on its list because the event is the thing
	// it stores, and it is the only entry — the record knows no store, no
	// task and no engine, so nothing here can decide anything about a run
	// or reach the state root to find one.
	"internal/db":  {"internal/record"},
	"internal/cli": {"internal/board", "internal/engine", "internal/export", "internal/flow", "internal/knowledge", "internal/logger", "internal/mcp", "internal/migrate", "internal/quota", "internal/repo", "internal/store", "internal/supervisor", "internal/task", "internal/tracker", "internal/ui", "internal/view", "internal/words"},
	// internal/logger is on internal/engine's list for the one thing this
	// package does that nothing else in Orbit does: it starts somebody
	// else's program. What that cost, how long it took and which of the
	// binaries on this machine it was are facts no other package sees. It
	// widens nothing else: engine still knows no record, store or task.
	"internal/engine": {"internal/logger"},
	// internal/export is internal/migrate read backwards, and it carries the
	// same three imports for the same three reasons: internal/db to read the
	// record, internal/record to turn a row back into the line it was
	// written as, internal/store to name where that line goes. It writes
	// nothing but files a person asked for, starts nothing, and — like the
	// migration — nothing imports it but the front door that triggers it.
	"internal/export": {"internal/db", "internal/record", "internal/store"},
	"internal/flow":   {},
	// internal/knowledge is what Orbit has learned: a fact, its scope and
	// where it came from. It imports nothing of Orbit's and that is the
	// point — the sentence an agent is told has to be traceable to a source,
	// and a package that could reach the record or the store could decide
	// things about a run instead of describing one.
	"internal/knowledge": {},
	"internal/logger":    {},
	// internal/mcp is the widest list on this map, and it is the same width
	// as internal/cli's for the same reason: it is a second front door onto
	// the very functions the command line calls, so it reaches internal/task
	// to act and internal/board to read. internal/record is on it for one
	// tool — orbit_inspect_task folds a task's events into the answer the
	// cockpit's inspector draws — and it is a read, not a write: nothing
	// here appends. internal/logger is on it because this is the door nobody
	// is standing at: a model drives it for hours from another process, so a
	// refusal answered to that model reached no terminal and no record. What
	// is absent is internal/engine: a supervisor starts runs, not models.
	// internal/knowledge is on internal/mcp's list because this is the door
	// the store grows through: an agent that hit something worth knowing
	// writes it down mid-task, and the next run against that code is told
	// before it starts. Reading it is the other half — a model that asks
	// before planning starts from what is known rather than finding it out
	// again.
	"internal/mcp": {"internal/board", "internal/flow", "internal/knowledge", "internal/logger", "internal/record", "internal/repo", "internal/store", "internal/supervisor", "internal/task", "internal/view"},
	// internal/migrate reads the files an older Orbit wrote and fills the
	// database from them, so it is the one package that touches the record
	// on both sides: internal/store to find the logs, internal/record to
	// read them, internal/db to write them down. It is a translation and
	// nothing else — it starts no run, and nothing imports it but the two
	// front doors that trigger it.
	"internal/migrate": {"internal/db", "internal/record", "internal/store"},
	"internal/quota":   {},
	"internal/record":  {},
	"internal/repo":    {"internal/store"},
	// internal/store lists internal/db for one method: Record, which opens
	// the SQLite file under the state root and hands back the same handle
	// every time. The handle has to be owned somewhere — db.Open pins each
	// one to a single connection so that one process is one writer, and two
	// handles in one process would be two writers contending for one lock —
	// and the store is where the state root already lives. The direction is
	// safe: internal/db lists only internal/record, so nothing comes back
	// the other way.
	// internal/logger is on internal/store's list for one line, and the line
	// is the argument. The settings lock is the only place in Orbit where a
	// process's death leaves a mark that another process silently repairs:
	// a lock file older than a minute is broken, the change goes through,
	// and both processes report success. Nobody is standing at that door
	// either. It widens nothing else — the store still decides nothing about
	// a run and still reaches no engine, no task and no board — and
	// internal/logger imports nothing of Orbit's, so there is no cycle.
	"internal/store": {"internal/db", "internal/logger"},
	// internal/supervisor is the one conversation in Orbit that belongs to
	// no task: a global, append-only thread hanging off the state root. It
	// lived inside internal/task for as long as there was nowhere else to
	// put it, which made a package whose doc says it turns a sentence into
	// a run also the home of a chat log, and put internal/task in the way
	// of every reader of that log.
	//
	// internal/engine is on its list because this package does start a
	// model — the supervisor is one — and internal/record and internal/store
	// because the thread is a file under the root. What is absent is
	// internal/task: the supervisor acts on tasks through the same front
	// doors everything else does, internal/cli and internal/mcp, and never
	// from in here. That absence is what keeps the direction one-way, and
	// with it there is no cycle to make: internal/task does not list this
	// package either.
	// internal/knowledge is on the supervisor's list because it answers with
	// the standing rules in front of it: one that directs a task without
	// them can contradict a gate an hour before the gate refuses the work.
	// internal/logger is there because a store it could not read costs the
	// facts and not the answer, and a loss that says nothing is the kind
	// nobody finds.
	"internal/supervisor": {"internal/engine", "internal/knowledge", "internal/logger", "internal/record", "internal/store"},
	// internal/logger is on internal/task's list for the same reason it is on
	// internal/ui's, and for one more: a run that is SIGKILLed writes nothing
	// about its own death, so the last line it managed to log is the only
	// account of it there is until a reader runs reconcile. It is a widening,
	// and it was argued rather than assumed. Nothing of Orbit's is imported
	// by internal/logger, so no cycle can be made of it, and what a run may
	// do is not widened at all: the log is a second copy of what the record
	// already took, written after the record took it, and no reader of Orbit
	// decides anything from it.
	// internal/knowledge is on internal/task's list because this is where a
	// fact reaches a model: the prompt of a phase, which this package
	// writes. It is a read — internal/task tells the engine what is known
	// and decides nothing about it.
	"internal/task":    {"internal/engine", "internal/flow", "internal/knowledge", "internal/logger", "internal/record", "internal/repo", "internal/store"},
	"internal/tracker": {},
	// internal/logger is on internal/ui's list for one reason: the window is
	// where a failure a reader saw arrives, and a failure nobody wrote down
	// ends as "it said something in red once, I think". It is a widening, and
	// it was argued rather than assumed. internal/logger imports nothing of
	// Orbit's — its own entry above is empty — so no cycle can be made of it.
	// What it does not widen is the three absences named at the top: the
	// window still cannot append to the record, still cannot build a path
	// under the state root, and still cannot start a model. The distinction
	// that keeps the second of those true is that internal/ui writes through
	// the package-level logger internal/cli opened, and never opens one
	// itself: where the file lives is internal/store's answer to give and
	// internal/cli's to ask for, here as everywhere else.
	// internal/knowledge is on internal/ui's list for one screen: the
	// supervisor draws what Orbit knows down its side, so that a rule
	// somebody is about to write sits beside the ones already standing. It
	// is a type and a read — the facts arrive through a port, because
	// reaching the state root to load them is what the window may not do.
	"internal/ui": {"internal/board", "internal/flow", "internal/knowledge", "internal/logger", "internal/repo", "internal/task", "internal/tracker", "internal/ui/layout", "internal/view", "internal/words"},
	// internal/ui/layout is widened to internal/view for one reason:
	// layout.Columns plans a row's columns from the board it is about to
	// draw, and the board is []view.Task. It is a widening, and it was
	// argued rather than assumed. internal/view imports only
	// internal/record and is pure data with no behaviour of its own, so
	// there is no cycle — nothing in internal/view imports anything under
	// internal/ui — and there is nothing to leak: a Task carries no handle
	// to the record it was folded from. The alternative was a width-only
	// struct built by internal/ui and handed down, which is a second
	// description of a row living one package away from the first, and two
	// descriptions of one thing drift. What must stay true and stay tested
	// is the line below this one: no tea import anywhere in
	// internal/ui/layout, so the geometry can never become a function of
	// anything but the numbers it was given.
	"internal/ui/layout": {"internal/view"},
	"internal/view":      {"internal/record"},
	"internal/words":     {},
}

// modulePath prefixes every import that is one of Orbit's own packages
// rather than a third party's.
const modulePath = "github.com/e1i0r/orbit"

// TestImportsFollowTheLayers walks every Go file, maps it to its package
// directory relative to the module root, and fails any import of
// github.com/e1i0r/orbit/... whose target is not on that package's list in
// arch.layers. A package that exists and has no entry is a failure, not a
// pass — a new package must be placed in the layering deliberately.
func TestImportsFollowTheLayers(t *testing.T) {
	modRoot := root(t)
	for _, path := range goFiles(t) {
		rel, err := filepath.Rel(modRoot, filepath.Dir(path))
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}

		pkg := filepath.ToSlash(rel)

		allowed, ok := layers[pkg]
		if !ok {
			t.Errorf("%s belongs to package %q, which has no entry in arch.layers — place it in the layering", path, pkg)
			continue
		}

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == modulePath || !strings.HasPrefix(importPath, modulePath+"/") {
				continue
			}

			target := strings.TrimPrefix(importPath, modulePath+"/")
			if !slices.Contains(allowed, target) {
				t.Errorf("%s imports %q, which %s does not list in arch.layers", path, importPath, pkg)
			}
		}
	}
}

// teaModule is the terminal's event loop. internal/ui/layout may not import
// it, in any package under it, in test files included.
const teaModule = "charm.land/bubbletea/v2"

// TestLayoutNeverImportsTheEventLoop is the other half of the widening
// argued in arch.layers: internal/ui/layout may read a view.Task, and it may
// not read a terminal.
//
// The property is that layout is a pure function of the numbers it is given.
// An import of bubbletea is the one thing that could quietly end that — a
// window size read from a message rather than taken as a parameter, a
// background colour asked of the terminal, a command returned from what is
// supposed to be arithmetic — and none of it would look wrong in a diff. It
// would show up as a layout that cannot be table-tested, which is a thing
// you notice a plan later.
//
// lipgloss is deliberately not banned: measuring a string in cells is the
// one terminal fact the arithmetic genuinely needs, it asks the terminal
// nothing to answer, and the alternative is counting bytes, which is the
// mistake TestUIMeasuresCellsNotBytes exists to prevent.
func TestLayoutNeverImportsTheEventLoop(t *testing.T) {
	modRoot := root(t)
	for _, path := range goFiles(t) {
		rel, err := filepath.Rel(modRoot, path)
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}

		if !strings.HasPrefix(filepath.ToSlash(rel), "internal/ui/layout/") {
			continue
		}

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == teaModule || strings.HasPrefix(importPath, teaModule+"/") {
				t.Errorf("%s imports %q — the layout is a pure function of the width and the height it is given, and an event loop is how that stops being true", rel, importPath)
			}
		}
	}
}
