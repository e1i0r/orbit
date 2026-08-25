package task

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// The two events that bracket a phase.
//
// They live beside each other, in their own file, because they are one
// decision seen from two ends: what a reader is owed about a phase before it
// runs, and what they are owed after it stopped. They were built inline in
// run.go until the file reached the size ceiling, which is the size ceiling
// doing its job — run.go is the interpreter that walks a flow, and the shape
// of an event is not that.

// phaseStart is the event that opens a phase, and the only place a run says
// what the phase was allowed to touch.
//
// The posture goes into the record rather than being left to the flow file
// because the log is the only account of a run that outlives it, and
// "the flow file said so at the time" is not an account. The names are joined
// with a comma and no space so the value stays one field of one line that a
// reader can grep for; the key is left out entirely when the phase asked for
// nothing, which is how the ending events already treat a session id and a
// cost they do not have. An empty string would read as a posture somebody
// wrote down, and nobody did.
func phaseStart(p flow.Phase, n int, notes []string) record.Event {
	data := map[string]string{"engine": p.Engine, "n": strconv.Itoa(n)}
	if p.Model != "" {
		data["model"] = p.Model
	}
	if p.Effort != "" {
		data["effort"] = p.Effort
	}
	if p.Thinking != "" {
		data["thinking"] = p.Thinking
	}
	if len(p.Permissions) > 0 {
		data["permissions"] = strings.Join(p.Permissions, ",")
	}
	if len(notes) > 0 {
		data["notes"] = strconv.Itoa(len(notes))
	}
	return record.Event{Kind: record.PhaseStarted, Phase: p.Name, Data: data}
}

// phaseEnd is the one event that ends a phase, whichever way it ended.
//
// The three endings carry the same facts because the same things are true of
// all three: the engine printed something, it may have cost money, it may
// have a session somebody wants to resume. Only the kind differs, and cause
// — the reason a phase that did not finish stopped.
//
// That the ending carries the output at all is a fix. run.go bound the
// engine's answer and then threw it away on the error path, so every failed
// or cancelled phase lost everything the agent had printed before it died,
// which is the case where a reader most wants it: on a cancellation it is
// the only evidence of what the run did before it was stopped. Claude.Run
// returns its captured stdout alongside its error precisely so this can keep
// it (claude.go:45-51).
func phaseEnd(kind, phase string, out engine.Result, cause error) record.Event {
	text, full := captured(out.Output)
	e := record.Event{Kind: kind, Phase: phase, Text: text}
	data := map[string]string{}
	if full > 0 {
		data["output_bytes"] = strconv.Itoa(full)
	}
	if out.SessionID != "" {
		data["session"] = out.SessionID
	}
	if out.Cost != 0 {
		data["cost"] = strconv.FormatFloat(out.Cost, 'f', -1, 64)
	}
	if cause != nil {
		// Why it stopped goes in Data rather than Text, because Text is now
		// what the engine printed, and a log that ends at phase.failed — it
		// can, the write after it is best-effort — must still say why. It is
		// cut to the same length for the same reason: one event is one line
		// and record.MaxLine is what a line may weigh, and an engine's error
		// can carry the whole of its stderr.
		msg, _ := captured(cause.Error())
		data["error"] = msg
	}
	if len(data) > 0 {
		e.Data = data
	}
	return e
}

func phaseThought(phase string, n int, text string) record.Event {
	c, full := captured(text)
	data := map[string]string{"n": strconv.Itoa(n)}
	if full > 0 {
		data["bytes"] = strconv.Itoa(full)
	}
	return record.Event{Kind: record.PhaseThought, Phase: phase, Text: c, Data: data}
}

func phaseToolCall(phase string, n int, tc engine.StreamToolCall) record.Event {
	c, full := captured(tc.Args)
	data := map[string]string{
		"n":    strconv.Itoa(n),
		"tool": tc.Name,
		"args": c,
	}
	if full > 0 {
		data["bytes"] = strconv.Itoa(full)
	}
	return record.Event{Kind: record.PhaseToolCall, Phase: phase, Text: c, Data: data}
}

func phaseRefused(phase string, n int, r engine.StreamRefusal) record.Event {
	c, full := captured(r.Input)
	data := map[string]string{
		"n":    strconv.Itoa(n),
		"tool": r.Tool,
	}
	if full > 0 {
		data["bytes"] = strconv.Itoa(full)
	}
	return record.Event{Kind: record.PhaseRefused, Phase: phase, Text: c, Data: data}
}

func runGates(ctx context.Context, s *store.Store, t Task, p flow.Phase, n int, wt string, out engine.Result) error {
	for _, g := range p.Gates {
		cmd := exec.CommandContext(ctx, "sh", "-c", g.Command)
		cmd.Dir = wt
		combined, err := cmd.CombinedOutput()
		text, full := captured(string(combined))
		data := map[string]string{
			"gate": g.Name,
			"n":    strconv.Itoa(n),
		}
		if full > 0 {
			data["bytes"] = strconv.Itoa(full)
		}
		if err == nil {
			data["exit"] = "0"
			if emitErr := emit(s, t, record.Event{Kind: record.GatePassed, Phase: p.Name, Text: text, Data: data}); emitErr != nil {
				return emitErr
			}
			continue
		}

		exitCode := 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		data["exit"] = strconv.Itoa(exitCode)
		_ = emit(s, t, record.Event{Kind: record.GateFailed, Phase: p.Name, Text: text, Data: data}) //nolint:errcheck // best-effort event emission on gate failure
		gateCause := fmt.Errorf("gate %q failed (exit %d)", g.Name, exitCode)
		_ = emit(s, t, phaseEnd(record.PhaseFailed, p.Name, out, gateCause)) //nolint:errcheck // best-effort event emission on gate failure
		return failed(s, t, fmt.Errorf("task %s, phase %q: %w", t.ID, p.Name, gateCause))
	}
	return nil
}
