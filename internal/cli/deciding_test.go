package cli

// What the cockpit reports about the decision engine, on both halves.
//
// The window draws its warning out of Deciding and the log line comes out
// of keyed, and neither can be checked from inside internal/ui: the setting
// lives in a file this package owns and the key lives in an environment
// internal/ui is not allowed to read. This is where both are reachable at
// once, which is the same reason the adapter answers them together.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/env"
	"github.com/e1i0r/orbit/internal/store"
)

// TestDecidingReportsTheSettingAndTheMissingKey.
func TestDecidingReportsTheSettingAndTheMissingKey(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	adapter, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}

	// Shipped: nothing switched on, so nothing to say whatever the
	// environment holds.
	t.Setenv(env.DecisionKey, "")

	if allowed, missing := adapter.Deciding(); allowed != "" {
		t.Errorf("a fresh settings file allows %q, want off to read as empty (missing %q)",
			allowed, missing)
	}

	for _, state := range []string{store.DecisionsShadow, store.DecisionsOn} {
		if _, err := adapter.Choose(nil, "decisions", state); err != nil {
			t.Fatalf("set decisions to %s: %v", state, err)
		}

		// No key: both halves are set, which is the pair the window draws.
		t.Setenv(env.DecisionKey, "")

		allowed, missing := adapter.Deciding()
		if allowed != state || missing != env.DecisionKey {
			t.Errorf("decisions %s with no key = (%q, %q), want (%q, %q)",
				state, allowed, missing, state, env.DecisionKey)
		}

		// A key, and the second half goes quiet.
		t.Setenv(env.DecisionKey, "apikey_whatever")

		if allowed, missing = adapter.Deciding(); allowed != state || missing != "" {
			t.Errorf("decisions %s with a key = (%q, %q), want (%q, \"\")",
				state, allowed, missing, state)
		}
	}

	// Back off, with the key still there: off is off.
	if _, err := adapter.Choose(nil, "decisions", store.DecisionsOff); err != nil {
		t.Fatalf("set decisions off: %v", err)
	}

	if allowed, _ := adapter.Deciding(); allowed != "" {
		t.Errorf("decisions off allows %q, want empty", allowed)
	}
}

// TestDecidingReadsTheKeyEveryTime.
//
// The settings half is cached on purpose, because the window asks several
// times a frame. The key is not, and must not be: a cached "no key" would
// outlive the only fix there is for it.
func TestDecidingReadsTheKeyEveryTime(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	adapter, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}

	if _, err := adapter.Choose(nil, "decisions", store.DecisionsOn); err != nil {
		t.Fatalf("set decisions on: %v", err)
	}

	t.Setenv(env.DecisionKey, "")

	if _, missing := adapter.Deciding(); missing == "" {
		t.Fatal("no key in the environment and Deciding says nothing is missing")
	}

	t.Setenv(env.DecisionKey, "apikey_whatever")

	if _, missing := adapter.Deciding(); missing != "" {
		t.Errorf("the key was exported and Deciding still reports %q missing", missing)
	}
}

// TestTheRunLogsWhichHalfItGot: the line a run writes before its first
// gate, in both directions.
//
// It names the variable rather than saying "no key", because a reader
// opening orbit.log after a run that waited on them needs to know which
// word to export, not that a word exists.
func TestTheRunLogsWhichHalfItGot(t *testing.T) {
	with := keyed(true)
	if strings.Contains(with, env.DecisionKey) || !strings.Contains(with, "decide") {
		t.Errorf("keyed(true) = %q, want it to say the engine will decide", with)
	}

	without := keyed(false)
	if !strings.Contains(without, env.DecisionKey) {
		t.Errorf("keyed(false) = %q, want the name of the variable to export", without)
	}

	if !strings.Contains(without, "waits for a person") {
		t.Errorf("keyed(false) = %q, want it to say what happens at a gate instead", without)
	}
}
