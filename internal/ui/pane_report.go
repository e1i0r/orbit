package ui

import (
	"github.com/e1i0r/orbit/internal/view"
)

// reportLines renders Pane 7: The engine's summary reports and conclusions.
func (m Model) reportLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	w, blocks := max(m.frame.Body.W, 1), 0
	var out []string
	var started view.Entry
	for _, e := range m.entries {
		if e.Attempted() {
			out = append(out, m.seam(e, w))
			started = view.Entry{}
			continue
		}
		if e.Phase == "" {
			continue
		}
		if e.What() == view.EntryStarted {
			started = e
			continue
		}
		switch e.What() {
		case view.EntryFinished, view.EntryFailed, view.EntryCancelled:
		default:
			continue
		}
		blocks++
		out = append(out, m.phaseHead(e, started))
		out = append(out, m.phaseBody(e)...)
	}

	if blocks == 0 {
		return []string{"", "  " + Paint(Dim).Render(p.T("report.empty", "no engine report available for this task"))}
	}

	return out
}
