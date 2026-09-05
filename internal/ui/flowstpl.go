package ui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/flow"
)

// applyFlowTemplate fills the builder from one of the presets.
//
// Every branch ends at loadedTemplate, and that is the point of it. A
// template that neither says it has loaded nor puts the cursor back on the
// first phase — Turbo Fix is one phase, pressed after a three-phase preset —
// leaves the band showing the previous template's sentence and the phase
// tabs pointing past the end of the flow.
func (m Model) applyFlowTemplate(tpl string) (Model, tea.Cmd) {
	st := &m.flows
	p := m.opts.Words

	switch tpl {
	case "TDD Fuzz & PR":
		st.flowName = "tdd-fuzz-pr"
		st.description = "Rigorous 3-step test-driven workflow with native Go fuzzing, property invariants (>=90% coverage), and automated PR creation on success."
		st.phases = []flow.Phase{
			{Name: "1-plan", Engine: "claude", Model: "opus", Effort: "high", Thinking: "adaptive", Prompt: "Analyze the task, architecture constraints (<300 lines/file), and design the test matrix with property invariants and edge cases.", Permissions: []string{"repo"}},
			{Name: "2-implement-fuzz", Engine: "claude", Model: "sonnet", Effort: "high", Thinking: "adaptive", FeedOutput: true, Prompt: "Implement the feature and write unit tests, property tests, and Go fuzz tests (testing.F) achieving >=90% test coverage. Verify with make check.", Permissions: []string{"repo"}},
			{Name: "3-review-pr", Engine: "claude", Model: "opus", Effort: "high", Thinking: "adaptive", FeedOutput: true, Wait: true, Prompt: "Review final diff, ensure zero lint errors, commit to branch orbit/<ID>, push to origin, and create a GitHub PR with gh pr create.", Permissions: []string{"repo"}},
		}

		return m.loadedTemplate(tpl), nil
	case "TDD Cycle":
		st.flowName = "tdd-cycle"
		st.description = "TDD cycle: 1. plan technical design, 2. implement unit tests and code, 3. review diff with human gate."
		st.phases = []flow.Phase{
			{Name: "1-plan", Engine: "claude", Model: "opus", Effort: "high", Thinking: "on", Prompt: "Analiza el problema y diseña el plan técnico.", Permissions: []string{"read"}},
			{Name: "2-implement", Engine: "claude", Model: "sonnet", Effort: "high", FeedOutput: true, Prompt: "Implementa el código y pruebas unitarias.", Permissions: []string{"repo"}},
			{Name: "3-review", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", FeedOutput: true, Wait: true, Prompt: "Audita el diff final y valida los chequeos.", Permissions: []string{"repo"}},
		}

		return m.loadedTemplate(tpl), nil
	case "Security Audit":
		st.flowName = "security-audit"
		st.description = "Security audit: 1. investigate repository vulnerabilities, 2. apply remediation patches."
		st.phases = []flow.Phase{
			{Name: "1-investigate", Engine: "claude", Model: "opus", Effort: "max", Thinking: "on", Prompt: "Inspecciona el repositorio por vulnerabilidades.", Permissions: []string{"read"}},
			{Name: "2-remediate", Engine: "claude", Model: "opus", Effort: "high", FeedOutput: true, Prompt: "Aplica parches para los hallazgos.", Permissions: []string{"repo"}},
		}

		return m.loadedTemplate(tpl), nil
	case "Turbo Fix":
		st.flowName = "turbo-fix"
		st.description = "Fast single-shot direct execution with sonnet and high effort."
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "high", Prompt: "Resuelve la tarea de forma directa.", Permissions: []string{"repo"}},
		}

		return m.loadedTemplate(tpl), nil
	case "ninguna":
		st.description = ""
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "default", Thinking: "adaptive", Prompt: "", Permissions: []string{"repo"}},
		}
		st.activePhase = 0

		return m.say(p.T("flows.template_blank", "blank flow (1 phase)")), nil
	}

	return m, nil
}

// loadedTemplate puts the cursor on the first phase and says which preset
// is now in the builder and how long it is.
func (m Model) loadedTemplate(name string) Model {
	m.flows.activePhase = 0
	n := len(m.flows.phases)

	return m.say(m.opts.Words.P("flows.template_loaded", n,
		"template {name} loaded ({n} phase)",
		"template {name} loaded ({n} phases)",
		about("name", name)))
}

