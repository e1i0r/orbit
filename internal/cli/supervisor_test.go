package cli

import (
	"strings"
	"testing"
)

func TestSupervisorCommandReadEmptyHistory(t *testing.T) {
	_, _ = workspace(t)

	code, out, errOut := run(t, "supervisor")
	if code != 0 {
		t.Fatalf("supervisor exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "empty") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestSupervisorCommandWriteAndReadHistory(t *testing.T) {
	_, _ = workspace(t)

	code, out, errOut := run(t, "supervisor", "-by", "elio", "please focus on unit tests")
	if code != 0 {
		t.Fatalf("supervisor post exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "recorded in supervisor thread") {
		t.Errorf("unexpected post response: %s", out)
	}

	code, out, errOut = run(t, "supervisor")
	if code != 0 {
		t.Fatalf("supervisor list exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "please focus on unit tests") || !strings.Contains(out, "[elio via cli]") {
		t.Errorf("unexpected history output: %s", out)
	}
}

// TestSupervisorCommandNumbersLinesAndTakesOneBack is the fix from the
// terminal's side. A line has no id, so the listing's own numbering is what
// -retract reads back.
func TestSupervisorCommandNumbersLinesAndTakesOneBack(t *testing.T) {
	_, _ = workspace(t)
	for _, text := range []string{"first thing", "the one I regret", "third thing"} {
		if code, _, errOut := run(t, "supervisor", "-by", "elio", text); code != 0 {
			t.Fatalf("supervisor %q exited %d: %s", text, code, errOut)
		}
	}

	code, out, errOut := run(t, "supervisor")
	if code != 0 {
		t.Fatalf("supervisor list exited %d: %s", code, errOut)
	}

	for _, want := range []string{"1  ", "2  ", "3  "} {
		if !strings.Contains(out, want) {
			t.Errorf("the listing is not numbered; %q is missing from:\n%s", want, out)
		}
	}

	code, out, errOut = run(t, "supervisor", "-retract", "2")
	if code != 0 {
		t.Fatalf("supervisor -retract 2 exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "took back line 2") {
		t.Errorf("unexpected retract response: %s", out)
	}

	code, out, errOut = run(t, "supervisor")
	if code != 0 {
		t.Fatalf("supervisor list exited %d: %s", code, errOut)
	}
	// Still shown, and shown as withdrawn: it was said.
	if !strings.Contains(out, "the one I regret") || !strings.Contains(out, "(retracted)") {
		t.Errorf("the withdrawn line is not listed as withdrawn:\n%s", out)
	}

	if strings.Count(out, "\n") != 3 {
		t.Errorf("the retraction became a line of the conversation:\n%s", out)
	}
}

func TestSupervisorCommandRefusesARetractionItCannotPlace(t *testing.T) {
	_, _ = workspace(t)
	// Nothing to point at yet.
	if code, _, errOut := run(t, "supervisor", "-retract", "1"); code == 0 {
		t.Error("supervisor -retract 1 over an empty thread exited 0")
	} else if errOut == "" {
		t.Error("supervisor -retract failed silently over an empty thread")
	}

	if code, _, errOut := run(t, "supervisor", "-by", "elio", "the only turn"); code != 0 {
		t.Fatalf("supervisor post exited %d: %s", code, errOut)
	}

	for _, n := range []string{"0", "-1", "2"} {
		if code, _, errOut := run(t, "supervisor", "-retract", n); code == 0 {
			t.Errorf("supervisor -retract %s exited 0 over a thread of one line", n)
		} else if errOut == "" {
			t.Errorf("supervisor -retract %s failed silently", n)
		}
	}

	if code, _, errOut := run(t, "supervisor", "-retract", "1"); code != 0 {
		t.Fatalf("supervisor -retract 1 exited %d: %s", code, errOut)
	}
	// Twice is not a second retraction, it is a mistake worth saying out loud.
	if code, _, errOut := run(t, "supervisor", "-retract", "1"); code == 0 {
		t.Error("retracting the same line twice exited 0")
	} else if !strings.Contains(errOut, "already") {
		t.Errorf("unexpected second-retraction error: %s", errOut)
	}
}

// TestTheThreadAnswersInTheReadersLanguage. The thread is where a reader
// tells the fleet what to do, and everything it said back to them — the
// refusals and the confirmations both — was English whatever they chose.
func TestTheThreadAnswersInTheReadersLanguage(t *testing.T) {
	answers := func(t *testing.T, language string) []string {
		t.Helper()
		t.Setenv("ORBIT_HOME", t.TempDir())

		if code, _, errOut := run(t, "set", "language", language); code != 0 {
			t.Fatalf("set language %s exited %d: %s", language, code, errOut)
		}

		var said []string

		for _, args := range [][]string{
			{"supervisor", "-by", "elio", "the one I regret"},
			{"supervisor", "-retract", "two"},
			{"supervisor", "-retract", "9"},
			{"supervisor", "-retract", "1"},
			{"supervisor", "-retract", "1"},
		} {
			_, out, errOut := run(t, args...)
			said = append(said, out+errOut)
		}

		return said
	}

	english, spanish := answers(t, "en"), answers(t, "es")
	for i, want := range english {
		if spanish[i] == want {
			t.Errorf("both readers are told %q", want)
		}
	}
}
