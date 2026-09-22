package flows

// The list that opens over the form, and the tab a flow is described in.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestAFilterNothingMatchesSaysSo. The count in the head said nought and
// the list was blank, which reads as a screen that broke rather than as a
// word that is in nothing.
func TestAFilterNothingMatchesSaysSo(t *testing.T) {
	s, e := designing(t, 110, 30)
	s = s.openPicker(flowFieldModel, e)

	for _, r := range "zzzzz" {
		s, _ = s.Key(tea.KeyPressMsg{Code: r, Text: string(r)}, e)
	}

	if ids, _ := s.pickerRows(e); len(ids) != 0 {
		t.Fatalf("zzzzz matched %d models, and this test needs none", len(ids))
	}

	drawn := ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
	if !strings.Contains(drawn, "nothing here is called that") {
		t.Errorf("a filter nothing matches draws no word about it:\n%s", drawn)
	}

	// And the way back is still on the screen.
	if !strings.Contains(drawn, "esc") {
		t.Errorf("the empty list does not say how to leave:\n%s", drawn)
	}
}

// TestThePageKeysWalkALongList. opencode answers to sixty models, and
// reaching the last of them one arrow at a time is sixty presses.
func TestThePageKeysWalkALongList(t *testing.T) {
	s, e := designing(t, 110, 30)
	s = s.openPicker(flowFieldModel, e)

	ids, _ := s.pickerRows(e)
	if len(ids) < 3 {
		t.Fatalf("the picker offers %d models, and this needs three", len(ids))
	}

	last := len(ids) - 1

	end, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyEnd}, e)
	if end.picker.sel != last {
		t.Errorf("end left the cursor on %d of %d", end.picker.sel, len(ids))
	}

	home, _ := end.Key(tea.KeyPressMsg{Code: tea.KeyHome}, e)
	if home.picker.sel != 0 {
		t.Errorf("home left the cursor on %d", home.picker.sel)
	}

	down, _ := s.Key(tea.KeyPressMsg{Code: tea.KeyPgDown}, e)
	if down.picker.sel != min(pickerPage, last) {
		t.Errorf("a page down left the cursor on %d, want %d", down.picker.sel, min(pickerPage, last))
	}

	if up, _ := down.Key(tea.KeyPressMsg{Code: tea.KeyPgUp}, e); up.picker.sel != 0 {
		t.Errorf("a page up from there left the cursor on %d, want the first", up.picker.sel)
	}

	// Both ends hold: a page past either is that end and not past it.
	if past, _ := end.Key(tea.KeyPressMsg{Code: tea.KeyPgDown}, e); past.picker.sel != last {
		t.Errorf("a page down from the last row left the cursor on %d", past.picker.sel)
	}
}

// TestEveryRowOfTheListIsTheChoiceItDraws.
func TestEveryRowOfTheListIsTheChoiceItDraws(t *testing.T) {
	s, e := designing(t, 110, 30)
	s = s.openPicker(flowFieldModel, e)

	ids, labels := s.pickerRows(e)
	lines := s.pickerLines(e.Frame.Body.H, e.Frame.Body.W, e)

	for row, l := range lines {
		if l.pick == noPick {
			continue
		}

		got := s.Hit(4, e.Frame.Body.Y+row, e)
		if got.Field != "pick" || got.Phase != l.pick {
			t.Errorf("row %d draws choice %d and answers %q/%d", row, l.pick, got.Field, got.Phase)
		}

		if want := cells.DialLabel(ids, labels, l.pick); !strings.Contains(ansi.Strip(l.text), want) {
			t.Errorf("row %d answers choice %d and draws %q", row, l.pick, ansi.Strip(l.text))
		}
	}
}

