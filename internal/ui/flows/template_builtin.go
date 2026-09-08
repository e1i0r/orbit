package flows

// The flows the designer offers as a starting point, written out.
//
// They are here rather than beside the designer's own keys because they are
// content and not behaviour: a phase list with its prompts is read when
// somebody wants to know what the template does, and edited when the advice
// in it has gone stale. What happens when one is chosen is in flowstpl.go.

import (
	"github.com/e1i0r/orbit/internal/flow"
)

// applyFlowTemplate fills the builder from one of the presets.
//
// Every branch ends at loadedTemplate, and that is the point of it. A
// template that neither says it has loaded nor puts the cursor back on the
// first phase — Turbo Fix is one phase, pressed after a three-phase preset —
// leaves the band showing the previous template's sentence and the phase
// tabs pointing past the end of the flow.
func (s State) applyFlowTemplate(tpl string, e Env) (State, Out) {
	p := e.Words

	switch tpl {
	case "TDD Fuzz & PR":
		s.flowName = "tdd-fuzz-pr"
		s.description = "Rigorous 3-step test-driven workflow with native Go fuzzing, property invariants (>=90% coverage), and automated PR creation on success."
		s.phases = []flow.Phase{
			{
				Name:        "1-plan",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "high",
				Thinking:    "adaptive",
				Prompt:      "Analyze the task, architecture constraints (<300 lines/file), and design the test matrix with property invariants and edge cases.",
				Permissions: []string{"repo"},
			},
			{
				Name:        "2-implement-fuzz",
				Engine:      "claude",
				Model:       "sonnet",
				Effort:      "high",
				Thinking:    "adaptive",
				FeedOutput:  true,
				Prompt:      "Implement the feature and write unit tests, property tests, and Go fuzz tests (testing.F) achieving >=90% test coverage. Verify with make check.",
				Permissions: []string{"repo"},
			},
			{
				Name:        "3-review-pr",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "high",
				Thinking:    "adaptive",
				FeedOutput:  true,
				Wait:        true,
				Prompt:      "Review final diff, ensure zero lint errors, commit to branch orbit/<ID>, push to origin, and create a GitHub PR with gh pr create.",
				Permissions: []string{"repo"},
			},
		}

		return s.loadedTemplate(tpl, e)
	case "TDD Cycle":
		s.flowName = "tdd-cycle"
		s.description = "TDD cycle: 1. plan technical design, 2. implement unit tests and code, 3. review diff with human gate."
		s.phases = []flow.Phase{
			{
				Name:        "1-plan",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "high",
				Thinking:    "on",
				Prompt:      "Analiza el problema y diseña el plan técnico.",
				Permissions: []string{"read"},
			},
			{
				Name:        "2-implement",
				Engine:      "claude",
				Model:       "sonnet",
				Effort:      "high",
				FeedOutput:  true,
				Prompt:      "Implementa el código y pruebas unitarias.",
				Permissions: []string{"repo"},
			},
			{
				Name:        "3-review",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "max",
				Thinking:    "on",
				FeedOutput:  true,
				Wait:        true,
				Prompt:      "Audita el diff final y valida los chequeos.",
				Permissions: []string{"repo"},
			},
		}

		return s.loadedTemplate(tpl, e)
	case "Security Audit":
		s.flowName = "security-audit"
		s.description = "Security audit: 1. investigate repository vulnerabilities, 2. apply remediation patches."
		s.phases = []flow.Phase{
			{
				Name:        "1-investigate",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "max",
				Thinking:    "on",
				Prompt:      "Inspecciona el repositorio por vulnerabilidades.",
				Permissions: []string{"read"},
			},
			{
				Name:        "2-remediate",
				Engine:      "claude",
				Model:       "opus",
				Effort:      "high",
				FeedOutput:  true,
				Prompt:      "Aplica parches para los hallazgos.",
				Permissions: []string{"repo"},
			},
		}

		return s.loadedTemplate(tpl, e)
	case "Turbo Fix":
		s.flowName = "turbo-fix"
		s.description = "Fast single-shot direct execution with sonnet and high effort."
		s.phases = []flow.Phase{
			{
				Name:        "1-implement",
				Engine:      "claude",
				Model:       "sonnet",
				Effort:      "high",
				Prompt:      "Resuelve la tarea de forma directa.",
				Permissions: []string{"repo"},
			},
		}

		return s.loadedTemplate(tpl, e)
	case "ninguna":
		s.description = ""
		s.phases = []flow.Phase{
			{
				Name:        "1-implement",
				Engine:      "claude",
				Model:       "sonnet",
				Effort:      "default",
				Thinking:    "adaptive",
				Prompt:      "",
				Permissions: []string{"repo"},
			},
		}
		s.activePhase = 0

		return s, Out{Said: p.T("flows.template_blank", "blank flow (1 phase)")}
	}

	return s, Out{}
}
