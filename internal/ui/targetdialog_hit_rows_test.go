package ui

// targetdialog_more_coverage_test.go continues target_dialog_coverage_test.go's
// walk of the hit-testers, split here to stay under the file's line ceiling:
// the task view's tab strip and pane, the start dialog's rows, the settings
// screen's option pills, and the repository picker.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/settings"
)

func TestHitDetailAndTabs(t *testing.T) {
	m := openOn(t, "ACME-2662")
	bodyY := m.frame.Body.Y

	if got := m.hitDetail(5, bodyY-10); got.Kind != point.None {
		t.Errorf("hitDetail outside the body = %+v, want point.None", got)
	}

	if got := m.hitDetail(5, bodyY); got.Kind != point.None {
		t.Errorf("hitDetail on the heading row = %+v, want point.None", got)
	}

	if got := m.hitDetail(5, bodyY+3); got.Kind != point.PaneBody {
		t.Errorf("hitDetail inside the pane = %+v, want point.PaneBody", got)
	}

	if got := m.hitDetail(5, bodyY+m.frame.Body.H-1); got.Kind != point.None {
		t.Errorf("hitDetail on the scroll-hint line = %+v, want point.None", got)
	}

	tabs := m.placeTabs()
	if len(tabs) == 0 {
		t.Fatal("placeTabs found no tabs")
	}

	if got := m.hitTabs(tabs[0].x); got.Kind != point.PaneTab || got.Pane != int(tabs[0].tab) {
		t.Errorf("hitTabs on the first tab = %+v, want pane %d", got, tabs[0].tab)
	}

	if got := m.hitTabs(-1); got.Kind != point.None {
		t.Errorf("hitTabs off every tab = %+v, want point.None", got)
	}
}

// drawnAt is where a tab starts in the strip, in cells.
//
// In cells and not in bytes, which is what strings.Index answers. A strip
// narrow enough to cut its names carries an ellipsis, and one of those is
// three bytes and one cell: measured in bytes, every tab after the first cut
// one reads as two cells further right than it is. The comparison below is
// against a cell position, so it has to be a cell position — otherwise this
// test fails on a strip that is correct, and would pass on one that is two
// cells out.
func drawnAt(strip, tag string) int {
	before, _, found := strings.Cut(strip, tag)
	if !found {
		return -1
	}

	return lipgloss.Width(before)
}

// TestEveryTabIsWhereTheStripDrewIt. A click is answered from placeTabs and
// the strip is drawn by tabStrip, so the two agreeing is the whole of whether
// pressing a tab opens the one under the pointer. They agree by both walking
// the same widths, which is what makes a change to the separator — brackets
// to none, one space to two — able to move every tab but the first.
func TestEveryTabIsWhereTheStripDrewIt(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.frame.Body.W = 140

	strip := ansi.Strip(m.tabStrip(m.frame.Body.W))
	tags := m.tabTags(m.frame.Body.W)

	placed := m.placeTabs()
	if len(placed) != len(tags) {
		t.Fatalf("the strip drew %d tabs and %d were placed", len(tags), len(placed))
	}

	for i, p := range placed {
		at := drawnAt(strip, tags[i].text)
		if at != p.x {
			t.Errorf("%q is drawn at cell %d and clicked at %d", tags[i].text, at, p.x)
		}

		if got := m.hitTabs(p.x + p.w - 1); got.Pane != int(p.tab) {
			t.Errorf("the last cell of %q opens pane %d, want %d", tags[i].text, got.Pane, p.tab)
		}
	}
}

func TestHitStartEveryRow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onto(t, m, "ACME-2698")
	next, _ := m.openStart()
	m = asModel(t, next)
	p := m.startLayout(m.frame.Body.W)
	y := m.frame.Body.Y

	// Column 5 is inside the label, before the first pill, so the row's
	// own answer is the one that comes back. A click on a pill is
	// TestAClickPicksTheFlowItLandedOn's.
	if got := m.hitStart(5, y+p.flow); got.Kind != point.DialogSwitch || got.Field != fieldFlow {
		t.Errorf("hitStart on the flow line = %+v, want the flow switch", got)
	}

	// The phase rows answer nothing: they are a preview of what the flow
	// will do, not a thing to choose. They used to answer a target nothing
	// routed, which is a cell that takes a click and drops it.
	if p.nPhases > 0 {
		if got := m.hitStart(5, y+p.phases); got.Kind != point.None {
			t.Errorf("hitStart on the first phase = %+v, want nothing", got)
		}
	}

	if got := m.hitStart(5, y+p.autopilot); got.Kind != point.DialogSwitch || got.Field != fieldAutopilotOn {
		t.Errorf("hitStart on the autopilot-on row = %+v, want the on switch", got)
	}

	if got := m.hitStart(5, y+p.autopilot+1); got.Kind != point.DialogSwitch || got.Field != fieldAutopilotOff {
		t.Errorf("hitStart on the autopilot-off row = %+v, want the off switch", got)
	}

	if got := m.hitStart(5, y+p.config); got.Kind != point.None {
		t.Errorf("hitStart on the config line = %+v, want point.None", got)
	}

	if got := m.hitStart(5, y-10); got.Kind != point.None {
		t.Errorf("hitStart outside the body = %+v, want point.None", got)
	}
}

