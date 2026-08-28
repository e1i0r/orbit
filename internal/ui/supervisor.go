package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

type supervisorState struct {
	prevScreen screen
	input      string
	offset     int
	lines      []view.SupervisorLine
	err        error

	// picking is the mode that takes a turn back: ↑↓ choose a line instead
	// of scrolling and ↵ withdraws it instead of sending. It is a mode
	// rather than a key on its own because on this screen every printable
	// key already types, and the arrows already scroll — there was no free
	// gesture left for "which line".
	picking bool
	pick    int
}

func (m Model) openSupervisor() Model {
	prev := m.screen
	if prev == screenSupervisor {
		prev = screenList
	}
	m.screen = screenSupervisor
	m.supervisor = supervisorState{
		prevScreen: prev,
		input:      "",
		offset:     999999,
	}
	return m.syncSupervisor()
}

func (m Model) abandonSupervisor() Model {
	target := m.supervisor.prevScreen
	if target == screenSupervisor {
		target = screenList
	}
	m.supervisor = supervisorState{}
	m.screen = target
	return m
}

func (m Model) syncSupervisor() Model {
	if m.opts.Reader == nil {
		return m
	}
	lines, err := m.opts.Reader.SupervisorLog()
	m.supervisor.lines = lines
	m.supervisor.err = err
	return m
}

func (m Model) supervisorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.supervisor.picking:
		return m.pickingKey(msg)
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back):
		return m.abandonSupervisor(), nil
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return m.startPicking(), nil
	case msg.Code == tea.KeyUp:
		if m.supervisor.offset > 0 {
			m.supervisor.offset--
		}
		return m, nil
	case msg.Code == tea.KeyDown:
		m.supervisor.offset++
		return m, nil
	case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Open):
		if msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0 {
			m.supervisor.input += "\n"
			return m, nil
		}
		text := strings.TrimSpace(m.supervisor.input)
		if text == "" {
			return m, nil
		}
		m.supervisor.input = ""
		return m.sendSupervisorMessage(text)
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if clip := readClipboard(); clip != "" {
			m.supervisor.input += clip
		}
		return m, nil
	case msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete:
		if len(m.supervisor.input) > 0 {
			runes := []rune(m.supervisor.input)
			m.supervisor.input = string(runes[:len(runes)-1])
		}
		return m, nil
	case (msg.Code == 'u' || msg.Code == 'U') && msg.Mod&tea.ModCtrl != 0:
		m.supervisor.input = ""
		return m, nil
	default:
		if msg.Text != "" {
			m.supervisor.input += msg.Text
			return m, nil
		}
	}
	return m, nil
}

func (m Model) sendSupervisorMessage(text string) (tea.Model, tea.Cmd) {
	if m.opts.RecordSupervisor != nil {
		// "operator" is who every other door writes, and the thread is one
		// conversation: a name hardcoded here made the same person read as
		// two participants depending on whether they typed in the window or
		// in a terminal, and put somebody else's name on the messages of
		// anyone who is not the author of this program.
		if err := m.opts.RecordSupervisor("operator", "tui", text); err != nil {
			return m.say(err.Error()), nil
		}
	}
	m = m.syncSupervisor()
	m.supervisor.offset = 999999
	m.supervisorBusy = true
	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	cmd := askSupervisorCmd(m.opts.AskSupervisor, eng, text)
	return m.say(m.opts.Words.T("supervisor.thinking", "supervisor is thinking...")), tea.Batch(cmd, spinnerTick())
}

// startPicking opens the mode that takes a turn back, on the last line said
// — which is the one somebody has just regretted nine times out of ten.
func (m Model) startPicking() Model {
	if len(m.supervisor.lines) == 0 || m.opts.RetractSupervisor == nil {
		return m
	}
	m.supervisor.picking = true
	m.supervisor.pick = len(m.supervisor.lines) - 1
	return m
}

// pickingKey is every key while a line is being picked.
func (m Model) pickingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back) ||
		((msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0):
		m.supervisor.picking = false
		return m, nil
	case msg.Code == tea.KeyUp:
		m.supervisor.pick = max(m.supervisor.pick-1, 0)
		return m, nil
	case msg.Code == tea.KeyDown:
		m.supervisor.pick = min(m.supervisor.pick+1, len(m.supervisor.lines)-1)
		return m, nil
	case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Open):
		return m.retractPicked(), nil
	}
	return m, nil
}

// retractPicked takes back the line under the cursor.
func (m Model) retractPicked() Model {
	m.supervisor.picking = false
	if m.opts.RetractSupervisor == nil || m.supervisor.pick >= len(m.supervisor.lines) {
		return m
	}
	l := m.supervisor.lines[m.supervisor.pick]
	if l.Retracted {
		return m.say(m.opts.Words.T("supervisor.already_back", "that line was already taken back"))
	}
	if err := m.opts.RetractSupervisor(l.At); err != nil {
		return m.say(err.Error())
	}
	m = m.syncSupervisor()
	return m.say(m.opts.Words.T("supervisor.took_back", "took that line back; the supervisor is no longer told it"))
}
