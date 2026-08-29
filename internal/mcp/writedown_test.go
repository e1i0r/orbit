package mcp

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// logs is the log and the errors file, as text, after fn has run against a
// logger of its own. The logger is a package-level global, so no test here
// runs in parallel with another.
func logs(t *testing.T, fn func()) (string, string) {
	t.Helper()

	dir := t.TempDir()
	all, bad := filepath.Join(dir, "orbit.log"), filepath.Join(dir, "errors.log")

	if err := logger.Init(all, bad); err != nil {
		t.Fatalf("logger.Init: %v", err)
	}

	fn()

	if err := logger.CloseGlobal(); err != nil {
		t.Fatalf("logger.CloseGlobal: %v", err)
	}

	return read(t, all), read(t, bad)
}

// TestEveryToolCallIsWrittenDownWithWhatItWasGiven. A tool nobody wrote down
// is a tool nobody can account for afterwards, and the arguments are half of
// what makes the line worth having: orbit_list_tasks against payments and
// orbit_list_tasks against every repository on the machine are the same
// sentence without them.
func TestEveryToolCallIsWrittenDownWithWhatItWasGiven(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "ORB-1")

	all, _ := logs(t, func() {
		sn.Call("orbit_list_tasks", map[string]any{"repo": r.Path, "limit": float64(5)})
	})

	for _, want := range []string{
		"[mcp/orbit_list_tasks]",
		"called with limit=5 repo=" + strconv.Quote(r.Path),
		"answered after",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the log does not contain %q:\n%s", want, all)
		}
	}
}

// TestTheArgumentsAreWrittenInOneOrderWhateverOrderTheyArrivedIn. A map
// ranges differently every run, so an unsorted line makes two identical
// calls look like two different ones and makes the file ungreppable.
func TestTheArgumentsAreWrittenInOneOrderWhateverOrderTheyArrivedIn(t *testing.T) {
	args := map[string]any{"zulu": "z", "alpha": "a", "mike": "m", "bravo": "b"}

	first := argsLine(args)
	for range 32 {
		if got := argsLine(args); got != first {
			t.Fatalf("argsLine answered %q and then %q for the same arguments", first, got)
		}
	}

	if want := `alpha="a" bravo="b" mike="m" zulu="z"`; first != want {
		t.Errorf("argsLine = %q, want %q", first, want)
	}
}

// TestARefusalIsAnErrorAndSaysWhatItRefused is the whole reason this file
// exists. A refusal is answered to the model that asked and to nobody else:
// without this line, a supervisor that spent an afternoon calling tools
// against task ids it invented leaves an Orbit that looked idle.
func TestARefusalIsAnErrorAndSaysWhatItRefused(t *testing.T) {
	_, sn, r := oneRepo(t)

	all, bad := logs(t, func() {
		refused(t, sn, "orbit_inspect_task", map[string]any{"repo": r.Path, "task_id": "ORB-404"})
	})

	if !strings.Contains(bad, "ORB-404") {
		t.Errorf("the errors file does not name the task that was refused:\n%s", bad)
	}

	if !strings.Contains(bad, "[ERROR]") || !strings.Contains(bad, "refused after") {
		t.Errorf("a refusal was not written down as a refusal:\n%s", bad)
	}

	if !strings.Contains(all, "ORB-404") {
		t.Errorf("the log kept a refusal out of the file that holds everything:\n%s", all)
	}
}

// TestAToolIsTimedFromBeforeItRunsAndNotAfterIt. noteCall answers with the
// moment it wrote, and that moment is what noteAnswer subtracts. Reading the
// clock inside noteAnswer instead would time nothing at all, and every tool
// in the file would have answered in under a millisecond.
func TestAToolIsTimedFromBeforeItRunsAndNotAfterIt(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "ORB-1")

	all, _ := logs(t, func() {
		start := noteCall("orbit_slow", nil)

		sn.Call("orbit_list_tasks", map[string]any{"repo": r.Path})
		time.Sleep(60 * time.Millisecond)
		noteAnswer("orbit_slow", start, done("ok"))
	})

	took := regexp.MustCompile(`\[mcp/orbit_slow\] answered after ([0-9.]+)ms`).FindStringSubmatch(all)
	if took == nil {
		t.Fatalf("no timed answer for orbit_slow in:\n%s", all)
	}

	ms, err := strconv.ParseFloat(took[1], 64)
	if err != nil {
		t.Fatalf("parse %q: %v", took[1], err)
	}

	if ms < 50 {
		t.Errorf("a tool that slept 60ms answered in %sms, so the clock is read after the work", took[1])
	}
}

// TestAnArgumentTooLongToReadIsCountedAndNotCopied. Several of this server's
// arguments are prose a model wrote — a prompt, a note, a directive, a
// correction — and the record already holds every one of them in full.
// Copying them here would make the log the size of the work and put a task's
// whole brief in a file that is read for what went wrong.
func TestAnArgumentTooLongToReadIsCountedAndNotCopied(t *testing.T) {
	brief := strings.Repeat("ship the thing. ", 40)

	all, _ := logs(t, func() {
		noteAnswer("orbit_x", noteCall("orbit_x", map[string]any{"prompt": brief}), done("ok"))
	})

	if strings.Contains(all, "ship the thing") {
		t.Errorf("a prompt was copied into the log:\n%s", all)
	}

	if want := "prompt=(640 characters)"; !strings.Contains(all, want) {
		t.Errorf("the log does not contain %q, so a long argument is not even counted:\n%s", want, all)
	}
}

