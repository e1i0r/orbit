package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) applyFlowTemplate(tpl string) (Model, tea.Cmd) {
	st := &m.flows
	switch tpl {
	case "TDD Fuzz & PR":
		st.flowName = "tdd-fuzz-pr"
		st.description = "Rigorous 3-step test-driven workflow with native Go fuzzing, property invariants (>=90% coverage), and automated PR creation on success."
		st.phases = []flow.Phase{
			{Name: "1-plan", Engine: "claude", Model: "opus", Effort: "high", Thinking: "adaptive", Prompt: "Analyze the task, architecture constraints (<300 lines/file), and design the test matrix with property invariants and edge cases.", Permissions: []string{"repo"}},
			{Name: "2-implement-fuzz", Engine: "claude", Model: "sonnet", Effort: "high", Thinking: "adaptive", FeedOutput: true, Prompt: "Implement the feature and write unit tests, property tests, and Go fuzz tests (testing.F) achieving >=90% test coverage. Verify with make check.", Permissions: []string{"repo"}},
			{Name: "3-review-pr", Engine: "claude", Model: "opus", Effort: "high", Thinking: "adaptive", FeedOutput: true, Wait: true, Prompt: "Review final diff, ensure zero lint errors, commit to branch orbit/<ID>, push to origin, and create a GitHub PR with gh pr create.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("plantilla TDD Fuzz & PR cargada (3 fases)"), nil
	case "TDD Cycle":
		st.flowName = "tdd-cycle"
		st.description = "TDD cycle: 1. plan technical design, 2. implement unit tests and code, 3. review diff with human gate."
		st.phases = []flow.Phase{
			{Name: "1-plan", Engine: "claude", Model: "opus", Effort: "high", Thinking: "on", Prompt: "Analiza el problema y diseña el plan técnico.", Permissions: []string{"read"}},
			{Name: "2-implement", Engine: "claude", Model: "sonnet", Effort: "high", FeedOutput: true, Prompt: "Implementa el código y pruebas unitarias.", Permissions: []string{"repo"}},
			{Name: "3-review", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", FeedOutput: true, Wait: true, Prompt: "Audita el diff final y valida los chequeos.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("plantilla TDD Cycle cargada (3 fases)"), nil
	case "Security Audit":
		st.flowName = "security-audit"
		st.description = "Security audit: 1. investigate repository vulnerabilities, 2. apply remediation patches."
		st.phases = []flow.Phase{
			{Name: "1-investigate", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", Prompt: "Inspecciona el repositorio por vulnerabilidades.", Permissions: []string{"read"}},
			{Name: "2-remediate", Engine: "claude", Model: "opus", Effort: "high", FeedOutput: true, Prompt: "Aplica parches para los hallazgos.", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("plantilla Security Audit cargada (2 fases)"), nil
	case "Turbo Fix":
		st.flowName = "turbo-fix"
		st.description = "Fast single-shot direct execution with sonnet and high effort."
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "high", Prompt: "Resuelve la tarea de forma directa.", Permissions: []string{"repo"}},
		}
	case "ninguna":
		st.description = ""
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "default", Thinking: "adaptive", Prompt: "", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
		return m.say("flujo en blanco (1 fase)"), nil
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
		Name:        name,
		Description: strings.TrimSpace(st.description),
		Phases:      st.phases,
	}
	if err := fl.Validate(); err != nil {
		return m.say(err.Error()), nil
	}

	dir := ""
	if m.opts.Flows != nil {
		dir = m.opts.Flows.FlowDir()
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME")
		}
		dir = filepath.Join(home, ".orbit", "flows")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return m.say(err.Error()), nil
	}
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
	m.flows.description = ""
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

func (m Model) editNamedFlow(name string) (Model, tea.Cmd) {
	return m.editFlow(name)
}

func (m Model) editFlow(name string) (Model, tea.Cmd) {
	fl, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return m.say(err.Error()), nil
	}
	m.flows.creating = true
	m.flows.isEditing = true
	m.flows.showingDetail = false
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = fl.Name
	m.flows.description = fl.Description
	m.flows.phases = fl.Phases
	m.flows.activePhase = 0
	m.flows.ensurePhase()
	p := m.opts.Words
	return m.say(p.T("flows.editing", "editing flow {name}", about("name", fl.Name))), nil
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
	p := m.opts.Words
	if origin == flow.OriginBuiltin {
		return m.say(p.T("flows.cannot_delete_builtin", "built-in flows cannot be deleted")), nil
	}
	m.flows.confirmDelete = true
	return m.say(p.T("flows.confirm_delete", "delete flow {name}? [y] yes / [n] no", about("name", name))), nil
}

func (m Model) confirmDeleteFlow() (Model, tea.Cmd) {
	st := &m.flows
	st.confirmDelete = false
	descriptors := flow.List(m.opts.Flows)
	if len(descriptors) == 0 || st.sel < 0 || st.sel >= len(descriptors) {
		return m, nil
	}
	d := descriptors[st.sel]
	p := m.opts.Words
	if d.Origin == flow.OriginBuiltin {
		return m.say(p.T("flows.cannot_delete_builtin", "built-in flows cannot be deleted")), nil
	}
	dir := ""
	if m.opts.Flows != nil {
		dir = m.opts.Flows.FlowDir()
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME")
		}
		dir = filepath.Join(home, ".orbit", "flows")
	}
	path := filepath.Join(dir, d.Name+".json")
	if err := os.Remove(path); err != nil {
		return m.say(err.Error()), nil
	}
	if st.sel > 0 {
		st.sel--
	}
	return m.say(p.T("flows.deleted", "flow {name} deleted", about("name", d.Name))), nil
}

func (m Model) handleFlowClick(t Target) (tea.Model, tea.Cmd) {
	p := m.opts.Words
	switch t.Field {
	case "create":
		return m.startCreateFlow(), nil
	case "details":
		return m.openFlowPreview(t.ID), nil
	case "edit":
		return m.editFlow(t.ID)
	case "delete":
		return m.deleteFlow(t.ID, flow.OriginUser)
	case "detail_select":
		if m.flows.fromScreen == screenCompose {
			m.compose.setFlow(m.flows.flowName)
			return m.abandonFlows(), nil
		}
		m.flows.showingDetail = false
		return m, nil
	case "detail_back":
		if m.flows.fromScreen == screenCompose {
			return m.abandonFlows(), nil
		}
		m.flows.showingDetail = false
		return m, nil
	case "paste_prompt":
		txt := strings.TrimSpace(readClipboard())
		if txt != "" {
			m.flows.cur().Prompt = txt
			m.flows.field = flowFieldPrompt
			return m.say(p.T("flows.paste_done", "📋 pasted {n} chars into phase {phase}",
				about("n", strconv.Itoa(len(txt))), about("phase", m.flows.cur().Name))), nil
		}
		return m.say(p.T("flows.clipboard_empty", "clipboard empty")), nil
	case "autogen_prompt":
		cur := m.flows.cur()
		draft := cur.Prompt
		cur.Prompt = generatePhasePrompt(draft, cur.Name, m.flows.flowName)
		m.flows.field = flowFieldPrompt
		if draft != "" {
			return m.say(p.T("flows.autogen_custom",
				"✨ prompt generated from your draft for phase {phase}",
				about("phase", cur.Name))), nil
		}
		return m.say(p.T("flows.autogen_role",
			"✨ prompt generated for role in phase {phase}",
			about("phase", cur.Name))), nil
	case "clear_prompt":
		m.flows.cur().Prompt = ""
		m.flows.field = flowFieldPrompt
		return m.say(p.T("flows.prompt_cleared",
			"🗑 prompt cleared for phase {phase}",
			about("phase", m.flows.cur().Name))), nil
	case "add_phase":
		m.flows.field = flowFieldAddPhase
		return m.handleFlowFieldAction()
	case "del_phase":
		m.flows.field = flowFieldDelPhase
		return m.handleFlowFieldAction()
	case "save":
		m.flows.field = flowFieldSave
		return m.handleFlowFieldAction()
	case "select_phase":
		m.flows.activePhase = t.Phase
		m.flows.field = flowFieldPhaseSelect
		cur := m.flows.cur()
		return m.say(p.T("flows.phase_selected",
			"phase {n} selected: {phase} ({engine}/{model})",
			about("n", strconv.Itoa(t.Phase+1)), about("phase", cur.Name),
			about("engine", cur.Engine), about("model", cur.Model))), nil
	}
	m.flows.field = t.Phase
	return m.handleFlowFieldAction()
}
