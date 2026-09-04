package task

// The six mid-phase writes, and what Run does when the log refuses one.
//
// Three of them come from the stream while the engine is still running, and
// three from the result when the engine never streamed at all. Every one of
// them fails the run rather than dropping the event, and none of them had a
// test: a phase whose thoughts and tool calls silently stopped being written
// halfway through is a record that reads as a phase that went quiet.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// unwritable is a payload the record cannot hold once JSON has escaped it.
//
// captured cuts an engine's words to a megabyte and record.MaxLine is four,
// so an honest overflow is not reachable that way -- but a megabyte of NUL
// bytes is six megabytes once every one of them is escaped. One event the log
// refuses, in a log that is otherwise perfectly writable, which is exactly
// the shape these branches are for.
func unwritable() string { return strings.Repeat("\x00", maxOutput) }

// onePhase is a flow of one phase against the named engine.
func onePhase(engineName string) flow.Flow {
	return flow.Flow{Name: "one", Phases: []flow.Phase{{Name: "implement", Engine: engineName}}}
}

// runWith walks a fresh task through one phase and hands back Run's verdict.
func runWith(t *testing.T, name string, eng engine.Engine) error {
	t.Helper()

	s, r := fixture(t)

	tk, err := Create(s, r, "STREAM-1", "a payload the record cannot hold", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return Run(context.Background(), s, tk, onePhase(name), map[string]engine.Engine{name: eng}, nil)
}

// TestAStreamedEventTheLogRefusesFailsTheRun.
//
// The engine goes on running after a stream event cannot be written -- the
// callback has nowhere to return an error to -- so the failure is held and
// answered once the phase is over. What it must not do is finish the phase
// as a success: the record would then show a phase that thought nothing and
// called no tools, and the only account of what the model actually did would
// be gone with no line anywhere saying it was ever there.
func TestAStreamedEventTheLogRefusesFailsTheRun(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event engine.StreamEvent
	}{
		{"a thought", engine.StreamEvent{Type: "thought", Thought: unwritable()}},
		{"a tool call", engine.StreamEvent{
			Type:     "tool_call",
			ToolCall: engine.StreamToolCall{Name: "Bash", Args: unwritable()},
		}},
		{"a refusal", engine.StreamEvent{
			Type:    "refusal",
			Refusal: engine.StreamRefusal{Tool: "Bash", Input: unwritable()},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eng := &engine.Fake{Output: "done", Events: []engine.StreamEvent{tc.event}}

			err := runWith(t, "fake", eng)
			if err == nil {
				t.Fatal("Run succeeded with a streamed event the log had refused")
			}

			if !strings.Contains(err.Error(), "stream event emit") {
				t.Errorf("the failure does not say a streamed event could not be written: %v", err)
			}
		})
	}
}

// quiet answers with a whole result and streams nothing, which is every
// engine that has no streaming mode and any engine whose stream was not
// parsed. Run writes those out afterwards instead, and those writes can fail
// the same way.
type quiet struct{ res engine.Result }

func (quiet) Name() string                                        { return "quiet" }
func (quiet) CanResume() bool                                     { return false }
func (quiet) CanThink() bool                                      { return false }
func (quiet) Transcript(string, time.Time) ([]engine.Turn, error) { return nil, nil }
func (quiet) Models() []engine.Choice                             { return nil }
func (quiet) Efforts() []engine.Choice                            { return nil }
func (quiet) Locate() (string, error)                             { return "quiet", nil }

func (q quiet) Run(context.Context, engine.Request) (engine.Result, error) {
	return q.res, nil
}

// TestAnUnstreamedEventTheLogRefusesFailsTheRun is the same rule on the other
// path in, and it is the one that decides it for engines that do not stream:
// the result is the only account there is, so losing a line of it quietly
// loses it for good.
func TestAnUnstreamedEventTheLogRefusesFailsTheRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  engine.Result
		says string
	}{
		{"a thought", engine.Result{Output: "done", Thoughts: []string{unwritable()}}, "fallback thought emit"},
		{
			"a refusal",
			engine.Result{Output: "done", Refusals: []engine.StreamRefusal{{Tool: "Bash", Input: unwritable()}}},
			"fallback refusal emit",
		},
		{
			"a tool call",
			engine.Result{Output: "done", ToolCalls: []engine.StreamToolCall{{Name: "Bash", Args: unwritable()}}},
			"fallback tool call emit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runWith(t, "quiet", quiet{res: tc.res})
			if err == nil {
				t.Fatal("Run succeeded with a result event the log had refused")
			}

			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the failure does not name which write was refused: %v", err)
			}
		})
	}
}
