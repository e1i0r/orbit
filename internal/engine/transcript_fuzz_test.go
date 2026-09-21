package engine

// A model's own output, read against bytes nothing here wrote.
//
// A stream is whatever the engine printed on its way through a run, and a
// transcript is a file another program wrote in a format it is free to
// change. Both end up in the record and in the next phase's prompt, so what
// these readings promise has to hold for anything at all: a run cannot fail
// because an engine printed something unexpected on line four hundred.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzStreamEvents is what a stream hands the record as it goes past.
//
// The events are written down one by one as they arrive, so a malformed one
// is not a bad line in a report — it is a row in the record. Every one of
// them has to be a kind a reader knows, and a thought has to be a thought:
// the filter that keeps text deltas out is the only thing standing between
// the thoughts pane and the prose of the answer arriving in it twice.
func FuzzStreamEvents(f *testing.F) {
	for _, seed := range []string{
		`{"type":"result","subtype":"success","result":"ok","total_cost_usd":0.01}`,
		`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"a thought"}]}}`,
		`{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"a thought"}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"an answer"}}`,
		`{"type":"content_block_delta","delta":{}}`,
		`{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`,
		"not json at all\n",
		"{\n",
		`{"type":"result"}`,
	} {
		f.Add(seed)
	}

	known := map[string]bool{"thought": true, "tool_call": true, "refusal": true, "result": true}

	f.Fuzz(func(t *testing.T, data string) {
		var seen []StreamEvent

		res, err := ParseStreamWithCallback(strings.NewReader(data), func(ev StreamEvent) {
			seen = append(seen, ev)
		})

		for i, ev := range seen {
			if !known[ev.Type] {
				t.Errorf("event %d of %q is of kind %q, which no reader knows", i, data, ev.Type)
			}

			// A thought with nothing in it is a keep-alive that reached the
			// record as a row saying the model thought nothing.
			if ev.Type == "thought" && ev.Thought == "" {
				t.Errorf("event %d of %q is a thought that says nothing", i, data)
			}

			if ev.Cost < 0 {
				t.Errorf("event %d of %q reports a cost of %v", i, data, ev.Cost)
			}
		}

		if err != nil {
			return
		}

		// An answer that parsed goes into the record, which is TEXT, and
		// into the next phase's prompt, which a model reads.
		if !utf8.ValidString(res.Output) {
			t.Errorf("parsing %q answered output that is not text", data)
		}

		if res.Cost < 0 {
			t.Errorf("parsing %q answered a cost of %v", data, res.Cost)
		}
	})
}

// FuzzNotSaid is the reading that keeps a machine's own words out of a
// conversation.
//
// What it hides was never typed by the reader or answered by the model, and
// all of it is written in the same place their words are. It has to answer
// about any text at all, because the text is whatever was in the file.
func FuzzNotSaid(f *testing.F) {
	for _, seed := range []string{
		"", "<command-name>/clear</command-name>", "  <system-reminder>",
		"an ordinary thing somebody said", "<", "<system", "Caveat: The messages below",
		"caveat: the messages below", "\n<system-reminder>",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		hidden := notSaid(text)

		// Nothing that begins with a letter is a machine's own line: every
		// prefix it hides opens with `<` or with the one sentence claude
		// writes at the top of a resumed session.
		if hidden && !strings.HasPrefix(text, "<") && !strings.HasPrefix(text, "Caveat:") {
			t.Errorf("notSaid(%q) hid something that does not open the way a machine's line does", text)
		}

		// And the reading is about the front of the text alone: the same
		// text with a word in front of it is somebody quoting it.
		if quoted := "somebody said: " + text; notSaid(quoted) {
			t.Errorf("notSaid(%q) hid it when somebody quoted it", text)
		}
	})
}

// FuzzTurnsAreOrdered is what every engine's own reading hands over: a
// conversation, oldest first.
//
// Each reading gathers turns per file or per row, and a session spread over
// more than one of either comes back interleaved. The panes draw them in the
// order they arrive.
func FuzzTurnsAreOrdered(f *testing.F) {
	f.Add(int64(0), int64(0), int64(0))
	f.Add(int64(3), int64(2), int64(1))
	f.Add(int64(-5), int64(0), int64(5))

	f.Fuzz(func(t *testing.T, a, b, c int64) {
		at := func(n int64) time.Time { return time.Unix(n%1_000_000, 0).UTC() }

		got := sorted([]Turn{
			{At: at(a), Text: "a"},
			{At: at(b), Text: "b"},
			{At: at(c), Text: "c"},
		})

		if len(got) != 3 {
			t.Fatalf("three turns came back as %d", len(got))
		}

		for i := 1; i < len(got); i++ {
			if got[i].At.Before(got[i-1].At) {
				t.Errorf("turn %d is older than the one before it: %v", i, got)
			}
		}
	})
}
