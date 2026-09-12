// Package cheat is the sheet that says what every key does.
//
// It is one screen and one subject: a reader who cannot find a gesture, and
// a window whose whole claim is that everything can be reached from the
// keyboard. What it lists it is handed — the verbs a task offers and the
// tabs the detail screen has — rather than keeping a copy: a sheet with its
// own list stops being true the day one of them changes, which is what the
// verbs line was, naming p, u and s long after s became skip.
package cheat

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the build it names, and the two lists it must not keep its
// own copy of.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	// Version is the build the masthead names, beside the mark. Empty
	// names no build: the sheet still opens, it just says orbit.
	Version string
	// Verbs is what a task offers, each with the sentence ? answers with.
	Verbs []Verb
	// Tabs is the detail screen's own list, in its own order.
	Tabs []Tab
}

// A Verb is one thing that can be done to a task: the key that does it,
// what that key does in words, and the family it belongs to for grouping.
type Verb struct {
	Key    string
	Says   string
	Family string
}

// A Tab is one of the detail screen's tabs, as its menu shows it.
type Tab struct {
	Glyph  string
	Title  string
	Detail string
}

// Out is what the screen asks the window for.
type Out struct {
	// Leave is the reader closing the sheet.
	Leave bool
	// Back is the screen it was opened from, as the window's own number.
	Back int
}

// State is the sheet: how far down it the reader has scrolled, and what they
// were looking at before they opened it.
type State struct {
	back   int
	offset int
}

// Open is the sheet coming up. back is the screen it was opened from, which
// leaving it returns to.
func Open(back int) State { return State{back: back} }

// Key is every key on this screen. There is nothing to choose, so they are
// the ones that scroll and the ones that leave — and ? leaves it too, which
// is the key that opened it.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case key.Matches(msg, e.Keys.Back), key.Matches(msg, e.Keys.Quit),
		key.Matches(msg, e.Keys.Help), key.Matches(msg, e.Keys.Open):
		return State{}, Out{Leave: true, Back: s.back}
	case key.Matches(msg, e.Keys.Up):
		if s.offset > 0 {
			s.offset--
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Down):
		s.offset++
		return s, Out{}
	}

	return s, Out{}
}

// markRows is the program's mark with its rings on: the same thirteen
// cells the command line prints under `orbit version`, art only and ASCII
// only — a mark like the header pill's ◉ reads two cells on fonts that
// render it so, and the line beside it comes out shifted. The language
// test names these rows to skip them: art has no Spanish to differ into.
var markRows = []string{
	"    _____",
	"   /     \\",
	"--(   o   )--",
	"   \\_____/",
}

// Wheel is the mouse doing what the arrows do: d rows down the sheet for a
// positive d, up for a negative one. The top stops it; the bottom is the
// drawing's to clamp, which is the only place that knows how many lines
// there are. Three rows a notch is the window's own wheelRows, passed in
// rather than repeated, so the hand learns one distance.
func (s State) Wheel(d int) State {
	s.offset += d
	if s.offset < 0 {
		s.offset = 0
	}

	return s
}

