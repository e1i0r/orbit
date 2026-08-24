package ui

// The flows screen: a screen listing every flow with its phases, and for
// each phase the engine, model, permissions and whether it waits.
//
// Read-only in this plan: editing a flow file is done in $ORBIT_HOME/flows/
// and the screen says so.

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/words"
)

type flowsState struct {
	sel    int
	offset int
}

func (m Model) openFlows() Model {
	m.screen = screenFlows
	m.flows = flowsState{}
	return m
}

func (m Model) abandonFlows() Model {
	m.flows = flowsState{}
	m.screen = screenList
	return m
}

func (m Model) flowsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := flow.List(m.opts.Flows)
	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		return m.abandonFlows(), nil
	case key.Matches(msg, m.keys.Up):
		if len(list) > 0 {
			m.flows.sel--
			if m.flows.sel < 0 {
				m.flows.sel = len(list) - 1
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if len(list) > 0 {
			m.flows.sel++
			if m.flows.sel >= len(list) {
				m.flows.sel = 0
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) flowsRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("flows.title", "Flows")),
		"  " + Paint(Dim).Render(p.T("flows.read_only", "flows are read-only; edit flow files under $ORBIT_HOME/flows/ to change them")),
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
			perms := "none"
			if len(ph.Permissions) > 0 {
				perms = strings.Join(ph.Permissions, ", ")
			}
			waitStr := p.T("flow.runs_auto", "runs automatically")
			if ph.Wait {
				waitStr = p.T("flow.stops_for_human", "stops for human")
			}

			phaseLine := fmt.Sprintf("%s%d. %s  %s  [%s]  (%s)",
				strings.Repeat(" ", gutter+2),
				idx+1,
				Paint(Accent).Render(ph.Name),
				Paint(Dim).Render(engineModel),
				Paint(Dim).Render(perms),
				Paint(Dim).Render(waitStr),
			)
			out = append(out, fit(phaseLine, w))
		}
		out = append(out, "")
	}

	waysOut := p.T("flows.ways_out", "{up_down} scroll · {back} back",
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
