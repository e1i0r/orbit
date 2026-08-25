package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
	case "ninguna":
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
		home, _ := os.UserHomeDir() //nolint:errcheck // fallback to home path
		dir = filepath.Join(home, ".orbit", "flows")
	}
	_ = os.MkdirAll(dir, 0755) //nolint:errcheck // directory creation before write
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
	m.flows.isEditing = true
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = fl.Name
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
		home, _ := os.UserHomeDir() //nolint:errcheck // fallback to home path
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
	if t.Field == "create" {
		return m.startCreateFlow(), nil
	}
	if t.Field == "edit" {
		return m.editFlow(t.ID)
	}
	if t.Field == "delete" {
		return m.deleteFlow(t.ID, flow.OriginUser)
	}
	if t.Field == "paste_prompt" {
		txt := strings.TrimSpace(readClipboard())
		if txt != "" {
			m.flows.cur().Prompt = txt
			m.flows.field = flowFieldPrompt
			return m.say(p.T("flows.paste_done", "📋 pasted {n} chars into phase {phase}", about("n", strconv.Itoa(len(txt))), about("phase", m.flows.cur().Name))), nil
		}
		return m.say(p.T("flows.clipboard_empty", "clipboard empty")), nil
	}
	if t.Field == "autogen_prompt" {
		cur := m.flows.cur()
		draft := cur.Prompt
		cur.Prompt = generatePhasePrompt(draft, cur.Name, m.flows.flowName)
		m.flows.field = flowFieldPrompt
		if draft != "" {
			return m.say(p.T("flows.autogen_custom", "✨ prompt generated from your draft for phase {phase}", about("phase", cur.Name))), nil
		}
		return m.say(p.T("flows.autogen_role", "✨ prompt generated for role in phase {phase}", about("phase", cur.Name))), nil
	}
	if t.Field == "clear_prompt" {
		m.flows.cur().Prompt = ""
		m.flows.field = flowFieldPrompt
		return m.say(p.T("flows.prompt_cleared", "🗑 prompt cleared for phase {phase}", about("phase", m.flows.cur().Name))), nil
	}
	if t.Field == "add_phase" {
		m.flows.field = flowFieldAddPhase
		return m.handleFlowFieldAction()
	}
	if t.Field == "del_phase" {
		m.flows.field = flowFieldDelPhase
		return m.handleFlowFieldAction()
	}
	if t.Field == "save" {
		m.flows.field = flowFieldSave
		return m.handleFlowFieldAction()
	}
	if t.Field == "select_phase" {
		m.flows.activePhase = t.Phase
		m.flows.field = flowFieldPhaseSelect
		cur := m.flows.cur()
		return m.say(p.T("flows.phase_selected", "phase {n} selected: {phase} ({engine}/{model})", about("n", strconv.Itoa(t.Phase+1)), about("phase", cur.Name), about("engine", cur.Engine), about("model", cur.Model))), nil
	}
	m.flows.field = t.Phase
	return m.handleFlowFieldAction()
}

func readClipboard() string {
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("pbpaste").Output(); err == nil {
			return string(out)
		}
	}
	if out, err := exec.Command("wl-paste").Output(); err == nil {
		return string(out)
	}
	if out, err := exec.Command("xclip", "-out", "-selection", "clipboard").Output(); err == nil {
		return string(out)
	}
	return ""
}

func generatePhasePrompt(userInput, phaseName, flowName string) string {
	raw := strings.TrimSpace(userInput)
	lower := strings.ToLower(raw)
	phLower := strings.ToLower(phaseName)

	if raw != "" {
		switch {
		case strings.Contains(lower, "valid") || strings.Contains(lower, "test") || strings.Contains(lower, "prob") || strings.Contains(lower, "verif") || strings.Contains(lower, "check"):
			return fmt.Sprintf("Valida exhaustivamente todo el código implementado: ejecuta las suites de pruebas automatizadas, verifica casos límite y asegura que no existan regresiones. Contexto: %s.", raw)
		case strings.Contains(lower, "sec") || strings.Contains(lower, "segur") || strings.Contains(lower, "audit") || strings.Contains(lower, "vuln"):
			//nolint:misspell // Spanish prompt template
			return fmt.Sprintf("Audita rigurosamente el código en busca de vulnerabilidades de seguridad, validación de entradas y manejo seguro de secretos. Contexto: %s.", raw)
		case strings.Contains(lower, "refactor") || strings.Contains(lower, "limp") || strings.Contains(lower, "clean") || strings.Contains(lower, "orden"):
			return fmt.Sprintf("Refactoriza y optimiza la estructura del código para máxima claridad, modularidad y rendimiento sin romper contratos existentes. Contexto: %s.", raw)
		case strings.Contains(lower, "fix") || strings.Contains(lower, "correg") || strings.Contains(lower, "repar") || strings.Contains(lower, "bug") || strings.Contains(lower, "error"):
			return fmt.Sprintf("Investiga la causa raíz de los errores reportados, aplica las correcciones necesarias y verifica que pasen todos los chequeos. Contexto: %s.", raw)
		case strings.Contains(lower, "doc") || strings.Contains(lower, "coment") || strings.Contains(lower, "readme"):
			return fmt.Sprintf("Genera documentación técnica detallada, clara y concisa explicando la arquitectura, configuración y ejemplos de uso. Contexto: %s.", raw)
		default:
			return fmt.Sprintf("Ejecuta con máxima precisión técnica la siguiente instrucción para la fase %s: %s. Respeta las directivas de arquitectura y calidad.", phaseName, raw)
		}
	}

	switch {
	case strings.Contains(phLower, "plan") || strings.Contains(phLower, "design") || strings.Contains(phLower, "arch"):
		return "Analiza en detalle los requisitos, examina el código existente y diseña un plan técnico estructurado con casos de prueba."
	case strings.Contains(phLower, "impl") || strings.Contains(phLower, "code") || strings.Contains(phLower, "dev") || strings.Contains(phLower, "build"):
		return "Implementa la solución completa siguiendo el plan acordado, asegurando calidad de código, modularidad y buenas prácticas."
	case strings.Contains(phLower, "test") || strings.Contains(phLower, "gate") || strings.Contains(phLower, "check") || strings.Contains(phLower, "qa"):
		return "Escribe y ejecuta pruebas automatizadas completas para verificar exhaustivamente la funcionalidad implementada."
	case strings.Contains(phLower, "review") || strings.Contains(phLower, "audit") || strings.Contains(phLower, "sec"):
		return "Audita el diff de cambios generados, buscando posibles vulnerabilidades, fugas de recursos o regresiones."
	case strings.Contains(phLower, "fix") || strings.Contains(phLower, "patch") || strings.Contains(phLower, "remed"):
		return "Corrige con precisión los errores y hallazgos reportados en la fase anterior hasta dejar el sistema impecable."
	default:
		if flowName != "" {
			return fmt.Sprintf("Ejecuta la fase %s para el flujo %s con máxima rigurosidad técnica.", phaseName, flowName)
		}
		return fmt.Sprintf("Ejecuta las tareas correspondientes a la fase %s de forma autónoma y precisa.", phaseName)
	}
}
