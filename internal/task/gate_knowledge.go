package task

// A fact that stops the work is a gate.
//
// This is where "stops" stops being a word in a prompt. A sentence in the
// context is advice: the model weighs it against everything else it was told,
// and sometimes it loses. A gate is a command that runs after the phase and
// sends the work back, and it needs no trust at all — it exits zero or it
// does not.
//
// Which is why a fact that asks to stop and brings no check is only ever a
// warning (knowledge.Fact.Action says so): there is nothing to run, so there
// is nothing to enforce, and pretending otherwise would put a rule in front
// of a reader that never fires.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
)

// gateName is how much of a fact's sentence names its gate. Long enough for
// the sentence to be recognised, short enough to sit in a heading.
const gateName = 90

// gatesOf is every gate that runs after a phase: the ones its flow declares,
// and then the standing rules.
//
// The phase's own come first because they are about the work it was asked to
// do — a build that does not compile is the more useful thing to be told
// about than a rule that was true before the task started, and the first
// refusal is the one the next attempt reads.
func gatesOf(p flow.Phase, knows []knowledge.Fact) []flow.Gate {
	return append(append([]flow.Gate{}, p.Gates...), knowledgeGates(knows)...)
}

// knowledgeGates turns the facts that can stop the work into gates.
//
// The scope is not consulted here, and that is deliberate: a check carries
// its own. "no diff under backend/ledger contains UPDATE ledger" says which
// paths it cares about inside the command, and on a task that never touched
// them it passes without being asked to. A second filter here would be a
// second opinion about where a rule applies, in a place that cannot see the
// diff the check can.
func knowledgeGates(knows []knowledge.Fact) []flow.Gate {
	var gates []flow.Gate

	for _, f := range knows {
		if f.Off || f.Action() != knowledge.Stops {
			continue
		}

		gates = append(gates, flow.Gate{Name: gateNamed(f), Command: f.Check})
	}

	return gates
}

// gateNamed is what the gate is called, which is the fact's own sentence.
//
// The refusal a later attempt reads names the gate and shows what it printed,
// and a check like `! git diff | grep -q X` prints nothing at all when it
// fails. So the name has to carry the meaning: "gate `No UPDATE or DELETE in
// ledger` refused it" tells a model what it broke, where "gate `rule-7`
// refused it, exit 1" is a wall with no sign on it.
func gateNamed(f knowledge.Fact) string {
	phrase := strings.TrimSpace(strings.SplitN(f.Phrase, "\n", 2)[0])
	if len(phrase) > gateName {
		phrase = strings.TrimSpace(phrase[:gateName]) + "…"
	}

	return phrase
}
