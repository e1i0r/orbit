package ui

// The root model: what one window remembers, the messages that change it,
// and the keystrokes that raise those messages. What it draws is in
// header.go, band.go, rows.go, cells.go and screen.go; what it can be asked
// for is in msg.go.

import (
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// gutter is the three cells the cursor's mark is drawn in, to the left of
// every row. It is subtracted before the columns are planned, because the
// plan is about a row and the mark is not part of one.
const gutter = 3

// messageLife is how long the band keeps what it was told before it goes
// back to saying what is running. Twenty seconds is long enough to be read
// after looking away and short enough that a stale sentence is never the
// thing on screen when something else starts.
const messageLife = 20 * time.Second

// screen is which of the window's three screens is on top: the board, the
// task view one level below it, and the dialog that decides what a run will
// be before anything runs.
type screen int

const (
	screenList screen = iota
	screenDetail
	screenStart
)

// confirm is the question the window is waiting for an answer to, and there
// is never more than one — a second question over the first is how a reader
// answers the wrong one.
type confirm int

const (
	confirmNone confirm = iota
	confirmCancel
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
	keys Keys

	board  board.Board
	seen   bool // a board has arrived, so the next crossing is worth a bell
	now    time.Time
	errs   int            // how many read failures the last board carried
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

	// start is the dialog that decides what a run will be, and taken is
	// which tasks this window has handed the terminal to an engine for.
	//
	// taken is a map for the reason expanded is, and carries the same
	// warning: it is cloned by took rather than written in place. Why the
	// fact lives here at all — rather than in the record, where it would
	// survive a restart — is argued at took, in gesture.go.
	start startModel
	taken map[string]bool

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
	detail   string
	tab      tab
	entries  []view.Entry
	logErr   error
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
	// when the view opens and carried from then on, and diffAsking is
	// whether a diff is out at git right now. Both exist for the clock: a
	// rescan every two seconds against a repository that takes twelve to
	// answer would otherwise have six diffs in flight and pay for six base
	// lookups, none of which can be cancelled. With these, at most one is
	// out at a time and the base is asked for once per open.
	diffBase   baseRef
	diffAsking bool
	// following is whether the log tab is taking every new entry as it
	// arrives. It is armed when the view opens and released the moment the
	// reader scrolls up, at the one site in scroll that reads the offset.
	following bool
	panes     [tabCount]viewport.Model
}

// New builds a window from its options. It reads nothing, asks the terminal
// nothing and starts no goroutine: everything that touches the world is a
// Cmd returned from Init.
//
// Width and height are zero for a window that will be told its size by the
// event loop, and set for one frame rendered by --once, which never receives
// a tea.WindowSizeMsg because it never runs a loop.
func New(o Options) Model {
	if o.Words == nil {
		o.Words = words.For("en")
	}
	m := Model{
		opts: o,
		keys: NewKeys(o.Words),
		now:  time.Now(),
		// NeedsYou and Running are open and the other two are shut,
		// because the window's question is "what needs me", and a screen
		// that opens on forty finished tasks has answered a different one.
		expanded: map[view.Band]bool{view.NeedsYou: true, view.Running: true},
		totals:   map[string]int{},
		taken:    map[string]bool{},
	}
	if o.Width > 0 && o.Height > 0 {
		m = m.resize(o.Width, o.Height)
	}
	return m
}

// Init starts the three clocks and asks the terminal, once, what colour it
// is.
//
// The background colour is asked for here and answered in Update, and that
// is the only way this program ever learns it. The synchronous alternatives
// — lipgloss.HasDarkBackground, compat.AdaptiveColor — read the terminal
// from wherever they are called, which inside a render is a blocking read
// in the middle of a frame.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, refresh(m.opts.Reader), tick(), rescanTick(), elapsedTick())
}

// Update is the whole of the window's behaviour, and every case in it is a
// row of the transition table in update_test.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height), nil
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		return m, nil
	case tickMsg:
		// The task view's log is on the same clock as the board, and for
		// the same reason: an append-only file that did not grow costs one
		// stat to find that out. A tail that only moved when the reader
		// pressed a key would not be a tail.
		cmds := []tea.Cmd{refresh(m.opts.Reader), tick()}
		if m.screen == screenDetail {
			cmds = append(cmds, logOf(m.opts.Reader, m.subject()))
		}
		return m, tea.Batch(cmds...)
	case rescanMsg:
		// The diff rides this clock rather than the log's tickMsg one,
		// because git diff is heavier than the stat a tick costs: at
		// board.RescanEvery this is a quarter the log's cadence, which is
		// slow enough to be cheap and still lets a live task's diff change
		// while the reader is looking at it, rather than only at the
		// moment the view was opened.
		//
		// The clock is slower than one diff's worst case all the same — up
		// to five seconds for the diff itself — so a tick that finds the
		// last one still out at git does not ask again. Without that, a
		// repository slow enough to need the bound is the one that gets a
		// second, third and sixth request piled on top of the first.
		cmds := []tea.Cmd{rescan(m.opts.Reader), rescanTick()}
		if m.screen == screenDetail && !m.diffAsking {
			m.diffAsking = true
			cmds = append(cmds, diffOf(m.opts.Reader, m.subject(), m.diffBase))
		}
		return m, tea.Batch(cmds...)
	case elapsedMsg:
		m.now = time.Time(msg)
		return m, elapsedTick()
	case boardMsg:
		return m.applyBoard(msg)
	case controlMsg:
		return m.say(m.controlSaid(msg)), nil
	case startedMsg:
		return m.say(m.startedSaid(msg)), nil
	case readMsg:
		return m.say(m.readSaid(msg)), nil
	case sessionMsg:
		return m.session(msg)
	case sessionEndedMsg:
		return m.sessionEnded(msg), nil
	case editorMsg:
		if msg.Err != nil {
			return m.say(msg.Err.Error()), nil
		}
		return m, nil
	case diffMsg:
		// A diff that arrives for a task the reader has since left is
		// stale, and dropping it is the whole guard: openKey has already
		// pointed m.detail at the new id and asked git about that one, so
		// writing this text in would put one task's changes under another
		// task's heading and nothing on screen would say so.
		if msg.ID != m.detail {
			return m, nil
		}
		m.diff, m.worktree = msg.Text, msg.Tree
		m.diffErr, m.diffKnown, m.diffNoBase = msg.Err, true, msg.NoBase
		m.diffBase, m.diffAsking = msg.Base, false
		return m.syncPanes(), nil
	case logMsg:
		// The same guard, for the same reason: a record that arrives for a
		// task the reader has since left would put one task's history under
		// another task's heading, and nothing on screen would say so.
		if msg.ID != m.detail {
			return m, nil
		}
		m.entries, m.logErr = msg.Entries, msg.Err
		return m.syncPanes(), nil
	case languageMsg:
		return m.language(msg.Lang), nil
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseMsg:
		// One case for all four pointer messages, which is what the
		// interface is for. Which of them this is, and what it means, is
		// mouse.go's question.
		return m.mouse(msg)
	}
	return m, nil
}
