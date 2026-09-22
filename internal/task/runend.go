package task

// How a run ends when it does not end by running out of phases.
//
// They are here and not in run.go because that file is at the size ceiling
// and these two are one idea: the last thing written about an attempt that
// stopped. Both are called from several places in the walk, and neither
// decides anything about the walk itself.

import (
	"context"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// stopped writes down that a run was stopped from outside.
func stopped(s *store.Store, t Task, phase string, out engine.Result, cause error) error {
	_ = emit(s, t, phaseEnd(record.PhaseCancelled, phase, out, nil)) //nolint:errcheck

	kind := record.TaskCancelled
	if errors.Is(cause, context.DeadlineExceeded) {
		kind = record.TaskTimedOut
	}

	_ = emit(s, t, record.Event{Kind: kind}) //nolint:errcheck

	return fmt.Errorf("task %s, phase %q: %w", t.ID, phase, cause)
}

// failed writes down that the run stopped and why.
func failed(s *store.Store, t Task, err error) error {
	text, _ := captured(err.Error())
	_ = emit(s, t, record.Event{Kind: record.TaskFailed, Text: text}) //nolint:errcheck

	return err
}
