package ui

// The form a task is written into, either manually or by importing from an issue tracker URL.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/tracker"
)

const (
	composeTabManual = 0
	composeTabURL    = 1
)

const (
	composeRepo = iota
	composeFlow
	composeEngine
	composeModel
	composeThinking
	composeEffort
	composeID
	composeText
	composeFields
)

const (
	composeURL = iota
	composeURLRepo
	composeURLFlow
	composeURLEngine
	composeURLModel
	composeURLThinking
	composeURLEffort
	composeURLFields
)

type composeState struct {
	tab         int // composeTabManual or composeTabURL
	field       int // active field index within current tab
	repo        string
	id          string
	text        string
	url         string
	repos       []repoItem
	repoIdx     int
	parsedIssue *tracker.Issue

	flows     []string
	flowIdx   int
	engines   []string
	engineIdx int
	// models and efforts are the ids these dials hold and modelLabels and
	// effortLabels are what those positions are drawn as. The two differ
	// for opencode, whose model ids are provider-qualified.
	models       []string
	modelLabels  []string
	modelIdx     int
	thinkings    []string
	thinkingIdx  int
	efforts      []string
	effortLabels []string
	effortIdx    int
}

// modelLabel and effortLabel are what the option at i is drawn as.
func (c composeState) modelLabel(i int) string {
	return dialLabel(c.models, c.modelLabels, i)
}

func (c composeState) effortLabel(i int) string {
	return dialLabel(c.efforts, c.effortLabels, i)
}

// openCompose brings the form up with the repository defaulted to the current task's repo.
func (m Model) openCompose() Model {
	m.screen = screenCompose
	repos := m.collectRepos()

	selectedRepo := ""
	if r, ok := m.selected(); ok && !r.head {
		selectedRepo = r.task.Repo
	}

	repoIdx := 0

	for i, r := range repos {
		if strings.EqualFold(r.name, selectedRepo) {
			repoIdx = i
			break
		}
	}

	initialRepo := ""
	if len(repos) > 0 {
		initialRepo = repos[repoIdx].name
	} else if selectedRepo != "" {
		initialRepo = selectedRepo
	}

	flowsListed := flow.List(m.opts.Flows)

	var flows []string
	for _, f := range flowsListed {
		flows = append(flows, f.Name)
	}

	if len(flows) == 0 {
		flows = []string{"task", "quick", "careful"}
	}

	// The engines and their two dials are the build's, not this form's:
	// see engine_table.go. This form used to carry a table of its own, and
	// it offered opencode a model called llama-3.3 — one no opencode has
	// ever answered to, so a task composed with it could not run.
	engines := m.engineNames()
	thinkings := []string{"adaptive", "off", "4000", "8000", "max"}

	m.compose = composeState{
		tab:       composeTabManual,
		repo:      initialRepo,
		repos:     repos,
		repoIdx:   repoIdx,
		flows:     flows,
		engines:   engines,
		thinkings: thinkings,
	}

	m = m.chooseComposeEngine(0)

	return m
}

func (m Model) abandonCompose() Model {
	m.compose = composeState{}
	m.screen = screenList

	return m
}

