package flows

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
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

// View is the designer drawn, in the room it was given.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	if s.creating {
		return s.flowsBuilderRows(h, w, e)
	}

	if s.showingDetail {
		return s.flowDetailRows(h, w, e)
	}

	return s.flowsListRows(h, w, e)
}

// flowsListRows is the list as the window draws it: a page of the lines
// below, the rail beside them, and the ways out pinned to the floor.
//
// The list is longer than the screen the moment a few flows have phases —
// fill cut the rest of it, so a reader with eight flows could not see the
// last three and had nothing on screen saying they were there.
func (s State) flowsListRows(h, w int, e Env) []string {
	lines := s.flowsListLines(w, e)
	ways := s.flowsWaysOut(w, e)

	rows := max(h-1, 0)
	start := s.flowsListStart(lines, rows)

	cw := max(w-2, 1)
	track := cells.Track(rows, len(lines), start)

	out := make([]string, 0, h)

	for i := range rows {
		at := start + i
		if at >= len(lines) {
			break
		}

		row := lines[at].text
		if track != nil {
			row = cells.PadRight(cells.Fit(row, cw), cw) + track[i]
		}

		out = append(out, row)
	}

	return append(cells.Fill(out, rows), ways)
}

// flowsListStart is the row the page begins at: wherever the wheel left it,
// and never past the flow the cursor is on.
func (s State) flowsListStart(lines []flowLine, rows int) int {
	if rows <= 0 || len(lines) <= rows {
		return 0
	}

	start := min(max(s.scroll, 0), len(lines)-rows)

	// Nothing chosen is the create button, which is at the top: the cursor
	// pulls the page to a flow, and -1 is not one. Without this the rows
	// that belong to no flow — which carry noFlow, the same -1 — all match
	// and the page jumps to the floor.
	if s.sel < 0 {
		return start
	}

	first, last := -1, -1

	for i, l := range lines {
		if l.at == s.sel {
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
func (s State) flowsWaysOut(w int, e Env) string {
	p := e.Words

	return cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.ways_out",
		"[⏎] inspect · [n] create · [e] edit · [d] delete · {up_down} scroll · {back} back",
		about("up_down", e.Keys.Up.Help().Key+e.Keys.Down.Help().Key),
		about("back", e.Keys.Back.Help().Key))), w)
}

// flowsListLines is every row of the list, in order.
func (s State) flowsListLines(w int, e Env) []flowLine {
	p := e.Words

	createBtn := "  " + theme.Paint(theme.Dim).Render(p.T("flows.create_btn_idle", "[+ Create Custom Flow] (press n)"))
	if s.sel == -1 {
		createBtn = "▸ " + theme.Pill(p.T("flows.create_btn", "+ Create Custom Flow"), "#FFFFFF", "#005F87") + "  " + theme.Paint(theme.Live).Render(p.T("flows.press_enter", "(press ⏎)"))
	}

	plain := func(text string) flowLine { return flowLine{text: text, at: noFlow} }

	out := []flowLine{
		plain(""),
		plain("  " + theme.Paint(theme.Accent).Render(p.T("flows.title", "Flows"))),
		// This line said flows were read-only and to go edit the files by
		// hand — printed directly above a Create button, and above an Edit
		// and a Delete pill on every row the cursor is on. What is true is
		// the half about where they live, and that a built-in is inside the
		// binary: saving one of your own under its name covers it, which is
		// the only way a shipped flow changes.
		plain("  " + theme.Paint(theme.Dim).Render(p.T("flows.where_they_live",
			"your own flows are files under $ORBIT_HOME/flows/; a built-in is inside orbit, and saving your own under its name covers it"))),
		plain(""),
		{text: createBtn, at: noFlow, create: true},
		plain(""),
	}

	descriptors := s.listed
	if len(descriptors) == 0 {
		out = append(out, plain("  "+theme.Paint(theme.Dim).Render(p.T("flows.none", "no flows found"))))
	}

	for i, d := range descriptors {
		mark := strings.Repeat(" ", cells.Gutter)
		if i == s.sel {
			mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
		}

		originStr := OriginSaid(p, d.Origin)

		headerLine := mark + theme.Paint(theme.Accent).Render(d.Name)
		if originStr != "" {
			headerLine += "  " + theme.Paint(theme.Dim).Render("("+originStr+")")
		}

		if i == s.sel {
			headerLine += "   " + theme.Pill("👁 "+p.T("flows.btn_view_details", "Details"), "#FFFFFF", "#0284C7")

			headerLine += " " + theme.Pill("✏ "+p.T("flows.btn_edit", "Edit"), "#FFFFFF", "#0C4A6E")
			if d.Origin != flow.OriginBuiltin {
				headerLine += " " + theme.Pill("🗑 "+p.T("flows.btn_delete", "Delete"), "#FFFFFF", "#7F1D1D")
			}
		}

		out = append(out, flowLine{text: cells.Fit(headerLine, w), at: i, head: true})

		got := s.shown(d.Name)

		fl, err := got.flow, got.err
		if err != nil {
			errLine := strings.Repeat(" ", cells.Gutter+2) + theme.Paint(theme.Bad).Render(err.Error())
			out = append(out, flowLine{text: cells.Fit(errLine, w), at: i})

			continue
		}

		if fl.Description != "" {
			descLine := strings.Repeat(" ", cells.Gutter+2) + theme.Paint(theme.OK).Render("↳ ") + theme.Paint(theme.Dim).Render(flatten(fl.Description))
			out = append(out, flowLine{text: cells.Fit(descLine, w), at: i})
		}

		for idx, ph := range fl.Phases {
			if ph.Loop != nil {
				out = append(out, flowLine{text: cells.Fit(s.loopLine(idx, ph, e), w), at: i})
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
				strings.Repeat(" ", cells.Gutter+2),
				idx+1,
				theme.Paint(theme.Accent).Render(ph.Name),
				theme.Paint(theme.Dim).Render(engineModel),
				theme.Paint(theme.Live).Render(feed),
				theme.Paint(theme.Dim).Render(waitStr),
			)
			if ph.Prompt != "" {
				phaseLine += "  " + theme.Paint(theme.Dim).Render(`"`+ph.Prompt+`"`)
			}

			out = append(out, flowLine{text: cells.Fit(phaseLine, w), at: i})
		}

		out = append(out, plain(""))
	}

	return out
}

// OriginSaid is where a flow came from, in words: built in, the reader's
// own, or theirs shadowing one of ours.
func OriginSaid(p *words.Printer, o flow.Origin) string {
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
