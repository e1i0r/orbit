package ui

import (
	"fmt"
	"strconv"

	"github.com/e1i0r/orbit/internal/view"
)

func (m Model) supervisorRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render("🛸 "+p.T("supervisor.title", "Supervisor & Cockpit Memory")),
		"  " + Paint(Dim).Render(p.T("supervisor.subtitle", "briefing, debriefing, autonomous feedback and standing directives")),
		"",
	}

	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	autoPip := Paint(Dim).Render("○ off")
	if m.autopilotOn() {
		autoPip = Paint(Live).Render("● on")
	}
	statusChips := fmt.Sprintf("  %s %s   %s %s   %s %s",
		Paint(Dim).Render("engine:"), Paint(Accent).Render(eng),
		Paint(Dim).Render("autopilot:"), autoPip,
		Paint(Dim).Render("messages:"), Paint(Dim).Render(strconv.Itoa(len(m.supervisor.lines))))
	out = append(out, fit(statusChips, w), "")

	inputHeight := 4
	threadHeight := h - len(out) - inputHeight
	if threadHeight < 3 {
		threadHeight = 3
	}

	threadRows := m.drawSupervisorThread(m.supervisor.lines, threadHeight, w)
	out = append(out, threadRows...)

	out = append(out, "")
	out = append(out, fit("  "+Paint(Accent).Bold(true).Render("💬 "+p.T("supervisor.input_prompt", "Say to Supervisor / Directive:")), w))
	inputText := m.supervisor.input
	if inputText == "" {
		inputText = Paint(Dim).Render(p.T("supervisor.placeholder", "type a briefing, question or standing directive..."))
	}
	cursor := Paint(Accent).Render("█")
	if m.supervisor.input == "" {
		cursor = ""
	}
	inputBox := "  " + Paint(Dim).Render("│ ") + inputText + cursor
	out = append(out, fit(inputBox, w))

	waysOut := p.T("supervisor.ways_out", "[⏎] send · [esc] back · {up_down} scroll",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))

	return fill(out, h)
}

func (m Model) drawSupervisorThread(lines []view.SupervisorLine, maxRows, w int) []string {
	p := m.opts.Words
	if len(lines) == 0 {
		emptyMsg := p.T("supervisor.empty", "No messages in supervisor thread yet. Type a briefing or instruction below.")
		return []string{"  " + Paint(Dim).Render(emptyMsg)}
	}

	rendered := make([]string, 0, len(lines)*2)
	for _, l := range lines {
		timeStr := Paint(Dim).Render(l.At.Format("15:04:05"))
		actorRole := Accent
		if l.By == "supervisor" || l.Channel == "autopilot" {
			actorRole = Live
		}
		actorBadge := Paint(actorRole).Bold(true).Render("[" + l.By + " via " + l.Channel + "]")
		taskTag := ""
		if l.TaskID != "" {
			taskTag = " " + Paint(Accent).Render("("+l.TaskID+")")
		}
		headerLine := fmt.Sprintf("  %s %s%s", timeStr, actorBadge, taskTag)
		rendered = append(rendered, fit(headerLine, w))
		msgLine := "    " + l.Text
		rendered = append(rendered, fit(msgLine, w))
	}

	if len(rendered) <= maxRows {
		return rendered
	}

	offset := m.supervisor.offset
	if offset > len(rendered)-maxRows {
		offset = len(rendered) - maxRows
	}
	if offset < 0 {
		offset = 0
	}
	return rendered[offset : offset+maxRows]
}
