package task

// Whether a flow can be walked at all, asked before anything is written.

import (
	"fmt"
	"slices"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// runnable answers whether this flow can be walked with these engines, and
// says what is wrong when it cannot.
//
// It is one question and it lives in one place, apart from Run, because it
// is not part of walking a flow: it is what has to be true before walking
// one is worth starting. Run is the interpreter and was carrying sixty lines
// of this before its file reached the size ceiling — the ceiling doing its
// job.
//
// Everything here is a fact about the flow and the registry and nothing
// about the state root, so it is answerable before a worktree exists, before
// a branch is cut, and before a model is billed for a phase whose second
// phase names an engine nobody configured. That is the whole point of asking
// it up front rather than at the phase that would trip over it.
func runnable(f flow.Flow, engines map[string]engine.Engine) error {
	if err := f.Validate(); err != nil {
		return err
	}

	for _, p := range f.Phases {
		eng, ok := engines[p.Engine]
		if !ok {
			return fmt.Errorf("phase %q wants the engine %q, which is not configured", p.Name, p.Engine)
		}

		if err := dialsFit(p, eng); err != nil {
			return err
		}
	}

	return nil
}

// dialsFit checks the three settings a phase may name against what its
// engine actually has.
//
// The empty catalogue is not checked for models, and that is the difference
// between the first check and the two below it. Models() empty means an
// engine that does not publish its names, not one that has none — so
// whatever the phase named goes through to the command line and the engine
// answers for it. Efforts() empty and CanThink() false are claims about a
// dial the engine does not have at all, which is why those two refuse.
func dialsFit(p flow.Phase, eng engine.Engine) error {
	if p.Model != "" && len(eng.Models()) > 0 && !offers(eng.Models(), p.Model) {
		return fmt.Errorf("phase %q names model %q, which engine %q does not offer", p.Name, p.Model, p.Engine)
	}

	if p.Effort != "" {
		efforts := eng.Efforts()
		if len(efforts) == 0 {
			return fmt.Errorf("phase %q names effort %q, but engine %q has no effort dial", p.Name, p.Effort, p.Engine)
		}

		if !offers(efforts, p.Effort) {
			return fmt.Errorf("phase %q names effort %q, which engine %q does not offer", p.Name, p.Effort, p.Engine)
		}
	}

	if p.Thinking != "" && !eng.CanThink() {
		return fmt.Errorf("phase %q configures thinking %q, but engine %q does not support thinking mode", p.Name, p.Thinking, p.Engine)
	}

	return nil
}

// offers reports whether a dial has a position with this id.
func offers(choices []engine.Choice, id string) bool {
	return slices.ContainsFunc(choices, func(c engine.Choice) bool { return c.ID == id })
}
