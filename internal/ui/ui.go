package ui

// The root model: what one window remembers, the messages that change it,
// and the keystrokes that raise those messages. What it draws is in
// header.go, band.go, rows.go, cells.go and screen.go; what it can be asked
// for is in msg.go.

import (
	"time"

	"charm.land/bubbles/v2/viewport"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/cheat"
	"github.com/e1i0r/orbit/internal/ui/compose"
	"github.com/e1i0r/orbit/internal/ui/engines"
	"github.com/e1i0r/orbit/internal/ui/flows"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/known"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/menu"
	"github.com/e1i0r/orbit/internal/ui/palette"
	"github.com/e1i0r/orbit/internal/ui/panes"
	"github.com/e1i0r/orbit/internal/ui/quota"
	"github.com/e1i0r/orbit/internal/ui/repos"
	"github.com/e1i0r/orbit/internal/ui/settings"
	"github.com/e1i0r/orbit/internal/ui/supervisor"
	"github.com/e1i0r/orbit/internal/view"
)

// messageLife is how long the band keeps what it was told before it goes
// back to saying what is running. Twenty seconds is long enough to be read
// after looking away and short enough that a stale sentence is never the
// thing on screen when something else starts.
const messageLife = 20 * time.Second

// confirm is the question the window is waiting for an answer to, and there
// is never more than one — a second question over the first is how a reader
// answers the wrong one.
type confirm int

const (
	confirmNone confirm = iota
	confirmCancel
	confirmPostCliTask
	confirmDeleteTask
	confirmRequeue
	confirmSkip
)

