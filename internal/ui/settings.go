package ui

// The settings screen: every row of the settings table on screen, with its
// current value, selectable options list, and description.

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

type settingsState struct {
	sel     int
	editing bool
	typed   string
	// flows is what the flow dial offers, read once when the screen opens
	// rather than while it draws.
	//
	// settingRowsList is called from View, from every keypress and from
	// every mouse event, so asking the flows directory inside it would be
	// one os.ReadDir per frame — and, worse, two readings taken at two
	// moments deciding the same dial. flows.go tells that story at length;
	// this is the same fix, made before it happened again.
	flows []string
}

type settingRow struct {
	key     string
	val     string
	options []string
	// labels is what each option is drawn as, when that is not the option
	// itself. Only the model dial needs it — opencode's models are stored
	// provider-qualified and shown without the provider — and every other
	// row leaves it nil.
	labels []string
	about  string
}

// label is what the option at i is drawn as.
func (r settingRow) label(i int) string {
	return dialLabel(r.options, r.labels, i)
}

func (m Model) openSettings() Model {
	m.screen = screenSettings
	m.settings = settingsState{flows: flow.Names(m.opts.Flows)}

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
	// The engines, the models and the efforts are the build's, not this
	// screen's: see engine_table.go. A settings file naming an engine this
	// build does not have keeps its own word rather than being quietly
	// moved onto the first one — that is the reader's setting, and the
	// dial simply has no pill lit.
	engines := m.engineNames()
	engineVal := orDef(s.Engine(), first(engines))
	flowVal := orDef(s.Flow(), flow.Default)
	themeVal := orDef(s.Theme(), defaultTheme)

	// The flows are the build's and the reader's, not this screen's. Names
	// written out by hand on this dial leave every flow they do not list —
	// the ones shipped inside the binary and the ones the reader wrote for
	// themselves — impossible to choose here at all.
	flows := m.settings.flows
	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	models, modelLabels := m.modelsFor(engineVal)

	modelVal := s.Model()
	if !slices.Contains(models, modelVal) && len(models) > 0 {
		modelVal = models[0]
	}

	efforts, effortLabels := m.effortsFor(engineVal)

	effortVal := m.knobs.Effort
	if !slices.Contains(efforts, effortVal) && len(efforts) > 0 {
		effortVal = efforts[0]
	}

	thinkingVal := orDef(m.knobs.Thinking, "adaptive")

	return []settingRow{
		{key: "language", val: langVal, options: []string{"en", "es"}, about: p.T("setting.language", "the language orbit speaks")},
		{key: "autopilot", val: autopilotStr, options: []string{"off", "on"}, about: p.T("setting.autopilot", "whether a run walks its whole flow without stopping")},
		{key: "unread-cap", val: strconv.Itoa(s.UnreadCap()), options: []string{"0", "3", "5", "10", "20"}, about: p.T("setting.unread_cap", "how many finished tasks may sit unread before nothing new starts")},
		{key: "engine", val: engineVal, options: engines, about: p.T("setting.engine", "the engine a task runs on when it names none")},
		{key: "model", val: modelVal, options: models, labels: modelLabels, about: p.T("setting.model", "the model a phase asks for when it names none")},
		{key: "effort", val: effortVal, options: efforts, labels: effortLabels, about: p.T("setting.effort", "the default reasoning effort level for engine sessions")},
		{key: "thinking", val: thinkingVal, options: []string{"adaptive", "on", "off"}, about: p.T("setting.thinking", "whether extended thinking mode is enabled for the engine")},
		{key: "flow", val: flowVal, options: flows, about: p.T("setting.flow", "the flow a new task is written against")},
		{key: "theme", val: themeVal, options: AvailableThemes(), about: p.T("setting.theme", "the visual color theme for the window")},
	}
}

// first is the head of a list, or "" when there is none.
func first(list []string) string {
	if len(list) == 0 {
		return ""
	}

	return list[0]
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

func (m Model) settingsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words
	rows := m.settingRowsList()
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("settings.title", "Settings")),
		"  " + Paint(Dim).Render(p.T("settings.subtitle", "changes take effect immediately")),
		"",
	}

	for i, r := range rows {
		isSelected := i == m.settings.sel
		mark := "    "
		keyRole := Accent
		descRole := Dim

		if isSelected {
			mark = "  " + Paint(Live).Bold(true).Render("▸ ")
			keyRole = Live
			descRole = Accent
		}

		var optViews []string
		if isSelected && m.settings.editing {
			optViews = append(optViews, Paint(Accent).Render(m.settings.typed)+Paint(Sel).Render(" "))
		} else {
			for i, opt := range r.options {
				if opt == r.val {
					optViews = append(optViews, Paint(Sel).Bold(true).Render(" ● "+r.label(i)+" "))
				} else {
					optViews = append(optViews, Paint(Dim).Render(" "+r.label(i)+" "))
				}
			}
		}

		optsFormatted := strings.Join(optViews, " ")

		keyCol := padRight(r.key, 14)
		headerLine := mark + Paint(keyRole).Bold(true).Render(keyCol) + "  " + optsFormatted
		descLine := "      " + Paint(descRole).Render(r.about)

		out = append(out, fit(headerLine, w), fit(descLine, w), "")
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

	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))

	return fill(out, h)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}

	return s + strings.Repeat(" ", width-w)
}
