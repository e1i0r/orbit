package ui

// The root model: what one window remembers, the messages that change it,
// and the keystrokes that raise those messages. What it draws is in
// header.go, rows.go and screen.go; what it can be asked for is in msg.go.

import (
	"errors"
	"strconv"
	"time"

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

// screen is which of the window's two screens is on top. There are two in
// this task; the task view fills out one level down.
type screen int

const (
	screenList screen = iota
	screenDetail
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
// The two maps are the exception and they are copied before they are
// written, not mutated in place. A map field survives the copy Update makes,
// so mutating one changes the model that was already handed to the renderer
// — which is the one shape of shared state a value model can still have.
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

	message   string
	messageAt time.Time
	notified  bool // a crossing has rung the bell, and the tests can see it

	detail string // the id the task view is open on
	diff   string
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
		return m, tea.Batch(refresh(m.opts.Reader), tick())
	case rescanMsg:
		return m, tea.Batch(rescan(m.opts.Reader), rescanTick())
	case elapsedMsg:
		m.now = time.Time(msg)
		return m, elapsedTick()
	case boardMsg:
		return m.applyBoard(msg)
	case controlMsg:
		return m.say(m.controlSaid(msg)), nil
	case startedMsg:
		return m.say(m.startedSaid(msg)), nil
	case editorMsg:
		if msg.Err != nil {
			return m.say(msg.Err.Error()), nil
		}
		return m, nil
	case diffMsg:
		m.detail, m.diff = msg.ID, msg.Text
		if msg.Err != nil {
			m.diff = msg.Err.Error()
		}
		return m, nil
	case languageMsg:
		return m.language(msg.Lang), nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// applyBoard takes the next board, or explains why there is not one.
//
// A boardMsg with a zero ReadAt is a read that failed or an enumeration that
// found nothing to say. The board already on screen is kept in both cases: a
// window that blanks because one stat failed has thrown away the answer it
// spent the last half-second holding.
func (m Model) applyBoard(msg boardMsg) (tea.Model, tea.Cmd) {
	if msg.Board.ReadAt.IsZero() {
		if len(msg.Board.Errs) > 0 {
			return m.say(msg.Board.Errs[0].Error()), nil
		}
		return m, nil
	}
	first := !m.seen
	m.board, m.seen = msg.Board, true
	m.totals = phaseTotals(msg.Board.Tasks)
	m = m.replan().clampCursor()
	if first {
		m.cursor = m.firstTask()
		m = m.follow()
	}
	// A read failure is said when the count of them changes and not on
	// every refresh, because the poll is twice a second and one unreadable
	// log would otherwise own the band for as long as it stayed unreadable.
	if n := len(msg.Board.Errs); n != m.errs {
		m.errs = n
		if n > 0 {
			m = m.say(m.opts.Words.P("msg.unreadable", n, "{n} record could not be read", "{n} records could not be read"))
		}
	}
	if first || len(msg.Changed.Entered) == 0 {
		return m, nil
	}
	m.notified = true
	m = m.say(m.opts.Words.P("msg.entered", len(msg.Changed.Entered), "{n} task needs you", "{n} tasks need you"))
	return m, tea.Raw("\a")
}

// resize takes the new geometry, or refuses it with both numbers.
func (m Model) resize(w, h int) Model {
	m.width, m.height = w, h
	f, err := layout.Fit(w, h)
	if err != nil {
		var narrow layout.TooNarrowError
		if !errors.As(err, &narrow) {
			narrow = layout.TooNarrowError{Need: layout.MinWidth, Got: w}
		}
		m.tooNarrow, m.narrow = true, narrow
		return m
	}
	m.tooNarrow, m.frame = false, f
	return m.replan().follow()
}

// replan re-plans the columns, from the whole board rather than from the
// rows currently shown: a column that changed width while a filter was being
// typed would move every field on screen between two keystrokes.
func (m Model) replan() Model {
	m.plan = layout.Columns(m.frame.Body.W-gutter, m.board.Tasks, m.opts.Words.Cells)
	return m
}

// say puts one sentence in the activity band. An empty sentence is not a
// sentence: it would blank the band, and a status area that goes blank reads
// as broken.
func (m Model) say(text string) Model {
	if text == "" {
		return m
	}
	m.message, m.messageAt = text, m.now
	return m
}

// controlSaid is what the band says about a word that was written.
func (m Model) controlSaid(msg controlMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	return m.opts.Words.T("msg.control_sent", "asked {id} to {word}",
		about("id", msg.ID), about("word", msg.Word))
}

// startedSaid is what the band says about a run that began.
func (m Model) startedSaid(msg startedMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	return m.opts.Words.T("msg.started", "{id} is running, as process {pid}",
		about("id", msg.ID), about("pid", strconv.Itoa(msg.Pid)))
}

// language rewrites the language, and everything built from it. The key map
// is rebuilt because a binding carries its own help text, and the help
// overlay reads the bindings.
func (m Model) language(lang string) Model {
	if m.opts.Settings != nil {
		if err := m.opts.Settings.SetLanguage(lang); err != nil {
			return m.say(err.Error())
		}
	}
	m.opts.Words = words.For(lang)
	m.keys = NewKeys(m.opts.Words)
	return m.replan()
}
