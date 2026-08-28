package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/e1i0r/orbit/internal/view"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m Model) supervisorRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words

	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}
	autoPip := Paint(Dim).Render("○ off")
	if m.autopilotOn() {
		autoPip = Paint(Live).Render("● on")
	}
	headerLine := fmt.Sprintf("  %s %s %s %s %s %s",
		Paint(Accent).Bold(true).Render("🛸 "+p.T("supervisor.title", "Supervisor & Cockpit Memory")),
		Paint(Dim).Render("·"),
		Paint(Accent).Render("["+eng+"]"),
		Paint(Dim).Render("· ⚡ autopilot: "+autoPip),
		Paint(Dim).Render("· messages: "+strconv.Itoa(len(m.supervisor.lines))),
		"",
	)
	out := []string{"", fit(headerLine, w), "  " + Paint(Dim).Render(strings.Repeat("─", min(w-4, 110))), ""}

	inputBoxHeight := 6
	threadHeight := h - len(out) - inputBoxHeight
	if threadHeight < 4 {
		threadHeight = 4
	}

	threadRows := m.drawSupervisorStream(m.supervisor.lines, threadHeight, w-6)
	out = append(out, threadRows...)

	out = append(out, m.drawSupervisorTextarea(w-4)...)
	return fill(out, h)
}

func (m Model) drawSupervisorStream(lines []view.SupervisorLine, maxRows, w int) []string {
	p := m.opts.Words
	var rendered []string

	if len(lines) == 0 && !m.supervisorBusy {
		emptyMsg := p.T("supervisor.empty", "No messages in supervisor thread yet. Type a briefing or instruction below.")
		rendered = append(rendered, "  "+Paint(Dim).Render(emptyMsg), "")
	} else {
		for _, l := range lines {
			isSupervisor := l.By == "supervisor" || l.By == "claude" || l.By == "codex" || l.By == "opencode" || l.By == "gemini"
			timeStr := Paint(Dim).Render(l.At.Format("15:04:05"))
			actorRole := Accent
			icon := "🧑‍💻"
			if isSupervisor {
				actorRole = Live
				icon = "🛸"
			}
			badge := Paint(actorRole).Bold(true).Render(icon + " " + l.By + " [" + l.Channel + "]")
			taskTag := ""
			if l.TaskID != "" {
				taskTag = " " + Paint(Accent).Render("("+l.TaskID+")")
			}

			rendered = append(rendered, fit(fmt.Sprintf("  %s  ❯ %s%s", timeStr, badge, taskTag), w))

			if isSupervisor {
				mdLines := renderMarkdown(l.Text, w, false)
				for _, mdl := range mdLines {
					rendered = append(rendered, fit(mdl, w))
				}
			} else {
				rawLines := strings.Split(strings.ReplaceAll(l.Text, "\r\n", "\n"), "\n")
				for _, rl := range rawLines {
					for _, wl := range splitIntoLines(formatInlineMarkdown(rl), max(20, w-6)) {
						rendered = append(rendered, fit("    "+wl, w))
					}
				}
			}
			rendered = append(rendered, "")
		}
	}

	if m.supervisorBusy {
		eng := m.knobs.Engine
		if eng == "" {
			eng = "claude"
		}
		frameIdx := int(m.now.UnixMilli()/120) % len(spinnerFrames)
		spin := spinnerFrames[frameIdx]
		timeStr := Paint(Dim).Render(m.now.Format("15:04:05"))
		spinLine := fmt.Sprintf("  %s  ❯ %s  %s %s",
			timeStr,
			Paint(Live).Bold(true).Render("🧠 "+eng+" [supervisor]"),
			Paint(Live).Render(spin),
			Paint(Live).Render("thinking & analyzing telemetry..."),
		)
		rendered = append(rendered, fit(spinLine, w), "")
	}

	if len(rendered) <= maxRows {
		return rendered
	}

	offset := m.supervisor.offset
	maxOffset := len(rendered) - maxRows
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return rendered[offset : offset+maxRows]
}

func (m Model) drawSupervisorTextarea(w int) []string {
	p := m.opts.Words
	boxW := min(w, 110)
	title := p.T("supervisor.input_prompt", "Say to Supervisor / Directive:")
	topBorder := "  " + Paint(Accent).Bold(true).Render("┌─ 💬 "+title+" ") +
		Paint(Dim).Render(strings.Repeat("─", max(boxW-lipgloss.Width(title)-10, 4))+"┐")

	var inputLines []string
	rawInput := m.supervisor.input
	if rawInput == "" {
		placeholder := p.T("supervisor.placeholder", "type a briefing, question or standing directive...")
		inputLines = append(inputLines, "  "+Paint(Dim).Render("│ ")+Paint(Dim).Render(placeholder))
	} else {
		lines := strings.Split(rawInput, "\n")
		for i, l := range lines {
			cursor := ""
			if i == len(lines)-1 {
				cursor = Paint(Accent).Render("█")
			}
			inputLines = append(inputLines, "  "+Paint(Dim).Render("│ ")+Paint(Accent).Render("> ")+l+cursor)
		}
	}

	for len(inputLines) < 2 {
		inputLines = append(inputLines, "  "+Paint(Dim).Render("│ "))
	}

	bottomText := p.T("supervisor.ways_out", "[Shift+↵] newline · [↵] send · [esc] back · [↑↓] scroll · [^V] paste",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key))
	bottomBorder := "  " + Paint(Dim).Render("└─ ") + Paint(Dim).Render(bottomText) + " " +
		Paint(Dim).Render(strings.Repeat("─", max(boxW-lipgloss.Width(bottomText)-6, 4))+"┘")

	out := []string{"", fit(topBorder, w)}
	for _, il := range inputLines {
		out = append(out, fit(il, w))
	}
	out = append(out, fit(bottomBorder, w))
	return out
}
