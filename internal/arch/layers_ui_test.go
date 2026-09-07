package arch

// What the window's own packages may import.
//
// They are here rather than in layers_test.go because that file was over the
// ceiling, and this is where the split falls: everything under internal/ui
// is one screen or one piece of vocabulary, and each line below says which.

var windowLayers = map[string][]string{
	// internal/ui/compose is the form a task is written into, by hand or
	// from the URL of an issue. It reads an issue through a door the window
	// hands it and answers with the task to write down: what a command does
	// is the window's business and not this form's.
	"internal/ui/compose": {
		"internal/flow", "internal/tracker", "internal/ui/cells", "internal/ui/clip",
		"internal/ui/keymap", "internal/ui/layout", "internal/ui/point", "internal/ui/theme",
		"internal/ui/typing", "internal/words",
	},
	// internal/ui/cheat is the sheet that says what every key does. What it
	// lists — the verbs a task offers, the tabs the detail screen has — it
	// is handed rather than keeping a copy: a sheet with its own list stops
	// being true the day one of them changes.
	"internal/ui/cheat": {"internal/ui/cells", "internal/ui/keymap", "internal/ui/theme", "internal/words"},
	// internal/ui/menu is what can be done to the thing under the pointer,
	// including what cannot and why not. It is handed the three lists it is
	// a menu of — the affordances, the command table, the panes of the task
	// being read — and answers with the row that was chosen: what a verb or
	// a command then does is the window's business and not this screen's.
	"internal/ui/menu": {
		"internal/ui/cells", "internal/ui/keymap", "internal/ui/layout", "internal/ui/point",
		"internal/ui/theme", "internal/words",
	},
	// internal/ui/panes is the body of the task screen: the twelve ways one
	// run can be read. Every one of them is a function of the record and of
	// what the reader has folded, which the window hands over — a pane can
	// reach no port and decide nothing about the task it is drawing.
	"internal/ui/panes": {
		"internal/ui/cells", "internal/ui/layout", "internal/ui/markdown", "internal/ui/theme",
		"internal/view", "internal/words",
	},
	// internal/ui/palette is the ':' line. It answers with the command the
	// reader chose and the window runs it: what a command does is not this
	// screen's business, which is why it names no port at all.
	"internal/ui/palette": {
		"internal/ui/cells", "internal/ui/keymap", "internal/ui/layout", "internal/ui/point",
		"internal/ui/theme", "internal/words",
	},
	// internal/ui/repos is the repository list, and the count each carries.
	// It reads the board because a repository with work going on in it is a
	// fact about the tasks, and it decides nothing about one.
	"internal/ui/repos": {
		"internal/board", "internal/ui/cells", "internal/ui/keymap", "internal/ui/theme",
		"internal/view", "internal/words",
	},
	// internal/ui/engines is the engine and model knobs: which engine a run
	// goes to, which of its models, how hard it thinks. It holds the dials
	// while it is up and hands them back; what to write down it asks the
	// window for, because the settings file is the window's door.
	"internal/ui/engines": {
		"internal/ui/cells", "internal/ui/keymap", "internal/ui/layout", "internal/ui/point",
		"internal/ui/roster", "internal/ui/theme", "internal/words",
	},
	// internal/ui/quota is the screen that says what is left of every
	// engine's windows. It holds nothing — there is nothing on it to
	// choose — so it is drawn from what its Env answers and left again.
	"internal/ui/quota": {
		"internal/ui/cells", "internal/ui/keymap", "internal/ui/roster",
		"internal/ui/theme", "internal/words",
	},
	// internal/ui/roster is the engines this build can run, their dials and
	// what is left of each one's quota: the vocabulary the window's ports
	// answer in. A screen that draws an engine has to be able to name one.
	"internal/ui/roster": {"internal/words"},
	// internal/ui/markdown is how a block of markdown is set: headings,
	// quotations, lists, and a fenced block in its well. It is typography
	// and nothing else — it knows the colours and the width it was given,
	// and never what the text is about.
	"internal/ui/markdown": {"internal/ui/cells", "internal/ui/theme"},
	// internal/ui/known is the screen that lists what Orbit knows: the one
	// place the store can be read whole, corrected, widened or turned off.
	// It reaches the store through three doors the window hands it and has
	// no board — which repository is being worked in arrives as a path.
	"internal/ui/known": {
		"internal/knowledge", "internal/ui/cells", "internal/ui/fact", "internal/ui/keymap",
		"internal/ui/theme", "internal/ui/typing", "internal/words",
	},
	// internal/ui/fact names a piece of knowledge in the window. Two
	// screens draw the same facts and both have to call a scope the same
	// thing, so the naming is one place rather than two.
	"internal/ui/fact": {"internal/knowledge"},
	// internal/ui/supervisor is the screen a conversation with the
	// supervisor is held on, with the column of what Orbit knows down its
	// side. It reads and writes the thread through the doors the window
	// hands it in an Env, and has no board: it can decide nothing about a
	// task.
	"internal/ui/supervisor": {
		"internal/knowledge", "internal/ui/cells", "internal/ui/clip", "internal/ui/fact",
		"internal/ui/keymap", "internal/ui/known", "internal/ui/layout", "internal/ui/markdown",
		"internal/ui/spoken", "internal/ui/theme", "internal/ui/typing", "internal/view",
		"internal/words",
	},
	// internal/ui/spoken is a line the operator typed into the supervisor,
	// taken apart: which of the gestures it is, what it is about, and what
	// is left once the gesture is off the front. It imports nothing — what
	// to do about a line is the window's, and this only says what the line
	// was.
	// internal/ui/settings is the first screen to leave: it holds its own
	// state, is handed what it needs in an Env, and asks the window for the
	// rest in an Out. It writes to the settings file through a port the
	// window passes and reaches nothing else.
	"internal/ui/settings": {"internal/flow", "internal/ui/cells", "internal/ui/keymap", "internal/ui/theme", "internal/words"},
	"internal/ui/spoken":   {},
	// internal/ui/theme is the whole vocabulary of colour: the seven roles,
	// the palettes that answer them, the paper each surface is drawn on, and
	// the lexer that decides which role a run of code takes. It imports
	// nothing of Orbit's — a palette is not a fact about a task — which is
	// why 79 files could start naming it without anything moving the other
	// way.
	"internal/ui/theme":  {},
	"internal/ui/typing": {"internal/ui/theme"},
	// internal/ui/upgrade asks GitHub what the newest release is and says
	// whether it is worth offering. It is the one package under
	// internal/ui that talks to the network, which is the reason it is its
	// own: the window asks and draws, and never reaches out itself.
	"internal/ui/upgrade": {},
}
