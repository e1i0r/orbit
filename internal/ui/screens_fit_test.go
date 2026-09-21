package ui

// Every screen, in every terminal it can be opened in: it draws the body it
// was given, no taller and no wider.
//
// A screen that answers with more rows than the body has is cut by whatever
// composes it, and what it loses is the end — which on most of these is the
// line that says how to leave. One that answers with fewer leaves the rows
// below it holding whatever was drawn there last. And a row wider than the
// body pushes the frame open or wraps, which moves every row under it, and
// rows are where clicks land.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// terminals is the range a reader actually works in: a narrow window beside
// an editor, a short one under one, and a full screen.
var terminals = []struct{ w, h int }{
	{80, 24},
	{100, 24},
	{100, 30},
	{120, 20},
	{140, 40},
	{100, 15},
	{70, 20},
	{60, 24},
	{80, 12},
}

// TestEveryScreenDrawsTheBodyItWasGiven.
func TestEveryScreenDrawsTheBodyItWasGiven(t *testing.T) {
	for _, c := range []struct {
		what string
		open func(Model) Model
		rows func(Model, int, int) []string
	}{
		{"the quota screen", Model.openQuota, Model.quotaRows},
		{"what Orbit knows", Model.openKnowledge, Model.knowledgeRows},
		{"the engine knobs", Model.openEngines, Model.enginesRows},
		{"the repository list", Model.openRepos, Model.repolistRows},
		{"the cheat sheet", Model.openHelp, Model.helpRows},
		{"the supervisor", Model.openSupervisor, Model.supervisorRows},
		{"the form", Model.openCompose, Model.composeRows},
		{"the flow designer", Model.openFlows, Model.flowsRows},
		{"the settings table", Model.openSettings, Model.settingsRows},
		{"a task's own screen", openFirstTask, Model.detailRows},
		{"the note a task carries", openFirstTask, Model.noteRows},
		{"the command palette", Model.openPalette, Model.paletteRows},
		{"the menu", openTaskMenu, Model.menuRows},
	} {
		for _, term := range terminals {
			m, _ := testModel(t, term.w, term.h)
			m.opts.Engines = enginesTestList
			m.opts.Quota = quotaFixture

			m = c.open(m)

			h, w := m.frame.Body.H, m.frame.Body.W

			rows := c.rows(m, h, w)
			if len(rows) != h {
				t.Errorf("%s at %dx%d drew %d rows into a body of %d",
					c.what, term.w, term.h, len(rows), h)
			}

			for i, row := range rows {
				if wide := lipgloss.Width(row); wide > w {
					t.Errorf("%s at %dx%d drew row %d %d cells wide into a body of %d:\n%q",
						c.what, term.w, term.h, i, wide, w, ansi.Strip(row))
				}

				if strings.Contains(row, "\n") {
					t.Errorf("%s at %dx%d drew a row %d holding a line break",
						c.what, term.w, term.h, i)
				}
			}
		}
	}
}

// openFirstTask is the window with a task's own screen up, which is the
// screen a reader spends the most time on.
func openFirstTask(m Model) Model {
	rows := m.rows()
	for _, r := range rows {
		if !r.head && !r.blank {
			m.screen, m.detail = screenDetail, r.task.ID

			return m
		}
	}

	return m
}

// openTaskMenu is the menu of verbs a task offers, which is drawn over the
// body like the palette.
func openTaskMenu(m Model) Model {
	at := openFirstTask(m)

	return at.openMenu(at.detail)
}
