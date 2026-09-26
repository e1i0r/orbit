package engine

import (
	"strings"
	"testing"
)

// clineRun is a run as cline 3.0.62 writes it with --json -v: deltas before
// each whole message, a tool that ran and one it refused, two turns counted.
const clineRun = `{"ts":"t","type":"run_start","providerId":"cline-pass","modelId":"cline-pass/glm-5.2","sessionId":"1789242437805_ejudv"}
{"ts":"t","type":"agent_event","event":{"type":"iteration_start","iteration":1}}
{"ts":"t","type":"agent_event","event":{"type":"content_start","contentType":"reasoning","reasoning":"Let","redacted":false}}
{"ts":"t","type":"agent_event","event":{"type":"content_end","contentType":"reasoning","reasoning":"Let me read it first."}}
{"ts":"t","type":"agent_event","event":{"type":"content_start","contentType":"text","text":"I'll ","accumulated":"I'll "}}
{"ts":"t","type":"agent_event","event":{"type":"content_end","contentType":"text","text":"I'll read the file."}}
{"ts":"t","type":"agent_event","event":{"type":"content_start","contentType":"tool","toolName":"editor","toolCallId":"t1","input":{"command":"create","path":"a.go"}}}
{"ts":"t","type":"agent_event","event":{"type":"content_end","contentType":"tool","toolName":"editor","toolCallId":"t1","output":"ok"}}
{"ts":"t","type":"agent_event","event":{"type":"content_start","contentType":"tool","toolName":"run_commands","toolCallId":"t2","input":{"commands":["rm -rf /"]}}}
{"ts":"t","type":"agent_event","event":{"type":"content_end","contentType":"tool","toolName":"run_commands","toolCallId":"t2","error":"Tool \"run_commands\" requires approval in a TTY session"}}
{"ts":"t","type":"agent_event","event":{"type":"usage","inputTokens":1200,"outputTokens":80,"cacheReadTokens":900,"cost":0.004,"totalInputTokens":1200,"totalCost":0.004}}
{"ts":"t","type":"agent_event","event":{"type":"usage","inputTokens":300,"outputTokens":20,"cost":0.001,"totalInputTokens":1500,"totalCost":0.005}}
{"ts":"t","type":"agent_event","event":{"type":"content_end","contentType":"text","text":"Done."}}
{"ts":"t","type":"agent_event","event":{"type":"done","reason":"completed","text":"Done.","iterations":2}}
{"ts":"t","type":"run_result","finishReason":"completed","iterations":2,"text":"Done."}
`

func TestClineStreamIsReadIntoWhatTheRunDid(t *testing.T) {
	var seen []string

	saw := func(ev StreamEvent) { seen = append(seen, ev.Type) }

	out, err := ParseClineStream(strings.NewReader(clineRun), saw)
	if err != nil {
		t.Fatalf("ParseClineStream: %v", err)
	}

	if out.SessionID != "1789242437805_ejudv" {
		t.Errorf("session = %q, want run_start's", out.SessionID)
	}

	// The whole messages, not the deltas that came before them.
	if out.Output != "I'll read the file.\n\nDone." {
		t.Errorf("output = %q", out.Output)
	}

	if len(out.Thoughts) != 1 || out.Thoughts[0] != "Let me read it first." {
		t.Errorf("thoughts = %q", out.Thoughts)
	}

	edit := out.ToolCalls[0]
	if len(out.ToolCalls) != 2 || edit.Name != "editor" || !strings.Contains(edit.Args, `"path":"a.go"`) {
		t.Errorf("tool calls = %+v", out.ToolCalls)
	}

	if len(out.Refusals) != 1 || out.Refusals[0].Tool != "run_commands" {
		t.Errorf("refusals = %+v, want the command cline refused", out.Refusals)
	}

	// Summed per turn, not the running total cline writes beside it.
	if out.Usage.Input != 1500 || out.Usage.Output != 100 || out.Usage.CacheRead != 900 {
		t.Errorf("usage = %+v", out.Usage)
	}

	if out.Cost < 0.0049 || out.Cost > 0.0051 {
		t.Errorf("cost = %v, want 0.005", out.Cost)
	}

	if strings.Join(seen, ",") != "thought,tool_call,tool_call,refusal,result,result" {
		t.Errorf("events = %v", seen)
	}
}

// TestAClineRunThatFailedSaysWhy. The error is the answer when nothing else
// was said.
func TestAClineRunThatFailedSaysWhy(t *testing.T) {
	failed := `{"ts":"t","type":"agent_event","event":{"type":"error","error":{"name":"Error","message":"ClinePass limit reached"},"errorClass":"unknown"}}
{"ts":"t","type":"run_result","finishReason":"error","text":"ClinePass limit reached"}
`

	out, err := ParseClineStream(strings.NewReader(failed), nil)
	if err != nil {
		t.Fatalf("ParseClineStream: %v", err)
	}

	if out.Output != "ClinePass limit reached" {
		t.Errorf("output = %q, want the error", out.Output)
	}

	if !NewCline().RanOut(out, nil) {
		t.Error("a spent subscription was not read as the allowance gone")
	}
}

// TestAClineStreamWithNothingInItIsSaid rather than read as an empty answer.
func TestAClineStreamWithNothingInItIsSaid(t *testing.T) {
	if _, err := ParseClineStream(strings.NewReader("not json\n"), nil); err == nil {
		t.Error("a stream with no cline events in it was read as a run")
	}
}
