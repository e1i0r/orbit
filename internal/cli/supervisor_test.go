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
