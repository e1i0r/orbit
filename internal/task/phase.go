package task

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
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
func phaseStart(p flow.Phase, n int) record.Event {
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
