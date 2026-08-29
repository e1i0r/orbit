package ui

// theme_help_coverage_test.go is currentPalette's fallback for a theme name
// nothing answers to, and the help overlay's own key map: which screen it
// remembers to return to, and how its offset scrolls and clamps.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCurrentPaletteFallsBackOnAnUnknownTheme(t *testing.T) {
	old := currentThemeName

	t.Cleanup(func() { currentThemeName = old })

	currentThemeName = "not-a-real-theme"

	if got := currentPalette(); got != themePalettes["monokai"] {
		t.Errorf("currentPalette with an unknown theme name = %+v, want the monokai fallback", got)
	}

	currentThemeName = "nord"

	if got := currentPalette(); got != themePalettes["nord"] {
		t.Errorf("currentPalette(nord) = %+v, want the nord palette", got)
	}
}

func TestOpenAndAbandonHelpRememberTheScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Opening from an ordinary screen remembers it.
	m.screen = screenSettings

	got := m.openHelp()
	if got.screen != screenHelp || got.help.prevScreen != screenSettings {
		t.Errorf("openHelp from settings = %+v, want screenHelp remembering settings", got)
	}

	// 2. Opening help while already on help does not remember help itself.
	got.screen = screenHelp

	got = got.openHelp()
	if got.help.prevScreen != screenList {
		t.Errorf("openHelp from help = %v, want it to fall back to screenList", got.help.prevScreen)
	}

	// 3. Abandoning returns to the remembered screen, and clears the state.
	got.help.prevScreen = screenStart

	got = got.abandonHelp()
	if got.screen != screenStart || got.help != (helpState{}) {
		t.Errorf("abandonHelp = screen %v help %+v, want screenStart and cleared state", got.screen, got.help)
	}

	// 4. A prevScreen that is itself screenHelp — reachable only by writing
	// the field directly, never by openHelp — still resolves to the list.
	got.help.prevScreen = screenHelp

	got = got.abandonHelp()
	if got.screen != screenList {
		t.Errorf("abandonHelp with prevScreen=screenHelp = %v, want screenList", got.screen)
	}
}

func TestHelpKeyEveryBinding(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openHelp()

	// 1. Down increments the offset with no ceiling of its own — helpRows
	// is what clamps it.
	next, _ := m.helpKey(tea.KeyPressMsg{Code: tea.KeyDown})

	got := asModel(t, next)
	if got.help.offset != 1 {
		t.Errorf("helpKey(down) offset = %d, want 1", got.help.offset)
	}

	// 2. Up decrements, and refuses to go below zero.
	next, _ = got.helpKey(tea.KeyPressMsg{Code: tea.KeyUp})

	got = asModel(t, next)
	if got.help.offset != 0 {
		t.Errorf("helpKey(up) offset = %d, want 0", got.help.offset)
	}

	next, _ = got.helpKey(tea.KeyPressMsg{Code: tea.KeyUp})

	got = asModel(t, next)
	if got.help.offset != 0 {
		t.Errorf("helpKey(up) at zero = %d, want it to stay at 0", got.help.offset)
	}

	// 3. Back, Quit, Help and Open all close the overlay.
	for _, code := range []rune{tea.KeyEscape, tea.KeyEnter} {
		got = got.openHelp()
		next, _ = got.helpKey(tea.KeyPressMsg{Code: code})

		got = asModel(t, next)
		if got.screen == screenHelp {
			t.Errorf("helpKey(%v) did not close the overlay", code)
		}
	}

	got = got.openHelp()
	next, _ = got.helpKey(tea.KeyPressMsg{Code: '?', Text: "?"})

	got = asModel(t, next)
	if got.screen == screenHelp {
		t.Error("helpKey('?') did not close the overlay")
	}

	// 4. An unmatched key is a no-op.
	got = got.openHelp()
	next, _ = got.helpKey(tea.KeyPressMsg{Code: 'z', Text: "z"})

	got = asModel(t, next)
	if got.screen != screenHelp {
		t.Error("an unmatched key closed the help overlay")
	}
}

func TestHelpRowsOffsetClampsToTheLastLine(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openHelp()

	// The title is asked for the way helpRows asks for it. Spelling it out
	// here in Spanish is what this line used to do, and it passed for a
	// model whose language is English only because the screen was Spanish
	// whatever the language said.
	title := m.opts.Words.T("help.title", "Help and keyboard shortcuts (cheat sheet)")

	full := m.helpRows(40, 100)
	if len(full) == 0 || !strings.Contains(strings.Join(full, "\n"), title) {
		t.Fatalf("helpRows at offset 0 = %v, want the help title", full)
	}

	m.help.offset = 3

	scrolled := m.helpRows(40, 100)
	if len(scrolled) != len(full) {
		t.Errorf("helpRows at offset 3 has %d lines, want it padded back to %d like offset 0", len(scrolled), len(full))
	}

	if strings.Contains(strings.Join(scrolled, "\n"), title) {
		t.Error("helpRows at offset 3 still shows the title, want it scrolled past")
	}

	// An offset past the end of the content clamps to the last line rather
	// than slicing out of range.
	m.help.offset = 100000
	if got := m.helpRows(40, 100); len(got) == 0 {
		t.Error("helpRows with an out-of-range offset returned nothing, want it clamped")
	}

	// h<=0 draws nothing at all.
	if got := m.helpRows(0, 100); got != nil {
		t.Errorf("helpRows(0, ...) = %v, want nil", got)
	}
}
