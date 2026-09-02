package task

// What the engine said while it was saying it, and what is written down when
// it would not stream at all.

import (
	"context"
	"fmt"

	"github.com/e1i0r/orbit/internal/engine"
)

// run puts the phase to its engine and writes down everything that comes
// back on the way.
//
// The three returns are three different things and the caller needs them
// apart: what the engine produced, the error the engine itself reported —
// which is not this function's to interpret — and an error of the record,
// which has already ended the run by the time it is handed back.
func (r phaseRun) run(ctx context.Context) (engine.Result, error, error) { //nolint:revive // the engine's error is a value here, not this function's failure
	var (
		streamErr                                             error
		streamedThoughts, streamedRefusals, streamedToolCalls int
	)

	out, runErr := r.eng.Run(ctx, engine.Request{
		Prompt:      prompt(r.task, r.phase, r.notes, r.prev, r.others, r.tried...),
		Model:       r.phase.Model,
		Effort:      r.phase.Effort,
		Thinking:    r.phase.Thinking,
		Dir:         r.wt,
		Permissions: r.phase.Permissions,
		Env:         childEnv(r.task),
		Resume:      lastSession(r.store, r.task, r.phase.Engine, r.eng),
		OnEvent: func(ev engine.StreamEvent) {
			var err error

			switch ev.Type {
			case "thought":
				streamedThoughts++
				err = emit(r.store, r.task, phaseThought(r.phase.Name, r.n, ev.Thought))
			case "tool_call":
				streamedToolCalls++
				err = emit(r.store, r.task, phaseToolCall(r.phase.Name, r.n, ev.ToolCall))
			case "refusal":
				streamedRefusals++
				err = emit(r.store, r.task, phaseRefused(r.phase.Name, r.n, ev.Refusal))
			}

			if err != nil && streamErr == nil {
				streamErr = err
			}
		},
	})
	if streamErr != nil {
		return out, nil, failed(r.store, r.task, fmt.Errorf("task %s, phase %q stream event emit: %w", r.task.ID, r.phase.Name, streamErr))
	}

	// The fallbacks are for an engine that answered in one piece rather than
	// in a stream: what it streamed is already in the record, and writing
	// the same thoughts twice would double the noisiest kinds there are.
	if err := r.fallback(streamedThoughts, streamedRefusals, streamedToolCalls, out); err != nil {
		return out, nil, err
	}

	return out, runErr, nil
}

// fallback writes down what an engine that did not stream reported at the
// end, and only the kinds it did not stream.
func (r phaseRun) fallback(thoughts, refusals, toolCalls int, out engine.Result) error {
	if thoughts == 0 {
		for _, th := range out.Thoughts {
			if err := emit(r.store, r.task, phaseThought(r.phase.Name, r.n, th)); err != nil {
				return failed(r.store, r.task, fmt.Errorf("task %s, phase %q fallback thought emit: %w", r.task.ID, r.phase.Name, err))
			}
		}
	}

	if refusals == 0 {
		for _, ref := range out.Refusals {
			if err := emit(r.store, r.task, phaseRefused(r.phase.Name, r.n, ref)); err != nil {
				return failed(r.store, r.task, fmt.Errorf("task %s, phase %q fallback refusal emit: %w", r.task.ID, r.phase.Name, err))
			}
		}
	}

	if toolCalls == 0 {
		for _, tc := range out.ToolCalls {
			if err := emit(r.store, r.task, phaseToolCall(r.phase.Name, r.n, tc)); err != nil {
				return failed(r.store, r.task, fmt.Errorf("task %s, phase %q fallback tool call emit: %w", r.task.ID, r.phase.Name, err))
			}
		}
	}

	return nil
}
