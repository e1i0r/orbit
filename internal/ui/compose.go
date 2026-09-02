package ui

// The form a task is written into, either manually or by importing from an issue tracker URL.

import (
	"path/filepath"
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
//
// Neither is the repository, and that one was the first field on the screen.
// Choosing one was a reader's first decision about a task, back when a task
// belonged to the checkout it was written under. It is worked in whatever it
// turns out to reach into now, so the field asked a question the answer to
// which was no longer the reader's to give — and a task that reached into
// three repositories still had one of them declared at the top of it as if
// that were the fact about it.
const (
	composeFlow = iota
	composeID
	composeText
	composeFields
)

// The URL tab, in the order the rows are drawn and the order the cursor
// walks them. The URL is last because it is what this tab is for: the
// reader pastes it, looks at the flow above to see what will be done with
// it, and saves — rather than reading upwards from the thing they came for.
const (
	composeURLFlow = iota
	composeURL
	composeURLFields
)

// firstComposeField is where the cursor lands when a tab is opened: the
// field that tab exists for, which on the URL tab is the last row and not
// the first.
func firstComposeField(tab int) int {
	if tab == composeTabURL {
		return composeURL
	}

	return composeFlow
}

type composeState struct {
	tab   int // composeTabManual or composeTabURL
	field int // active field index within current tab

	// Where the first phase runs, which the form picks rather than asks.
	repoPath string

	id          input
	text        input
	url         input
	parsedIssue *tracker.Issue

	flows   []string
	flowIdx int
}

// openCompose brings the form up.
func (m Model) openCompose() Model {
	m.screen = screenCompose

	under := ""
	if r, ok := m.selected(); ok && !r.head {
		under = r.task.Repo
	}

	path := m.startsIn(under)

	flowsListed := flow.List(m.opts.Flows)

	var flows []string
	for _, f := range flowsListed {
		flows = append(flows, f.Name)
	}

	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	m.compose = composeState{
		tab:      composeTabManual,
		repoPath: path,
		flows:    flows,
	}

	return m
}

// startsIn is the repository the first phase runs in: the one the hint
// names, by name or by path, and otherwise the first Orbit knows.
//
// The form used to ask, and now it picks. The preference is the task the
// cursor was on, because a reader writing a task while looking at another
// one is usually writing about the same code — but it is a guess either
// way, and a wrong guess costs one repo.joined event: the task goes on into
// whatever it is actually worked in, and says so.
func (m Model) startsIn(hint string) string {
	repos := m.collectRepos()

	if hint != "" {
		for _, r := range repos {
			if r.path != "" && (strings.EqualFold(r.name, hint) || r.path == hint) {
				return r.path
			}
		}

		// A checkout Orbit has never listed, named by the only thing that
		// can name one it does not know: where it is. This is the session a
		// reader opened by hand somewhere else and is now writing about.
		if strings.ContainsRune(hint, filepath.Separator) {
			return hint
		}
	}

	// A repository with no path is one the board knows only by name —
	// a checkout a task was carried into, which view.Task names and does
	// not locate. It is somewhere work happened and not somewhere work can
	// be started, so the fall back is the first one Orbit can actually
	// reach, and a board where it can reach none is answered with nothing.
	for _, r := range repos {
		if r.path != "" {
			return r.path
		}
	}

	return ""
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

		m.compose.field = firstComposeField(m.compose.tab)

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
				m.compose.field = composeURL
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
		limit = composeURL
	}

	if m.compose.field < limit {
		m.compose.field++
		return m, nil
	}

	return m.composeSubmit(startNow)
}
