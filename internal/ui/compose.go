package ui

// The form a task is written into, either manually or by importing from an issue tracker URL.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"strings"

	"github.com/e1i0r/orbit/internal/tracker"
)

const (
	composeTabManual = 0
	composeTabURL    = 1
)

const (
	composeRepo = iota
	composeID
	composeText
	composeFields
)

const (
	composeURL = iota
	composeURLRepo
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

	m.compose = composeState{
		tab:     composeTabManual,
		repo:    initialRepo,
		repos:   repos,
		repoIdx: repoIdx,
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
		return m.composeNext(false)
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return m.composeSubmit(true)
	case msg.Code == tea.KeyUp || key.Matches(msg, m.keys.Up):
		return m.composeMove(-1), nil
	case msg.Code == tea.KeyDown || key.Matches(msg, m.keys.Down):
		return m.composeMove(1), nil
	case msg.Code == tea.KeyLeft:
		if m.isComposeRepoField() {
			return m.cycleComposeRepo(-1), nil
		}
	case msg.Code == tea.KeyRight:
		if m.isComposeRepoField() {
			return m.cycleComposeRepo(1), nil
		}
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

	if (msg.Text == "1" || msg.Text == "2") && (m.isComposeRepoField() || m.compose.get() == "") {
		if msg.Text == "1" {
			m.compose.tab = composeTabManual
		} else {
			m.compose.tab = composeTabURL
		}
		m.compose.field = 0
		return m, nil
	}

	if msg.Text != "" {
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
		if strings.HasPrefix(cur, "http://") || strings.HasPrefix(cur, "https://") || strings.HasPrefix(cur, "linear.app/") {
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

func (m Model) isComposeRepoField() bool {
	return (m.compose.tab == composeTabManual && m.compose.field == composeRepo) ||
		(m.compose.tab == composeTabURL && m.compose.field == composeURLRepo)
}

func (m Model) cycleComposeRepo(d int) Model {
	if len(m.compose.repos) == 0 {
		return m
	}
	m.compose.repoIdx = (m.compose.repoIdx + d + len(m.compose.repos)) % len(m.compose.repos)
	m.compose.repo = m.compose.repos[m.compose.repoIdx].name
	return m
}

func (c *composeState) get() string {
	if c.tab == composeTabURL {
		if c.field == composeURL {
			return c.url
		}
		return c.repo
	}
	switch c.field {
	case composeRepo:
		return c.repo
	case composeID:
		return c.id
	}
	return c.text
}

func (c *composeState) set(v string) {
	if c.tab == composeTabURL {
		if c.field == composeURL {
			c.url = v
		} else {
			c.repo = v
		}
		return
	}
	switch c.field {
	case composeRepo:
		c.repo = v
	case composeID:
		c.id = v
	default:
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
	if d > 0 && m.compose.field == composeRepo && m.compose.tab == composeTabManual {
		if completed, done := m.composeComplete(); done {
			return completed
		}
	}
	return m.composeMove(d)
}

func (m Model) composeComplete() (Model, bool) {
	prefix := strings.ToLower(strings.TrimSpace(m.compose.repo))
	if prefix == "" {
		return m, false
	}
	for _, t := range m.board.Tasks {
		if strings.HasPrefix(strings.ToLower(t.Repo), prefix) && !strings.EqualFold(t.Repo, prefix) {
			m.compose.repo = t.Repo
			return m, true
		}
	}
	return m, false
}

func (m Model) composeNext(startNow bool) (tea.Model, tea.Cmd) {
	limit := composeText
	if m.compose.tab == composeTabURL {
		limit = composeURLRepo
	}
	if m.compose.field < limit {
		m.compose.field++
		return m, nil
	}
	return m.composeSubmit(startNow)
}
