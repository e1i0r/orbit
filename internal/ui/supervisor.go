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
		offset:     0,
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
	case key.Matches(msg, m.keys.Back):
		return m.abandonSupervisor(), nil
	case key.Matches(msg, m.keys.Up):
		if m.supervisor.offset > 0 {
			m.supervisor.offset--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.supervisor.offset++
		return m, nil
	case key.Matches(msg, m.keys.Open):
		text := strings.TrimSpace(m.supervisor.input)
		if text == "" {
			return m, nil
		}
		m.supervisor.input = ""
		return m.sendSupervisorMessage(text)
	}

	switch msg.String() {
	case "backspace", "ctrl+h":
		if len(m.supervisor.input) > 0 {
			runes := []rune(m.supervisor.input)
			m.supervisor.input = string(runes[:len(runes)-1])
		}
		return m, nil
	case "ctrl+u":
		m.supervisor.input = ""
		return m, nil
	default:
		if r := msg.Text; r != "" {
			m.supervisor.input += r
			return m, nil
		}
	}
	return m, nil
}

func (m Model) sendSupervisorMessage(text string) (tea.Model, tea.Cmd) {
	if m.opts.RecordSupervisor != nil {
		if err := m.opts.RecordSupervisor("elio", "tui", text); err != nil {
			return m.say(err.Error()), nil
		}
	}
	m = m.syncSupervisor()
	if len(m.supervisor.lines) > 0 {
		m.supervisor.offset = len(m.supervisor.lines)
	}
	return m.say(m.opts.Words.T("supervisor.sent", "message recorded in supervisor thread")), nil
}
