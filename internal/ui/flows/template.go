package flows

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/prompt"
)

// loadedTemplate puts the cursor on the first phase and says which preset
// is now in the builder and how long it is.
func (s State) loadedTemplate(name string, e Env) (State, Out) {
	s.activePhase = 0
	n := len(s.phases)

	return s, said(e.Words.P("flows.template_loaded", n,
		"template {name} loaded ({n} phase)",
		"template {name} loaded ({n} phases)",
		about("name", name)))
}

func (s State) saveCustomFlow(e Env) (State, Out) {
	p := e.Words

	name := strings.TrimSpace(s.flowName)
	if name == "" {
		return s, Out{Said: p.T("flows.name_required", "give the flow a name")}
	}

	s.ensurePhase()

	if len(s.phases) == 0 {
		return s, Out{Said: p.T("flows.min_phases_required", "the flow must have at least one phase")}
	}

	fl := flow.Flow{
		Name:        name,
		Description: strings.TrimSpace(s.description),
		Attempts:    s.attempts,
		Phases:      s.phases,
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
	if _, err := flow.Save(e.Flows, fl); err != nil {
		return s, Out{Said: err.Error()}
	}

	s.creating = false
	s.phases = nil
	s.flowName = ""
	s.description = ""
	s.refresh(e.Flows)

	return s, Out{Said: p.T("flows.saved", "flow {name} saved", about("name", name))}
}

func (s State) editSelectedFlow(e Env) (State, Out) {
	descriptors := s.listed
	if len(descriptors) == 0 || s.sel < 0 || s.sel >= len(descriptors) {
		return s, Out{}
	}

	d := descriptors[s.sel]

	return s.editFlow(d.Name, e)
}

func (s State) editNamedFlow(name string, e Env) (State, Out) {
	return s.editFlow(name, e)
}

func (s State) editFlow(name string, e Env) (State, Out) {
	fl, err := flow.Resolve(e.Flows, name)
	if err != nil {
		return s, Out{Said: err.Error()}
	}

	s.creating = true
	s.engine = cells.OrDef("", e.Engine)
	s.isEditing = true
	s.showingDetail = false
	s.confirmDiscard = false
	s.confirmDelete = false
	s.field = 0
	s.template = "ninguna"
	s.flowName = fl.Name
	s.description = fl.Description
	s.phases = fl.Phases
	s.attempts = fl.Attempts
	s.activePhase = 0
	// Editing opens on the fields: this flow already exists, and the tab
	// that writes one from a sentence would replace every phase of it.
	s.tab = flowTabFields
	s.scroll = 0
	s.ensurePhase()

	p := e.Words

	return s, Out{Said: p.T("flows.editing", "editing flow {name}", about("name", fl.Name))}
}

func (s State) deleteSelectedFlow(e Env) (State, Out) {
	descriptors := s.listed
	if len(descriptors) == 0 || s.sel < 0 || s.sel >= len(descriptors) {
		return s, Out{}
	}

	d := descriptors[s.sel]

	return s.deleteFlow(d.Name, d.Origin, e)
}

func (s State) deleteFlow(name string, origin flow.Origin, e Env) (State, Out) {
	p := e.Words
	if origin == flow.OriginBuiltin {
		return s, Out{Said: p.T("flows.cannot_delete_builtin", "built-in flows cannot be deleted")}
	}

	s.confirmDelete = true

	return s, Out{Said: p.T("flows.confirm_delete", "delete flow {name}? [y/n]", about("name", name))}
}

func (s State) confirmDeleteFlow(e Env) (State, Out) {
	s.confirmDelete = false

	descriptors := s.listed
	if len(descriptors) == 0 || s.sel < 0 || s.sel >= len(descriptors) {
		return s, Out{}
	}

	d := descriptors[s.sel]

	p := e.Words
	if d.Origin == flow.OriginBuiltin {
		return s, Out{Said: p.T("flows.cannot_delete_builtin", "built-in flows cannot be deleted")}
	}

	// flow.Delete, for the reasons saveCustomFlow gives, and for one more
	// of its own: it says whether a built-in of that name was underneath.
	// Removing a shadow does not remove the flow, it restores the shipped
	// one, and a task written against that name goes on running —
	// differently. os.Remove cannot report that, so the window said
	// "deleted" and left the reader to find out by running it.
	revealed, err := flow.Delete(e.Flows, d.Name)
	if err != nil {
		return s, Out{Said: err.Error()}
	}

	if s.sel > 0 {
		s.sel--
	}

	s.refresh(e.Flows)

	if revealed {
		return s, said(p.T("flows.deleted_revealed",
			"flow {name} deleted; the built-in of that name is showing again",
			about("name", d.Name)))
	}

	return s, Out{Said: p.T("flows.deleted", "flow {name} deleted", about("name", d.Name))}
}

// pastedPrompt puts what was on the clipboard into the phase being edited,
// and says how much of it arrived.
//
// The count is of characters. It was len(), which is bytes: a paragraph of
// Spanish was reported a third longer than it is, and one emoji as four
// characters. Nobody counts a prompt to check, which is exactly why the
// number has to be right.
func (s State) pastedPrompt(txt string, e Env) (State, Out) {
	p := e.Words
	if txt == "" {
		return s, Out{Said: p.T("flows.clipboard_empty", "clipboard empty")}
	}

	s.cur().Prompt = txt
	s.field = flowFieldPrompt

	return s, said(p.T("flows.paste_done", "📋 pasted {n} chars into phase {phase}",
		about("n", strconv.Itoa(utf8.RuneCountInString(txt))),
		about("phase", s.cur().Name)))
}

// Click is one click landing on what Hit named.
func (s State) Click(t point.Target, e Env) (State, Out) {
	p := e.Words

	switch t.Field {
	case "create":
		return s.startCreateFlow(e), Out{}
	case "details":
		return Preview(t.ID, s.from, e), Out{}
	case "edit":
		return s.editFlow(t.ID, e)
	case "delete":
		return s.deleteFlow(t.ID, flow.OriginUser, e)
	case "detail_select":
		if s.from == FromCompose {
			next, out := s.leave()
			out.Chose = s.flowName

			return next, out
		}

		s.showingDetail = false

		return s, Out{}
	case "detail_back":
		if s.from == FromCompose {
			return s.leave()
		}

		s.showingDetail = false

		return s, Out{}
	case "paste_prompt":
		return s.pastedPrompt(strings.TrimSpace(clip.Read()), e)
	case "autogen_prompt":
		cur := s.cur()
		draft := cur.Prompt
		cur.Prompt = prompt.Phase(draft, cur.Name, s.flowName)

		s.field = flowFieldPrompt
		if draft != "" {
			return s, said(p.T("flows.autogen_custom",
				"✨ prompt generated from your draft for phase {phase}",
				about("phase", cur.Name)))
		}

		return s, said(p.T("flows.autogen_role",
			"✨ prompt generated for role in phase {phase}",
			about("phase", cur.Name)))
	case "clear_prompt":
		s.cur().Prompt = ""
		s.field = flowFieldPrompt

		return s, said(p.T("flows.prompt_cleared",
			"🗑 prompt cleared for phase {phase}",
			about("phase", s.cur().Name)))
	case "add_phase":
		s.field = flowFieldAddPhase
		return s.handleFlowFieldAction(e)
	case "del_phase":
		s.field = flowFieldDelPhase
		return s.handleFlowFieldAction(e)
	case "save":
		s.field = flowFieldSave
		return s.handleFlowFieldAction(e)
	case "say_engine":
		s.sayFocus = sayOnEngine
		return s.openPicker(flowFieldSayEngine, e), Out{}
	case "say_model":
		s.sayFocus = sayOnModel
		return s.openPicker(flowFieldSayModel, e), Out{}
	case "draft":
		next, cmd := s.draftFlow(e)
		return next, cmd
	case "tab":
		s.tab = t.Phase
		s.scroll = 0

		return s, Out{}
	case "pick":
		return s.takePick(t.Phase, e), Out{}
	case "select_phase":
		s.activePhase = t.Phase
		s.field = flowFieldPhaseSelect
		cur := s.cur()

		return s, said(p.T("flows.phase_selected",
			"phase {n} selected: {phase} ({engine}/{model})",
			about("n", strconv.Itoa(t.Phase+1)), about("phase", cur.Name),
			about("engine", cur.Engine), about("model", cur.Model)))
	}

	s.field = t.Phase

	return s.handleFlowFieldAction(e)
}
