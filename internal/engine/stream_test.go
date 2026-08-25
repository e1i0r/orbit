package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stream opens one of the checked-in fixtures.
//
// Every test in this package reads bytes off the disk and never runs the
// claude binary. That is not squeamishness about slow tests: a real headless
// run spends real money and needs a network, and the program this replaces
// shipped a suite with a paid call in it, which made the suite unsafe to run
// and so nobody ran it. ParseStream is a function rather than a few lines
// inside Run for exactly this reason, the same reason claudeArgs was split
// out before it.
func stream(t *testing.T, name string) *bytes.Reader {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return bytes.NewReader(raw)
}

func TestParseStreamReadsTheSessionAndTheCostFromTheResultObject(t *testing.T) {
	got, err := ParseStream(stream(t, "stream_normal.jsonl"))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if got.SessionID != "9c1f8f2a-4d3b-4a77-9a52-2f0f6f9b5c31" {
		t.Errorf("SessionID = %q — without it there is no taking the keyboard", got.SessionID)
	}
	if got.Cost != 0.0431 {
		t.Errorf("Cost = %v, want 0.0431", got.Cost)
	}
	if got.Output != "ACME-1: retries on 5xx are in place in the payments client." {
		t.Errorf("Output = %q — the record keeps the result field, not the whole stream", got.Output)
	}
}

// TestParseStreamKeepsOnlyTheResultTextNotTheWholeStream pins what a reader
// of the record actually sees. The stream carries every assistant turn and
// every tool result; the human-readable answer is the result object's own
// result field, and putting the raw JSON in the record instead would make
// phase.finished unreadable in a pager.
func TestParseStreamKeepsOnlyTheResultTextNotTheWholeStream(t *testing.T) {
	got, err := ParseStream(stream(t, "stream_normal.jsonl"))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if strings.Contains(got.Output, `"type"`) {
		t.Errorf("Output carries raw stream JSON: %q", got.Output)
	}
	if strings.Contains(got.Output, "Reading the payments client") {
		t.Errorf("Output carries an intermediate assistant turn: %q", got.Output)
	}
}

// TestParseStreamRefusesAStreamThatEndsWithNoResultObject is the fixture
// that matters most. A stream with no result object means claude died, was
// killed, or changed its output shape, and every one of those is a phase
// whose session and cost are simply unknown. Returning a zero Result and no
// error would write that unknown into the record as "this phase cost
// nothing", which is a lie the log would carry for ever.
func TestParseStreamRefusesAStreamThatEndsWithNoResultObject(t *testing.T) {
	_, err := ParseStream(stream(t, "stream_no_result.jsonl"))
	if err == nil {
		t.Fatal("a stream with no result object parsed as a success")
	}
	if !strings.Contains(err.Error(), "result") {
		t.Errorf("the error does not name what was missing: %v", err)
	}
}

// TestParseStreamAcceptsAResultObjectWithNoCost keeps the distinction the
// Result doc comment draws: an engine that does not report a number is a
// fact about that engine, not a failure of the run. The session id is still
// there and still worth having.
func TestParseStreamAcceptsAResultObjectWithNoCost(t *testing.T) {
	got, err := ParseStream(stream(t, "stream_no_cost.jsonl"))
	if err != nil {
		t.Fatalf("ParseStream refused a result object that reported no cost: %v", err)
	}
	if got.Cost != 0 {
		t.Errorf("Cost = %v, want 0 — nothing in the fixture says what it cost", got.Cost)
	}
	if got.SessionID != "4f2a1c88-6b5d-42e7-8c30-7ab19d0e5c62" {
		t.Errorf("SessionID = %q — a missing cost lost the session id with it", got.SessionID)
	}
	if got.Output == "" {
		t.Error("Output is empty — a missing cost lost the answer with it")
	}
}

func TestParseStreamRefusesAnEmptyStream(t *testing.T) {
	if _, err := ParseStream(strings.NewReader("")); err == nil {
		t.Error("an engine that printed nothing at all parsed as a success")
	}
}

// TestParseStreamIgnoresLinesThatAreNotJSON allows for a binary that prints
// a warning on stdout before it starts streaming. Such a line is not this
// function's failure to report — the failure it reports is the absence of a
// result object, which is checked above, and which a stream of nothing but
// noise still hits.
func TestParseStreamIgnoresLinesThatAreNotJSON(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "stream_normal.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	noisy := "warning: a newer version is available\n\n" + string(raw)
	got, err := ParseStream(strings.NewReader(noisy))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if got.SessionID == "" {
		t.Error("a line of noise before the stream lost the whole result")
	}
}

// TestParseStreamTakesTheLastResultObject pins which object wins if a stream
// somehow carries two. The last one is the terminal one, and the terminal
// one is the summary of the run.
func TestParseStreamTakesTheLastResultObject(t *testing.T) {
	two := `{"type":"result","subtype":"success","result":"first","session_id":"aaa","total_cost_usd":0.01}` + "\n" +
		`{"type":"result","subtype":"success","result":"second","session_id":"bbb","total_cost_usd":0.02}` + "\n"
	got, err := ParseStream(strings.NewReader(two))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}
	if got.SessionID != "bbb" || got.Output != "second" {
		t.Errorf("took %+v, want the terminal result object", got)
	}
}

func TestParseStreamWithCallbackEmitsEvents(t *testing.T) {
	streamData := `{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"analyzing code"}]}}` + "\n" +
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls -la"}}]}}` + "\n" +
		`{"type":"result","subtype":"success","result":"done","session_id":"sess-123","total_cost_usd":0.05}` + "\n"

	var received []StreamEvent
	res, err := ParseStreamWithCallback(strings.NewReader(streamData), func(ev StreamEvent) {
		received = append(received, ev)
	})
	if err != nil {
		t.Fatalf("ParseStreamWithCallback: %v", err)
	}
	if res.Output != "done" {
		t.Errorf("Output = %q, want 'done'", res.Output)
	}
	if len(received) != 3 {
		t.Fatalf("received %d events, want 3", len(received))
	}
	if received[0].Type != "thought" || received[0].Thought != "analyzing code" {
		t.Errorf("received[0] = %+v, want thought", received[0])
	}
	if received[1].Type != "tool_call" || received[1].ToolCall.Name != "Bash" {
		t.Errorf("received[1] = %+v, want tool_call Bash", received[1])
	}
	if received[2].Type != "result" || received[2].Cost != 0.05 {
		t.Errorf("received[2] = %+v, want result cost 0.05", received[2])
	}
}
