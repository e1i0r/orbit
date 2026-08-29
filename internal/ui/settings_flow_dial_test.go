package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
)

// saveFlow writes one more flow into a directory that already exists, which
// userFlows cannot do: it fills its directory when it is called, and two of
// the tests below need a flow to arrive after that.
func saveFlow(t *testing.T, src flow.Source, name string) {
	t.Helper()

	body := `{"name":"` + name + `","phases":[{"name":"implement","engine":"claude","model":"sonnet"}]}`
	if err := os.WriteFile(filepath.Join(src.FlowDir(), name+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("save the flow %q: %v", name, err)
	}
}

// flowDialOf is the settings screen's flow row.
func flowDialOf(t *testing.T, m Model) settingRow {
	t.Helper()

	for _, r := range m.settingRowsList() {
		if r.key == "flow" {
			return r
		}
	}

	t.Fatal("the settings screen has no flow row")

	return settingRow{}
}

// TestTheFlowDialOffersEveryFlowTheBuildShips.
//
// The dial was three names written out by hand — task, quick, careful —
// while the binary ships four. The fourth could not be chosen from this
// screen at all: `orbit set flow` took it and the compose dialog offered it,
// and here it did not exist. Every other dial on this screen is already
// built from the build, and the comment above them says why.
func TestTheFlowDialOffersEveryFlowTheBuildShips(t *testing.T) {
	m, _ := testModel(t, 120, 40)
	m = m.openSettings()

	got := flowDialOf(t, m).options
	for _, name := range flow.BuiltinNames() {
		if !slices.Contains(got, name) {
			t.Errorf("the flow dial offers %v, and this build ships %q", got, name)
		}
	}
}

// TestTheFlowDialOffersAFlowTheReaderWrote. A flow somebody saved is one
// they can compose a single task against, so it has to be one they can make
// the default for all of them.
func TestTheFlowDialOffersAFlowTheReaderWrote(t *testing.T) {
	m, _ := testModel(t, 120, 40)
	m.opts.Flows = userFlows(t, "midnight")
	m = m.openSettings()

	if got := flowDialOf(t, m).options; !slices.Contains(got, "midnight") {
		t.Errorf("the flow dial offers %v, and the reader wrote midnight", got)
	}
}

// TestTheFlowDialIsReadWhenTheScreenOpens.
//
// settingRowsList runs from View, from every keypress and from every mouse
// event, so a dial that asked the flows directory for itself would be one
// os.ReadDir per frame — and two readings, taken at two moments, deciding
// the same dial. That is the bug flows.go already had and already fixed: a
// flow saved between the draw and the click moved every row out from under
// the cursor.
func TestTheFlowDialIsReadWhenTheScreenOpens(t *testing.T) {
	m, _ := testModel(t, 120, 40)
	m.opts.Flows = userFlows(t)
	m = m.openSettings()

	saveFlow(t, m.opts.Flows, "midnight")

	if got := flowDialOf(t, m).options; slices.Contains(got, "midnight") {
		t.Errorf("the flow dial is %v, so it is reading the directory while it draws", got)
	}
}
