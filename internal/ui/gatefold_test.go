package ui

// Folding a check and a refusal. Both tabs answer one question first — did
// it pass, was it stopped — and both carry a paragraph behind that answer:
// the command a gate ran and why it failed, the sentence the sandbox wrote
// down. A screen that sets every paragraph open is a screen the answer is
// lost on.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// wordy is a sentence too long for any row of a detail pane, in words that
// can be broken between, so that what folds is the wrapping and not a run of
// characters nothing can break.
const wordy = "the command ran to the end and came back with a status nobody " +
	"on this side of the gate can do anything about, which is the whole of " +
	"what there is to say about it and rather more than one row holds"

// onTab is the window on one tab over a record a test chose, and the rows it
// draws.
func onTab(t *testing.T, which tab, entries []view.Entry) (Model, []string) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", entries)
	m = showing(t, m, which)

	return m, screenRows(m)
}

// TestACheckWithMoreToSayOffersToOpen. The verdict is the row; the command
// and the reason are what the reader asks for after reading the verdict.
func TestACheckWithMoreToSayOffersToOpen(t *testing.T) {
	m, lines := onTab(t, tabGates, []view.Entry{
		{Kind: "gate.passed", Gate: "lint", Text: "make lint"},
		{Kind: "gate.failed", Gate: "check", Text: "make check", Cause: wordy},
	})

	pass := rowOf(lines, "lint")
	if pass < 0 {
		t.Fatalf("the check that passed is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	// A check whose whole story fits beside its name is offered no arrow:
	// opening it would put nothing on the screen that was not already there.
	if strings.Contains(lines[pass], foldShut) {
		t.Errorf("a check with nothing more to say offers to open: %q", lines[pass])
	}

	fail := rowOf(lines, "check")
	if fail < 0 {
		t.Fatalf("the check that failed is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[fail], foldShut) {
		t.Errorf("the check that failed does not offer to open: %q", lines[fail])
	}

	if strings.Contains(strings.Join(lines, "\n"), "on this side of the gate") {
		t.Errorf("the reason is on the screen before it was asked for:\n%s", strings.Join(lines, "\n"))
	}

	at := m.hit(30, fail)
	if at.Kind != TargetPaneRow || at.Pane != 1 {
		t.Fatalf("pointing at the check that failed = %+v, want row 1 of the pane", at)
	}

	opened := screenRows(clicked(t, m, at))
	if rowOf(opened, "on this side of the gate") < 0 {
		t.Errorf("opening the check did not say why it failed:\n%s", strings.Join(opened, "\n"))
	}
}

// TestACheckIsWrappedAndCutToItsPane. A gate is a shell command and a reason
// is whatever a runner printed: neither was written to fit a column, and a
// path with nothing to break at runs off the pane even wrapped.
func TestACheckIsWrappedAndCutToItsPane(t *testing.T) {
	run := strings.Repeat("x", 200)
	m, lines := onTab(t, tabGates, []view.Entry{
		{Kind: "gate.failed", Gate: "check", Text: run, Cause: wordy + " " + run},
	})

	m = clicked(t, m, m.hit(30, node(t, lines, "check")))

	for i, l := range m.gatesLines() {
		plain := strings.TrimRight(ansi.Strip(l), " ")

		if w := lipgloss.Width(plain); w > m.frame.Body.W {
			t.Errorf("row %d of the gates runs to %d cells on a pane of %d: %q", i, w, m.frame.Body.W, plain)
		}
	}
}

// TestARefusalSaysWhatTheSandboxSaid. What was reached for is the row; what
// the sandbox wrote back is a paragraph, and a paragraph set on one row loses
// the half that says why.
func TestARefusalSaysWhatTheSandboxSaid(t *testing.T) {
	m, lines := onTab(t, tabRefused, []view.Entry{
		{Kind: "phase.refused", Tool: "git push", Text: wordy},
	})

	y := rowOf(lines, "git push")
	if y < 0 {
		t.Fatalf("the refusal is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[y], foldShut) {
		t.Errorf("the refusal does not offer to open: %q", lines[y])
	}

	if strings.Contains(strings.Join(lines, "\n"), "rather more than one row holds") {
		t.Errorf("the whole refusal is on the screen before it was asked for:\n%s", strings.Join(lines, "\n"))
	}

	at := m.hit(30, y)
	if at.Kind != TargetPaneRow || at.Pane != 0 {
		t.Fatalf("pointing at the refusal = %+v, want row 0 of the pane", at)
	}

	opened := screenRows(clicked(t, m, at))
	if rowOf(opened, "rather more than one row holds") < 0 {
		t.Errorf("opening the refusal did not say what the sandbox said:\n%s", strings.Join(opened, "\n"))
	}

	// The standing rules are the tab's other half and belong to no refusal:
	// a fold that swallowed them would take the answer to "why was it
	// stopped" off the screen with the refusal it was opened from.
	if rowOf(opened, "git remote / config") < 0 {
		t.Errorf("opening a refusal took the sandbox rules off the tab:\n%s", strings.Join(opened, "\n"))
	}
}

// TestARefusalWithoutAToolIsStillNamed. The record does not always say what
// was reached for, and a row that names nothing is a row a reader cannot act
// on.
func TestARefusalWithoutAToolIsStillNamed(t *testing.T) {
	_, lines := onTab(t, tabRefused, []view.Entry{{Kind: "phase.refused", Text: "denied"}})

	y := rowOf(lines, "command")
	if y < 0 {
		t.Fatalf("the unnamed refusal is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[y], "denied") {
		t.Errorf("the unnamed refusal does not say what the sandbox said: %q", lines[y])
	}

	// One row of it and nothing to open: an arrow that opens onto the row it
	// is already showing is an arrow that lies.
	if strings.Contains(lines[y], foldShut) {
		t.Errorf("a refusal with nothing more to say offers to open: %q", lines[y])
	}
}

// TestALongNoteIsClosedAndSaysHowMuchIsUnderIt. A note is Markdown the
// operator wrote and is as long as they made it; a tab that sets ten of them
// open is a tab the eleventh cannot be found on.
func TestALongNoteIsClosedAndSaysHowMuchIsUnderIt(t *testing.T) {
	m, lines := onTab(t, tabNotes, []view.Entry{
		{
			Kind: "task.noted", At: ago(3 * time.Minute), Attempt: 1,
			Text: "the first thing\nthe second thing\nthe third thing",
		},
	})

	y := rowOf(lines, "OPERATOR")
	if y < 0 {
		t.Fatalf("the note is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[y], foldShut) {
		t.Errorf("the note does not offer to open: %q", lines[y])
	}

	if rowOf(lines, "the third thing") >= 0 {
		t.Errorf("the whole note is on the screen before it was asked for:\n%s", strings.Join(lines, "\n"))
	}

	// What is behind the arrow is said as a count, not left to be guessed:
	// a reader skimming a thread decides which note to open by how much of
	// it there is.
	if rowOf(lines, "2 more lines") < 0 {
		t.Errorf("the closed note does not count what is under it:\n%s", strings.Join(lines, "\n"))
	}

	at := m.hit(30, y)
	if at.Kind != TargetPaneRow || at.Pane != 0 {
		t.Fatalf("pointing at the note = %+v, want row 0 of the pane", at)
	}

	opened := screenRows(clicked(t, m, at))
	if rowOf(opened, "the third thing") < 0 {
		t.Errorf("opening the note did not show the rest of it:\n%s", strings.Join(opened, "\n"))
	}

	if rowOf(opened, "more lines") >= 0 {
		t.Errorf("the open note still counts what is under it:\n%s", strings.Join(opened, "\n"))
	}
}

// TestAOneLineNoteHasNothingToOpen. An arrow that opens onto the row it is
// already showing is an arrow that lies, and a thread of them is a thread
// nobody trusts an arrow on again.
func TestAOneLineNoteHasNothingToOpen(t *testing.T) {
	_, lines := onTab(t, tabNotes, []view.Entry{
		{Kind: "task.noted", At: ago(time.Minute), Text: "one line and no more"},
	})

	y := rowOf(lines, "OPERATOR")
	if y < 0 {
		t.Fatalf("the note is not on the tab:\n%s", strings.Join(lines, "\n"))
	}

	if strings.Contains(lines[y], foldShut) {
		t.Errorf("a note with nothing more to say offers to open: %q", lines[y])
	}

	if rowOf(lines, "one line and no more") < 0 {
		t.Errorf("the note that does not fold is not showing what it says:\n%s", strings.Join(lines, "\n"))
	}
}

// TestAColumnIsPaddedOnWhatIsSeen. A width verb counts the bytes of an
// escape sequence as characters, so a painted string padded to a column is
// not padded at all. Both panes that draw a table are checked together
// because the mistake is one mistake.
func TestAColumnIsPaddedOnWhatIsSeen(t *testing.T) {
	m, _ := onTab(t, tabCost, fixtureEntries())

	for name, rows := range map[string][]string{
		"cost": m.costLines(), "artifacts": m.artifactsLines(),
	} {
		var body []string

		for _, l := range rows {
			if plain := strings.TrimRight(ansi.Strip(l), " "); strings.HasPrefix(plain, "    ") {
				body = append(body, plain[4:])
			}
		}

		if len(body) < 2 {
			t.Fatalf("the %s table drew %d rows to compare, so this proves nothing", name, len(body))
		}

		// The widest row says where the columns are; every other row has to
		// start its fields at one of them, or the table is not a table.
		cols := map[int]bool{}
		for _, r := range body {
			if starts := columnStarts(r); len(starts) > len(cols) {
				cols = map[int]bool{}
				for _, c := range starts {
					cols[c] = true
				}
			}
		}

		for _, r := range body {
			for _, c := range columnStarts(r) {
				if !cols[c] {
					t.Errorf("a row of the %s table starts a field at cell %d, off every column %v: %q", name, c, cols, r)
				}
			}
		}
	}

	// The figures are the column a reader runs an eye down, and a total that
	// does not land under the costs it totals is the one misalignment nobody
	// can read past.
	money := map[int]string{}

	for _, l := range m.costLines() {
		plain := ansi.Strip(l)
		if at := strings.Index(plain, "$"); at >= 0 {
			money[lipgloss.Width(plain[:at])] = plain
		}
	}

	if len(money) > 1 {
		t.Errorf("the cost figures start in %d different places: %v", len(money), money)
	}
}

// columnStarts is the cell each of a row's fields begins at. A field ends
// where two spaces do, which is the gap a padded column always leaves and a
// value never contains.
func columnStarts(row string) []int {
	var (
		out []int
		at  int
	)

	for rest := row; rest != ""; {
		trimmed := strings.TrimLeft(rest, " ")
		at += lipgloss.Width(rest[:len(rest)-len(trimmed)])

		gap := strings.Index(trimmed, "  ")
		if gap < 0 {
			return append(out, at)
		}

		out = append(out, at)
		at += lipgloss.Width(trimmed[:gap])
		rest = trimmed[gap:]
	}

	return out
}