func TestHitSettingsEveryOutcome(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings

	rows := m.settingRowsList()
	if len(rows) == 0 {
		t.Fatal("the fixture settings port produced no rows")
	}

	y := m.frame.Body.Y

	if got := m.hitSettings(5, y); got.Kind != point.None {
		t.Errorf("hitSettings above the table = %+v, want point.None", got)
	}

	// A heading is not a setting.
	if got := m.hitSettings(10, y+4); got.Kind != point.None {
		t.Errorf("hitSettings on the first group's heading = %+v, want point.None", got)
	}

	// The language row, which offers pills, brought into view the way a
	// reader brings it, and on the line the screen says it drew it on.
	lang := settingRow(t, m, "language")
	m.settings = m.settings.Point(lang, m.settingsEnv())

	line, shown := m.settings.LineOf(lang, m.settingsEnv())
	if !shown {
		t.Fatal("the language row is not on the screen")
	}

	if got := m.hitSettings(10, y+line); got.Kind != point.SettingsRow || got.Field != "" {
		t.Errorf("hitSettings left of the pills = %+v, want the row with no field", got)
	}

	// The first cell of the first pill, measured from the same constant the
	// drawing pads the name column to.
	at := settings.PillsAt + 1

	got := m.hitSettings(at, y+line)
	if got.Kind != point.SettingsRow || got.Field != rows[lang].Options[0] {
		t.Errorf("hitSettings on the first pill = %+v, want field %q", got, rows[lang].Options[0])
	}

	if got := m.hitSettings(5000, y+line); got.Kind != point.SettingsRow || got.Field != "" {
		t.Errorf("hitSettings past every pill = %+v, want the row with no field", got)
	}

	if got := m.hitSettings(10, y+4+3*len(rows)+50); got.Kind != point.None {
		t.Errorf("hitSettings past every row = %+v, want point.None", got)
	}
}

func TestHitReposEveryOutcome(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openRepos()

	repos := m.collectRepos()
	if len(repos) == 0 {
		t.Fatal("the fixture board has no repositories")
	}

	y := m.frame.Body.Y

	if got := m.hitRepos(5, y-10); got.Kind != point.None {
		t.Errorf("hitRepos outside the body = %+v, want point.None", got)
	}

	if got := m.hitRepos(5, y+4); got.Kind != point.Repo || got.ID != repos[0].Name {
		t.Errorf("hitRepos on the first row = %+v, want %q", got, repos[0].Name)
	}

	if got := m.hitRepos(5, y+4+len(repos)+50); got.Kind != point.None {
		t.Errorf("hitRepos past every row = %+v, want point.None", got)
	}
}

// TestHitSettingsFollowsTheScrolledTable. The table is taller than the body,
// so a click has to be measured from where the table now starts. Measured
// from the top of it, a click lands on whichever dial used to be drawn
// there — the one gesture in this window that turns a knob nobody pointed at.
func TestHitSettingsFollowsTheScrolledTable(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSettings()

	rows := m.settingRowsList()
	if len(rows) == 0 {
		t.Fatal("the fixture settings port produced no rows")
	}

	// The cursor walks to the last dial, which is what pulls the table up.
	for range len(rows) - 1 {
		m.settings = m.settings.Scroll(1, m.settingsEnv())
	}

	last := len(rows) - 1

	line := settingsLine(t, m, rows[last].Key)
	if line >= m.frame.Body.H {
		t.Fatalf("the last dial is drawn on row %d of a body of %d", line, m.frame.Body.H)
	}

	got := m.hitSettings(5, m.frame.Body.Y+line)
	if got.Kind != point.SettingsRow || got.Pane != last {
		t.Errorf("a click on the last dial = %+v, want row %d", got, last)
	}

	// And the row it named is the row that is drawn there, which is the
	// half of this a coordinate alone cannot check.
	drawn := m.settingsRows(m.frame.Body.H, m.frame.Body.W)
	if line >= len(drawn) || !strings.Contains(drawn[line], rows[last].Key) {
		t.Errorf("line %d draws %q, want the %q row", line, safeLine(drawn, line), rows[last].Key)
	}
}

// safeLine is one line of a screen, for an error message that must not panic
// while saying what went wrong.
func safeLine(rows []string, i int) string {
	if i < 0 || i >= len(rows) {
		return ""
	}

	return rows[i]
}
