package task

// What a run says about itself in the log, as opposed to in the record.
//
// The record is the account every reader of Orbit agrees on, and it says
// what happened to the task. The log is narrower and cheaper: it says what
// happened to the process, in time order, in one file, next to whatever else
// on this machine was going wrong at the same moment. The two are worth
// keeping apart — a task's record is read by a reader who already knows
// which task they are asking about, and the log is read by one who does not
// know that yet.
//
// Every line here comes off emit, which is the one funnel every event in
// this package goes through. Nothing in run.go, gate.go or reconcile.go
// writes a line of its own, so a kind added later is logged the day it is
// recorded and nobody has to remember to do it twice.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
)

// streamed is the three kinds a phase writes while the model is talking.
// They are the whole of a model's stream — hundreds of events for one phase
// — and the record already has them, whole. In the log they would bury the
// twenty lines that say what became of the run, which is what the log is
// for.
var streamed = map[string]bool{
	record.PhaseThought:  true,
	record.PhaseToolCall: true,
	record.PhaseRefused:  true,
}

// wentWrong is every kind that also belongs in the errors file. Cancelled is
// deliberately not one of them: a reader who typed cancel got what they
// asked for, and a file of what went wrong is worth less for every line in
// it that nothing went wrong about.
var wentWrong = map[string]bool{
	record.TaskFailed:    true,
	record.PhaseFailed:   true,
	record.TaskTimedOut:  true,
	record.TaskAbandoned: true,
	record.GateFailed:    true,
	record.TaskStuck:     true,
}

// noteEvent writes down an event that reached the record. It is called after
// the append and not before it, so the log holds what the record holds
// rather than what this process meant to write.
func noteEvent(id string, e record.Event) {
	if streamed[e.Kind] {
		return
	}

	where := ""
	if e.Phase != "" {
		where = " in phase " + e.Phase
	}

	switch {
	case wentWrong[e.Kind]:
		logger.Error("task/run", "%s: %s%s: %s", id, e.Kind, where, oneLine(e))
	case e.Kind == record.TaskCancelled || e.Kind == record.PhaseCancelled:
		logger.Warn("task/run", "%s: %s%s", id, e.Kind, where)
	default:
		logger.Info("task/run", "%s: %s%s", id, e.Kind, where)
	}
}

// oneLine is as much of a failure as belongs on one line of the log.
//
// phase.failed keeps why it stopped in Data["error"] and what the engine
// printed in Text, so the reason is asked for first and the output is the
// fallback — the other way round gives a line of the model's prose and no
// error at all. The whole of both is in the record, which is where a reader
// who needs the rest goes; this is for recognising the failure in a file
// that has everything else on the machine in it too.
func oneLine(e record.Event) string {
	text := e.Data["error"]
	if text == "" {
		text = e.Text
	}

	text = strings.TrimSpace(text)
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}

	if r := []rune(text); len(r) > noteWidth {
		text = string(r[:noteWidth])
	}

	return text
}

// noteWidth is how much of a failure one line carries. It is a bound and not
// a judgement about what matters: the record holds the whole of it, and a
// log entry wide enough to wrap in a terminal is one nobody skims.
const noteWidth = 200

// noted writes a failure down and hands it back, so that a caller can log
// and return in the one line the caller already had.
func noted(id string, err error) error {
	logger.Error("task/run", "%s: %v", id, err)

	return err
}
