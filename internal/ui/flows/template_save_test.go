package flows

// flowstpl_save_coverage_test.go is every way saveCustomFlow can fail or
// succeed. Every case that reaches os.MkdirAll or os.WriteFile is pointed
// at a directory of the test's own. The one case that is about the
// fallback under $HOME redirects HOME to a temp dir with t.Setenv rather
// than touching the real one.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCustomFlowNoName(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "   "

	m2, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd when the name is blank")
	}

	if !m2.creating {
		t.Fatalf("a rejected save should leave the builder open")
	}

	wantBand(t, out, "give the flow a name")
}

func TestSaveCustomFlowInvalid(t *testing.T) {
	s, e := designer(t)
	e.Flows = flowsTestDir(t.TempDir())
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "no-engine"
	s.phases[0].Engine = ""

	_, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd for a flow that fails Validate")
	}

	wantBand(t, out, "no-engine")
}

func TestSaveCustomFlowMkdirAllFails(t *testing.T) {
	dir := t.TempDir()

	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	s, e := designer(t)
	e.Flows = flowsTestDir(blocked)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "wont-save"

	m2, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd when MkdirAll fails")
	}

	if !m2.creating {
		t.Fatalf("a failed save should leave the builder open")
	}

	if out.Said == "" {
		t.Fatalf("expected the MkdirAll error to reach the band")
	}
}

func TestSaveCustomFlowWriteFileFails(t *testing.T) {
	dir := t.TempDir()
	// A directory sitting where the flow file would be written turns
	// os.WriteFile into a failure of its own, distinct from MkdirAll's.
	if err := os.MkdirAll(filepath.Join(dir, "in-the-way.json"), 0o755); err != nil {
		t.Fatalf("seed dir: %v", err)
	}

	s, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "in-the-way"

	_, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd when WriteFile fails")
	}

	if out.Said == "" {
		t.Fatalf("expected the WriteFile error to reach the band")
	}
}

func TestSaveCustomFlowSucceeds(t *testing.T) {
	dir := t.TempDir()
	s, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "my-new-flow"

	m2, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd on a successful save")
	}

	if m2.creating {
		t.Errorf("a successful save should close the builder")
	}

	if m2.phases != nil || m2.flowName != "" {
		t.Errorf("a successful save should clear the form")
	}

	wantBand(t, out, "my-new-flow")

	written, err := os.ReadFile(filepath.Join(dir, "my-new-flow.json"))
	if err != nil {
		t.Fatalf("expected the flow file on disk: %v", err)
	}

	if len(written) == 0 {
		t.Errorf("expected a non-empty flow file")
	}
}

// TestAWindowWithNowhereToSaveSaysSoInsteadOfGuessing.
//
// A window opened without a state root — e.Flows nil, which is the
// fixture's own default — must not fall back to $HOME/.orbit/flows and
// report the flow saved. On any machine with $ORBIT_HOME pointed elsewhere,
// and that is the whole reason the variable exists, the flow would go into a
// directory nothing reads: the band saying saved, the list never showing it
// again, and no error ever raised. Refusing is the honest answer, and it
// is the one internal/flow already gives.
//
// HOME is redirected for the length of the test so that a regression writes
// into a temp directory rather than the machine's real home.
func TestAWindowWithNowhereToSaveSaysSoInsteadOfGuessing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ORBIT_HOME", filepath.Join(home, "elsewhere"))

	s, e := designer(t)
	e.Flows = nil // nowhere to keep one
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "home-fallback-flow"

	m2, out := s.saveCustomFlow(e)
	if out.Cmd != nil {
		t.Fatalf("expected nil cmd")
	}

	wantBand(t, out, "nowhere to keep a flow")

	if m2.creating != true {
		t.Error("a save that did not happen has to leave the builder open")
	}

	if _, err := os.Stat(filepath.Join(home, ".orbit", "flows", "home-fallback-flow.json")); err == nil {
		t.Error("the flow was written under ~/.orbit/flows, a directory this window has no business naming")
	}
}

// TestAFlowNameCannotClimbOutOfTheFlowDirectory.
//
// The name is a text field, and it was joined onto the flow directory and
// written with no check at all. filepath.Join resolves the .. as it goes, so
// a flow called ../notes was written one level above the flows directory —
// over whatever was there. flow.ValidName refuses the name instead, and it
// is the same check `orbit set` and the MCP server already make.
func TestAFlowNameCannotClimbOutOfTheFlowDirectory(t *testing.T) {
	root := t.TempDir()

	flows := filepath.Join(root, "flows")
	if err := os.MkdirAll(flows, 0o700); err != nil {
		t.Fatalf("seed the flow dir: %v", err)
	}

	for _, name := range []string{"../escaped", "sub/escaped", ".."} {
		s, e := designer(t)
		e.Flows = flowsTestDir(flows)
		s = s.startCreateFlow(e).OnFields()
		s.flowName = name

		if _, out := s.saveCustomFlow(e); out.Cmd != nil {
			t.Fatalf("expected nil cmd for %q", name)
		}

		if _, err := os.Stat(filepath.Join(root, "escaped.json")); err == nil {
			t.Fatalf("the flow named %q was written outside the flow directory", name)
		}
	}
}

// TestASavedFlowIsNoWiderThanTheStateRoot. internal/flow spells out 0700 and
// 0600 and says why: a flow directory that ended up group-readable because
// two packages disagreed about a number would be a quiet widening. The
// window was the second package, and it disagreed — 0755 and 0644.
func TestASavedFlowIsNoWiderThanTheStateRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "flows")

	s, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "narrow"

	if _, out := s.saveCustomFlow(e); out.Cmd != nil {
		t.Fatalf("expected nil cmd on a successful save")
	}

	for _, c := range []struct {
		path string
		want os.FileMode
	}{
		{dir, 0o700},
		{filepath.Join(dir, "narrow.json"), 0o600},
	} {
		info, err := os.Stat(c.path)
		if err != nil {
			t.Fatalf("stat %s: %v", c.path, err)
		}

		if got := info.Mode().Perm(); got != c.want {
			t.Errorf("%s is %o, want %o", c.path, got, c.want)
		}
	}
}
