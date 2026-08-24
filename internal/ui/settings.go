package ui

// The settings screen: every row of the settings table on screen, with its
// current value, selectable options list, and description.

import (
	"bytes"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type settingsState struct {
	sel     int
	editing bool
	typed   string
}

type settingRow struct {
	key     string
	val     string
	options []string
	about   string
}

func (m Model) openSettings() Model {
	m.screen = screenSettings
	m.settings = settingsState{}
	return m
}

func (m Model) abandonSettings() Model {
	m.settings = settingsState{}
	m.screen = screenList
	return m
}

func (m Model) settingRowsList() []settingRow {
	p := m.opts.Words
	s := m.opts.Settings
	if s == nil {
		return nil
	}
	autopilotStr := "off"
	if s.Autopilot() {
		autopilotStr = "on"
	}
	langVal := orDef(s.Language(), "en")
	engineVal := orDef(s.Engine(), "claude")
	modelVal := orDef(s.Model(), "opus")
	flowVal := orDef(s.Flow(), "task")
	themeVal := orDef(s.Theme(), "monokai")

	return []settingRow{
		{key: "language", val: langVal, options: []string{"en", "es"}, about: p.T("setting.language", "the language orbit speaks")},
		{key: "autopilot", val: autopilotStr, options: []string{"off", "on"}, about: p.T("setting.autopilot", "whether a run walks its whole flow without stopping")},
		{key: "unread-cap", val: strconv.Itoa(s.UnreadCap()), options: []string{"0", "3", "5", "10", "20"}, about: p.T("setting.unread_cap", "how many finished tasks may sit unread before nothing new starts")},
		{key: "engine", val: engineVal, options: []string{"claude", "codex", "opencode"}, about: p.T("setting.engine", "the engine a task runs on when it names none")},
		{key: "model", val: modelVal, options: []string{"opus", "sonnet", "haiku", "o3-mini", "o1", "deepseek-r1", "qwen-2.5-coder"}, about: p.T("setting.model", "the model a phase asks for when it names none")},
		{key: "flow", val: flowVal, options: []string{"task", "quick", "careful"}, about: p.T("setting.flow", "the flow a new task is written against")},
		{key: "theme", val: themeVal, options: AvailableThemes(), about: p.T("setting.theme", "the visual color theme for the window")},
	}
}

func orDef(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (m Model) settingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.settingRowsList()
	if len(rows) == 0 {
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
			return m.abandonSettings(), nil
		}
		return m, nil
	}

	if m.settings.editing {
		switch {
		case key.Matches(msg, m.keys.Back):
			m.settings.editing = false
			m.settings.typed = ""
			return m, nil
		case key.Matches(msg, m.keys.Open):
			return m.settingsSubmit()
		case msg.Code == tea.KeyBackspace:
			m.settings.typed = trimLastRune(m.settings.typed)
			return m, nil
		}
		if msg.Text != "" {
			m.settings.typed += msg.Text
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		return m.abandonSettings(), nil
	case key.Matches(msg, m.keys.Up), msg.Text == "k":
		m.settings.sel--
		if m.settings.sel < 0 {
			m.settings.sel = len(rows) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Down), msg.Text == "j":
		m.settings.sel++
		if m.settings.sel >= len(rows) {
			m.settings.sel = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.Open), msg.Text == " ", msg.Code == tea.KeyRight, msg.Text == "l":
		return m.cycleSetting(1)
	case msg.Code == tea.KeyLeft, msg.Text == "h":
		return m.cycleSetting(-1)
	case msg.Text == "e":
		m.settings.editing = true
		m.settings.typed = rows[m.settings.sel].val
		return m, nil
	}
	return m, nil
}

func (m Model) cycleSetting(delta int) (tea.Model, tea.Cmd) {
	rows := m.settingRowsList()
	if m.settings.sel < 0 || m.settings.sel >= len(rows) {
		return m, nil
	}
	r := rows[m.settings.sel]
	if len(r.options) == 0 {
		return m, nil
	}
	idx := 0
	for i, opt := range r.options {
		if opt == r.val {
			idx = i
			break
		}
	}
	nextIdx := (idx + delta) % len(r.options)
	if nextIdx < 0 {
		nextIdx += len(r.options)
	}
	return m.applySetting(r.key, r.options[nextIdx])
}

func (m Model) settingsSubmit() (tea.Model, tea.Cmd) {
	rows := m.settingRowsList()
	if m.settings.sel < 0 || m.settings.sel >= len(rows) {
		m.settings.editing = false
		return m, nil
	}
	keyName := rows[m.settings.sel].key
	val := strings.TrimSpace(m.settings.typed)
	m.settings.editing = false
	m.settings.typed = ""
	return m.applySetting(keyName, val)
}

func (m Model) applySetting(keyName, val string) (tea.Model, tea.Cmd) {
	s := m.opts.Settings
	if s != nil {
		switch keyName {
		case "language":
			_ = s.SetLanguage(val)
		case "autopilot":
			_ = s.SetAutopilot(val == "on")
		case "unread-cap":
			if n, err := strconv.Atoi(val); err == nil {
				_ = s.SetUnreadCap(n)
			}
		case "engine":
			_ = s.SetEngine(val)
		case "model":
			_ = s.SetModel(val)
		case "flow":
			_ = s.SetFlow(val)
		case "theme":
			_ = s.SetTheme(val)
			SetCurrentTheme(val)
		}
	}
	if m.opts.Do != nil {
		var buf bytes.Buffer
		_ = m.opts.Do("set", []string{keyName, val}, &buf)
	}
	if keyName == "language" {
		return m.say("language changed to " + val), func() tea.Msg { return languageMsg{Lang: val} }
	}
	return m.say(keyName + " is now " + val), nil
}

func (m Model) settingsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	rows := m.settingRowsList()
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("settings.title", "Settings")),
		"  " + Paint(Dim).Render(p.T("settings.subtitle", "changes take effect immediately")),
		"",
	}

	for i, r := range rows {
		mark := "    "
		if i == m.settings.sel {
			mark = "  " + Paint(Accent).Render("▸ ")
		}

		var optViews []string
		if i == m.settings.sel && m.settings.editing {
			optViews = append(optViews, Paint(Accent).Render(m.settings.typed)+Paint(Sel).Render(" "))
		} else {
			for _, opt := range r.options {
				if opt == r.val {
					optViews = append(optViews, Paint(Sel).Render(" "+opt+" "))
				} else {
					optViews = append(optViews, Paint(Dim).Render(opt))
				}
			}
		}
		optsFormatted := strings.Join(optViews, " ")

		keyCol := padRight(r.key, 12)
		line := mark + Paint(Accent).Render(keyCol) + "  " + padRight(optsFormatted, 40) + "  " + Paint(Dim).Render(r.about)
		out = append(out, fit(line, w))
	}

	waysOut := p.T("settings.ways_out", "{open} edit · {up_down} move · {back} back",
		about("open", m.keys.Open.Help().Key),
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	if m.settings.editing {
		waysOut = p.T("settings.ways_out_edit", "{open} save · {back} cancel",
			about("open", m.keys.Open.Help().Key),
			about("back", m.keys.Back.Help().Key))
	}
	out = append(out, "", fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
