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

	renderSection(p.T("help.live.title", "⚡ 2. LIVE CONTROL AND SETTINGS"), [][2]string{
		{p.T("help.live.autopilot_key", "[A] / ⚡ click"), p.T("help.live.autopilot", "Toggle autopilot: tasks in to do start on their own")},
		{p.T("help.live.engine_key", "[M] / 🧠 click"), p.T("help.live.engine", "Engine dial: claude, codex, opencode, effort and thinking")},
		{"[S]", p.T("help.live.settings", "Settings screen: language, theme, limits")},
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
		"🔍 3. TASK DETAIL ({n} tab)", "🔍 3. TASK DETAIL ({n} tabs)")

	tabs = append(tabs,
		[2]string{"[Tab / shift+tab]", p.T("help.tabs.cycle", "Next tab / previous tab")},
		[2]string{"[p] / [u] / [s]", p.T("help.tabs.control", "Pause / unblock / add an operator note")},
	)
	renderSection(tabsTitle, tabs)

	renderSection(p.T("help.global.title", "⌨️ 4. GLOBAL COMMANDS"), [][2]string{
		{"[:]", p.T("help.global.palette", "Open the command palette (orbit new, flows, set...)")},
		{"[?]", p.T("help.global.help", "Open or close this help window")},
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
