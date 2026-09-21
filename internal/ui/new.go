package ui

// How a window starts: what it is built from, and what it asks the world
// for once it is running.
//
// Apart from ui.go, which holds what a window remembers: a struct of a
// hundred and fifty fields and the two functions that bring one to life are
// two subjects, and the file was over the ceiling with both in it.

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/upgrade"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

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
		keys: keymap.New(o.Words),
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
	return tea.Batch(tea.RequestBackgroundColor, refresh(m.opts.Reader), tick(), rescanTick(), elapsedTick(), upgrade.Check(m.opts.Version), upgrade.Tick())
}
