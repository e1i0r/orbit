package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/words"
)

func (m Model) flowsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	if m.flows.creating {
		return m.flowsBuilderRows(h, w)
	}
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("flows.title", "Flows")),
		"  " + Paint(Dim).Render(p.T("flows.read_only", "flows are read-only; edit flow files under $ORBIT_HOME/flows/ to change them")),
		"",
		"  " + Pill(p.T("flows.create_btn", "+ Create Custom Flow"), "#FFFFFF", "#005F87") + "  " + Paint(Dim).Render("(press n)"),
		"",
	}

	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("flows.none", "no flows found")))
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
			headerLine += "   " + Pill("✏ Editar", "#FFFFFF", "#0C4A6E")
			if d.Origin != flow.OriginBuiltin {
				headerLine += " " + Pill("🗑 Borrar", "#FFFFFF", "#7F1D1D")
			}
		}
		out = append(out, fit(headerLine, w))

		fl, err := flow.Resolve(m.opts.Flows, d.Name)
		if err != nil {
			errLine := strings.Repeat(" ", gutter+2) + Paint(Bad).Render(err.Error())
			out = append(out, fit(errLine, w))
			continue
		}

		for idx, ph := range fl.Phases {
			engineModel := ph.Engine
			if ph.Model != "" {
				engineModel += " / " + ph.Model
			}
			feed := ""
			if ph.FeedOutput {
				feed = " [feeds input]"
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
			out = append(out, fit(phaseLine, w))
		}
		out = append(out, "")
	}

	waysOut := p.T("flows.ways_out", "[n] create · [e] edit · [d] delete · {up_down} scroll · {back} back",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
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