// Model is one window. It is a value: Update takes one and returns the next,
// and nothing here is a pointer into something a Cmd could still be writing
// to on another goroutine.
//
// The maps are the exception and they are copied before they are written,
// not mutated in place. A map field survives the copy Update makes, so
// mutating one changes the model that was already handed to the renderer —
// which is the one shape of shared state a value model can still have.
type Model struct {
	opts Options
	keys keymap.Keys

	board        board.Board
	seen         bool      // a board has arrived, so the next crossing is worth a bell
	now, logoTap time.Time // logoTap is the last badge click; header.go answers it for 400ms
	// brokeAt is when the newest run of the stuck streak that last took
	// autopilot off had stopped. It is what keeps the breaker from arguing
	// with a reader who turns the switch back on: the same three stuck runs
	// are the same evidence, and only a run that stopped after this one is
	// new.
	brokeAt time.Time
	// world is the pane world of the pass syncPanes is inside; paneEnv says why.
	world  *panes.Env
	errs   int            // how many read failures the last board carried
	noted  notedErrs      // the last failure each clocked source wrote down
	totals map[string]int // phases per flow name, for "review 2/3"

	width, height int
	frame         layout.Frame
	plan          layout.Plan
	tooNarrow     bool
	narrow        layout.TooNarrowError
	dark          bool

	screen    screen
	cursor    int // an index into rows(), which includes the band headers
	offset    int // the first row of rows() the body is showing
	expanded  map[view.Band]bool
	confirm   confirm
	confirmID string
	filtering bool
	// filter is what has been typed into the filter, as a plain string.
	//
	// It is not bubbles/textinput, and the reason is a hard one rather than
	// a preference: that component imports github.com/atotto/clipboard, a
	// module this build does not have and may not add. A one-line filter is
	// a rune appended and a rune removed, which is what filterKey does; the
	// day it needs selection or a cursor is the day the dependency is worth
	// arguing for.
	filter string

	// palette is the ':' line and the list above it, while it is up. Its
	// input is a plain string for the same reason filter is, and its
	// whole shape lives in palette.go.
	palette palette.State

	// menu is what can be done to the thing it was opened on — a task,
	// or the board itself — including what cannot, with the reason. It
	// lives in menu.go and owns the body while it is up, exactly as the
	// palette does. The two never show at once: whichever is up owns the
	// keyboard, and the other's opening key is swallowed by it.
	menu menu.State

	// compose is the form a task is written into, pendingID the id it just
	// wrote — the board polls twice a second, so the new row is selected
	// the moment it shows up — and writing whether that write is still out,
	// because a failed one goes back to the form. All in compose.go.
	compose   compose.State
	pendingID string
	pendTries int
	writing   bool

	// tip is the reader asking what a key does, with ? and then that key.
	// It lives in tip.go, beside the sentences it answers with.
	tip tipState

	// await is the gesture this window is waiting to see land, so that a
	// verb can say what it is doing and then say that it happened. One at
	// a time: see landing.go.
	await awaited

	note           noteState
	settings       settings.State
	flows          flows.State
	repolist       repos.State
	repoFilter     string
	queueFilter    *view.Band
	engines        engines.State
	knobs          engines.Knobs
	help           cheat.State
	quota          quota.State
	supervisor     supervisor.State
	supervisorBusy bool
	// supervisorAt is when the question went out, for the line in the band
	// that says how long it has been: see waiting.go.
	supervisorAt time.Time
	knowledge    known.State
	delivering   deliverPending
	// spinning is whether an animation frame is already on its way.
	// The frame clock is a chain of one-shot ticks, so two starters
	// mean two chains and a spinner that turns twice as fast for the
	// rest of the session. nextFrame, in spinner.go, is the only thing
	// that sets this and the tick is the only thing that clears it.
	spinning         bool
	rawText          bool
	expandedDetail   bool
	upgradeAvailable string

	// folds is which sections of the task view the reader has closed, by
	// the keys in pane_overview.go. Absent is open: a window nobody has
	// folded anything in shows everything it has.
	//
	// It is a map for the reason taken is, and carries the same warning —
	// fold clones it rather than writing it in place, because a Model is
	// copied by every method that returns one and a map shared between two
	// copies is one map.
	folds map[string]bool

	// opened is which rows of each pane the reader has opened, by whatever
	// that pane indexes its rows with — an entry of the record on the
	// timeline and the thinking pane, a phase of the flow on the flow tree.
	// Absent is closed, the other way round from folds: a section is a
	// heading and one of these is a paragraph, and a screen that opens every
	// paragraph at once is the wall this folding is here to take down. The
	// map a pane holds is cloned rather than written in place, for the
	// reason above.
	//
	// heads is which row of each pane an entry's own row was drawn on,
	// written by syncPanes as it lays the panes out. The pointer reads the
	// map the render produced rather than counting the rows again, because a
	// second count is a second opinion about where a row is. It is one map
	// per tab because the timeline and the thinking pane both draw entries,
	// and a pane that draws none has none.
	opened [tabCount]map[int]bool
	heads  [tabCount]map[int]int

	// tree is where the keyboard is standing on the flow tab, in
	// treecursor.go. It and the map are the panes the arrows walk rather
	// than scroll, because they are the panes made of things to open.
	tree treeAt
	// mapAt is the file the map's cursor is on, by its path.
	mapAt string
	// diffMemo is the last drawing of the diff pane, in diffmemo.go.
	diffMemo *diffDrawn

	// errandOut is the delivery a line to the supervisor is being sent for,
	// set only for the length of the send: see askSupervisorTo.
	errandOut Errand

	// shutAttempts is which attempts the reader has closed, by their number.
	// Absent is open, the way folds is and the way opened is not: an attempt
	// is a heading over a block of the record, and a reader who opens the
	// report is opening it to read the report.
	//
	// seams is which row of each pane an attempt's rule was drawn on. It is
	// one map per tab because two panes draw those rules and a reader points
	// at whichever one is up, and it is written by syncPanes for the reason
	// logHeads is.
	shutAttempts map[int]bool
	seams        [tabCount]map[int]int

	// start is the dialog that decides what a run will be, and taken is
	// which tasks this window has handed the terminal to an engine for.
	//
	// taken is a map for the reason expanded is, and carries the same
	// warning: it is cloned by took rather than written in place. Why the
	// fact lives here at all — rather than in the record, where it would
	// survive a restart — is argued at took, in gesture.go.
	start startModel
	taken map[string]bool

	// watching is a palette command that is out running right now, and
	// watchUp is whether its output is on screen. The watch is a pointer
	// because the Cmd that runs the command and every poll that reads its
	// buffer are outside the value model's copy discipline; the buffer is
	// its own mutex-guarded thing precisely so that no field of the model
	// is ever written from another goroutine. watchUp going down does not
	// stop the run — Orbit cannot cancel what it did not spawn a handle
	// for — so the done sentence still reaches the band afterwards.
	watching *commandWatch
	watchUp  bool
	output   string

	// held is the pointer's button, if one is down, and what it went down
	// on. It is on the model rather than in the mouse handler because the
	// press and the release are two messages, and what happens between them
	// — a refresh, a resize, a task finishing — goes through Update like
	// everything else.
	held hold

	message   string
	messageAt time.Time
	notified  bool // a crossing has rung the bell, and the tests can see it

	// The task view, one level down. detail is the id it is open on, and
	// every late answer is checked against it before it is written in.
	//
	// The panes are an array and not a map for the reason the maps above
	// carry a warning: an array is copied with the model, so a Cmd that is
	// still running cannot scroll a pane the renderer has already been
	// handed. Their content is rebuilt by syncPanes whenever a fact behind
	// them moves, and their scroll positions are the only thing on this
	// screen the reader owns.
	detail  string
	tab     tab
	entries []view.Entry
	logErr  error
	// files is what the task's own directory holds, filesKnown is whether
	// an answer has landed at all, and filesErr is what it said if that
	// answer was a failure. The three are kept apart for the reason the
	// diff's three are: an empty listing before the first answer and an
	// empty listing after a real one are different facts.
	files      []view.File
	filesErr   error
	filesKnown bool
	// read is what each file the reader has opened turned out to hold. It
	// is a map for the reason expanded is, and readFile clones it rather
	// than writing it in place. Absent means nobody has opened that file,
	// which is not the same as a file that was read and is empty.
	read     map[string]fileRead
	diff     string
	worktree string
	// diffKnown is whether a diffMsg has landed at all, and diffErr is what
	// it said if the answer was a failure. Neither is folded into diff
	// itself: an empty diff before the first answer and an empty diff after
	// a real "nothing changed" are two different facts, and collapsing them
	// is how a hung git ends up asserting one it never observed.
	diffErr    error
	diffKnown  bool
	diffNoBase bool
	// diffBase is the branch the diff is measured against, looked up once
	// when the view opens and carried from then on, and diffClock is what
	// the rescan keeps between beats. Both exist for the clock: a rescan
	// every two seconds against a repository that takes twelve to answer
	// would otherwise have six diffs in flight and pay for six base
	// lookups, none of which can be cancelled. With these, at most one is
	// out at a time and the base is asked for once per open.
	diffBase          baseRef
	diffClock         diffClock
	hideDiffRationale bool
	collapsedFiles    map[string]bool
	diffFilePicker    bool
	diffFileCursor    int
	// weigh is the impact pane and shape the map pane, both declared beside
	// what reads them: impact.go and tree.go.
	weigh weighed
	shape mapped

	// following is whether the log tab is taking every new entry as it
	// arrives. It is armed when the view opens and released the moment the
	// reader scrolls up, at the one site in scroll that reads the offset.
	following bool
	panes     [tabCount]viewport.Model
}

// Errand is the task and the verb a line to the supervisor is carrying out,
// and empty for a line somebody typed. The side that runs the engine writes
// every step it takes onto that task, under that verb, which is what the
// tree and the band show while the verb is out.
type Errand struct {
	Task view.Task
	Verb string
}

// askPort is the shape of Options.AskSupervisor.
type askPort = func(engineName, conversation, prompt string, about Errand) (string, error)
