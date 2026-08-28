package engine

import (
	"strings"
	"testing"
)

// TestAResultObjectDoesNotWipeTheSessionID is the fix.
//
// The session id is announced on claude's init line and the result object at
// the end of the stream does not repeat it. The result branch assigned
// out.SessionID = env.SessionID unconditionally, so the last line of every
// run overwrote the captured id with an empty string: the run finished, the
// record said it could not be resumed, and the evidence that it could had
// been read and thrown away one line earlier.
func TestAResultObjectDoesNotWipeTheSessionID(t *testing.T) {
	stream := `{"type":"system","subtype":"init","session_id":"sess-abc"}
{"type":"assistant","message":{"content":[{"type":"text","text":"working"}]}}
{"type":"result","result":"done","total_cost_usd":0.25}`

	got, err := ParseStream(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if got.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc — the result object has none, and it overwrote the one the init line gave", got.SessionID)
	}

	if got.Output != "done" || got.Cost != 0.25 {
		t.Errorf("Output = %q, Cost = %v, want done and 0.25", got.Output, got.Cost)
	}
}

// TestAResultObjectThatCarriesASessionIDStillSetsIt. The guard must not turn
// into "first one wins": a stream whose only session id is on the result
// line still has to report it.
func TestAResultObjectThatCarriesASessionIDStillSetsIt(t *testing.T) {
	got, err := ParseStream(strings.NewReader(
		`{"type":"result","result":"done","session_id":"sess-late"}`))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if got.SessionID != "sess-late" {
		t.Errorf("SessionID = %q, want sess-late", got.SessionID)
	}
}

// TestConnectionRefusedIsNotAPermissionRefusal is the fix.
//
// The bare word "refused" was on the match list, and it is the one thing a
// network stack says when nothing is listening. A phase that failed because
// a database was down was recorded — and drawn on the screen — as a phase
// whose posture was too narrow, pointing whoever debugged it at the
// permissions and away from the port.
func TestConnectionRefusedIsNotAPermissionRefusal(t *testing.T) {
	for _, s := range []string{
		"dial tcp 127.0.0.1:5432: connect: connection refused",
		"curl: (7) Failed to connect to localhost port 8080: Connection refused",
		"ssh: connect to host build.internal port 22: Connection refused",
	} {
		if isPermissionRefusal(s) {
			t.Errorf("%q was recorded as the sandbox denying a permission", s)
		}
	}

	for _, s := range []string{
		"Error: permission denied",
		"EPERM: operation not permitted, open '/etc/hosts'",
		"Claude refused permission to use Bash",
		"this tool is not allowed in the current mode",
	} {
		if !isPermissionRefusal(s) {
			t.Errorf("%q is a permission being denied and was not recorded as one", s)
		}
	}
}

// TestParseCodexStream reads the shape a real `codex exec --json` prints.
func TestParseCodexStream(t *testing.T) {
	stream := `Reading additional input from stdin...
{"type":"thread.started","thread_id":"01a04a53-fc89-7893-8a69-8f2b9adcc3a5"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"i0","type":"reasoning","text":"deciding"}}
{"type":"item.started","item":{"id":"i1","type":"command_execution","command":"echo hello"}}
{"type":"item.completed","item":{"id":"i1","type":"command_execution","command":"echo hello","exit_code":0}}
{"type":"item.completed","item":{"id":"i2","type":"agent_message","text":"it printed hello"}}
{"type":"turn.completed","usage":{"input_tokens":15163,"output_tokens":5}}`

	got, err := ParseCodexStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ParseCodexStream: %v", err)
	}

	if got.SessionID != "01a04a53-fc89-7893-8a69-8f2b9adcc3a5" {
		t.Errorf("SessionID = %q — codex spells it thread_id, and nothing was reading it", got.SessionID)
	}

	if got.Output != "it printed hello" {
		t.Errorf("Output = %q, want the agent message", got.Output)
	}

	if len(got.Thoughts) != 1 || got.Thoughts[0] != "deciding" {
		t.Errorf("Thoughts = %v, want the reasoning item", got.Thoughts)
	}

	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Args != "echo hello" {
		t.Errorf("ToolCalls = %+v, want the one shell command", got.ToolCalls)
	}
}

