package ui

// palette_more_coverage_test.go is the ':' line's own arithmetic:
// which candidate a typed prefix keeps, where the pointer's rows land, and
// what the line itself draws — the parts
// palette_and_menu_coverage_test.go's lifecycle walk never has reason to
// hit.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

func paletteCommands() []Command {
	return []Command{
		{Name: "new", Args: "<repo> <title>", About: func(p *words.Printer) string { return "write a task" }},
		{Name: "settings"},
		{Name: "nope", Refused: true, Because: func(p *words.Printer) string { return "not from here" }},
	}
}

func TestMatchesSettingsAliasAndCandidatesFindTheAlias(t *testing.T) {
	for _, want := range []string{"conf", "config", "set", "ajus", "ajustes"} {
		if !matchesSettingsAlias(want) {
			t.Errorf("matchesSettingsAlias(%q) = false, want true", want)
		}
	}

	if matchesSettingsAlias("zzz") {
		t.Error("matchesSettingsAlias(\"zzz\") = true, want false")
	}

	m, _ := testModel(t, 100, 30)
	m.opts.Commands = paletteCommands()
	m.palette.open, m.palette.typed = true, "conf"
	all := m.palette.candidates(m.opts.Commands)
	found := false

	for _, c := range all {
		if c.Name == "settings" {
			found = true
		}
	}

	if !found {
		t.Errorf("candidates(%q) = %v, want settings reached through its alias", m.palette.typed, all)
	}
}

func TestCommandIndexFindsAPositionOrSaysThereIsNone(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = paletteCommands()
	m.palette.open = true

	if i, ok := m.commandIndex("settings"); !ok || i != 1 {
		t.Errorf("commandIndex(\"settings\") = %d,%v, want 1,true", i, ok)
	}

	if _, ok := m.commandIndex("no-such-command"); ok {
		t.Error("commandIndex on a name not in the table answered true")
	}
}

func TestEnsureVisibleKeepsTheSelectionOnScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. h<=0: the offset collapses to zero.
	m.frame.Body.H = 0
	m.palette.offset = 4

	after := m.ensureVisible()
	if after.palette.offset != 0 {
		t.Errorf("ensureVisible with h<=0 left offset=%d, want 0", after.palette.offset)
	}

	// 2. The selection above the window pulls the offset up to it.
	m.frame.Body.H = 5
	m.palette.sel, m.palette.offset = 1, 6

	after = m.ensureVisible()
	if after.palette.offset != 1 {
		t.Errorf("ensureVisible with sel above the window = offset %d, want 1", after.palette.offset)
	}

	// 3. The selection below the window pushes the offset down to it.
	m.palette.sel, m.palette.offset = 10, 0

	after = m.ensureVisible()
	if want := 10 - 5 + 1; after.palette.offset != want {
		t.Errorf("ensureVisible with sel below the window = offset %d, want %d", after.palette.offset, want)
	}

	// 4. Already on screen: nothing moves.
	m.palette.sel, m.palette.offset = 2, 1

	after = m.ensureVisible()
	if after.palette.offset != 1 {
		t.Errorf("ensureVisible with sel already visible = offset %d, want 1 unchanged", after.palette.offset)
	}

	// 5. A window taller than the selection's own index: the subtraction
	// that pushes the offset down to the selection lands below zero, and
	// the final clamp catches it.
	m.frame.Body.H = 1000
	m.palette.sel, m.palette.offset = 5, -2000

	after = m.ensureVisible()
	if after.palette.offset != 0 {
		t.Errorf("ensureVisible with a window taller than sel = %d, want it clamped to 0", after.palette.offset)
	}
}

func TestCompleteFillsTheLineWithTheSelection(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = paletteCommands()
	m.palette.open, m.palette.typed, m.palette.sel = true, "n", 0

	after := m.complete()
	if after.palette.typed != "new" {
		t.Errorf("complete() typed=%q, want \"new\"", after.palette.typed)
	}

	if after.palette.sel != 0 || after.palette.offset != 0 {
		t.Errorf("complete() left sel=%d offset=%d, want both reset to 0", after.palette.sel, after.palette.offset)
	}

	// Nothing selected: the line is left exactly as it was.
	m.palette.typed, m.palette.sel = "zzz-no-match", 0

	after = m.complete()
	if after.palette.typed != "zzz-no-match" {
		t.Errorf("complete() with nothing selected changed typed to %q", after.palette.typed)
	}
}

func TestPaletteInputLineDrawsThePlaceholderOrTheTyped(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if line := m.paletteInputLine(40); !strings.Contains(line, "type a command") {
		t.Errorf("paletteInputLine on an empty line = %q, want the placeholder", line)
	}

	m.palette.typed = "set"
	if line := m.paletteInputLine(40); !strings.Contains(line, "set") {
		t.Errorf("paletteInputLine on %q = %q, want it to contain what was typed", m.palette.typed, line)
	}
}

func TestPaletteRowDrawsRefusalsAndUsage(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	cmds := paletteCommands()

	row := m.paletteRow(cmds[0], false, 60) // new: has Args and About
	if !strings.Contains(row, "new") || !strings.Contains(row, "<repo>") {
		t.Errorf("paletteRow(new) = %q, want the name and its usage", row)
	}

	refused := m.paletteRow(cmds[2], false, 60) // nope: refused
	if !strings.Contains(refused, "not from here") {
		t.Errorf("paletteRow(nope) = %q, want the refusal in place of the description", refused)
	}

	selected := m.paletteRow(cmds[0], true, 60)
	if selected == row {
		t.Error("paletteRow(selected=true) drew the same line as unselected")
	}
}

func TestHitPaletteMapsRowsToCommands(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = paletteCommands()
	m = m.openPalette()

	// 1. Past the body entirely.
	if tgt := m.hitPalette(0, 9999); tgt.Kind != TargetNone {
		t.Errorf("hitPalette far past the body = %+v, want TargetNone", tgt)
	}

	// 2. The first row names the first candidate.
	tgt := m.hitPalette(0, m.frame.Body.Y)
	if tgt.Kind != TargetCommand || tgt.Key != "new" {
		t.Errorf("hitPalette on the first row = %+v, want TargetCommand keyed \"new\"", tgt)
	}

	// 3. A row past however many candidates there are answers nothing.
	m.palette.typed = "new"
	if tgt := m.hitPalette(0, m.frame.Body.Y+3); tgt.Kind != TargetNone {
		t.Errorf("hitPalette past the filtered list = %+v, want TargetNone", tgt)
	}
}
