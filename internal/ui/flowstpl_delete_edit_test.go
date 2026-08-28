package ui

// flowstpl_coverage_test.go exercises the flow lifecycle actions that touch
// a directory: delete, edit and the template presets. The fixture board has
// no flow directory of its own, so every test here that is about a reader's
// own flow builds one with flowsTestDir.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

// flowsTestDir is a reader's own flow directory, as flow.Source asks for
// it. It has a name of this file's own rather than a shared fixture's,
// since three agents are editing this package at the same time and a
// generic name risks a duplicate symbol.
type flowsTestDir string

func (d flowsTestDir) FlowDir() string { return string(d) }

// writeFlowFile writes one flow file into dir, in the shape flow.Load
// expects.
func writeFlowFile(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestDeleteSelectedFlowAndDeleteFlow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.flows.refresh(m.opts.Flows)

	// 1. Nothing selected: a no-op.
	m.flows.sel = -1

	m2, cmd := m.deleteSelectedFlow()
	if cmd != nil || m2.flows.confirmDelete {
		t.Fatalf("expected a no-op with nothing selected, got confirmDelete=%v cmd=%v", m2.flows.confirmDelete, cmd)
	}

	// 2. A built-in flow cannot be deleted. "careful" sorts first among the
	// three built-ins the fixture ships.
	m.flows.sel = 0
	m2, _ = m.deleteSelectedFlow()
	wantBand(t, m2, "cannot be deleted")

	if m2.flows.confirmDelete {
		t.Fatalf("a built-in should never reach the confirmation")
	}

	// 3. A reader's own flow asks for confirmation instead of acting.
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)
	m.opts.Flows = flowsTestDir(dir)
	m.flows.refresh(m.opts.Flows)
	// Sorted: careful, quick, task, tdd-fuzz-pr, zzz-mine.
	m.flows.sel = 4

	m2, _ = m.deleteSelectedFlow()
	if !m2.flows.confirmDelete {
		t.Fatalf("expected confirmDelete after asking to delete a reader's own flow")
	}

	wantBand(t, m2, "zzz-mine")

	// 4. deleteFlow itself, called directly for each origin.
	m3, _ := m.deleteFlow("built-in-name", flow.OriginBuiltin)
	wantBand(t, m3, "cannot be deleted")

	m4, _ := m.deleteFlow("zzz-mine", flow.OriginUser)
	if !m4.flows.confirmDelete {
		t.Fatalf("expected confirmDelete for a user-origin flow")
	}
}

func TestConfirmDeleteFlow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.flows.refresh(m.opts.Flows)

	// 1. Out of range: a no-op that still clears confirmDelete.
	m.flows.confirmDelete = true
	m.flows.sel = 99

	m2, cmd := m.confirmDeleteFlow()
	if cmd != nil || m2.flows.confirmDelete {
		t.Fatalf("expected confirmDelete cleared and no cmd for an out-of-range selection")
	}

	// 2. A built-in flow refuses even once confirmed.
	m.flows.sel = 0
	m2, _ = m.confirmDeleteFlow()
	wantBand(t, m2, "cannot be deleted")

	// 3. A reader's own flow is actually removed, and the selection steps
	// back when it was not already at the top.
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)
	m.opts.Flows = flowsTestDir(dir)
	m.flows.refresh(m.opts.Flows)
	m.flows.sel = 4 // careful, quick, task, tdd-fuzz-pr, zzz-mine
	m2, _ = m.confirmDeleteFlow()
	wantBand(t, m2, "zzz-mine")

	if m2.flows.sel != 3 {
		t.Errorf("sel after delete = %d, want 3 (stepped back)", m2.flows.sel)
	}

	if _, err := os.Stat(filepath.Join(dir, "zzz-mine.json")); !os.IsNotExist(err) {
		t.Errorf("expected zzz-mine.json to be gone, stat err = %v", err)
	}

	// 4. Deleting the first entry leaves the selection at zero rather than
	// going negative.
	dir2 := t.TempDir()
	writeFlowFile(t, dir2, "aaa-mine", `{"name":"aaa-mine","phases":[{"name":"implement","engine":"claude"}]}`)
	m.opts.Flows = flowsTestDir(dir2)
	m.flows.refresh(m.opts.Flows)
	m.flows.sel = 0 // aaa-mine sorts before every built-in
	m2, _ = m.confirmDeleteFlow()
	wantBand(t, m2, "aaa-mine")

	if m2.flows.sel != 0 {
		t.Errorf("sel after delete = %d, want 0", m2.flows.sel)
	}
}