// TestParseCodexStreamReportsAFailedTurn. codex answers a bad model or a bad
// effort with an error object and a turn.failed, and both carry the sentence
// a reader needs. A run that says nothing at all is the one thing this must
// not do with them.
func TestParseCodexStreamReportsAFailedTurn(t *testing.T) {
	got, err := ParseCodexStream(strings.NewReader(
		`{"type":"thread.started","thread_id":"t1"}
{"type":"turn.failed","error":{"message":"unsupported value: 'minimal' is not supported with this model"}}`), nil)
	if err != nil {
		t.Fatalf("ParseCodexStream: %v", err)
	}

	if !strings.Contains(got.Output, "minimal") {
		t.Errorf("Output = %q, want the reason codex gave", got.Output)
	}
}

// TestParseOpenCodeStream reads the shape a real `opencode run --format
// json` prints.
func TestParseOpenCodeStream(t *testing.T) {
	stream := `{"type":"step_start","sessionID":"ses_fb5af5","part":{"type":"step-start"}}
{"type":"text","sessionID":"ses_fb5af5","part":{"type":"text","text":"writing it"}}
{"type":"tool_use","sessionID":"ses_fb5af5","part":{"type":"tool","tool":"write","callID":"c1","state":{"status":"completed","input":{"filePath":"/tmp/probe.txt"}}}}
{"type":"step_finish","sessionID":"ses_fb5af5","part":{"type":"step-finish","reason":"stop","cost":0.02}}
{"type":"step_finish","sessionID":"ses_fb5af5","part":{"type":"step-finish","reason":"stop","cost":0.03}}`

	got, err := ParseOpenCodeStream(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatalf("ParseOpenCodeStream: %v", err)
	}

	if got.SessionID != "ses_fb5af5" {
		t.Errorf("SessionID = %q — opencode spells it sessionID, and nothing was reading it", got.SessionID)
	}

	if got.Output != "writing it" {
		t.Errorf("Output = %q, want the text part", got.Output)
	}

	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "write" {
		t.Errorf("ToolCalls = %+v, want the one write", got.ToolCalls)
	}

	// opencode prices each step as it finishes, so a run of two steps
	// reports two figures and the run cost the sum.
	if got.Cost != 0.05 {
		t.Errorf("Cost = %v, want 0.05 — the two steps summed", got.Cost)
	}
}

// TestAStreamWithNothingInItSaysSo. A phase that produced nothing and a
// phase whose event shape moved under us both end with an empty Result, and
// only one of them is a phase. Saying so is what makes the second noticed.
func TestAStreamWithNothingInItSaysSo(t *testing.T) {
	for _, c := range []struct {
		name  string
		parse func(*strings.Reader, func(StreamEvent)) (Result, error)
	}{
		{"codex", func(r *strings.Reader, f func(StreamEvent)) (Result, error) { return ParseCodexStream(r, f) }},
		{"opencode", func(r *strings.Reader, f func(StreamEvent)) (Result, error) { return ParseOpenCodeStream(r, f) }},
	} {
		if _, err := c.parse(strings.NewReader("some prose\n{\"type\":\"who_knows\"}\n"), nil); err == nil {
			t.Errorf("%s read a stream with no event it knows and called it a finished phase", c.name)
		}
	}
}

// TestTheThinkingBudgetIsANumberOrNothing.
//
// MAX_THINKING_TOKENS was set to whatever the dial said, so a phase whose
// thinking field held a word passed that word to claude as a token budget.
// The comment above it already promised "a positive integer".
func TestTheThinkingBudgetIsANumberOrNothing(t *testing.T) {
	for _, c := range []struct {
		thinking string
		want     string
	}{
		{"", ""},
		{"adaptive", ""},
		{"on", ""},
		{"off", "MAX_THINKING_TOKENS=0"},
		{"none", "MAX_THINKING_TOKENS=0"},
		{"0", "MAX_THINKING_TOKENS=0"},
		{"31999", "MAX_THINKING_TOKENS=31999"},
		{"lots", ""},
		{"-5", ""},
		{"12.5", ""},
		{"1e6", ""},
	} {
		got := claudeEnv(Request{Thinking: c.thinking})

		var have string
		if len(got) > 0 {
			have = got[0]
		}

		if have != c.want {
			t.Errorf("thinking %q set %q, want %q", c.thinking, have, c.want)
		}
	}
}
