package arch

// layers says which of Orbit's own packages each package may import.
// A package may import anything on its list, and nothing else of Orbit's.
//
// The load-bearing entries are the absences. internal/ui does not list
// internal/record, so the window cannot append an event; it does not list
// internal/store, so the window cannot build a path under the state root;
// it does not list internal/engine, so the window cannot start a model.
// "The window derives everything and holds no authority" is those three
// missing lines, and nothing else makes it true.
var layers = mergedLayers()

// mergedLayers is the whole map, written down in three files because one was
// over the ceiling: the packages that do the work here, the four ways in to
// them in layers_waysin_test.go, and the window's own screens in
// layers_ui_test.go. The splits are where the program's own seams are.
func mergedLayers() map[string][]string {
	all := make(map[string][]string, len(program)+len(waysIn)+len(windowLayers))

	for _, one := range []map[string][]string{program, waysIn, windowLayers} {
		for pkg, may := range one {
			all[pkg] = may
		}
	}

	return all
}

var program = map[string][]string{
	// web/build writes the landing page and imports nothing of Orbit's: it is
	// a template, two catalogues of sentences and the standard library. It is
	// in the module rather than beside it so that `go test ./...` checks the
	// committed pages are what the template says — a generated file nobody
	// checks is a generated file that has already drifted.
	"web/build":     {},
	"cmd/orbit":     {"internal/cli"},
	"internal/arch": {"internal/verb"},
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
	"internal/db": {"internal/record"},
	// ui is the browser half, and the Go in it is one file: the built
	// window, embedded. It imports nothing because there is nothing for it
	// to import — what it carries is bytes.
	"ui": {},
	// The flow tests and the stand-in engine they put on PATH import nothing
	// of Orbit's, and that is the whole of what makes them what they are:
	// they reach the program the way a person does, by running the binary. A
	// test that imported internal/task would be asking the program about
	// itself, and would go on passing after the command line stopped being
	// able to ask the same question.
	"test/integration":            {},
	"test/integration/fakeengine": {},
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
	// internal/learn is where a sentence waits between being said and being
	// agreed with: the record because the tray is a table, and
	// internal/knowledge because a sentence somebody keeps becomes a fact.
	"internal/learn":  {"internal/db", "internal/knowledge", "internal/logger", "internal/store"},
	"internal/logger": {},
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
	//
	// internal/knowledge is on its list for the same reason it is on
	// internal/task's: this is a second place a fact reaches a model. The
	// supervisor answers with the standing rules in front of it, so that it
	// cannot direct a task into something a gate would refuse an hour later,
	// and so that it can tell whether what the operator just said is already
	// written down. It is a read — this package loads facts and writes none.
	// internal/logger is on it because a store that cannot be read costs the
	// facts and not the answer: the supervisor keeps answering, and the line
	// in the log is the only account of what it was answering without.
	// internal/learn is on internal/supervisor's list for one call, at the
	// door every line of the thread goes through: a rule is said in the
	// middle of talking, so noticing it belongs where the sentence arrives —
	// the same place for the cockpit, a command and a tool call alike.
	"internal/supervisor": {
		"internal/engine", "internal/knowledge", "internal/learn",
		"internal/logger", "internal/record", "internal/store",
	},
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
	// internal/learn is on the list because a directive is one of the two
	// places a rule gets said: half of what somebody types at a run is not
	// about that run. It is a write to the tray and never to what Orbit
	// knows — a sentence there is not knowledge until somebody agrees with
	// it — and internal/learn imports nothing of this package's, so no
	// cycle can be made of it.
	"internal/task":    {"internal/engine", "internal/flow", "internal/knowledge", "internal/learn", "internal/logger", "internal/record", "internal/repo", "internal/store"},
	"internal/tracker": {},
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
	// internal/ui/typing is the field somebody types into: a value, a caret,
	// and the other end of a selection, with the wrapping that decides which
	// drawn line a caret is on. It imports nothing, not even internal/words,
	// because it holds runes and not sentences — what the window puts around
	// it is the window's business, and the caret arithmetic is the part that
	// was worth being able to test on its own.
	// internal/ui/prompt is what the window asks an engine for, in the
	// engine's own words: the six deliver verbs, the instruction a phase is
	// given, and the shape a flow drafted from a sentence must come back in.
	// It imports nothing, because a prompt is a string and everything that
	// decides which one to send is the window's.
	// internal/ui/clip is the pasteboard, which is three commands that may
	// not be installed rather than a library: pbcopy and pbpaste on a mac,
	// wl-copy and xclip on the two Linux display servers. It imports nothing.
	// internal/ui/cells is how the window measures: a terminal draws a wide
	// rune in two columns and a combining mark in none, so a cut made on the
	// byte count leaves a row of the wrong width and a broken escape with
	// it. Everything here counts what a reader can see.
	"internal/ui/cells": {"internal/ui/theme"},
	"internal/ui/clip":  {},
	// internal/ui/keymap is which key does what, and — the half that is
	// actually the decision — which verbs a task offers right now and the
	// reason each refused one gives. It reads a view.Task and says what can
	// be done to it; doing any of it is the window's, which is why nothing
	// here reaches the record or the store.
	"internal/ui/keymap": {"internal/view", "internal/words"},
	// internal/ui/patch is a diff read: which files it touches, how much of
	// each, and what the record says the change to each was for. It reads
	// text git wrote and says what is in it; what to draw of that is the
	// window's.
	"internal/ui/patch": {"internal/ui/theme", "internal/view", "internal/words"},
	// internal/ui/point is what one cell of the terminal holds, so that a
	// click can be answered. Every screen writes these and the window reads
	// them: a screen that is its own package still has to be able to say
	// "a click here means this flow".
	"internal/ui/point": {"internal/ui/layout", "internal/view"},
	// internal/ui/flows is the whole flow designer: the list, the form, the
	// diagram and the tab that turns a sentence into a flow. It reads and
	// writes flows through the source the window hands it and asks an
	// engine through a port; it has never heard of a task or the board.
	"internal/ui/flows": {
		"internal/flow", "internal/ui/cells", "internal/ui/clip", "internal/ui/keymap",
		"internal/ui/layout", "internal/ui/point", "internal/ui/prompt", "internal/ui/theme",
		"internal/words",
	},
	"internal/ui/prompt": {},
	"internal/view":      {"internal/record"},
	"internal/words":     {},
}
