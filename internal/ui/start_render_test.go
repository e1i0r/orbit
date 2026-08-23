package ui

// start_render_test.go is the dialog measured and the dialog compared: the
// frame it is specified as in both languages, and the same three assertions
// the board's measured render makes, pointed at the one screen in this
// window whose content is not a list.
//
// It is a file of its own for the reason detail_render_test.go is: the
// transition table and the frame are two different questions, and the
// 300-line ceiling does not have both in one file.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestTheStartDialogIsTheScreenItWasSpecifiedAs is the frame itself, in both
// languages the module ships.
func TestTheStartDialogIsTheScreenItWasSpecifiedAs(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			m := modelWith(t, printerFor(t, lang), fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
			m, _ = dialog(t, m, "ACME-2662")
			golden(t, "start-100x30-"+lang, renderAt(t, m, 100, 30))
		})
	}
}

// TestEveryStartDialogFitsTheTerminalItWasGiven is the measured render, run
// over the dialog rather than over the board.
//
// It is the same helper and the same three assertions task 11 wrote, pointed
// at the one screen in this window whose content is not a list: a phase name
// and a model name are data, they are as long as the flow file says, and a
// Spanish label in front of them is longer than the English one.
func TestEveryStartDialogFitsTheTerminalItWasGiven(t *testing.T) {
	for _, lang := range []string{"en", "es", "qps"} {
		printer := printerFor(t, lang)
		for _, size := range sizes {
			name := lang + "/" + strconv.Itoa(size.w) + "x" + strconv.Itoa(size.h)
			t.Run(name, func(t *testing.T) {
				m := modelWith(t, printer, fixtureBoard(fixtureTasks(), 4), size.w, size.h, nil)
				m, _ = dialog(t, m, "ACME-2662")
				rows := renderAt(t, m, size.w, size.h)
				if len(rows) != size.h {
					t.Fatalf("the dialog is %d rows, want %d", len(rows), size.h)
				}
				for i, row := range rows {
					if cells := ansi.StringWidth(row); cells > size.w {
						t.Errorf("row %d is %d cells wide, want at most %d: %q", i, cells, size.w, row)
					}
				}
				if !strings.Contains(strings.Join(rows, "\n"), "ACME-2662") {
					t.Error("the dialog does not say which task it would start")
				}
			})
		}
	}
}

// TestTheStartDialogOffersOnlyKeysItHandles is the footer's whole claim.
//
// It is drawn from the same bindings the dialog matches against, so a key
// printed under the phases is a key that does something. Pressing every one
// of them and finding the window unmoved is how a footer that has drifted
// from its own key map is caught.
func TestTheStartDialogOffersOnlyKeysItHandles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m, _ = dialog(t, m, "ACME-2662")
	rows := renderAt(t, m, 100, 30)
	bar := rows[len(rows)-1]
	for _, glyph := range []string{"space", " e ", "[e]"} {
		if strings.Contains(bar, glyph) {
			t.Errorf("the footer offers %q, which this dialog does not handle: %q", glyph, bar)
		}
	}
	for _, b := range m.startBindings() {
		for _, keystroke := range b.Keys() {
			after, cmd := advance(t, m, press(keystroke))
			if cmd == nil && after.screen == m.screen && after.start.at == m.start.at &&
				after.message == m.message && after.autopilotOn() == m.autopilotOn() {
				t.Errorf("%q is in the footer and moves nothing", keystroke)
			}
		}
	}
}

// flowDir is a reader's own flow directory, as internal/flow asks for it.
type flowDir string

func (d flowDir) FlowDir() string { return string(d) }

// userFlows writes flow files into a directory of this test's own and hands
// back the Source that finds them.
//
// Nothing here goes near $ORBIT_HOME or the home directory: main_test.go
// points HOME at a temporary directory and unsets ORBIT_HOME, and this
// Source is handed to the window rather than looked up by it.
func userFlows(t *testing.T, names ...string) flow.Source {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		body := `{"name":"` + n + `","phases":[{"name":"implement","engine":"claude","model":"sonnet"}]}`
		if err := os.WriteFile(filepath.Join(dir, n+".json"), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
	return flowDir(dir)
}

// TestAFlowOfTheReadersOwnIsMarkedTheWayOrbitFlowsMarksIt is the one fact
// the fixture board cannot show, because the fixture has no flow directory:
// the cycle offers a reader's own flows, and says which they are.
//
// The two marks are asserted as the English `orbit flows` prints, and that
// is the whole point of the assertion — a reader who ran the command and
// then opened this dialog is looking at one fact, and two spellings of it
// would read as two.
func TestAFlowOfTheReadersOwnIsMarkedTheWayOrbitFlowsMarksIt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Flows = userFlows(t, "mine", "task")
	m, _ = dialog(t, m, "ACME-2662")

	marks := map[string]string{}
	for _, f := range m.start.flows {
		marks[f.name] = m.flowMark(f)
	}
	for name, want := range map[string]string{
		"mine":    "yours",
		"task":    "yours, shadowing the built-in",
		"careful": "",
	} {
		if marks[name] != want {
			t.Errorf("the cycle marks %q as %q, want %q", name, marks[name], want)
		}
	}
	for _, name := range []string{"mine", "task", "careful"} {
		if _, listed := marks[name]; !listed {
			t.Errorf("the cycle does not offer %q at all, so its mark says nothing", name)
		}
	}
	// A shadowed built-in is one entry and not two, for the reason
	// `orbit flows` lists it once: there is one flow that name resolves to.
	var seen int
	for _, f := range m.start.flows {
		if f.name == "task" {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("the cycle lists a shadowed built-in %d times, want once", seen)
	}
}

// The mark reaches the screen, and not only the model.
func TestTheDialogSaysAFlowIsTheReadersOwn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Flows = userFlows(t, "mine")
	m, _ = dialog(t, m, "ACME-2662")
	for range m.start.flows {
		if m.start.chosen().name == "mine" {
			break
		}
		m = m.cycleFlow()
	}
	if m.start.chosen().name != "mine" {
		t.Fatalf("f never reaches the reader's own flow; the cycle is %v", m.start.flows)
	}
	if line := ansi.Strip(m.flowLine(100)); !strings.Contains(line, "yours") {
		t.Errorf("the flow line is %q, want it to say the flow is the reader's own", line)
	}
}
