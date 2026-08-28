package ui

// targetdialog_more_coverage_test.go continues target_dialog_coverage_test.go's
// walk of the hit-testers, split here to stay under the file's line ceiling:
// the task view's tab strip and pane, the start dialog's rows, the settings
// screen's option pills, and the repository picker.

import (
	"testing"
)

func TestHitDetailAndTabs(t *testing.T) {
	m := openOn(t, "ACME-2662")
	bodyY := m.frame.Body.Y

	if got := m.hitDetail(5, bodyY-10); got.Kind != TargetNone {
		t.Errorf("hitDetail outside the body = %+v, want TargetNone", got)
	}

	if got := m.hitDetail(5, bodyY); got.Kind != TargetNone {
		t.Errorf("hitDetail on the heading row = %+v, want TargetNone", got)
	}

	if got := m.hitDetail(5, bodyY+3); got.Kind != TargetPaneBody {
		t.Errorf("hitDetail inside the pane = %+v, want TargetPaneBody", got)
	}

	if got := m.hitDetail(5, bodyY+m.frame.Body.H-1); got.Kind != TargetNone {
		t.Errorf("hitDetail on the scroll-hint line = %+v, want TargetNone", got)
	}

	tabs := m.placeTabs()
	if len(tabs) == 0 {
		t.Fatal("placeTabs found no tabs")
	}

	if got := m.hitTabs(tabs[0].x); got.Kind != TargetPaneTab || got.Pane != int(tabs[0].tab) {
		t.Errorf("hitTabs on the first tab = %+v, want pane %d", got, tabs[0].tab)
	}

	if got := m.hitTabs(-1); got.Kind != TargetNone {
		t.Errorf("hitTabs off every tab = %+v, want TargetNone", got)
	}
}

func TestHitStartEveryRow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onto(t, m, "ACME-2698")
	next, _ := m.openStart()
	m = asModel(t, next)
	p := m.startLayout(m.frame.Body.W)
	y := m.frame.Body.Y

	if got := m.hitStart(5, y+p.flow); got.Kind != TargetDialogSwitch || got.Field != fieldFlow {
		t.Errorf("hitStart on the flow line = %+v, want the flow switch", got)
	}

	if p.nPhases > 0 {
		if got := m.hitStart(5, y+p.phases); got.Kind != TargetDialogPhase || got.Phase != 0 {
			t.Errorf("hitStart on the first phase = %+v, want phase 0", got)
		}
	}

	if got := m.hitStart(5, y+p.autopilot); got.Kind != TargetDialogSwitch || got.Field != fieldAutopilotOn {
		t.Errorf("hitStart on the autopilot-on row = %+v, want the on switch", got)
	}

	if got := m.hitStart(5, y+p.autopilot+1); got.Kind != TargetDialogSwitch || got.Field != fieldAutopilotOff {
		t.Errorf("hitStart on the autopilot-off row = %+v, want the off switch", got)
	}

	if got := m.hitStart(5, y+p.config); got.Kind != TargetNone {
		t.Errorf("hitStart on the config line = %+v, want TargetNone", got)
	}

	if got := m.hitStart(5, y-10); got.Kind != TargetNone {
		t.Errorf("hitStart outside the body = %+v, want TargetNone", got)
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

	if got := m.hitSettings(5, y); got.Kind != TargetNone {
		t.Errorf("hitSettings above row 4 = %+v, want TargetNone", got)
	}

	if got := m.hitSettings(10, y+4); got.Kind != TargetSettingsRow || got.Field != "" {
		t.Errorf("hitSettings left of the pills = %+v, want the row with no field", got)
	}

	if got := m.hitSettings(21, y+4); got.Kind != TargetSettingsRow || got.Field != rows[0].options[0] {
		t.Errorf("hitSettings on the first pill = %+v, want field %q", got, rows[0].options[0])
	}

	if got := m.hitSettings(5000, y+4); got.Kind != TargetSettingsRow || got.Field != "" {
		t.Errorf("hitSettings past every pill = %+v, want the row with no field", got)
	}

	if got := m.hitSettings(10, y+4+3*len(rows)+50); got.Kind != TargetNone {
		t.Errorf("hitSettings past every row = %+v, want TargetNone", got)
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

	if got := m.hitRepos(5, y-10); got.Kind != TargetNone {
		t.Errorf("hitRepos outside the body = %+v, want TargetNone", got)
	}

	if got := m.hitRepos(5, y+4); got.Kind != TargetRepo || got.ID != repos[0].name {
		t.Errorf("hitRepos on the first row = %+v, want %q", got, repos[0].name)
	}

	if got := m.hitRepos(5, y+4+len(repos)+50); got.Kind != TargetNone {
		t.Errorf("hitRepos past every row = %+v, want TargetNone", got)
	}
}