func (m Model) saveCustomFlow() (Model, tea.Cmd) {
	st := &m.flows

	p := m.opts.Words

	name := strings.TrimSpace(st.flowName)
	if name == "" {
		return m.say(p.T("flows.name_required", "give the flow a name")), nil
	}

	st.ensurePhase()

	if len(st.phases) == 0 {
		return m.say(p.T("flows.min_phases_required", "the flow must have at least one phase")), nil
	}

	fl := flow.Flow{
		Name:        name,
		Description: strings.TrimSpace(st.description),
		Attempts:    st.attempts,
		Phases:      st.phases,
	}

	// flow.Save, and not a second copy of it here.
	//
	// This encoded the flow, made the directory and wrote the file itself,
	// and every one of those steps disagreed with the package that owns
	// them. It never asked flow.ValidName, so a flow named ../notes was
	// filepath.Join'd straight out of the flow directory and written
	// wherever that landed. It made the directory 0755 and the file 0644
	// where internal/flow spells out 0700 and 0600 — the quiet widening
	// that package's own comment warns about, arriving from the one place
	// it could not see. And when the window was given no flow source it
	// invented ~/.orbit/flows, which is the wrong directory on any machine
	// with $ORBIT_HOME set: the flow saved, said so, and was never listed
	// again. flow.Save refuses that case instead of guessing.
	if _, err := flow.Save(m.opts.Flows, fl); err != nil {
		return m.say(err.Error()), nil
	}

	m.flows.creating = false
	m.flows.phases = nil
	m.flows.flowName = ""
	m.flows.description = ""
	m.flows.refresh(m.opts.Flows)

	return m.say(p.T("flows.saved", "flow {name} saved", about("name", name))), nil
}

func (m Model) editSelectedFlow() (Model, tea.Cmd) {
	descriptors := m.flows.listed
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
	m.flows.engine = m.dialEngine("")
	m.flows.isEditing = true
	m.flows.showingDetail = false
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = fl.Name
	m.flows.description = fl.Description
	m.flows.phases = fl.Phases
	m.flows.attempts = fl.Attempts
	m.flows.activePhase = 0
	// Editing opens on the fields: this flow already exists, and the tab
	// that writes one from a sentence would replace every phase of it.
	m.flows.tab = flowTabFields
	m.flows.scroll = 0
	m.flows.ensurePhase()
	p := m.opts.Words

	return m.say(p.T("flows.editing", "editing flow {name}", about("name", fl.Name))), nil
}

func (m Model) deleteSelectedFlow() (Model, tea.Cmd) {
	descriptors := m.flows.listed
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

	descriptors := m.flows.listed
	if len(descriptors) == 0 || st.sel < 0 || st.sel >= len(descriptors) {
		return m, nil
	}

	d := descriptors[st.sel]

	p := m.opts.Words
	if d.Origin == flow.OriginBuiltin {
		return m.say(p.T("flows.cannot_delete_builtin", "built-in flows cannot be deleted")), nil
	}

	// flow.Delete, for the reasons saveCustomFlow gives, and for one more
	// of its own: it says whether a built-in of that name was underneath.
	// Removing a shadow does not remove the flow, it restores the shipped
	// one, and a task written against that name goes on running —
	// differently. os.Remove cannot report that, so the window said
	// "deleted" and left the reader to find out by running it.
	revealed, err := flow.Delete(m.opts.Flows, d.Name)
	if err != nil {
		return m.say(err.Error()), nil
	}

	if st.sel > 0 {
		st.sel--
	}

	st.refresh(m.opts.Flows)

	if revealed {
		return m.say(p.T("flows.deleted_revealed",
			"flow {name} deleted; the built-in of that name is showing again",
			about("name", d.Name))), nil
	}

	return m.say(p.T("flows.deleted", "flow {name} deleted", about("name", d.Name))), nil
}

// pastedPrompt puts what was on the clipboard into the phase being edited,
// and says how much of it arrived.
//
// The count is of characters. It was len(), which is bytes: a paragraph of
// Spanish was reported a third longer than it is, and one emoji as four
// characters. Nobody counts a prompt to check, which is exactly why the
// number has to be right.
func (m Model) pastedPrompt(txt string) Model {
	p := m.opts.Words
	if txt == "" {
		return m.say(p.T("flows.clipboard_empty", "clipboard empty"))
	}

	m.flows.cur().Prompt = txt
	m.flows.field = flowFieldPrompt

	return m.say(p.T("flows.paste_done", "📋 pasted {n} chars into phase {phase}",
		about("n", strconv.Itoa(utf8.RuneCountInString(txt))),
		about("phase", m.flows.cur().Name)))
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
		return m.pastedPrompt(strings.TrimSpace(readClipboard())), nil
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
	case "say_engine":
		return m.turnSayEngine(1), nil
	case "draft":
		next, cmd := m.draftFlow()
		return next, cmd
	case "tab":
		m.flows.tab = t.Phase
		m.flows.scroll = 0

		return m, nil
	case "pick":
		return m.takePick(t.Phase), nil
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
