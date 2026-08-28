package ui

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
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.flowName = "   "

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd when the name is blank")
	}

	if !m2.flows.creating {
		t.Fatalf("a rejected save should leave the builder open")
	}

	wantBand(t, m2, "give the flow a name")
}

func TestSaveCustomFlowInvalid(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.flowName = "no-engine"
	m.flows.phases[0].Engine = ""

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd for a flow that fails Validate")
	}

	wantBand(t, m2, "no-engine")
}

func TestSaveCustomFlowMkdirAllFails(t *testing.T) {
	dir := t.TempDir()

	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(blocked)
	m = m.startCreateFlow()
	m.flows.flowName = "wont-save"

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd when MkdirAll fails")
	}

	if !m2.flows.creating {
		t.Fatalf("a failed save should leave the builder open")
	}

	if m2.message == "" {
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

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m = m.startCreateFlow()
	m.flows.flowName = "in-the-way"

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd when WriteFile fails")
	}

	if m2.message == "" {
		t.Fatalf("expected the WriteFile error to reach the band")
	}
}

func TestSaveCustomFlowSucceeds(t *testing.T) {
	dir := t.TempDir()
	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m = m.startCreateFlow()
	m.flows.flowName = "my-new-flow"

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd on a successful save")
	}

	if m2.flows.creating {
		t.Errorf("a successful save should close the builder")
	}

	if m2.flows.phases != nil || m2.flows.flowName != "" {
		t.Errorf("a successful save should clear the form")
	}

	wantBand(t, m2, "my-new-flow")

	written, err := os.ReadFile(filepath.Join(dir, "my-new-flow.json"))
	if err != nil {
		t.Fatalf("expected the flow file on disk: %v", err)
	}

	if len(written) == 0 {
		t.Errorf("expected a non-empty flow file")
	}
}

// TestSaveCustomFlowDefaultHomeDir is a window opened without a state root
// at all — m.opts.Flows is nil, the same as the fixture's own default — so
// saveCustomFlow falls back to $HOME/.orbit/flows. HOME is redirected to a
// temp dir for the length of the test, so the fallback is exercised without
// writing anywhere near the machine's real home directory.
func TestSaveCustomFlowDefaultHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.flowName = "home-fallback-flow"

	m2, cmd := m.saveCustomFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd on a successful save")
	}

	wantBand(t, m2, "home-fallback-flow")

	if _, err := os.Stat(filepath.Join(home, ".orbit", "flows", "home-fallback-flow.json")); err != nil {
		t.Errorf("expected the flow saved under the redirected home dir: %v", err)
	}
}
