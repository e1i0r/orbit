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

	// follow is whether the thread is pinned to its own end. It replaces a
	// sentinel offset of 999999, which was the bug behind "the scroll does
	// not work": one press of ↑ took it to 999998, still far past the
	// bottom, so the thread did not move until the key had been pressed a
	// million times. Every movement is clamped where it is made now, and
	// this says what "at the bottom" means without a magic number.
	follow bool
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
		follow:     true,
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
		return m.scrollThread(-1), nil
	case msg.Code == tea.KeyDown:
		return m.scrollThread(1), nil
	case msg.Code == tea.KeyPgUp:
		return m.scrollThread(-m.threadPage()), nil
	case msg.Code == tea.KeyPgDown:
		return m.scrollThread(m.threadPage()), nil
	case msg.Code == tea.KeyHome:
		m.supervisor.offset, m.supervisor.follow = 0, false
		return m, nil
	case msg.Code == tea.KeyEnd:
		m.supervisor.follow = true
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
	m.supervisor.follow = true
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

// scrollThread moves the thread by d rows and lands somewhere real.
//
// The clamp is here, at the press, and not only at the drawing. An offset
// allowed to run past either end is what makes a scroll feel broken: the
// number keeps moving while the screen does not, and then the first ten
// presses of the other arrow appear to do nothing while it walks back.
//
// Reaching the end is what turns following back on, so a reader who scrolls
// down to the newest message is carried by the ones that arrive after it,
// and a reader who has scrolled up is left where they were reading.
func (m Model) scrollThread(d int) Model {
	total, view := m.threadSize()
	last := max(total-view, 0)
	offset := m.supervisor.offset
	if m.supervisor.follow {
		offset = last
	}
	offset = min(max(offset+d, 0), last)
	m.supervisor.offset = offset
	m.supervisor.follow = offset >= last
	return m
}

// threadSize is how many rows the conversation is and how many are on
// screen, asked of the same functions that draw it.
func (m Model) threadSize() (total, view int) {
	boxW, threadH := m.supervisorLayout(max(m.frame.Body.H, 1), max(m.frame.Body.W, 1))
	rows, _ := m.threadLines(boxContentWidth(boxW))
	return len(rows), max(threadH-2, 1)
}

// threadRows is how many rows of conversation are on screen.
func (m Model) threadRows() int {
	_, view := m.threadSize()
	return view
}

// threadPage is one press of page up or down: a screenful less one row, so
// that the line you were reading is still there to pick the thread up from.
func (m Model) threadPage() int {
	return max(m.threadRows()-1, 1)
}
