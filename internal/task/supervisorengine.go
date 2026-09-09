package task

// The engine a supervisor works with, carried into what it starts.
//
// The supervisor sits above the board: a reader who dials it to agy and then
// says "fix task 2900" is asking that person to go and fix it, not asking
// the task to run itself again the way it always has. Task 2900 was written
// against claude, and claude is out of session — which is the whole reason
// the reader changed the dial. A retry that read the task's own engine put
// them straight back into the wall they had just walked around.
//
// So the name travels. SuperviseIn puts it in the environment of the engine
// it runs; the engine passes its environment to the `orbit mcp` server it
// spawns, and to any orbit command it runs itself; and StartWith reads it
// here when the caller named no engine of its own. A window, a phase or a
// person at a terminal has nothing in this variable and is untouched.
//
// It changes one attempt and not the task. The task keeps its engine, its
// flow and its history, and phase.started records the engine that actually
// ran — so the record says an attempt was made by somebody else, which is
// what happened.

import (
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
)

// supervisorEngine is the engine the supervisor asking for this is running
// on, or "" when the caller is not a supervisor. engine.SupervisorVar is the
// name both ends spell.
func supervisorEngine() string {
	return strings.TrimSpace(os.Getenv(engine.SupervisorVar))
}

// withoutSupervisorEngine is an environment with the variable taken out.
//
// A run started by a supervisor is not itself a supervisor. Left in, the
// name would travel from the run to its engine, from that engine to the mcp
// server it spawns, and a task that asked to start another task would start
// it on the supervisor's engine for reasons nobody could see from the board.
// One hop is the whole of what this is for.
func withoutSupervisorEngine(env []string) []string {
	kept := make([]string, 0, len(env))

	for _, kv := range env {
		if name, _, found := strings.Cut(kv, "="); found && name == engine.SupervisorVar {
			continue
		}

		kept = append(kept, kv)
	}

	return kept
}
