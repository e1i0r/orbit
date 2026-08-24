package ui

// The key map, which is also the help overlay: bubbles/help renders from the
// same key.Binding values the window matches against, so a gesture cannot be
// bound one way and documented another.

import (
	"charm.land/bubbles/v2/key"

	"github.com/e1i0r/orbit/internal/words"
)

// Keys is one binding per gesture, each carrying the help text it is
// documented with.
//
// Case carries meaning throughout: lowercase acts on the task under the
// cursor, uppercase changes a standing setting. The program this replaces
// assigned case arbitrarily, so there was nothing to learn from it — a
// reader who knew twenty of its keys still could not guess the twenty-first.
//
// The fields are grouped in the order a reader meets them: moving about,
// opening things, doing something to a task, and the settings that are not
// about any one task.
type Keys struct {
	Up, Down, First, Last, PageUp, PageDown           key.Binding
	Open, Back, NextTab, PrevTab, Sideways            key.Binding
	Start, Run, ChangeFlow, Menu, Compose             key.Binding
	Pause, Resume, Cancel                             key.Binding
	Take, Hand, Ask, MarkRead, Edit                   key.Binding
	Filter, Commands, Repos, Autopilot, Language, Help, Quit key.Binding
}

// NewKeys builds the key map, with every description translated.
//
// The Printer is a parameter and not a package variable, which is the whole
// discipline internal/words exists to enforce; the cost is that the key map
// is rebuilt when the reader changes language, and that is correct — the
// help text is part of the binding.
//
// The descriptions carry no cell budget, unlike the state words and the band
// names. A budget the English at the call site already exceeds is not a
// budget, it is a trap for whoever writes the Spanish, and "take the
// keyboard" is seventeen cells before anybody translates it. The key bar
// drops whole hints rather than truncating them, so the hints are not
// width-constrained the way a column is.
func NewKeys(p *words.Printer) Keys {
	return Keys{
		Up:    binding("↑", p.T("key.up", "up"), "up", "k"),
		Down:  binding("↓", p.T("key.down", "down"), "down", "j"),
		First: binding("g", p.T("key.first", "first"), "g"),
		// End is bound alongside G because one level down "last" means the
		// newest entry in a live log, and End is the key a reader who has
		// never used a modal editor reaches for to get there.
		Last:     binding("G", p.T("key.last", "last"), "G", "end"),
		PageUp:   binding("PgUp", p.T("key.page_up", "page up"), "pgup"),
		PageDown: binding("PgDn", p.T("key.page_down", "page down"), "pgdown"),

		// One key, two things, and it is not an overload: on a band header
		// it expands the band in place and on a row it opens the task, and
		// the cursor is on exactly one of the two.
		Open:    binding("⏎", p.T("key.open", "open"), "enter"),
		Back:    binding("esc", p.T("key.back", "back"), "esc", "left"),
		NextTab: binding("tab", p.T("key.next_tab", "next tab"), "tab"),
		PrevTab: binding("⇧tab", p.T("key.prev_tab", "previous tab"), "shift+tab"),
		// Sideways shares ← with Back, and detailKey matches it first: in a
		// pane that scrolls horizontally ← has to be the scroll. Nothing is
		// lost by it, because esc is the glyph the key bar and the help
		// overlay have always printed for going back.
		Sideways: binding("←→", p.T("key.sideways", "scroll sideways"), "left", "right"),

		// Start opens the dialog that decides what a run will be; it does
		// not write a task. Writing one is N, which opens the compose form
		// — the same screen :new and the board menu land on.
		Start:   binding("n", p.T("key.start", "start a run"), "n"),
		Compose: binding("N", p.T("key.compose", "write a task"), "N"),
		// Run and ChangeFlow belong to that dialog and to nothing else.
		// They share ⏎ and f with nothing on the board, because the dialog
		// takes every keystroke while it is up.
		Run:        binding("⏎", p.T("key.run", "run it"), "enter"),
		ChangeFlow: binding("f", p.T("key.change_flow", "change flow"), "f"),

		// Menu is the right-click's keyboard half: everything that can be
		// done to the thing under the cursor, including what cannot, with
		// the reason. It is bound beside Start because both are about the
		// row, not about a standing setting.
		Menu: binding("m", p.T("key.menu", "menu"), "m"),

		Pause:  binding("p", p.T("key.pause", "pause"), "p"),
		Resume: binding("r", p.T("key.resume", "resume"), "r"),
		Cancel: binding("x", p.T("key.cancel", "cancel"), "x"),

		Take:     binding("t", p.T("key.take", "take the keyboard"), "t"),
		Hand:     binding("h", p.T("key.hand", "hand it back"), "h"),
		Ask:      binding("a", p.T("key.ask", "ask"), "a"),
		MarkRead: binding("d", p.T("key.read", "mark read"), "d"),
		Edit:     binding("o", p.T("key.edit", "open in $EDITOR"), "o"),

		Filter: binding("/", p.T("key.filter", "filter"), "/"),
		// Commands is the palette, and it is bound beside the filter
		// because the two are siblings: both are a line the reader types
		// into, and one narrows what is already on screen while the other
		// reaches everything no key was ever given to.
		Commands:  binding(":", p.T("key.commands", "commands"), ":"),
		Repos:     binding("R", p.T("key.repos", "repositories"), "R"),
		Autopilot: binding("A", p.T("key.autopilot", "autopilot"), "A"),
		Language:  binding("L", p.T("key.language", "language"), "L"),
		Help:      binding("?", p.T("key.help", "help"), "?"),
		Quit:      binding("q", p.T("key.quit", "quit"), "q"),
	}
}

// binding is one gesture: the glyph the bar prints, the description the help
// overlay prints, and the keystrokes that trigger it.
//
// The glyph is separate from the keystrokes because they answer different
// questions — "⏎" is what a reader recognises and "enter" is what the
// terminal sends — and because the glyph is the same in every language while
// the description is not.
func binding(glyph, desc string, keys ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(glyph, desc))
}
