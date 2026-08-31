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

// The fields of the form. Which engine, which model, how much thinking and
// how much effort are not among them: the seat already answers those, and a
// phase of a flow answers them again for the work it runs. A form that asks
// a third time is a third answer to keep in sync with the other two.
const (
	composeRepo = iota
	composeFlow
	composeID
	composeText
	composeFields
)

const (
	composeURL = iota
	composeURLRepo
	composeURLFlow
	composeURLFields
)

type composeState struct {
	tab         int // composeTabManual or composeTabURL
	field       int // active field index within current tab
	repo        string
	id          input
	text        input
	url         input
	repos       []repoItem
	repoIdx     int
	parsedIssue *tracker.Issue

	flows   []string
	flowIdx int
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
		flows = flow.BuiltinNames()
	}

	m.compose = composeState{
		tab:     composeTabManual,
		repo:    initialRepo,
		repos:   repos,
		repoIdx: repoIdx,
		flows:   flows,
	}

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
				m.compose.text.insert("\n")
				return m, nil
			}
		}

		return m.composeNext(false)
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return m.composeSubmit(true)
	case (msg.Code == 'a' || msg.Code == 'A') && msg.Mod&tea.ModCtrl != 0:
		return m.composeCaret((*input).selectAll), nil
	case (msg.Code == 'c' || msg.Code == 'C') && msg.Mod&tea.ModCtrl != 0:
		return m.composeCopy(false), nil
	case (msg.Code == 'x' || msg.Code == 'X') && msg.Mod&tea.ModCtrl != 0:
		return m.composeCopy(true), nil
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
		return m.composeVertical(-1, msg.Mod), nil
	case msg.Code == tea.KeyDown || key.Matches(msg, m.keys.Down):
		return m.composeVertical(1, msg.Mod), nil
	case msg.Code == tea.KeyLeft:
		return m.composeArrow(-1, msg.Mod), nil
	case msg.Code == tea.KeyRight:
		return m.composeArrow(1, msg.Mod), nil
	case key.Matches(msg, m.keys.PrevTab):
		return m.composeMove(-1), nil
	case msg.Code == tea.KeyTab || key.Matches(msg, m.keys.NextTab):
		if msg.Mod&tea.ModShift != 0 {
			return m.composeMove(-1), nil
		}

		return m.composeTab(1), nil
	case msg.Code == tea.KeyBackspace:
		return m.composeEdit(func(in *input) { in.backspace() }), nil
	case msg.Code == tea.KeyDelete:
		return m.composeEdit(func(in *input) { in.deleteForward() }), nil
	case msg.Code == tea.KeyHome:
		return m.composeJump((*input).lineStart, msg.Mod), nil
	case msg.Code == tea.KeyEnd:
		return m.composeJump((*input).lineEnd, msg.Mod), nil
	}

	if (msg.Text == "1" || msg.Text == "2") && (m.isPillField() || m.compose.typed() == "") {
		if msg.Text == "1" {
			m.compose.tab = composeTabManual
		} else {
			m.compose.tab = composeTabURL
		}

		m.compose.field = 0

		return m, nil
	}

	if msg.Text != "" && !m.isPillField() {
		return m.composeEdit(func(in *input) { in.insert(msg.Text) }), nil
	}

	return m, nil
}

func (m *Model) onComposeChanged() {
	if m.compose.tab == composeTabURL && m.compose.field == composeURL {
		raw := strings.TrimSpace(m.compose.url.String())
		if raw != "" {
			if issue, err := tracker.Parse(raw); err == nil {
				m.compose.parsedIssue = &issue

				m.compose.id.setValue(issue.ID)

				if issue.Title != "" {
					m.compose.text.setValue(issue.Title)
				}
			} else {
				m.compose.parsedIssue = nil
			}
		} else {
			m.compose.parsedIssue = nil
		}
	} else if m.compose.tab == composeTabManual {
		cur := strings.TrimSpace(m.compose.typed())
		if strings.HasPrefix(cur, "http://") || strings.HasPrefix(cur, "https://") ||
			strings.HasPrefix(cur, "linear.app/") {
			if issue, err := tracker.Parse(cur); err == nil {
				m.compose.tab = composeTabURL
				m.compose.url.setValue(cur)
				m.compose.parsedIssue = &issue

				m.compose.id.setValue(issue.ID)

				if issue.Title != "" {
					m.compose.text.setValue(issue.Title)
				}
			}
		}
	}
}

// active is the field being typed into, or nothing when the form is on a
// row of pills. Every key that writes, deletes or moves a caret goes
// through it, so which field a keystroke lands in is answered once.
func (c *composeState) active() *input {
	if c.tab == composeTabURL {
		if c.field == composeURL {
			return &c.url
		}

		return nil
	}

	switch c.field {
	case composeID:
		return &c.id
	case composeText:
		return &c.text
	}

	return nil
}

// typed is what the field being typed into holds, for the screens that only
// want to read it.
func (c *composeState) typed() string {
	if in := c.active(); in != nil {
		return in.String()
	}

	return ""
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
		limit = composeURLFlow
	}

	if m.compose.field < limit {
		m.compose.field++
		return m, nil
	}

	return m.composeSubmit(startNow)
}
