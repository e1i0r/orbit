package ui

// The form a task is written into, so that writing one stops being the
// reason a reader opens a second terminal.
//
// It is a screen of its own because it is one: three fields and a caret,
// with the keyboard it owns while it is up. What it does at the end is not
// decided here — the write goes through the same command the palette runs,
// `orbit new`, so the id rules, the flow default and the error sentences
// are the command line's own and this file carries none of them.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"strings"
)

// composeField counts the fields the form has. The repository comes first
// because it decides where the id will live, the id second because it
// names the thing, and the text last because it is the longest to type.
const (
	composeRepo = iota
	composeID
	composeText
	composeFields
)

// composeState is the form while it is up. Every field is a plain string,
// for the reason filter and the palette line are: a rune appended and a
// rune removed, and no dependency asked for.
type composeState struct {
	repo, id, text string
	field          int // which field the caret is in
}

// openCompose brings the form up, with the repository defaulted to the one
// the cursor's task belongs to — writing the next task against the repo
// you are already looking at is the common case, and the form should open
// with it already answered.
//
// The model is copied whole and only the screen and the form change: every
// other field is exactly what the reader was looking at, so esc back to
// the board lands where they left it.
func (m Model) openCompose() Model {
	m.screen = screenCompose
	m.compose = composeState{}
	if r, ok := m.selected(); ok && !r.head {
		m.compose.repo = r.task.Repo
	}
	return m
}

// abandonCompose takes the form down without a confirmation, because there
// is nothing to confirm: nothing has been written.
func (m Model) abandonCompose() Model {
	m.compose = composeState{}
	m.screen = screenList
	return m
}

// composeKey feeds the form, which owns every key it is not given a reason
// to give up.
func (m Model) composeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		return m.abandonCompose(), nil
	case key.Matches(msg, m.keys.Open):
		return m.composeNext()
	case key.Matches(msg, m.keys.NextTab):
		return m.composeTab(1), nil
	case key.Matches(msg, m.keys.PrevTab):
		return m.composeTab(-1), nil
	case msg.Code == tea.KeyBackspace:
		m.compose.set(trimLastRune(m.compose.get()))
		return m, nil
	}
	if msg.Text != "" {
		m.compose.set(m.compose.get() + msg.Text)
		return m, nil
	}
	return m, nil
}

// get is the value of the field the caret is in.
func (c *composeState) get() string {
	switch c.field {
	case composeRepo:
		return c.repo
	case composeID:
		return c.id
	}
	return c.text
}

// set replaces it.
func (c *composeState) set(v string) {
	switch c.field {
	case composeRepo:
		c.repo = v
	case composeID:
		c.id = v
	default:
		c.text = v
	}
}

// composeTab moves the caret between fields. On the repository field, tab
// completes first and moves only when nothing matches what is typed — the
// one field whose answer is a name somebody else chose, which is where
// completion earns its keep.
func (m Model) composeTab(d int) Model {
	if d > 0 && m.compose.field == composeRepo {
		if done := m.composeComplete(); done {
			return m
		}
	}
	m.compose.field += d
	if m.compose.field < 0 {
		m.compose.field = 0
	}
	if m.compose.field >= composeFields {
		m.compose.field = composeFields - 1
	}
	return m
}

// composeComplete finishes the repository's name from the board's own list,
// and answers whether it did — a prefix that matches nothing leaves the
// caret where it is and lets tab mean move.
//
// The names come off the tasks the board is holding, which is the list the
// reader can already see; the walk's own count is a number, and a number
// completes nothing.
func (m Model) composeComplete() bool {
	prefix := strings.ToLower(m.compose.repo)
	for _, t := range m.board.Tasks {
		if strings.HasPrefix(strings.ToLower(t.Repo), prefix) {
			m.compose.repo = t.Repo
			return true
		}
	}
	return false
}

// composeNext moves to the following field, and on the last one submits.
func (m Model) composeNext() (tea.Model, tea.Cmd) {
	if m.compose.field < composeText {
		m.compose.field++
		return m, nil
	}
	return m.composeSubmit()
}

// composeSubmit checks the three answers locally where the sentence is
// cheap, then writes through `orbit new` — the same command, the same
// flags, and every deeper rule (an id already taken, a flow that is not
// there) answered by the command itself, verbatim.
func (m Model) composeSubmit() (tea.Model, tea.Cmd) {
	p := m.opts.Words
	repo := strings.TrimSpace(m.compose.repo)
	id := strings.TrimSpace(m.compose.id)
	text := strings.TrimSpace(m.compose.text)
	switch {
	case repo == "":
		return m.say(p.T("compose.repo_required",
			"the repository is required; which one is this task against?")), nil
	case id == "":
		return m.say(p.T("compose.id_required",
			"the id is required; what is this task called?")), nil
	case text == "":
		return m.say(p.T("compose.text_required",
			"the task needs something written in it")), nil
	}
	if m.opts.ValidID != nil {
		if err := m.opts.ValidID(id); err != nil {
			return m.say(err.Error()), nil
		}
	}
	// The repository is typed as its name and passed as its path, which is
	// the board's own mapping: a name on the board is a path the board
	// read it from.
	path := repo
	for _, t := range m.board.Tasks {
		if t.Repo == repo {
			path = t.RepoPath
			break
		}
	}
	m.screen = screenList
	m.pendingID, m.pendTries = id, 0
	return m.runWatched(Command{Name: "new"}, []string{"-repo", path, "-id", id, text})
}

// composeRows draws the form: one row per field, the caret's field marked,
// and the two ways out said at the bottom rather than left to be guessed.
func (m Model) composeRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	type field struct {
		label, value, placeholder string
	}
	fields := []field{
		{p.T("compose.repo", "repository"), m.compose.repo, "which repository?"},
		{p.T("compose.id", "id"), m.compose.id, "what is it called?"},
		{p.T("compose.text", "task"), m.compose.text, "what is to be done?"},
	}
	out := []string{""}
	for i, f := range fields {
		value := f.value
		role := Accent
		if value == "" {
			value, role = f.placeholder, Dim
		}
		mark := strings.Repeat(" ", gutter)
		if i == m.compose.field {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}
		line := mark + Paint(Dim).Render(f.label+": ") + Paint(role).Render(value)
		if i == m.compose.field {
			line += Paint(Sel).Render(" ")
		}
		out = append(out, fit(line, w))
	}
	out = append(out, "", fit("  "+Paint(Dim).Render(p.T("compose.ways_out",
		"{open} writes it · {back} abandons it — nothing is written until then",
		about("open", m.keys.Open.Help().Key), about("back", m.keys.Back.Help().Key))), w))
	return fill(out, h)
}

// selectPending is the board's answer to a write that just succeeded: the
// new task is chosen the moment the poll that carries it lands, and after
// two refreshes without it the wait is given up in silence — the write
// answered no error, so there is nothing to report.
func (m Model) selectPending() Model {
	if m.pendingID == "" {
		return m
	}
	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == m.pendingID {
			m.pendingID, m.pendTries = "", 0
			return m.moveTo(i)
		}
	}
	m.pendTries++
	if m.pendTries > 2 {
		m.pendingID, m.pendTries = "", 0
	}
	return m
}
