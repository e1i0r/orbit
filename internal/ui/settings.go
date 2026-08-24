package ui

// The settings screen: every row of the settings table on screen, with its
// current value and description, allowing changing them without leaving Orbit.

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
	key   string
	val   string
	about string
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
	return []settingRow{
		{
			key:   "language",
			val:   s.Language(),
			about: p.T("setting.language", "the language orbit speaks"),
		},
		{
			key:   "autopilot",
			val:   autopilotStr,
			about: p.T("setting.autopilot", "whether a run walks its whole flow without stopping"),
		},
		{
			key:   "unread-cap",
			val:   strconv.Itoa(s.UnreadCap()),
			about: p.T("setting.unread_cap", "how many finished tasks may sit unread before nothing new starts"),
		},
		{
			key:   "engine",
			val:   s.Engine(),
			about: p.T("setting.engine", "the engine a task runs on when it names none"),
		},
		{
			key:   "model",
			val:   s.Model(),
			about: p.T("setting.model", "the model a phase asks for when it names none"),
		},
		{
			key:   "flow",
			val:   s.Flow(),
			about: p.T("setting.flow", "the flow a new task is written against"),
		},
	}
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
	case key.Matches(msg, m.keys.Up):
		m.settings.sel--
		if m.settings.sel < 0 {
			m.settings.sel = len(rows) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.settings.sel++
		if m.settings.sel >= len(rows) {
			m.settings.sel = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.Open), msg.Text == " ":
		row := rows[m.settings.sel]
		if row.key == "autopilot" {
			newVal := "on"
			if row.val == "on" {
				newVal = "off"
			}
			return m.applySetting("autopilot", newVal)
		}
		m.settings.editing = true
		m.settings.typed = row.val
		return m, nil
	}
	return m, nil
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
	if m.opts.Do == nil {
		return m.say("this window was opened without a way to change settings"), nil
	}
	var buf bytes.Buffer
	err := m.opts.Do("set", []string{keyName, val}, &buf)
	if err != nil {
		return m.say(err.Error()), nil
	}
	msg := strings.TrimSpace(buf.String())
	if keyName == "language" {
		return m.say(msg), func() tea.Msg { return languageMsg{Lang: val} }
	}
	return m.say(msg), nil
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
		mark := strings.Repeat(" ", gutter)
		if i == m.settings.sel {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}
		valStr := r.val
		if valStr == "" {
			valStr = "—"
		}
		valRole := Accent
		if i == m.settings.sel && m.settings.editing {
			valStr = m.settings.typed
			valRole = Accent
		}
		valRendered := Paint(valRole).Render(valStr)
		if i == m.settings.sel && m.settings.editing {
			valRendered += Paint(Sel).Render(" ")
		}

		line := mark + Paint(Accent).Render(padRight(r.key, 12)) + "  " + padRight(valRendered, 16) + "  " + Paint(Dim).Render(r.about)
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
