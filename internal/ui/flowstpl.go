package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) applyFlowTemplate(tpl string) (Model, tea.Cmd) {
	st := &m.flows
	switch tpl {
	case "TDD Cycle":
		st.flowName = "tdd-cycle"
		st.phases = []flow.Phase{
			{Name: "1-plan", Engine: "claude", Model: "opus", Effort: "high", Thinking: "on", Prompt: "Analiza el problema y diseña el plan técnico.", Permissions: []string{"read"}},
			{Name: "2-implement", Engine: "claude", Model: "sonnet", Effort: "high", FeedOutput: true, Prompt: "Implementa el código y pruebas unitarias.", Permissions: []string{"repo"}},
		}
		st.phaseName = "3-review"
		st.engine = "claude"
		st.model = "opus"
		st.effort = "max"
		st.thinking = "on"
		st.feedOutput = true
		st.wait = true
		st.prompt = "Audita el diff final y valida los chequeos."
		return m.say("plantilla TDD Cycle cargada"), nil
	case "Security Audit":
		st.flowName = "security-audit"
		st.phases = []flow.Phase{
			{Name: "1-investigate", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", Prompt: "Inspecciona el repositorio por vulnerabilidades.", Permissions: []string{"read"}},
		}
		st.phaseName = "2-remediate"
		st.engine = "claude"
		st.model = "opus"
		st.effort = "high"
		st.feedOutput = true
		st.wait = false
		st.prompt = "Aplica parches para los hallazgos."
		return m.say("plantilla Security Audit cargada"), nil
	case "Turbo Fix":
		st.flowName = "turbo-fix"
		st.phases = nil
		st.phaseName = "implement"
		st.engine = "claude"
		st.model = "sonnet"
		st.effort = "high"
		st.feedOutput = false
		st.wait = false
		st.prompt = "Resuelve la tarea de forma directa."
		return m.say("plantilla Turbo Fix cargada"), nil
	}
	return m, nil
}

func (m Model) saveCustomFlow() (Model, tea.Cmd) {
	st := &m.flows
	name := strings.TrimSpace(st.flowName)
	if name == "" {
		return m.say("indica un nombre para el flujo"), nil
	}
	phases := st.phases
	if len(phases) == 0 && st.phaseName != "" {
		phases = append(phases, st.currentPhase())
	}
	if len(phases) == 0 {
		return m.say("el flujo debe tener al menos una fase"), nil
	}
	fl := flow.Flow{
		Name:   name,
		Phases: phases,
	}
	if err := fl.Validate(); err != nil {
		return m.say(err.Error()), nil
	}

	dir := ""
	if m.opts.Flows != nil {
		dir = m.opts.Flows.FlowDir()
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".orbit", "flows")
	}
	_ = os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return m.say(err.Error()), nil
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return m.say(err.Error()), nil
	}
	m.flows.creating = false
	m.flows.phases = nil
	m.flows.flowName = ""
	p := m.opts.Words
	return m.say(p.T("flows.saved", "flow {name} saved", about("name", name))), nil
}
