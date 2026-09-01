package engine

import (
	"strings"
	"testing"
)

// What a phase spent in tokens, read off each engine's own stream.
//
// The three count in three vocabularies — claude names the cache pair beside
// the input, codex folds the cached prefix into it, opencode nests it under
// tokens — and the record keeps one shape, so each adapter is checked against
// lines in the spelling its CLI actually prints.

// claude reports the turn on the result line, and the cache pair is the
// point: read as zero across a session it says something early in the
// context is changing between turns, which nothing in a total shows.
func TestClaudeReportsTheTurnsTokensAndItsCache(t *testing.T) {
	stream := `{"type":"system","subtype":"init","session_id":"sess-abc"}
{"type":"assistant","message":{"content":[{"type":"text","text":"working"}],"usage":{"input_tokens":10,"output_tokens":2}}}
{"type":"result","result":"done","total_cost_usd":0.25,"usage":{"input_tokens":1200,"output_tokens":340,"cache_read_input_tokens":18000,"cache_creation_input_tokens":900}}`

	got, err := ParseStream(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	want := Usage{Input: 1200, Output: 340, CacheRead: 18000, CacheWrite: 900}
	if got.Usage != want {
		t.Errorf("Usage = %+v, want %+v", got.Usage, want)
	}
}

// A run that was killed has no result line and still spent what it spent.
// The messages are summed only then: taken as well as the result line, every
// assistant turn would be counted twice.
func TestAKilledClaudeRunStillReportsWhatItSpent(t *testing.T) {
	stream := `{"type":"assistant","message":{"content":[{"type":"text","text":"one"}],"usage":{"input_tokens":10,"output_tokens":2,"cache_read_input_tokens":100}}}
{"type":"assistant","message":{"content":[{"type":"text","text":"two"}],"usage":{"input_tokens":30,"output_tokens":5}}}`

	got, err := ParseStream(strings.NewReader(stream))
	if err == nil {
		t.Fatal("ParseStream on a stream with no result object returned no error")
	}

	want := Usage{Input: 40, Output: 7, CacheRead: 100}
	if got.Usage != want {
		t.Errorf("Usage = %+v, want %+v", got.Usage, want)
	}
}

// codex counts the cached prefix inside input_tokens. Kept as printed, a
// task's total would count those tokens once as input and again as a cache
// read, and Input would mean one thing under codex and another under claude.
func TestCodexInputIsWhatWasNotAlreadyCached(t *testing.T) {
	stream := `{"type":"thread.started","thread_id":"th-1"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"ok"}}
{"type":"turn.completed","usage":{"input_tokens":15163,"cached_input_tokens":14080,"output_tokens":330}}`

	got, err := ParseCodexStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ParseCodexStream: %v", err)
	}

	want := Usage{Input: 1083, Output: 330, CacheRead: 14080}
	if got.Usage != want {
		t.Errorf("Usage = %+v, want %+v", got.Usage, want)
	}
}

// A codex that reports more cached than it sent is a codex that changed its
// mind about which of the two contains the other. The answer to give then is
// zero input, not a negative one.
func TestCodexNeverReportsLessThanNoInput(t *testing.T) {
	got, err := ParseCodexStream(strings.NewReader(
		`{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":900,"output_tokens":7}}`), nil)
	if err != nil {
		t.Fatalf("ParseCodexStream: %v", err)
	}

	if got.Usage.Input != 0 {
		t.Errorf("Input = %d, want 0", got.Usage.Input)
	}
}

// opencode counts a step at a time, so a run of several steps reports
// several counts and the phase spent all of them — the same reason its cost
// is summed rather than read off one line.
func TestOpenCodeAddsUpEveryStepsTokens(t *testing.T) {
	stream := `{"type":"step_start","sessionID":"ses_1","part":{"type":"step-start"}}
{"type":"text","sessionID":"ses_1","part":{"type":"text","text":"ok"}}
{"type":"step_finish","sessionID":"ses_1","part":{"reason":"stop","cost":0.01,"tokens":{"input":120,"output":40,"cache":{"read":8000,"write":300}}}}
{"type":"step_finish","sessionID":"ses_1","part":{"reason":"stop","cost":0.02,"tokens":{"input":80,"output":10,"cache":{"read":9000,"write":0}}}}`

	got, err := ParseOpenCodeStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ParseOpenCodeStream: %v", err)
	}

	want := Usage{Input: 200, Output: 50, CacheRead: 17000, CacheWrite: 300}
	if got.Usage != want {
		t.Errorf("Usage = %+v, want %+v", got.Usage, want)
	}
}

// An engine that says nothing about tokens says nothing: zero is "not
// reported" and Any is what tells the two apart, so a record is not written
// claiming a phase sent no prompt.
func TestAnEngineThatCountsNothingSaysNothing(t *testing.T) {
	got, err := ParseCodexStream(strings.NewReader(
		`{"type":"turn.completed"}`), nil)
	if err != nil {
		t.Fatalf("ParseCodexStream: %v", err)
	}

	if got.Usage.Any() {
		t.Errorf("Usage = %+v, want nothing said", got.Usage)
	}
}
