package verb

// The names a verb answered to before it joined a family.

import (
	"strings"
	"testing"
)

// TestAnOldNameStillFindsItsVerb.
//
// A rename that breaks every script somebody wrote is not a rename; it is a
// removal with something new standing next to it.
func TestAnOldNameStillFindsItsVerb(t *testing.T) {
	for was, now := range map[string]string{
		"merge":    "pr merge",
		"close-pr": "pr close",
		"set":      "settings set",
	} {
		v, found := One(was)
		if !found {
			t.Errorf("%q finds nothing, and it is what %q used to be called", was, now)
			continue
		}

		if v.Path() != now {
			t.Errorf("%q finds %q, want %q", was, v.Path(), now)
		}
	}
}

// TestAnOldNameSaysWhatToTypeInstead, once, above whatever the verb
// answered. A script left on a name that is going away with nothing to tell
// whoever wrote it is how a rename becomes a breakage six months later.
func TestAnOldNameSaysWhatToTypeInstead(t *testing.T) {
	on := map[string]string{"key": "autopilot", "value": "on"}

	out, err := asked(t, trayOf(t), "set", on)
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	if !strings.HasPrefix(out.Said, "set is now settings set") {
		t.Errorf("asked by its old name, it said %q", out.Said)
	}
}

// TestTheNameItHasNowSaysNothingExtra. The notice is for whoever typed the
// old one; everybody else has a sentence to read that is about what they
// asked for.
func TestTheNameItHasNowSaysNothingExtra(t *testing.T) {
	on := map[string]string{"key": "autopilot", "value": "on"}

	out, err := asked(t, trayOf(t), "settings set", on)
	if err != nil {
		t.Fatalf("settings set: %v", err)
	}

	if strings.Contains(out.Said, "the old name still works") {
		t.Errorf("asked by the name it has, it said %q", out.Said)
	}
}

// TestNoNameMeansTwoThings.
//
// An old name is looked up the same way a current one is, so one that
// collides with a verb's own name would shadow it — and two verbs claiming
// the same old name would make which one answers depend on the order they
// happen to be declared in.
func TestNoNameMeansTwoThings(t *testing.T) {
	claimed := map[string]string{}

	for _, v := range Every() {
		claimed[v.Path()] = v.Path()
	}

	for _, v := range Every() {
		if v.Was == "" {
			continue
		}

		if by, taken := claimed[v.Was]; taken {
			t.Errorf("%q is what %q used to be called, and %q answers to it", v.Was, v.Path(), by)
		}

		claimed[v.Was] = v.Path()
	}
}
