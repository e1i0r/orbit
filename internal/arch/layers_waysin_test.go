package arch

// What the four ways in to Orbit may import.
//
// The command line, the browser, the MCP server and the declaration they all
// derive from. They are here rather than in layers_test.go because that file
// was over the ceiling, and this is where the second split falls: these four
// sit over everything that does the work, and their lists are long for that
// reason rather than by accident.
//
// The window is the fourth, and only the window itself is here: the screens
// and the vocabulary underneath it are in layers_ui_test.go, which is the
// first split and the same idea.

var waysIn = map[string][]string{
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
	// internal/supervisor is on internal/ui's list for one thing: the words
	// the window hands the supervisor's errands in. The deliver verbs live
	// beside the thread they are said in so that all four doors hand over
	// the same words, and the window reads them rather than keeping a copy
	// that would drift. It is strings and nothing else — reaching the
	// thread itself stays behind the ports, and the three absences at the
	// top still hold.
	"internal/ui": {"internal/board", "internal/flow", "internal/knowledge", "internal/logger", "internal/repo", "internal/task", "internal/tracker", "internal/ui/cells", "internal/ui/cheat", "internal/ui/clip", "internal/ui/compose", "internal/ui/engines", "internal/ui/fact", "internal/ui/flows", "internal/ui/keymap", "internal/ui/known", "internal/ui/layout", "internal/ui/markdown", "internal/ui/menu", "internal/ui/palette", "internal/ui/panes", "internal/ui/prose", "internal/ui/patch", "internal/ui/point", "internal/ui/prompt", "internal/ui/quota", "internal/ui/repos", "internal/ui/roster", "internal/ui/settings", "internal/ui/spoken", "internal/supervisor", "internal/ui/supervisor", "internal/ui/theme", "internal/ui/typing", "internal/ui/upgrade", "internal/verb", "internal/view", "internal/words"},
	"internal/cli": {
		"internal/board", "internal/engine", "internal/export", "internal/flow",
		"internal/knowledge", "internal/learn", "internal/logger", "internal/mcp",
		"internal/migrate", "internal/quota", "internal/record", "internal/repo",
		"internal/store", "internal/supervisor", "internal/task", "internal/tracker",
		"internal/ui", "internal/ui/fact", "internal/ui/known", "internal/ui/roster",
		"internal/verb", "internal/view", "internal/web", "internal/words", "ui",
	},
	// internal/web is a second reader of the fold, beside internal/ui/panes
	// and not on top of them: it reads the board and the view and answers
	// JSON. It knows no store, no task and no engine — where a worktree
	// lives arrives through a port its caller fills, the way the window is
	// given one.
	"internal/web": {"internal/board", "internal/flow", "internal/repo", "internal/view"},
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
	"internal/mcp": {
		"internal/board", "internal/export", "internal/flow", "internal/knowledge",
		"internal/logger", "internal/record", "internal/repo", "internal/store",
		"internal/supervisor", "internal/task", "internal/verb", "internal/view",
		"internal/words",
	},
	// internal/verb is every action Orbit can be asked for, declared once
	// and done once. It sits under the four ways in and over the packages
	// that do the work, which is why its list is long: it is the one place
	// allowed to know what a verb means, so that the command line, the
	// window, the browser and the MCP server can stop each having an
	// opinion about it.
	// internal/ui/theme is on the list for one constant: the theme the
	// window draws in when the settings name none, which the settings
	// reading has to print. It is a package of names and colours that
	// imports nothing of Orbit's, so it widens nothing else — and the
	// alternative was a second copy of the word, which is exactly how the
	// settings table came to print monokai for a cockpit drawing frauddi.
	"internal/verb": {
		"internal/board", "internal/engine", "internal/flow", "internal/knowledge",
		"internal/learn", "internal/quota", "internal/repo", "internal/store",
		"internal/supervisor", "internal/task", "internal/ui/theme", "internal/view", "internal/words",
	},
}
