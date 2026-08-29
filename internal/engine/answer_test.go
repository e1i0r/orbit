package engine

// What a phase said and what it thought on the way are two facts, and the
// record keeps them apart: thoughts are what the thinking pane draws, and
// the answer is what stands beside "finished". The claude reader used to
// file a text block under both.

import (
	"strings"
	"testing"
)

// claudeSaid is a stream in claude's shape: it reasons, it answers in two
// blocks, and the caller says whether the run reached its result line.
func claudeSaid(withResult bool) string {
	lines := []string{
		`{"type":"system","subtype":"init","session_id":"sess-abc"}`,
		`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"the retry belongs in the client"}]}}`,
		`{"type":"content_block_start","content_block":{"type":"text","text":"I moved the retry into the client."}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"I moved "}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Two tests cover the 5xx path.\n"}]}}`,
	}

	if withResult {
		lines = append(lines, `{"type":"result","result":"I moved the retry into the client.","total_cost_usd":0.1}`)
	}

	return strings.Join(lines, "\n")
}

// TestWhatAPhaseSaidIsNotAlsoWhatItThought. The answer comes back on the
// result line as Output, so a text block kept in Thoughts as well is the
// same paragraph written down twice — once in the thinking pane and once in
// the report.
func TestWhatAPhaseSaidIsNotAlsoWhatItThought(t *testing.T) {
	var streamed []string

	got, err := ParseStreamWithCallback(strings.NewReader(claudeSaid(true)), func(ev StreamEvent) {
		if ev.Type == "thought" {
			streamed = append(streamed, ev.Thought)
		}
	})
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if len(got.Thoughts) != 1 || got.Thoughts[0] != "the retry belongs in the client" {
		t.Errorf("Thoughts = %q, want the reasoning and nothing the phase answered", got.Thoughts)
	}

	// The events are what the record is written from while the run goes, so
	// a text block sent on as a thought lands there just as surely as one
	// left on the Result.
	if len(streamed) != 1 || streamed[0] != "the retry belongs in the client" {
		t.Errorf("the stream sent on %q as thoughts, want the reasoning alone", streamed)
	}

	if got.Output != "I moved the retry into the client." {
		t.Errorf("Output = %q, want what the result line said", got.Output)
	}
}

// TestAPhaseKilledBeforeItsResultLineStillSaysWhatItSaid. A run cancelled
// mid-phase never prints a result object, and its text blocks are the whole
// of what it managed to say.
func TestAPhaseKilledBeforeItsResultLineStillSaysWhatItSaid(t *testing.T) {
	got, err := ParseStream(strings.NewReader(claudeSaid(false)))
	if err == nil {
		t.Fatal("a stream with no result object parsed as a success")
	}

	want := "I moved the retry into the client.\n\nTwo tests cover the 5xx path."
	if got.Output != want {
		t.Errorf("Output = %q, want the two blocks the phase said, a blank line apart and trimmed", got.Output)
	}

	if len(got.Thoughts) != 1 || got.Thoughts[0] != "the retry belongs in the client" {
		t.Errorf("Thoughts = %q, want the reasoning alone", got.Thoughts)
	}
}

// TestAResultLineIsTheAnswerEvenWhereTextBlocksAgree. The fallback is for a
// stream that never reported one, and a stream that did has its summary
// taken from it — the blocks are the pieces, the result is the phase's own
// account of them.
func TestAResultLineIsTheAnswerEvenWhereTextBlocksAgree(t *testing.T) {
	got, err := ParseStream(strings.NewReader(strings.Join([]string{
		`{"type":"assistant","message":{"content":[{"type":"text","text":"first half"}]}}`,
		`{"type":"result","result":"the whole of it","session_id":"sess-abc"}`,
	}, "\n")))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if got.Output != "the whole of it" {
		t.Errorf("Output = %q, want what the result line said", got.Output)
	}
}
