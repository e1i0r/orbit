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

// mergedLayers is the whole map: the program's packages here, and the
// window's own in layers_ui_test.go. It is two files because one was over
// the ceiling, and the split is where the window starts.
func mergedLayers() map[string][]string {
	all := make(map[string][]string, len(program)+len(windowLayers))
	for pkg, may := range program {
		all[pkg] = may
	}

	for pkg, may := range windowLayers {
		all[pkg] = may
	}

	return all
}

var program = map[string][]string{
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
	"internal/cli": {"internal/board", "internal/engine", "internal/export", "internal/flow", "internal/knowledge", "internal/logger", "internal/mcp", "internal/migrate", "internal/quota", "internal/repo", "internal/store", "internal/supervisor", "internal/task", "internal/tracker", "internal/ui", "internal/ui/roster", "internal/view", "internal/words"},
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
	"internal/supervisor": {
		"internal/engine", "internal/knowledge", "internal/logger",
		"internal/record", "internal/store",
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
	"internal/ui": {"internal/board", "internal/flow", "internal/knowledge", "internal/logger", "internal/repo", "internal/task", "internal/tracker", "internal/ui/cells", "internal/ui/cheat", "internal/ui/clip", "internal/ui/compose", "internal/ui/engines", "internal/ui/fact", "internal/ui/flows", "internal/ui/keymap", "internal/ui/known", "internal/ui/layout", "internal/ui/markdown", "internal/ui/menu", "internal/ui/palette", "internal/ui/panes", "internal/ui/prose", "internal/ui/patch", "internal/ui/point", "internal/ui/prompt", "internal/ui/quota", "internal/ui/repos", "internal/ui/roster", "internal/ui/settings", "internal/ui/spoken", "internal/ui/supervisor", "internal/ui/theme", "internal/ui/typing", "internal/ui/upgrade", "internal/view", "internal/words"},
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
