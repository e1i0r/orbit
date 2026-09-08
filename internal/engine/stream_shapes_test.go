package engine

// The shapes claude's stream actually arrives in, as against the ones a
// reader would design.

import (
	"strings"
	"testing"
)

// TestOneMessageIsCountedOnce. claude repeats an assistant message across
// several records carrying the identical usage object, and a killed phase
// was billed once per repeat.
func TestOneMessageIsCountedOnce(t *testing.T) {
	stream := `{"type":"assistant","message":{"id":"msg_1","content":[{"type":"text","text":"one"}],"usage":{"input_tokens":10,"output_tokens":2}}}
{"type":"assistant","message":{"id":"msg_1","content":[{"type":"text","text":"one"}],"usage":{"input_tokens":10,"output_tokens":2}}}
{"type":"assistant","message":{"id":"msg_2","content":[{"type":"text","text":"two"}],"usage":{"input_tokens":30,"output_tokens":5}}}`

	got, err := ParseStream(strings.NewReader(stream))
	if err == nil {
		t.Fatal("ParseStream on a stream with no result object returned no error")
	}

	want := Usage{Input: 40, Output: 7}
	if got.Usage != want {
		t.Errorf("Usage = %+v, want %+v — the repeated message was counted twice", got.Usage, want)
	}
}

// TestARefusalArrivesInEitherShape. A tool_result's content is a string on
// some blocks and an array of blocks on others; typed as a string, the whole
// line failed to unmarshal and the refusal in it was never seen.
func TestARefusalArrivesInEitherShape(t *testing.T) {
	stream := `{"type":"assistant","message":{"id":"msg_1","content":[{"type":"tool_use","id":"call_1","name":"Bash","input":{}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"call_1","is_error":true,"content":[{"type":"text","text":"permission denied: Bash"}]}]}}
{"type":"result","result":"done","session_id":"s1"}`

	got, err := ParseStream(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if len(got.Refusals) != 1 {
		t.Fatalf("the stream carried %d refusals, want the one in the array-shaped result", len(got.Refusals))
	}

	if got.Refusals[0].Tool != "Bash" {
		t.Errorf("the refusal names the tool %q, want the call it answered", got.Refusals[0].Tool)
	}

	if !strings.Contains(got.Refusals[0].Input, "permission denied") {
		t.Errorf("the refusal carries %q", got.Refusals[0].Input)
	}
}
