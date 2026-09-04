package engine

// Reading agy's stream, against the lines a real run printed.

import (
	"strings"
	"testing"
)

// agyRun is the stream of a run that thought, called a tool, was denied
// another, and answered. Every line is one agy 1.1.25 printed, with the
// prose shortened.
const agyRun = `{"event":"init","conversation_id":"4eee6f79","init":{"cwd":"/tmp","permission_mode":"request-review"}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":0,"state":"DONE","step_type":"user_input"}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":1,"state":"ACTIVE","step_type":"tool","tool_name":"list_dir","tool_info":{"name":"list_dir","parameters":{"DirectoryPath":"/tmp"}}}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":1,"state":"DONE","step_type":"tool","tool_name":"list_dir","tool_info":{"name":"list_dir","parameters":{"DirectoryPath":"/tmp"}}}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":2,"state":"ACTIVE","step_type":"tool","tool_name":"run_command","tool_info":{"name":"run_command","parameters":{"CommandLine":"ls -la"}}}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":2,"state":"ERROR","step_type":"tool","tool_name":"run_command","tool_info":{"name":"run_command","parameters":{"CommandLine":"ls -la"},"error":{"type":"TOOL_ERROR","message":"permission check failed for command \"ls -la\": user denied permission to run command"}}}}
{"event":"step_update","step_update":{"conversation_id":"4eee6f79","step_index":3,"state":"DONE","step_type":"agent_response","text_delta":"the directory holds one file","usage":{"input_tokens":100,"output_tokens":20,"thinking_tokens":15,"cache_read_tokens":5,"total_tokens":120}}}
{"event":"result","result":{"conversation_id":"4eee6f79","status":"SUCCESS","response":"the directory holds one file","num_turns":1,"usage":{"input_tokens":100,"output_tokens":20,"thinking_tokens":15,"cache_read_tokens":5,"total_tokens":125}}}
`

func TestAgyStreamReadsARun(t *testing.T) {
	res, err := ParseAgyStream(strings.NewReader(agyRun), nil)
	if err != nil {
		t.Fatalf("ParseAgyStream: %v", err)
	}

	if res.SessionID != "4eee6f79" {
		t.Errorf("SessionID = %q, want the conversation id the run opened with", res.SessionID)
	}

	if res.Output != "the directory holds one file" {
		t.Errorf("Output = %q, want the answer once", res.Output)
	}

	want := Usage{Input: 100, Output: 20, CacheRead: 5}
	if res.Usage != want {
		t.Errorf("Usage = %+v, want the run's own totals %+v", res.Usage, want)
	}
}

// TestAgyStreamRecordsAToolOnce: a step is printed on every state it passes
// through, and a timeline that drew each of them would show one command
// twice.
func TestAgyStreamRecordsAToolOnce(t *testing.T) {
	res, err := ParseAgyStream(strings.NewReader(agyRun), nil)
	if err != nil {
		t.Fatalf("ParseAgyStream: %v", err)
	}

	if len(res.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %+v, want the two tools the run asked for", res.ToolCalls)
	}

	if res.ToolCalls[0].Name != "list_dir" || !strings.Contains(res.ToolCalls[0].Args, "/tmp") {
		t.Errorf("the first tool call is %+v, want list_dir with what it was pointed at", res.ToolCalls[0])
	}
}

// TestAgyStreamReadsADenialAsARefusal, not as a tool that failed: the phase
// asked for something its posture did not buy, and that is what a reader
// needs to be told.
func TestAgyStreamReadsADenialAsARefusal(t *testing.T) {
	var events []StreamEvent

	res, err := ParseAgyStream(strings.NewReader(agyRun), func(ev StreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("ParseAgyStream: %v", err)
	}

	if len(res.Refusals) != 1 || res.Refusals[0].Tool != "run_command" {
		t.Fatalf("Refusals = %+v, want the command agy was denied", res.Refusals)
	}

	if !strings.Contains(res.Output, "the directory holds one file") {
		t.Errorf("Output = %q, want the answer rather than the denial", res.Output)
	}

	var refusals int

	for _, ev := range events {
		if ev.Type == "refusal" {
			refusals++
		}
	}

	if refusals != 1 {
		t.Errorf("%d refusals reached the window as they happened, want 1", refusals)
	}
}

// TestAgyStreamFallsBackToTheResultLine: a run that answers in one breath
// streams no text at all, and the whole answer is on the line that ends it.
func TestAgyStreamFallsBackToTheResultLine(t *testing.T) {
	const oneBreath = `{"event":"step_update","step_update":{"conversation_id":"c1","step_index":0,"state":"DONE","step_type":"user_input"}}
{"event":"result","result":{"conversation_id":"c1","status":"SUCCESS","response":"ok","usage":{"input_tokens":7,"output_tokens":1}}}
`

	res, err := ParseAgyStream(strings.NewReader(oneBreath), nil)
	if err != nil {
		t.Fatalf("ParseAgyStream: %v", err)
	}

	if res.Output != "ok" {
		t.Errorf("Output = %q, want what the result line said", res.Output)
	}

	if res.Usage.Input != 7 {
		t.Errorf("Usage = %+v, want the count off the result line", res.Usage)
	}
}

// TestAgyStreamKeepsWhatAKilledRunSpent: no result line ever arrives, and
// the steps are the only place the count was written.
func TestAgyStreamKeepsWhatAKilledRunSpent(t *testing.T) {
	const killed = `{"event":"step_update","step_update":{"conversation_id":"c2","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"half an","usage":{"input_tokens":10,"output_tokens":2}}}
{"event":"step_update","step_update":{"conversation_id":"c2","step_index":2,"state":"DONE","step_type":"agent_response","text_delta":"answer","usage":{"input_tokens":30,"output_tokens":4}}}
`

	res, err := ParseAgyStream(strings.NewReader(killed), nil)
	if err != nil {
		t.Fatalf("ParseAgyStream: %v", err)
	}

	if res.Usage.Input != 40 || res.Usage.Output != 6 {
		t.Errorf("Usage = %+v, want the steps summed", res.Usage)
	}

	if res.Output != "half an\n\nanswer" {
		t.Errorf("Output = %q, want both pieces that arrived", res.Output)
	}
}

// TestAgyStreamSaysSoWhenItUnderstoodNothing. A stream whose shape moved
// under this parser has to be reported: silence would read as a phase that
// ran and did nothing.
func TestAgyStreamSaysSoWhenItUnderstoodNothing(t *testing.T) {
	if _, err := ParseAgyStream(strings.NewReader("not json\n{\"event\":\"unknown\"}\n"), nil); err == nil {
		t.Error("a stream with nothing this parser reads came back without an error")
	}
}