// TestConfirmDeleteFlowRemoveError forces os.Remove to fail by taking write
// permission off the directory the flow file lives in — deleting a file
// needs write on its parent, not on the file itself — so the error branch
// is reached without guessing at a filesystem race.
func TestConfirmDeleteFlowRemoveError(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "blocked-mine", `{"name":"blocked-mine","phases":[{"name":"implement","engine":"claude"}]}`)
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:errcheck

	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m.flows.refresh(m.opts.Flows)
	descriptors := flow.List(m.opts.Flows)
	idx := -1

	for i, d := range descriptors {
		if d.Name == "blocked-mine" {
			idx = i
		}
	}

	if idx == -1 {
		t.Fatalf("blocked-mine not listed among %v", descriptors)
	}

	m.flows.sel = idx

	m2, cmd := m.confirmDeleteFlow()
	if cmd != nil {
		t.Fatalf("expected nil cmd on a remove failure")
	}

	if m2.message == "" {
		t.Fatalf("expected the removal error to reach the band")
	}

	if _, err := os.Stat(filepath.Join(dir, "blocked-mine.json")); err != nil {
		t.Errorf("file should still be there after a failed remove: %v", err)
	}
}

func TestEditSelectedFlowOutOfRange(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.flows.sel = -1

	m2, cmd := m.editSelectedFlow()
	if cmd != nil || m2.flows.creating {
		t.Fatalf("expected a no-op with nothing selected")
	}
}

func TestEditSelectedFlowOpensTheBuilder(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.flows.refresh(m.opts.Flows)
	m.flows.sel = 0 // "careful"

	m2, _ := m.editSelectedFlow()
	if !m2.flows.creating || !m2.flows.isEditing {
		t.Fatalf("expected the builder open in edit mode")
	}

	if m2.flows.flowName != "careful" {
		t.Errorf("flowName = %q, want careful", m2.flows.flowName)
	}

	wantBand(t, m2, "careful")
}

func TestApplyFlowTemplate(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	cases := []struct {
		tpl        string
		wantName   string
		wantPhases int
	}{
		{"TDD Cycle", "tdd-cycle", 3},
		{"Security Audit", "security-audit", 2},
		{"Turbo Fix", "turbo-fix", 1},
		{"ninguna", "", 1},
	}
	for _, c := range cases {
		m2, _ := m.applyFlowTemplate(c.tpl)
		if m2.flows.flowName != c.wantName {
			t.Errorf("applyFlowTemplate(%q).flowName = %q, want %q", c.tpl, m2.flows.flowName, c.wantName)
		}

		if len(m2.flows.phases) != c.wantPhases {
			t.Errorf("applyFlowTemplate(%q) has %d phases, want %d", c.tpl, len(m2.flows.phases), c.wantPhases)
		}
	}

	// An unrecognised preset changes nothing.
	before := m.flows

	m2, cmd := m.applyFlowTemplate("not-a-real-preset")
	if cmd != nil {
		t.Errorf("expected nil cmd for an unknown template")
	}

	if len(m2.flows.phases) != len(before.phases) || m2.flows.flowName != before.flowName {
		t.Errorf("an unknown template should leave the form alone")
	}
}

// TestDeletingAShadowSaysTheBuiltInIsBack.
//
// A flow of the reader's own named after a built-in does not replace it, it
// covers it. Deleting the file therefore does not remove the flow — it
// restores the shipped one, and every task written against that name goes on
// running, differently. The window removed the file with os.Remove, which
// cannot report that, so the band said "deleted" and left the reader to
// discover the change by running it. flow.Delete has always answered this
// question; nothing was asking it.
func TestDeletingAShadowSaysTheBuiltInIsBack(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "quick", `{"name":"quick","phases":[{"name":"implement","engine":"claude"}]}`)

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m.flows.refresh(m.opts.Flows)

	sel := -1

	for i, d := range flow.List(m.opts.Flows) {
		if d.Name == "quick" {
			sel = i
		}
	}

	if sel < 0 {
		t.Fatal("the flow shadowing quick is not in the list")
	}

	m.flows.sel = sel

	m2, _ := m.confirmDeleteFlow()
	wantBand(t, m2, "showing again")
}