func (m Model) composeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back):
		return m.abandonCompose(), nil
	case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Open):
		if msg.Mod&tea.ModCtrl != 0 {
			return m.composeSubmit(true)
		}

		if msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0 {
			if m.compose.tab == composeTabManual && m.compose.field == composeText {
				m.compose.text += "\n"
				return m, nil
			}
		}

		return m.composeNext(false)
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return m.composeSubmit(true)
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if clip := readClipboard(); clip != "" {
			return m.paste(clip), nil
		}

		return m, nil
	case msg.Text == "+":
		if m.isComposeFlowField() {
			return m.openFlows(), nil
		}
	case msg.Text == "i" || msg.Text == "I":
		if m.isComposeFlowField() {
			return m.openFlowPreview(m.compose.chosenFlow()), nil
		}
	case (msg.Text == "a" || msg.Text == "A" || key.Matches(msg, m.keys.Autopilot)) && m.isPillField():
		return m.autopilot()
	case msg.Code == tea.KeyUp || key.Matches(msg, m.keys.Up):
		return m.composeMove(-1), nil
	case msg.Code == tea.KeyDown || key.Matches(msg, m.keys.Down):
		return m.composeMove(1), nil
	case msg.Code == tea.KeyLeft:
		return m.handleComposeLeft(), nil
	case msg.Code == tea.KeyRight:
		return m.handleComposeRight(), nil
	case key.Matches(msg, m.keys.PrevTab):
		return m.composeMove(-1), nil
	case msg.Code == tea.KeyTab || key.Matches(msg, m.keys.NextTab):
		if msg.Mod&tea.ModShift != 0 {
			return m.composeMove(-1), nil
		}

		return m.composeTab(1), nil
	case msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete:
		m.compose.set(trimLastRune(m.compose.get()))
		m.onComposeChanged()

		return m, nil
	}

	if (msg.Text == "1" || msg.Text == "2") && (m.isPillField() || m.compose.get() == "") {
		if msg.Text == "1" {
			m.compose.tab = composeTabManual
		} else {
			m.compose.tab = composeTabURL
		}

		m.compose.field = 0

		return m, nil
	}

	if msg.Text != "" && !m.isPillField() {
		m.compose.set(m.compose.get() + msg.Text)
		m.onComposeChanged()

		return m, nil
	}

	return m, nil
}

func (m *Model) onComposeChanged() {
	if m.compose.tab == composeTabURL && m.compose.field == composeURL {
		raw := strings.TrimSpace(m.compose.url)
		if raw != "" {
			if issue, err := tracker.Parse(raw); err == nil {
				m.compose.parsedIssue = &issue

				m.compose.id = issue.ID
				if issue.Title != "" {
					m.compose.text = issue.Title
				}
			} else {
				m.compose.parsedIssue = nil
			}
		} else {
			m.compose.parsedIssue = nil
		}
	} else if m.compose.tab == composeTabManual {
		cur := strings.TrimSpace(m.compose.get())
		if strings.HasPrefix(cur, "http://") || strings.HasPrefix(cur, "https://") ||
			strings.HasPrefix(cur, "linear.app/") {
			if issue, err := tracker.Parse(cur); err == nil {
				m.compose.tab = composeTabURL
				m.compose.url = cur
				m.compose.parsedIssue = &issue

				m.compose.id = issue.ID
				if issue.Title != "" {
					m.compose.text = issue.Title
				}
			}
		}
	}
}

func (c *composeState) get() string {
	if c.tab == composeTabURL {
		if c.field == composeURL {
			return c.url
		}

		return ""
	}

	switch c.field {
	case composeID:
		return c.id
	case composeText:
		return c.text
	}

	return ""
}

func (c *composeState) set(v string) {
	if c.tab == composeTabURL {
		if c.field == composeURL {
			c.url = v
		}

		return
	}

	switch c.field {
	case composeID:
		c.id = v
	case composeText:
		c.text = v
	}
}

func (m Model) composeMove(d int) Model {
	maxFields := composeFields
	if m.compose.tab == composeTabURL {
		maxFields = composeURLFields
	}

	m.compose.field += d
	if m.compose.field < 0 {
		m.compose.field = 0
	}

	if m.compose.field >= maxFields {
		m.compose.field = maxFields - 1
	}

	return m
}

func (m Model) composeTab(d int) Model {
	return m.composeMove(d)
}

func (m Model) composeNext(startNow bool) (tea.Model, tea.Cmd) {
	limit := composeText
	if m.compose.tab == composeTabURL {
		limit = composeURLEffort
	}

	if m.compose.field < limit {
		m.compose.field++
		return m, nil
	}

	return m.composeSubmit(startNow)
}
