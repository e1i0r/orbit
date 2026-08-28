package cli

// flows() branches the hand-written flows_test.go never reaches: a bad
// flag, a state root store.Open cannot create, a settings file it cannot
// read, and flowMark's default case, which List (internal/flow) never
// actually returns but which this file's own doc comment says is answered
// with "" rather than a panic.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

func TestFlowsEarlyExitOnBadFlag(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	if code, _, errOut := run(t, "flows", "-nosuchflag"); code == 0 {
		t.Error("flows with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("flows failed silently on a bad flag")
	}
}

func TestFlowsFailsWhenTheStateRootCannotBeCreated(t *testing.T) {
	// A regular file where $ORBIT_HOME would go: store.Open's os.MkdirAll
	// fails on it, since a file cannot be turned into a directory.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ORBIT_HOME", filepath.Join(blocker, "orbit"))

	code, _, errOut := run(t, "flows")
	if code == 0 {
		t.Error("flows over an unmakeable state root exited 0")
	}

	if errOut == "" {
		t.Error("flows failed silently over an unmakeable state root")
	}
}

func TestFlowsFailsWhenSettingsCannotBeRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)
	// A directory where settings.json goes fails the read for every user.
	if err := os.MkdirAll(filepath.Join(home, "settings.json"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	code, _, errOut := run(t, "flows")
	if code == 0 {
		t.Error("flows over unreadable settings exited 0")
	}

	if errOut == "" {
		t.Error("flows failed silently over unreadable settings")
	}
}

func TestFlowMarkAnswersEmptyForAnUnclassifiedOrigin(t *testing.T) {
	p := printer()
	if got := flowMark(p, flow.OriginUnknown); got != "" {
		t.Errorf("flowMark(OriginUnknown) = %q, want empty", got)
	}
}