// TestADraftIsAskedForWhereTheButtonIs. Drafting sends a question to an
// engine, and the whole width of the window answered it: a click on the
// blank forty columns to the right of the button asked one.
func TestADraftIsAskedForWhereTheButtonIs(t *testing.T) {
	s, e := designing(t, 110, 34)
	s.tab = flowTabSay

	row, text := -1, ""

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
	for i := start; i < len(lines); i++ {
		if lines[i].act == "draft" {
			row, text = i-start, ansi.Strip(lines[i].text)

			break
		}
	}

	if row < 0 {
		t.Fatal("the draft button is not drawn at all")
	}

	wide := lipgloss.Width(text)

	if got := s.Hit(2, e.Frame.Body.Y+row, e); got.Field != "draft" {
		t.Errorf("the button itself answers %q, want draft", got.Field)
	}

	for _, x := range []int{wide, wide + 10, e.Frame.Body.W - 1} {
		if got := s.Hit(x, e.Frame.Body.Y+row, e); got.Kind != point.None {
			t.Errorf("column %d, past the %d the row is drawn in, answers %q", x, wide, got.Field)
		}
	}
}

// TestTheDescribeTabsDialsChooseWhatIsPointedAt, and open the list from
// anywhere else on the row — which is what the row says ⏎ does.
func TestTheDescribeTabsDialsChooseWhatIsPointedAt(t *testing.T) {
	s, e := designing(t, 110, 34)
	s.tab = flowTabSay

	for _, c := range []struct {
		act   string
		field int
	}{
		{"say_engine", flowFieldSayEngine},
		{"say_model", flowFieldSayModel},
	} {
		opts, current, ok := s.choices(c.field, e)
		if !ok {
			continue
		}

		row := -1

		lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)
		for i := start; i < len(lines); i++ {
			if lines[i].act == c.act {
				row = i - start

				break
			}
		}

		if row < 0 {
			t.Fatalf("the %s row is not drawn", c.act)
		}

		at := valueAt

		for i, o := range opts {
			wide := lipgloss.Width(o.label)
			if o.id == current {
				wide += 2
			}

			got := s.Hit(at, e.Frame.Body.Y+row, e)
			if got.Field != "dial" || got.Phase != c.field || got.Pane != i {
				t.Errorf("%s: column %d is %q/%d/%d, want option %d",
					c.act, at, got.Field, got.Phase, got.Pane, i)
			}

			at += wide + 1
		}

		// The label to the left of the pills opens the list.
		if got := s.Hit(4, e.Frame.Body.Y+row, e); got.Field != c.act {
			t.Errorf("%s: the label answers %q, want the list", c.act, got.Field)
		}
	}
}

// TestChoosingAnEngineInTheDescribeTabLetsItsModelGo, which is the rule the
// list itself follows: a model is one engine's own name for it.
func TestChoosingAnEngineInTheDescribeTabLetsItsModelGo(t *testing.T) {
	s, e := designing(t, 110, 34)
	s.tab = flowTabSay
	s.sayModel = "one"

	opts, current, ok := s.choices(flowFieldSayEngine, e)
	if !ok || len(opts) < 2 {
		t.Skipf("this build offers %d engines", len(opts))
	}

	other := 0

	for i, o := range opts {
		if o.id != current {
			other = i

			break
		}
	}

	next, _ := s.Click(point.Target{
		Kind: point.FlowItem, Field: "dial", Phase: flowFieldSayEngine, Pane: other,
	}, e)

	if next.sayEngine != opts[other].id {
		t.Errorf("clicking engine %q set %q", opts[other].id, next.sayEngine)
	}

	if next.sayModel != "" {
		t.Errorf("the model stayed %q across a change of engine", next.sayModel)
	}
}

