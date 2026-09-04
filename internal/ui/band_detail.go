package ui

import (
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// detailBandLine answers "what is happening with this task right now" for the
// task detail screen, ensuring the activity band stays attached to the viewed task
// rather than falling back to global board tasks or generic idle messages.
func (m Model) detailBandLine(t view.Task) string {
	p := m.opts.Words

	// 1. In-flight delivery action for this task.
	if m.delivering.verb != "" && m.delivering.task.ID == t.ID {
		by := m.delivering.cmd
		if by == "" {
			by = deliverBySupervisor
		}

		said := p.T("overview.deliver_out", "{verb} is out with {by}",
			about("verb", m.delivering.verb), about("by", by))

		pieces := []string{Paint(Accent).Render(t.ID), Paint(Live).Render(said)}
		if age := elapsed(m.now, t.Since); age != "" {
			pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
		}

		return m.spinner(Live) + strings.Join(pieces, dot)
	}

	// 2. Uncompleted delivery recorded in the task history.
	steps := m.byHand()
	for i := len(steps) - 1; i >= 0; i-- {
		st := steps[i]
		if st.done {
			continue
		}

		said := p.T("overview.deliver_out_bare", "{verb} is out", about("verb", st.verb))
		if st.by != "" {
			said = p.T("overview.deliver_out", "{verb} is out with {by}",
				about("verb", st.verb), about("by", st.by))
		}

		pieces := []string{Paint(Accent).Render(t.ID), Paint(Live).Render(said)}
		if ago := elapsed(m.now, st.at); ago != "" {
			pieces = append(pieces, p.T("overview.deliver_ago", "asked {ago} ago",
				about("ago", ago)))
		}

		return m.spinner(Live) + strings.Join(pieces, dot)
	}

	// 3. Supervisor active on this task.
	if m.supervisorBusy && (m.delivering.task.ID == t.ID || m.detail == t.ID) {
		said := p.T("supervisor.thinking", "supervisor is thinking...")
		pieces := []string{Paint(Accent).Render(t.ID), Paint(Live).Render(said)}

		return m.spinner(Live) + strings.Join(pieces, dot)
	}

	// 4. Per-band rendering for the viewed task.
	switch view.BandOf(t) {
	case view.Running:
		return m.detailRunningLine(t)
	case view.NeedsYou:
		return m.detailNeedsYouLine(t)
	case view.ToDo:
		return m.detailToDoLine(t)
	case view.Done:
		return m.detailDoneLine(t)
	default:
		return m.runningLine(t)
	}
}

func (m Model) detailRunningLine(t view.Task) string {
	p := m.opts.Words

	if t.Reason.Key == view.ReasonHeld {
		held := p.T("reason.held", "held: {phase}", reasonArgs(t.Reason)...)

		pieces := []string{Paint(Accent).Render(t.ID), Paint(Warn).Render(held)}
		if age := elapsed(m.now, t.Since); age != "" {
			pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
		}

		if eng := engineAndModel(t); eng != "" {
			pieces = append(pieces, eng)
		}

		if t.Flow != "" {
			pieces = append(pieces, t.Flow)
		}

		pieces = append(pieces, Paint(Dim).Render("⏸"))

		return strings.Join(pieces, dot)
	}

	pieces := []string{Paint(Accent).Render(t.ID), Paint(Live).Render(m.phaseWord(t))}
	if age := elapsed(m.now, t.Since); age != "" {
		pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
	}

	if t.CurrentAction != "" {
		pieces = append(pieces, Paint(Live).Render(
			actionGlyph(t.ActionKind)+fit(t.CurrentAction, actionCells)))
	} else if t.CurrentThought != "" {
		first := strings.TrimSpace(strings.Split(t.CurrentThought, "\n")[0])
		if first != "" {
			pieces = append(pieces, Paint(Live).Render("🧠 "+fit(first, actionCells)))
		}
	}

	if eng := engineAndModel(t); eng != "" {
		pieces = append(pieces, eng)
	}

	if t.Flow != "" {
		pieces = append(pieces, t.Flow)
	}

	return m.spinner(Live) + strings.Join(pieces, dot)
}

func (m Model) detailNeedsYouLine(t view.Task) string {
	p := m.opts.Words
	state, role := m.stateWord(t)
	pieces := []string{Paint(Accent).Render(t.ID), Paint(role).Render(state)}

	if age := elapsed(m.now, t.Since); age != "" {
		pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
	}

	if t.Reason.Key == view.ReasonGate {
		if m.autopilotOn() {
			pieces = append(pieces, Paint(Live).Render(
				p.T("why.pause_autopilot_is_lifting",
					"autopilot is lifting this gate; press A to turn it off")))
		} else {
			pieces = append(pieces, Paint(Dim).Render(
				p.T("why.pause_already_waiting",
					"this phase is already waiting for you; press r to let it go")))
		}
	}

	if eng := engineAndModel(t); eng != "" {
		pieces = append(pieces, eng)
	}

	if t.Flow != "" {
		pieces = append(pieces, t.Flow)
	}

	return strings.Join(pieces, dot)
}

func (m Model) detailToDoLine(t view.Task) string {
	p := m.opts.Words

	pieces := []string{
		Paint(Accent).Render(t.ID),
		Paint(Dim).Render(p.T("state.not_started", "not started")),
	}

	if t.Flow != "" {
		pieces = append(pieces, t.Flow)
	}

	if t.Repo != "" {
		pieces = append(pieces, t.Repo)
	}

	return strings.Join(pieces, dot)
}

func (m Model) detailDoneLine(t view.Task) string {
	p := m.opts.Words
	state, role := m.stateWord(t)
	pieces := []string{Paint(Accent).Render(t.ID), Paint(role).Render(state)}

	if age := elapsed(m.now, t.Since); age != "" {
		pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
	}

	if eng := engineAndModel(t); eng != "" {
		pieces = append(pieces, eng)
	}

	if t.Flow != "" {
		pieces = append(pieces, t.Flow)
	}

	return strings.Join(pieces, dot)
}
