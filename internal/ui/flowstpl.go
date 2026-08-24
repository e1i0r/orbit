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
			{Name: "3-review", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", FeedOutput: true, Wait: true, Prompt: "Audita el diff final y valida los chequeos.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("plantilla TDD Cycle cargada (3 fases)"), nil
	case "Security Audit":
		st.flowName = "security-audit"
		st.phases = []flow.Phase{
			{Name: "1-investigate", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", Prompt: "Inspecciona el repositorio por vulnerabilidades.", Permissions: []string{"read"}},
			{Name: "2-remediate", Engine: "claude", Model: "opus", Effort: "high", FeedOutput: true, Prompt: "Aplica parches para los hallazgos.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("plantilla Security Audit cargada (2 fases)"), nil
	case "Turbo Fix":
		st.flowName = "turbo-fix"
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "high", Prompt: "Resuelve la tarea de forma directa.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
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
	st.ensurePhase()
	if len(st.phases) == 0 {
		return m.say("el flujo debe tener al menos una fase"), nil
	}
	fl := flow.Flow{
		Name:   name,
		Phases: st.phases,
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

func (m Model) editSelectedFlow() (Model, tea.Cmd) {
	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 || m.flows.sel < 0 || m.flows.sel >= len(descriptors) {
		return m, nil
	}
	d := descriptors[m.flows.sel]
	return m.editFlow(d.Name)
}

func (m Model) editFlow(name string) (Model, tea.Cmd) {
	fl, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return m.say(err.Error()), nil
	}
	m.flows.creating = true
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = fl.Name
	m.flows.phases = fl.Phases
	m.flows.activePhase = 0
	m.flows.ensurePhase()
	return m.say("editando flujo " + fl.Name), nil
}

func (m Model) deleteSelectedFlow() (Model, tea.Cmd) {
	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 || m.flows.sel < 0 || m.flows.sel >= len(descriptors) {
		return m, nil
	}
	d := descriptors[m.flows.sel]
	return m.deleteFlow(d.Name, d.Origin)
}

func (m Model) deleteFlow(name string, origin flow.Origin) (Model, tea.Cmd) {
	if origin == flow.OriginBuiltin {
		return m.say("los flujos de serie no se pueden borrar"), nil
	}
	m.flows.confirmDelete = true
	return m.say("¿Borrar flujo " + name + "? [y] sí / [n] no"), nil
}

func (m Model) confirmDeleteFlow() (Model, tea.Cmd) {
	st := &m.flows
	st.confirmDelete = false
	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 || st.sel < 0 || st.sel >= len(descriptors) {
		return m, nil
	}
	d := descriptors[st.sel]
	if d.Origin == flow.OriginBuiltin {
		return m.say("los flujos de serie no se pueden borrar"), nil
	}
	dir := ""
	if m.opts.Flows != nil {
		dir = m.opts.Flows.FlowDir()
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".orbit", "flows")
	}
	path := filepath.Join(dir, d.Name+".json")
	if err := os.Remove(path); err != nil {
		return m.say(err.Error()), nil
	}
	if st.sel > 0 {
		st.sel--
	}
	p := m.opts.Words
	return m.say(p.T("flows.deleted", "flow {name} deleted", about("name", d.Name))), nil
}