// TestTheChosenRowOfTheListIsOnTheScreen, at every height and wherever the
// cursor has been walked to. The list is longer than the screen and the row
// being chosen is the one that has to be on it.
func TestTheChosenRowOfTheListIsOnTheScreen(t *testing.T) {
	s, e := designing(t, 110, 45)
	s = s.openPicker(flowFieldModel, e)

	ids, _ := s.pickerRows(e)
	if len(ids) < 3 {
		t.Fatalf("the picker offers %d models, and this needs three", len(ids))
	}

	for _, h := range []int{8, 12, 20, 40} {
		for sel := range len(ids) {
			at := s
			at.picker.sel = sel

			lines := at.pickerLines(h, e.Frame.Body.W, e)
			if len(lines) > h {
				t.Fatalf("at %d rows the list drew %d lines", h, len(lines))
			}

			marked, chosen := 0, false

			for _, l := range lines {
				if l.pick == sel {
					chosen = true
				}

				if strings.Contains(ansi.Strip(l.text), "▸") {
					marked++

					if l.pick != sel {
						t.Errorf("at %d rows with %d chosen, the mark is on row %d", h, sel, l.pick)
					}
				}
			}

			if !chosen {
				t.Errorf("at %d rows, choice %d of %d is on no row of the list", h, sel, len(ids))
			}

			if marked != 1 {
				t.Errorf("at %d rows with %d chosen, %d rows carry the mark", h, sel, marked)
			}
		}
	}
}

// TestTheListSaysWhichOneIsInUse, and says it of one row only.
func TestTheListSaysWhichOneIsInUse(t *testing.T) {
	s, e := designing(t, 110, 45)

	s.edited().Model = "zeta/two"
	s = s.openPicker(flowFieldModel, e)

	ids, _ := s.pickerRows(e)

	at, said := -1, 0

	for i, id := range ids {
		if id == "zeta/two" {
			at = i
		}
	}

	if at < 0 {
		t.Fatalf("the model in use is on no row of the list: %v", ids)
	}

	for _, l := range s.pickerLines(e.Frame.Body.H, e.Frame.Body.W, e) {
		if !strings.Contains(ansi.Strip(l.text), "in use") {
			continue
		}

		said++

		if l.pick != at {
			t.Errorf("row %d is marked as in use and the model is on row %d", l.pick, at)
		}
	}

	if said != 1 {
		t.Errorf("%d rows say they are in use, want one", said)
	}
}

// TestEveryPillOfAFlowRowIsWhereItIsDrawn. The pills on the row the cursor
// is on were measured two cells wider than they are, with no space counted
// between them, so every zone sat one to three cells right of the pill it
// was named for: the first cell of Edit inspected the flow, and three cells
// of blank air past it edited.
func TestEveryPillOfAFlowRowIsWhereItIsDrawn(t *testing.T) {
	e := world(t)
	s := Open(FromBoard, e)

	for at := range s.listed {
		s.sel = at
		d := s.listed[at]

		row := -1

		for i, l := range s.flowsListLines(e.Frame.Body.W, e) {
			if l.at == at && l.head {
				row = i

				break
			}
		}

		if row < 0 {
			t.Fatalf("flow %q is on no row of the list", d.Name)
		}

		x := rowPillsStart(d, e)

		for _, pill := range rowPills(d, e) {
			wide := lipgloss.Width(pill.drawn)

			for _, col := range []int{x, x + wide/2, x + wide - 1} {
				got := s.Hit(col, e.Frame.Body.Y+row, e)
				if got.Field != pill.field || got.ID != d.Name {
					t.Errorf("%s: column %d of the %s pill answers %q for %q",
						d.Name, col, pill.field, got.Field, got.ID)
				}
			}

			// The space between two pills belongs to the row, which
			// inspects — not to either of them.
			if got := s.Hit(x+wide, e.Frame.Body.Y+row, e); got.Field == "delete" {
				t.Errorf("%s: the gap at column %d deletes it", d.Name, x+wide)
			}

			x += wide + rowPillGap
		}

		// And a built-in offers no way to delete it, at any column.
		if d.Origin != flow.OriginBuiltin {
			continue
		}

		for col := range e.Frame.Body.W {
			if got := s.Hit(col, e.Frame.Body.Y+row, e); got.Field == "delete" {
				t.Errorf("%s is a built-in and column %d offers to delete it", d.Name, col)
			}
		}
	}
}
