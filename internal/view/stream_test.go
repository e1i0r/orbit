package view

// What a reader gets out of an engine's stream. The fixtures below are the
// shapes the three engines actually wrote — the claude one is a phase of
// ORB-102 that was cancelled mid-run, 76 kilobytes of frames whose whole
// human content was two sentences.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// killedStream is the frame order a cancelled claude phase leaves behind: a
// hook fires before the model is asked anything, the model thinks, calls
// tools and reports the result of each one, and the counters run the whole
// time.
const killedStream = `{"type":"system","subtype":"init","session_id":"a1","tools":["Bash","Read"],"slash_commands":["/graphify"]}
{"type":"system","subtype":"hook_started","hook_name":"SessionStart","prompt":"You have superpowers. Below is the full content of your using-superpowers skill..."}
{"type":"system","subtype":"hook_response","hook_name":"SessionStart","response":{"systemMessage":"skills loaded"}}
{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"The user wants a byte formatter. Let me check for one first."}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Using test-driven-development for this. First, checking whether a byte formatter already exists."}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"grep -rn FormatBytes ."}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"no matches"}]}}
{"type":"system","subtype":"thinking_tokens","tokens":1841}
{"type":"rate_limit_event","rate_limit":{"status":"allowed"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"No FormatBytes exists yet. Writing the test first (RED):"}]}}`

// TestAKilledStreamReadsAsWhatTheModelSaid.
func TestAKilledStreamReadsAsWhatTheModelSaid(t *testing.T) {
	got := unframe(killedStream)

	want := "Using test-driven-development for this. First, checking whether a byte formatter already exists.\n\n" +
		"No FormatBytes exists yet. Writing the test first (RED):"
	if got != want {
		t.Errorf("a cancelled phase reads as\n%q\nwant\n%q", got, want)
	}

	for _, junk := range []string{"superpowers", "hook_started", "thinking_tokens", "rate_limit", "tool_use", "grep -rn", "no matches", "The user wants"} {
		if strings.Contains(got, junk) {
			t.Errorf("the framing survived: %q is still on the screen", junk)
		}
	}
}

// TestOnlyTheBlocksThatSayTheyArePoseAreRead. The block kinds orbit has seen
// keep their content under a key of their own — thinking under "thinking",
// a call under "input", a result under "content" — so the kind is what says
// a block is prose and the presence of text is not. An engine that grows a
// block kind carrying text is the case this guard is here for.
func TestOnlyTheBlocksThatSayTheyAreProseAreRead(t *testing.T) {
	stream := `{"type":"assistant","message":{"content":[` +
		`{"type":"tool_use","name":"Bash","text":"rm -rf /tmp/build"},` +
		`{"type":"text","text":"Cleaning the build directory."}]}}`

	if got := unframe(stream); got != "Cleaning the build directory." {
		t.Errorf("the turn reads as %q, want only the block that said it was prose", got)
	}
}

// TestProseIsLeftAsItWasWritten. A phase that reached its own end writes the
// model's answer and nothing around it, and a tool call's arguments are an
// object that never says what kind of thing it is — neither is a stream, and
// a reader who is shown less than was written has been lied to.
func TestProseIsLeftAsItWasWritten(t *testing.T) {
	for _, text := range []string{
		"## What I did\n\nRewrote `formatBytes` and added a test.\n",
		`{"command":"go test ./...","description":"run the tests"}`,
		"",
		"   \n\n  ",
		"{ this line opens with a brace and is not JSON",
	} {
		if got := unframe(text); got != text {
			t.Errorf("unframe(%q) = %q, want it unchanged", text, got)
		}
	}
}

// TestTheResultStandsForTheRunThatReachedIt. The result frame is the
// engine's own account of the whole run, and it repeats the last thing the
// model said — printing both would show the answer twice.
func TestTheResultStandsForTheRunThatReachedIt(t *testing.T) {
	stream := `{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."}]}}
{"type":"result","subtype":"success","total_cost_usd":0.42,"result":"Done: the retry now backs off."}`

	if got := unframe(stream); got != "Done: the retry now backs off." {
		t.Errorf("a finished stream reads as %q, want the result", got)
	}

	// A result frame with nothing in it is an engine that ended without
	// saying anything, which is not a reason to drop the turns before it.
	empty := `{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."}]}}
{"type":"result","subtype":"error_during_execution","result":""}`
	if got := unframe(empty); got != "Working on it." {
		t.Errorf("an empty result reads as %q, want the turn that came before it", got)
	}
}

// TestEveryEngineIsRead. codex puts one completed item on the line and
// opencode one part, so a reader is told what the model said whoever ran it.
func TestEveryEngineIsRead(t *testing.T) {
	for _, c := range []struct {
		engine string
		stream string
		want   string
	}{
		{
			"codex",
			`{"type":"thread.started","thread_id":"t1"}
{"type":"item.completed","item":{"type":"reasoning","text":"checking the tests"}}
{"type":"item.completed","item":{"type":"command_execution","command":"go test ./...","aggregated_output":"ok"}}
{"type":"item.completed","item":{"type":"agent_message","text":"The tests pass."}}`,
			"The tests pass.",
		},
		{
			"opencode",
			`{"type":"text","sessionID":"s1","part":{"type":"reasoning","text":"reading the diff"}}
{"type":"tool_use","part":{"type":"tool","tool":"bash"}}
{"type":"text","sessionID":"s1","part":{"type":"text","text":"The diff is clean."}}`,
			"The diff is clean.",
		},
	} {
		if got := unframe(c.stream); got != c.want {
			t.Errorf("%s reads as %q, want %q", c.engine, got, c.want)
		}
	}
}

// TestAStreamCutMidFrameKeepsWhatCameBefore. A process killed while it was
// writing leaves half a line, and half a line is not a reason to lose the
// nine whole ones above it.
func TestAStreamCutMidFrameKeepsWhatCameBefore(t *testing.T) {
	stream := `{"type":"assistant","message":{"content":[{"type":"text","text":"Reading the record."}]}}
{"type":"assistant","message":{"content":[{"type":"text","tex`

	if got := unframe(stream); got != "Reading the record." {
		t.Errorf("a stream cut mid-frame reads as %q, want the frame that was whole", got)
	}
}

// TestTheLogCarriesWhatWasSaidBesideWhatWasWritten. Both are fields of the
// entry: the panes draw Said, and `[v] raw` and the byte counts are the
// record itself.
func TestTheLogCarriesWhatWasSaidBesideWhatWasWritten(t *testing.T) {
	entries := Log([]record.Event{
		{At: at(0), Kind: record.TaskCreated, Text: "Retry the webhook on 5xx"},
		{At: at(1), Kind: record.PhaseCancelled, Phase: "build", Text: killedStream},
	})

	if entries[0].Said() != entries[0].Text {
		t.Errorf("the task's own words read as %q, want %q", entries[0].Said(), entries[0].Text)
	}

	if !strings.HasPrefix(entries[1].Said(), "Using test-driven-development") {
		t.Errorf("the cancelled phase says %q, want the model's own words", entries[1].Said())
	}

	if entries[1].Text != killedStream {
		t.Error("the entry no longer carries what the engine actually printed")
	}

	if entries[1].Kept != len(killedStream) {
		t.Errorf("the entry was written down as %d bytes, want the %d the record holds", entries[1].Kept, len(killedStream))
	}
}
