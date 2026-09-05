package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/words"
)

// flowLine is one drawn row of the list and the flow it belongs to, so that
// scrolling and clicking read the same layout rather than each counting the
// rows for themselves.
type flowLine struct {
	text string
	at   int // the flow this row is part of, or noFlow
	// head is the flow's own row, the one carrying its name and the pills
	// that inspect, edit and delete it. create is the button above them.
	head   bool
	create bool
}

// noFlow is a row that belongs to no flow: the title, a blank, the button.
const noFlow = -1

func (m Model) flowsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	if m.flows.creating {
		return m.flowsBuilderRows(h, w)
	}

	if m.flows.showingDetail {
		return m.flowDetailRows(h, w)
	}

	return m.flowsListRows(h, w)
}

// flowsListRows is the list as the window draws it: a page of the lines
// below, the rail beside them, and the ways out pinned to the floor.
//
// The list is longer than the screen the moment a few flows have phases —
// fill cut the rest of it, so a reader with eight flows could not see the
// last three and had nothing on screen saying they were there.
func (m Model) flowsListRows(h, w int) []string {
	lines := m.flowsListLines(w)
	ways := m.flowsWaysOut(w)

	rows := max(h-1, 0)
	start := m.flowsListStart(lines, rows)

	cw := max(w-2, 1)
	track := scrollTrack(rows, len(lines), start)

	out := make([]string, 0, h)

	for i := range rows {
		at := start + i
		if at >= len(lines) {
			break
		}

		row := lines[at].text
		if track != nil {
			row = padRight(fit(row, cw), cw) + track[i]
		}

		out = append(out, row)
	}

	return append(fill(out, rows), ways)
}

// flowsListStart is the row the page begins at: wherever the wheel left it,
// and never past the flow the cursor is on.
func (m Model) flowsListStart(lines []flowLine, rows int) int {
	if rows <= 0 || len(lines) <= rows {
		return 0
	}

	start := min(max(m.flows.scroll, 0), len(lines)-rows)

	// Nothing chosen is the create button, which is at the top: the cursor
	// pulls the page to a flow, and -1 is not one. Without this the rows
	// that belong to no flow — which carry noFlow, the same -1 — all match
	// and the page jumps to the floor.
	if m.flows.sel < 0 {
		return start
	}

	first, last := -1, -1

	for i, l := range lines {
		if l.at == m.flows.sel {
			if first < 0 {
				first = i
			}

			last = i
		}
	}

	switch {
	case first < 0:
		return start
	case first < start:
		return first
	case last >= start+rows:
		return min(last-rows+1, len(lines)-rows)
	}

	return start
}

// flowsWaysOut is the line along the floor, drawn outside the page so that
// scrolling never takes it off the screen.
func (m Model) flowsWaysOut(w int) string {
	p := m.opts.Words

	return fit("  "+Paint(Dim).Render(p.T("flows.ways_out",
		"[⏎] inspect · [n] create · [e] edit · [d] delete · {up_down} scroll · {back} back",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))), w)
}

// flowsListLines is every row of the list, in order.
func (m Model) flowsListLines(w int) []flowLine {
	p := m.opts.Words

	createBtn := "  " + Paint(Dim).Render(p.T("flows.create_btn_idle", "[+ Create Custom Flow] (press n)"))
	if m.flows.sel == -1 {
		createBtn = "▸ " + Pill(p.T("flows.create_btn", "+ Create Custom Flow"), "#FFFFFF", "#005F87") + "  " + Paint(Live).Render(p.T("flows.press_enter", "(press ⏎)"))
	}

	plain := func(text string) flowLine { return flowLine{text: text, at: noFlow} }

	out := []flowLine{
		plain(""),
		plain("  " + Paint(Accent).Render(p.T("flows.title", "Flows"))),
		// This line said flows were read-only and to go edit the files by
		// hand — printed directly above a Create button, and above an Edit
		// and a Delete pill on every row the cursor is on. What is true is
		// the half about where they live, and that a built-in is inside the
		// binary: saving one of your own under its name covers it, which is
		// the only way a shipped flow changes.
		plain("  " + Paint(Dim).Render(p.T("flows.where_they_live",
			"your own flows are files under $ORBIT_HOME/flows/; a built-in is inside orbit, and saving your own under its name covers it"))),
		plain(""),
		{text: createBtn, at: noFlow, create: true},
		plain(""),
	}

	descriptors := m.flows.listed
	if len(descriptors) == 0 {
		out = append(out, plain("  "+Paint(Dim).Render(p.T("flows.none", "no flows found"))))
	}

	for i, d := range descriptors {
		mark := strings.Repeat(" ", gutter)
		if i == m.flows.sel {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}

		originStr := flowOriginString(p, d.Origin)

		headerLine := mark + Paint(Accent).Render(d.Name)
		if originStr != "" {
			headerLine += "  " + Paint(Dim).Render("("+originStr+")")
		}

		if i == m.flows.sel {
			headerLine += "   " + Pill("👁 "+p.T("flows.btn_view_details", "Details"), "#FFFFFF", "#0284C7")

			headerLine += " " + Pill("✏ "+p.T("flows.btn_edit", "Edit"), "#FFFFFF", "#0C4A6E")
			if d.Origin != flow.OriginBuiltin {
				headerLine += " " + Pill("🗑 "+p.T("flows.btn_delete", "Delete"), "#FFFFFF", "#7F1D1D")
			}
		}

		out = append(out, flowLine{text: fit(headerLine, w), at: i, head: true})

		got := m.flows.shown(d.Name)

		fl, err := got.flow, got.err
		if err != nil {
			errLine := strings.Repeat(" ", gutter+2) + Paint(Bad).Render(err.Error())
			out = append(out, flowLine{text: fit(errLine, w), at: i})

			continue
		}

		if fl.Description != "" {
			descLine := strings.Repeat(" ", gutter+2) + Paint(OK).Render("↳ ") + Paint(Dim).Render(flatten(fl.Description))
			out = append(out, flowLine{text: fit(descLine, w), at: i})
		}

		for idx, ph := range fl.Phases {
			if ph.Loop != nil {
				out = append(out, flowLine{text: fit(m.loopLine(idx, ph), w), at: i})
				continue
			}

			engineModel := ph.Engine
			if ph.Model != "" {
				engineModel += " / " + ph.Model
			}

			feed := ""
			if ph.FeedOutput {
				feed = " " + p.T("flows.feeds_input", "[feeds input]")
			}

			waitStr := p.T("flow.runs_auto", "runs automatically")
			if ph.Wait {
				waitStr = p.T("flow.stops_for_human", "stops for human")
			}

			phaseLine := fmt.Sprintf("%s%d. %s  %s%s  (%s)",
				strings.Repeat(" ", gutter+2),
				idx+1,
				Paint(Accent).Render(ph.Name),
				Paint(Dim).Render(engineModel),
				Paint(Live).Render(feed),
				Paint(Dim).Render(waitStr),
			)
			if ph.Prompt != "" {
				phaseLine += "  " + Paint(Dim).Render(`"`+ph.Prompt+`"`)
			}

			out = append(out, flowLine{text: fit(phaseLine, w), at: i})
		}

		out = append(out, plain(""))
	}

	return out
}

func flowOriginString(p *words.Printer, o flow.Origin) string {
	switch o {
	case flow.OriginBuiltin:
		return p.T("flow.built_in", "built in")
	case flow.OriginUser:
		return p.T("flow.yours", "yours")
	case flow.OriginShadow:
		return p.T("flow.shadowing", "yours, shadowing the built-in")
	}

	return ""
}
