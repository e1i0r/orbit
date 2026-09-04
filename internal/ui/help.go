package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type helpState struct {
	prevScreen screen
	offset     int
}

func (m Model) openHelp() Model {
	prev := m.screen
	if prev == screenHelp {
		prev = screenList
	}

	m.screen = screenHelp
	m.help = helpState{prevScreen: prev}

	return m
}

func (m Model) abandonHelp() Model {
	target := m.help.prevScreen
	if target == screenHelp {
		target = screenList
	}

	m.help = helpState{}
	m.screen = target

	return m
}

func (m Model) helpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Open):
		return m.abandonHelp(), nil
	case key.Matches(msg, m.keys.Up):
		if m.help.offset > 0 {
			m.help.offset--
		}

		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.help.offset++
		return m, nil
	}

	return m, nil
}

func (m Model) helpRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("help.title", "Help and keyboard shortcuts (cheat sheet)")),
		"  " + Paint(Dim).Render(p.T("help.subtitle", "every function can be reached from the keyboard or by clicking it")),
		"",
	}

	renderSection := func(title string, items [][2]string) {
		out = append(out, "  "+Paint(Live).Bold(true).Render(title))

		for _, item := range items {
			k := pad(item[0], 28, false)
			line := "    " + Paint(Accent).Render(k) + " " + Paint(Dim).Render(item[1])
			out = append(out, fit(line, w))
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
	bindings := m.keys.taskVerbs()

	verbs := make([][2]string, 0, len(bindings))
	for _, b := range bindings {
		for i, line := range splitIntoLines(m.meaning(firstKey(b)), max(w-36, 20)) {
			glyph := ""
			if i == 0 {
				glyph = "[" + b.Help().Key + "]"
			}

			verbs = append(verbs, [2]string{glyph, line})
		}
	}

	renderSection(p.T("help.verbs.title", "🎛️ 2. WHAT YOU CAN DO TO A TASK"), verbs)

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
	tabs := make([][2]string, 0, len(m.tabNames())+2)
	for _, e := range m.tabMenuEntries() {
		tabs = append(tabs, [2]string{e.glyph + " " + e.title, e.detail})
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
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))

	if m.help.offset > 0 {
		if m.help.offset >= len(out) {
			m.help.offset = len(out) - 1
		}

		out = out[m.help.offset:]
	}

	return fill(out, h)
}