// View is the sheet drawn.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words

	name := "orbit"
	if e.Version != "" {
		name += " " + e.Version
	}

	// The masthead: the mark with the build's name and the sheet's title
	// beside it, the way `orbit version` draws them. The body paints in
	// the titles' own colour and the rings in the keys', so the block
	// reads as the sheet's and not as a picture hung beside it.
	body := theme.Paint(theme.Live).Render
	rings := theme.Paint(theme.Accent).Render
	head := theme.Paint(theme.Live).Bold(true).Render
	out := []string{
		"",
		body(cells.PadRight(markRows[0], 13)) + "  " + theme.Paint(theme.Accent).Bold(true).Render(name),
		body(cells.PadRight(markRows[1], 13)) + "  " + head(p.T("help.title", "Help and keyboard shortcuts (cheat sheet)")),
		rings("--(   ") + body("o") + rings("   )--"),
		body(markRows[3]),
		// Breathing room under the mark: the subtitle sitting against the
		// rings reads as the logo's slogan, and it is the sheet's.
		"",
		"  " + theme.Paint(theme.Dim).Render(p.T("help.subtitle", "every function can be reached from the keyboard or by clicking it")),
		"",
	}

	renderSection := func(title string, items [][2]string) {
		out = append(out, "  "+theme.Paint(theme.Live).Bold(true).Render(title))

		for _, item := range items {
			k := cells.Pad(item[0], 28, false)
			line := "    " + theme.Paint(theme.Accent).Render(k) + " " + theme.Paint(theme.Dim).Render(item[1])
			out = append(out, cells.Fit(line, w))
		}

		out = append(out, "")
	}

	renderSection(p.T("help.board.title", "📋 1. BOARD AND QUEUES"), [][2]string{
		{"[↑↓] [j/k]", p.T("help.board.move", "Move between tasks and board sections")},
		{p.T("help.board.open_key", "[⏎ enter] / click"), p.T("help.board.open", "Open the selected task, or fold a queue away")},
		{"[n]", p.T("help.board.run", "Start a new run of the selected task")},
		{"[N] (shift+n)", p.T("help.board.write", "Write a new task in the current repository")},
		{"[c]", p.T("help.board.cli", "Open an interactive CLI session")},
		{"[/]", p.T("help.board.filter", "Search and filter as you type (ID, title, repo)")},
		{"[Esc]", p.T("help.board.escape", "Clear the active filters, or go back to the previous view")},
		{p.T("help.board.reset_key", "◉ orbit (click)"), p.T("help.board.reset", "Reset every filter and show the whole board")},
		{p.T("help.board.queue_key", "[📋⚡💬🏁] (click)"), p.T("help.board.queue", "Show only the tasks in that queue")},
	})

	// The verbs are built out of the bindings and the sentences ? answers
	// with, rather than written down a second time here: a sheet that keeps
	// its own copy of them stops being true the day one of them changes,
	// which is what the line below this section was — it named p, u and s
	// for pause, unblock and note long after s became skip.
	//
	// The sentences are whole. They are longer than a column, so they are
	// wrapped here and the continuation rows are given no key of their own.
	//
	// One section per family, in the order a reader meets them: what the
	// keys do first, then what only a command does. A sheet that kept its
	// own list stops being true the day one of them changes — the grouping
	// is the families', read off each row, and never a second list.
	groups := []string{"task", "pr", "board"}
	byFamily := map[string][][2]string{}

	for _, v := range e.Verbs {
		family := v.Family
		if family == "" {
			family = "task"
		}

		for i, line := range cells.Lines(v.Says, max(w-36, 20)) {
			glyph := ""
			if i == 0 {
				glyph = "[" + v.Key + "]"
			}

			byFamily[family] = append(byFamily[family], [2]string{glyph, line})
		}
	}

	out = append(out, "  "+theme.Paint(theme.Live).Bold(true).Render(p.T("help.verbs.title", "🎛️ 2. WHAT YOU CAN DO TO A TASK")))
	familyTitle := map[string]string{
		"task":  p.T("help.verbs.task", "task"),
		"pr":    p.T("help.verbs.pr", "pr"),
		"board": p.T("help.verbs.board", "board"),
	}

	for _, family := range groups {
		if len(byFamily[family]) == 0 {
			continue
		}

		out = append(out, "    "+theme.Paint(theme.Dim).Render(familyTitle[family]))

		for _, item := range byFamily[family] {
			k := cells.Pad(item[0], 28, false)
			line := "    " + theme.Paint(theme.Accent).Render(k) + " " + theme.Paint(theme.Dim).Render(item[1])
			out = append(out, cells.Fit(line, w))
		}
	}

	out = append(out, "")

	renderSection(p.T("help.live.title", "⚡ 3. LIVE CONTROL AND SETTINGS"), [][2]string{
		{p.T("help.live.autopilot_key", "[A] / ⚡ click"), p.T("help.live.autopilot", "Toggle autopilot: tasks in to do start on their own")},
		{p.T("help.live.engine_key", "[M] / 🧠 click"), p.T("help.live.engine", "Engine dial: claude, codex, opencode, effort and thinking")},
		{"[S]", p.T("help.live.supervisor", "The supervisor: what it has said about the board, and the line you answer it on")},
		{p.T("help.live.repos_key", "[R] / 📦 click"), p.T("help.live.repos", "Pick which of the connected repositories the board shows")},
		{p.T("help.live.quota_key", "[Q] / ⏳ click"), p.T("help.live.quota", "What is left of each engine's windows, and when each comes back")},
		{p.T("help.live.lang_key", "🌐 ES / EN (click)"), p.T("help.live.lang", "Switch the language of the whole window, live")},
	})

	// The tab rows are the tab list itself, not a copy of it: same key, same
	// name, same description the tab menu shows. The list this replaces was
	// written out by hand, and by the time it was read it announced eleven
	// tabs and then named eight of them — refused, artifacts and notes had
	// been added to the detail screen and never to the cheat sheet.
	tabs := make([][2]string, 0, len(e.Tabs)+2)
	for _, t := range e.Tabs {
		tabs = append(tabs, [2]string{t.Glyph + " " + t.Title, t.Detail})
	}

	tabsTitle := p.P("help.tabs.title", len(tabs),
		"🔍 4. TASK DETAIL ({n} tab)", "🔍 4. TASK DETAIL ({n} tabs)")

	tabs = append(tabs,
		[2]string{"[Tab / shift+tab]", p.T("help.tabs.cycle", "Next tab / previous tab")},
		[2]string{"[r] / [s] / [a]", p.T("help.tabs.control", "Let a stopped run go / skip the phase it waits at / write it a note")},
	)
	renderSection(tabsTitle, tabs)

	renderSection(p.T("help.global.title", "⌨️ 5. GLOBAL COMMANDS"), [][2]string{
		{"[:]", p.T("help.global.palette", "Open the command palette (orbit new, flows, set...)")},
		{"[?]", p.T("help.global.help", "Ask what a key does, then press it — ? again is this whole sheet")},
		{"[q]", p.T("help.global.quit", "Leave Orbit")},
	})

	waysOut := p.T("help.ways_out", "{up_down} scroll · {back} back",
		about("up_down", e.Keys.Up.Help().Key+e.Keys.Down.Help().Key),
		about("back", e.Keys.Back.Help().Key))
	out = append(out, cells.Fit("  "+theme.Paint(theme.Dim).Render(waysOut), w))

	if s.offset > 0 {
		if s.offset >= len(out) {
			s.offset = len(out) - 1
		}

		out = out[s.offset:]
	}

	return cells.Fill(out, h)
}
