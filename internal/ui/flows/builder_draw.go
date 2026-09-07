package flows

import (
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
)

func (s State) startCreateFlow(e Env) State {
	s.creating = true
	s.engine = cells.OrDef("", e.Engine)
	s.isEditing = false
	s.confirmDiscard = false
	s.confirmDelete = false
	s.field = 0
	s.template = "ninguna"
	s.flowName = ""
	s.phases = []flow.Phase{
		{Name: "1-implement", Engine: s.engine, Thinking: "adaptive", Permissions: []string{"repo"}},
	}
	s.activePhase = 0
	s.checksTyped = false
	s.scroll = 0
	// A new flow opens on the tab where you say what it should do: there is
	// nothing to edit yet, and a sentence is a faster first draft than
	// eleven fields. Editing one opens on the fields, because that flow
	// already exists and the sentence would replace it.
	s.tab = flowTabSay

	return s
}

// flowsBuilderRows is the form as the window draws it: the rows of
// builderLines, from wherever the window starts.
func (s State) flowsBuilderRows(h, w int, e Env) []string {
	lines, start := s.builderView(h, w, e)

	out := make([]string, 0, h)
	for _, l := range lines[min(start, len(lines)):] {
		out = append(out, l.text)
	}

	return cells.Fill(out, h)
}