// TestWhatIsShortEnoughToReadIsWrittenOutInFull. The counting above is for
// paragraphs. A repository path, a task id, a phase name and a boolean are
// the facts that make a line worth reading, and every one of them is short.
func TestWhatIsShortEnoughToReadIsWrittenOutInFull(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
		why  string
	}{
		{"ORB-12", `"ORB-12"`, "a task id is quoted so an empty one is visible as one"},
		{"", `""`, "an argument that arrived empty is a fact, not an absence"},
		{true, "true", "a boolean is not a string and is not quoted like one"},
		{float64(5), "5", "JSON numbers arrive as float64 and read as numbers"},
		{strings.Repeat("x", argChars), strconv.Quote(strings.Repeat("x", argChars)), "the limit itself is still written out"},
		{strings.Repeat("x", argChars+1), "(201 characters)", "one rune past the limit is counted instead"},
		{strings.Repeat("é", argChars+1), "(201 characters)", "the limit is runes, not bytes"},
	} {
		if got := argValue(c.v); got != c.want {
			t.Errorf("argValue(%v) = %q, want %q — %s", c.v, got, c.want, c.why)
		}
	}

	if got := argsLine(nil); got != "no arguments" {
		t.Errorf("argsLine(nil) = %q, want a sentence saying there were none", got)
	}
}

// TestARequestThisServerCouldNotAnswerIsAWarningAndNotAFailure. The four
// faults below are things a client sent, not things Orbit did, and the
// session goes on after every one of them. An errors file that fills up with
// handled faults is an errors file nobody reads.
func TestARequestThisServerCouldNotAnswerIsAWarningAndNotAFailure(t *testing.T) {
	_, sn, _ := oneRepo(t)

	for _, c := range []struct {
		in   string
		want string
	}{
		{"{not json", "would not parse"},
		{`{"jsonrpc":"2.0","id":1,"method":"tools/fly"}`, "no method"},
		{`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":7}`, "would not decode"},
		{`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{}}`, "no tool was named"},
	} {
		all, bad := logs(t, func() {
			exchange(t, sn, c.in)
		})

		if !strings.Contains(all, c.want) {
			t.Errorf("%q left nothing about %q in the log:\n%s", c.in, c.want, all)
		}

		if !strings.Contains(all, "[WARN] [mcp/transport]") {
			t.Errorf("%q was not written down as a warning from the transport:\n%s", c.in, all)
		}

		if strings.TrimSpace(bad) != "" {
			t.Errorf("%q, which this server answered, was written to the errors file:\n%s", c.in, bad)
		}
	}
}

// TestAToolThatCrashedDoesNotEndTheSession. Nineteen handlers share one
// pipe, and a model drives that pipe for hours. Without the recover, a panic
// in any of them closes the conversation mid-sentence: the client sees the
// stream end and cannot tell which of its requests was the last one
// answered. The refusal below is what it gets instead.
func TestAToolThatCrashedDoesNotEndTheSession(t *testing.T) {
	var after CallToolResult

	all, bad := logs(t, func() {
		crashed := guard("orbit_boom", map[string]any{"task_id": "ORB-1"}, func() CallToolResult {
			panic("a nil map somewhere")
		})

		if !crashed.IsError {
			t.Errorf("a tool that panicked answered without an error: %+v", crashed)
		}

		if !strings.Contains(text(t, crashed), "crashed") {
			t.Errorf("the refusal does not say the tool crashed: %s", text(t, crashed))
		}

		// The next call is the point: the goroutine is still here.
		after = guard("orbit_fine", nil, func() CallToolResult { return done("still here") })
	})

	if after.IsError || !strings.Contains(text(t, after), "still here") {
		t.Errorf("the call after the crash did not run: %+v", after)
	}

	for _, want := range []string{"panicked: a nil map somewhere", "runtime/debug", "mcp/orbit_boom"} {
		if !strings.Contains(bad, want) {
			t.Errorf("the errors file does not contain %q, so the stack was lost:\n%s", want, bad)
		}
	}

	if !strings.Contains(all, "orbit_fine") {
		t.Errorf("the log stopped at the crash:\n%s", all)
	}
}

// TestACrashIsStillTimedAndStillWrittenDownAsAnAnswer. The recover runs
// inside the same deferred function that writes the answer line, so a tool
// that died is accounted for the way every other one is — one call line, one
// answer line — rather than leaving a call in the log that nothing closes.
func TestACrashIsStillTimedAndStillWrittenDownAsAnAnswer(t *testing.T) {
	all, _ := logs(t, func() {
		guard("orbit_boom", nil, func() CallToolResult { panic("x") })
	})

	if n := strings.Count(all, "[mcp/orbit_boom]"); n != 3 {
		t.Errorf("a crashed tool left %d lines, want the call, the panic and the answer:\n%s", n, all)
	}

	if !strings.Contains(all, "refused after") {
		t.Errorf("a crash was not written down as an answer:\n%s", all)
	}
}
