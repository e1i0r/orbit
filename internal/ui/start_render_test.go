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
	"regexp"
	"strconv"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"

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
// It reads the bar rather than the key map, and that is the whole of what
// makes it a test. The bar is hints() plus a tail — [?] and [q] — appended
// outside them and never dropped, so a version of this test that walked
// startBindings() was structurally unable to see the tail, and the dialog
// printed [?] for a key it did not answer for as long as that was true.
//
// Every glyph the bar prints has to belong to a binding this screen handles,
// and every one of those keys has to move something when it is pressed.
func TestTheStartDialogOffersOnlyKeysItHandles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m, _ = dialog(t, m, "ACME-2662")
	rows := renderAt(t, m, 100, 30)
	bar := ansi.Strip(rows[len(rows)-1])

	handled := map[string]key.Binding{}
	for _, b := range append(m.startBindings(), m.keys.Help, m.keys.Quit) {
		handled[b.Help().Key] = b
	}

	glyphs := inBrackets(bar)
	if len(glyphs) == 0 {
		t.Fatalf("the bar offers no keys at all, so this test asserts nothing: %q", bar)
	}

	for _, glyph := range glyphs {
		b, ok := handled[glyph]
		if !ok {
			t.Errorf("the bar offers %q, which this dialog does not handle: %q", glyph, bar)
			continue
		}

		for _, keystroke := range b.Keys() {
			after, cmd := advance(t, m, press(keystroke))
			// The dials count as movement. They were left out of this
			// comparison while the footer was long enough in Spanish that
			// [o] fell off the end of the bar, so the one key here that
			// moves nothing but a dial was never actually pressed.
			if cmd == nil && after.screen == m.screen && after.start.at == m.start.at &&
				after.message == m.message && after.autopilotOn() == m.autopilotOn() &&
				after.knobs == m.knobs {
				t.Errorf("%q is in the footer and moves nothing", keystroke)
			}
		}
	}
}

// inBrackets is every [glyph] in one rendered line, in the order they were
// drawn. It is how a test reads a key bar: the bar draws a key as its glyph
// in square brackets and its meaning after it, and the glyph is the part a
// reader is being invited to press.
func inBrackets(line string) []string {
	var out []string
	for _, m := range bracketed.FindAllStringSubmatch(line, -1) {
		out = append(out, m[1])
	}

	return out
}

var bracketed = regexp.MustCompile(`\[([^\[\]]+)\]`)

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
// It is named for an equivalence, so it asserts one rather than writing both
// sides down. The classification comes from flow.List — the one function
// that decides where a name came from, and the one `orbit flows` asks — and
// the sentence comes from the same catalogue key that listing prints
// through. Change the rule and both sides move; change the dialog alone and
// this fails.
//
// Both languages, and each catches a different drift. In Spanish the words
// come from es.json, so a dialog that quietly grew a key of its own falls
// back to English and is caught. In English T returns what the caller wrote,
// so the literal here and the literal in flowMark are compared directly —
// and they are not two copies free to differ, because
// TestEveryTranslationKeyIsHonest fails the build when two call sites give
// one key two different English sentences. That is what welds this test, the
// dialog and `orbit flows` to one wording.
//
// What the test still says on its own is which key belongs to which origin.
// That is the contract between the two screens, and stating it is the point.
func TestAFlowOfTheReadersOwnIsMarkedTheWayOrbitFlowsMarksIt(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			p := printerFor(t, lang)
			m := modelWith(t, p, fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
			m.opts.Flows = userFlows(t, "mine", "task")
			m, _ = dialog(t, m, "ACME-2662")

			marks, seen := map[string]string{}, map[string]int{}
			for _, f := range m.start.flows {
				marks[f.name], seen[f.name] = m.flowMark(f), seen[f.name]+1
			}

			listed := flow.List(m.opts.Flows)
			if len(listed) == 0 {
				t.Fatal("flow.List offers nothing, so the comparison would assert nothing")
			}

			for _, l := range listed {
				var want string

				switch l.Origin {
				case flow.OriginUser:
					want = p.T("flow.yours", "yours")
				case flow.OriginShadow:
					want = p.T("flow.shadowing", "yours, shadowing the built-in")
				}

				if seen[l.Name] == 0 {
					t.Errorf("the cycle does not offer %q at all, so its mark says nothing", l.Name)
					continue
				}

				if marks[l.Name] != want {
					t.Errorf("the cycle marks %q as %q, want %q", l.Name, marks[l.Name], want)
				}
				// A shadowed built-in is one entry and not two, for the
				// reason `orbit flows` lists it once: there is one flow
				// that name resolves to.
				if seen[l.Name] != 1 {
					t.Errorf("the cycle lists %q %d times, want once", l.Name, seen[l.Name])
				}
			}

			if marks["task"] == marks["mine"] {
				t.Errorf("a shadowing flow and an ordinary one of the reader's own are marked the same: %q", marks["task"])
			}

			if marks["careful"] != "" {
				t.Errorf("a built-in is marked %q; a mark on every line is not a mark", marks["careful"])
			}
		})
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
