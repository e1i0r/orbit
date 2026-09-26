package engine

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestClineInterface(t *testing.T) {
	c := NewCline()
	if c.Name() != "cline" {
		t.Errorf("Name() = %q, want cline", c.Name())
	}

	// --id opens cline's terminal and drops the prompt, so a headless run
	// cannot carry on a session.
	if c.CanResume() {
		t.Error("CanResume() = true; cline cannot resume a session headless")
	}

	if c.CanThink() {
		t.Error("CanThink() = true; the thinking level is the effort")
	}

	for _, m := range c.Models() {
		if m.ID != "" && !strings.HasPrefix(m.ID, clinePass+"/") {
			t.Errorf("model %q is not one of cline-pass's", m.ID)
		}

		if m.ID != "" && strings.Contains(m.Label, "/") {
			t.Errorf("model %q is labelled with its provider, %q", m.ID, m.Label)
		}
	}
}

// TestClineArgs is the argv cline 3.0.62 parses for a headless run: its
// event stream with the run_start line that carries the session, every tool
// approved, the directory, the provider the model belongs to, and the
// thinking level as the effort.
func TestClineArgs(t *testing.T) {
	got, err := clineArgs(Request{
		Prompt:      "refactor the handler",
		Model:       "cline-pass/glm-5.2",
		Effort:      "high",
		Dir:         "/w/ACME-1",
		Permissions: []string{PermissionRead, PermissionRepo},
	})
	if err != nil {
		t.Fatalf("clineArgs: %v", err)
	}

	want := []string{
		"--json", "-v", "--yolo", "--cwd", "/w/ACME-1",
		"-P", "cline-pass", "-m", "cline-pass/glm-5.2",
		"--thinking", "high", "refactor the handler",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("clineArgs =\n%q\nwant\n%q", got, want)
	}

	// A model with no provider cline-pass would answer to goes to whichever
	// provider cline was signed into, as it is.
	got, err = clineArgs(Request{Prompt: "p", Model: "claude-sonnet-4-6", Permissions: []string{PermissionRepo}})
	if err != nil {
		t.Fatalf("clineArgs: %v", err)
	}

	if strings.Contains(strings.Join(got, " "), "-P") {
		t.Errorf("a bare model was sent with a provider: %q", got)
	}
}

// TestClineRefusesAPostureItCannotHold. A headless cline denies every tool
// it would have to ask about, so a read posture would be recorded and not
// enforced.
func TestClineRefusesAPostureItCannotHold(t *testing.T) {
	if _, err := clineArgs(Request{Prompt: "p", Permissions: []string{PermissionRead}}); err == nil {
		t.Error("a read-only phase was given a cline command line")
	}
}

// TestClineRunsHere. Left to itself cline hands a run to its background hub,
// outside the phase's process and environment.
func TestClineRunsHere(t *testing.T) {
	if env := clineEnv(Request{}); !reflect.DeepEqual(env, []string{"CLINE_SESSION_BACKEND_MODE=local"}) {
		t.Errorf("clineEnv = %q", env)
	}
}

// TestClineSaysWhenItsAllowanceIsGone, in its own subscription's words and in
// a provider's.
func TestClineSaysWhenItsAllowanceIsGone(t *testing.T) {
	c := NewCline()

	for _, said := range []string{
		"ClinePass limit reached. Try again later.",
		"Daily free model limit reached",
		"429 Too Many Requests",
	} {
		if !c.RanOut(Result{Output: said}, nil) {
			t.Errorf("%q was not read as the allowance gone", said)
		}
	}

	if c.RanOut(Result{}, errors.New("exit status 1: go test failed")) {
		t.Error("a failing test was read as the allowance gone")
	}
}
