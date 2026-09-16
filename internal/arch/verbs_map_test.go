package arch

// The excuses: every verb one way in does not offer, and why.
//
// Apart from verbs_test.go because it is a list that grows and a test that
// does not, and a file where the two share a ceiling is a file where adding
// an excuse means splitting a test.

// notThere is a verb one way in does not offer, and the reason it does not.
//
// The key is "<way in>:<verb>", and a verb that belongs to a family is both
// of its words: "mcp:rules keep". A verb missing from a surface with no line
// here fails, and a line here for a verb that is offered fails too: a
// stale excuse reads as a decision somebody made.
var notThere = map[string]string{
	// The window is a terminal, and a terminal is where a person already
	// is: what it cannot do is what needs a second program in front of it.
	"window:knowledge learn": "the knowledge screen writes facts through its own port, " +
		"not as a task verb",
	"window:board reconcile": "opening the window reconciles every task in the state root, " +
		"so there is nothing left for a gesture to ask for",
	"window:export": "it writes the record into a directory the reader names, " +
		"and the window has nowhere to type a path that is not a task's",
	"window:supervisor retract": "it points at a line by its number in a listing, and the window " +
		"draws the thread as a conversation rather than a numbered list — a number " +
		"typed against a screen that does not number its lines takes back whatever is there",
	"window:task join": "it names the task with -task because the caller it was written for is " +
		"an engine inside a run, where the id is already in the environment; the menu " +
		"passes a task positionally, the way every other verb about one takes it",
	"window:pr show": "the deliver toolbar acts on the pull request rather than listing it; " +
		"which ones are open is the command line's reading",
	"window:rules history": "the review draws what a rule has been through as sentences rather " +
		"than as rows — the reader there is deciding, not auditing, and the rows are the " +
		"command line's reading",
	"window:rules read": "it is the first thing done to a repository and not something done from a " +
		"screen about one — the gesture belongs beside adding the checkout, and what it produces " +
		"lands in the tray this screen already draws",
	"window:rules draft": "it spends money with no task behind it, and the knowledge screen has no " +
		"gesture that costs anything; what it produces lands in the tray this screen already draws",
	"window:rules repeated": "every row of the knowledge screen is one somebody answers — kept, " +
		"dropped, corrected — and a habit is not answerable yet: what turns it into a rule " +
		"is 3.2, and what that produces is a proposal in the tray this screen already draws",

	// A chat carries one message at a time to somebody holding a phone.
	// What it cannot do is hand over a terminal, and what it should not do
	// is answer in columns nobody can read at that width.
	"chat:task take": "this hands a terminal to an engine, and a chat has no terminal to hand over",
	"chat:task diff": "a diff is read in columns against a wide window; a phone would get the " +
		"first file and a scroll bar",
	"chat:task tree":   "the same: a tree of a repository is a shape, not a paragraph",
	"chat:task impact": "the same, and it is the slowest reading there is",
	"chat:export":      "it writes the record into a directory the reader names, and a chat has no filesystem",
	"chat:task compare": "it runs the flow's checks on both sides of a change, which takes minutes " +
		"and answers in columns",

	// The MCP server is spoken to by a model, and these are the four a
	// model has no business asking for on its own — plus the one it could
	// not do if it wanted to.
	"mcp:pr":           "opening a pull request is a person's decision, not a model's",
	"mcp:pr merge":     "merging is a person's decision, not a model's",
	"mcp:pr close":     "closing a pull request is a person's decision, not a model's",
	"mcp:task approve": "accepting a library a task reached for is the question the gate asked a person",
	"mcp:task take":    "this hands a terminal to an engine, and a tool call has no terminal to hand over",
	"mcp:rules draft": "it spends money with no task behind it, so a model asking for it would be " +
		"making Orbit pay for another model on nobody's say-so",
	"mcp:rules read": "it spends money with no task behind it, and a model reading the file another " +
		"model wrote about this project is a loop nobody asked to pay for",
	"mcp:rules pause": "a model that could pause a rule could quietly clear away the ones it keeps " +
		"running into; pausing is the answer a person gives after reading it",
	"mcp:rules resume": "it is the other half of pausing, and belongs to whoever paused it",
}
