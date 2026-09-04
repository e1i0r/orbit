package ui

// Folding an attempt: the rule between one run of a task and the next is
// also the lid on it. A task tried three times holds three reports, and two
// of them are history the moment the third one starts.

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// attemptRule is how one attempt's seam reads with the colour taken off it.
func attemptRule(n int) string { return "── attempt " + strconv.Itoa(n) }

// onReport is the window on the report tab, and the rows it draws.
func onReport(t *testing.T, entries []view.Entry) (Model, []string) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", entries)
	m = showing(t, m, tabReport)

	return m, screenRows(m)
}

// clicked is the window after the pointer went down and came up on a target.
func clicked(t *testing.T, m Model, at Target) Model {
	t.Helper()

	next, _ := m.leftClick(at)

	return asModel(t, next)
}

// TestAnAttemptSaysItCanBeClosed. Both panes that draw the rule draw the
// same one, because the two tabs are two readings of one record and a reader
// who learns the gesture on one of them has learnt it on the other.
func TestAnAttemptSaysItCanBeClosed(t *testing.T) {
	_, timelineRows := timeline(t, fixtureEntries())
	_, reportRows := onReport(t, fixtureEntries())

	for name, lines := range map[string][]string{"timeline": timelineRows, "report": reportRows} {
		y := rowOf(lines, attemptRule(1))
		if y < 0 {
			t.Errorf("%s: the first attempt has no rule:\n%s", name, strings.Join(lines, "\n"))
			continue
		}

		if !strings.Contains(lines[y], foldOpen) {
			t.Errorf("%s: an attempt nobody has closed is not drawn as open: %q", name, lines[y])
		}
	}
}

// TestClosingAnAttemptTakesItsReportOffThePane, end to end: the rule is
// where the pointer is answered, what it closes is the attempt under it, and
// the attempt beside it is left alone.
func TestClosingAnAttemptTakesItsReportOffThePane(t *testing.T) {
	m, lines := onReport(t, fixtureEntries())

	if rowOf(lines, "wrote retry.go") < 0 {
		t.Fatalf("the first attempt's report is not on the pane:\n%s", strings.Join(lines, "\n"))
	}

	at := m.hit(30, rowOf(lines, attemptRule(1)))
	if at.Kind != TargetSeam || at.Pane != 1 {
		t.Fatalf("the first attempt's rule answers as %+v, want the first attempt", at)
	}

	rows := screenRows(clicked(t, m, at))

	if rowOf(rows, "wrote retry.go") >= 0 {
		t.Errorf("a closed attempt printed its report anyway:\n%s", strings.Join(rows, "\n"))
	}

	if rowOf(rows, "b41d07") < 0 {
		t.Errorf("closing one attempt took the next one's report with it:\n%s", strings.Join(rows, "\n"))
	}

	head := rowOf(rows, attemptRule(1))
	if head < 0 || !strings.Contains(rows[head], foldShut) {
		t.Fatalf("a closed attempt has no rule to open it again:\n%s", strings.Join(rows, "\n"))
	}

	if next := rowOf(rows, attemptRule(2)); next != head+1 {
		t.Errorf("the two rules are %d rows apart, want one:\n%s", next-head, strings.Join(rows, "\n"))
	}

	back := screenRows(clicked(t, clicked(t, m, at), at))
	if rowOf(back, "wrote retry.go") < 0 {
		t.Errorf("clicking a closed attempt did not open it again:\n%s", strings.Join(back, "\n"))
	}
}

// TestClosingAnAttemptTakesItsEntriesOffTheTimeline. The same gesture on the
// log, where what folds away is every line the attempt wrote — and not the
// lines written before any attempt began, which belong to the task.
func TestClosingAnAttemptTakesItsEntriesOffTheTimeline(t *testing.T) {
	m, lines := timeline(t, fixtureEntries())

	at := m.hit(30, rowOf(lines, attemptRule(1)))
	if at.Kind != TargetSeam || at.Pane != 1 {
		t.Fatalf("the first attempt's rule answers as %+v, want the first attempt", at)
	}

	rows := screenRows(clicked(t, m, at))

	head, next := rowOf(rows, attemptRule(1)), rowOf(rows, attemptRule(2))
	if head < 0 || next < 0 {
		t.Fatalf("closing an attempt took a rule with it:\n%s", strings.Join(rows, "\n"))
	}

	if next != head+1 {
		t.Errorf("a closed attempt kept %d rows of the log:\n%s", next-head-1, strings.Join(rows, "\n"))
	}

	if rowOf(rows, "written down") < 0 {
		t.Errorf("closing an attempt took the lines written before any attempt:\n%s", strings.Join(rows, "\n"))
	}
}

// TestAReportFoldedUpStillOffersItsAttempts. The pane says a task has no
// report when the record holds none; a reader who has shut every rule on it
// has to be able to open one again.
func TestAReportFoldedUpStillOffersItsAttempts(t *testing.T) {
	m, _ := onReport(t, fixtureEntries())

	for _, n := range []int{1, 2} {
		y := rowOf(screenRows(m), attemptRule(n))
		if y < 0 {
			t.Fatalf("attempt %d has no rule to close", n)
		}

		m = clicked(t, m, m.hit(30, y))
	}

	rows := screenRows(m)
	if rowOf(rows, "no engine report") >= 0 {
		t.Errorf("a report the reader folded up reads as a task that has none:\n%s", strings.Join(rows, "\n"))
	}

	for _, n := range []int{1, 2} {
		y := rowOf(rows, attemptRule(n))
		if y < 0 {
			t.Fatalf("attempt %d lost the rule that opens it again:\n%s", n, strings.Join(rows, "\n"))
		}

		if !strings.Contains(rows[y], foldShut) {
			t.Errorf("a closed attempt is not drawn as closed: %q", rows[y])
		}
	}
}

// TestOnlyTheRuleAnswersForTheAttempt. The rows an attempt runs on belong to
// the pane, or a reader who points anywhere in a report closes the report.
func TestOnlyTheRuleAnswersForTheAttempt(t *testing.T) {
	m, lines := onReport(t, fixtureEntries())

	inside := rowOf(lines, "wrote retry.go")
	if inside < 0 {
		t.Fatalf("the first attempt's report is not on the pane:\n%s", strings.Join(lines, "\n"))
	}

	if got := m.hit(30, inside); got.Kind == TargetSeam {
		t.Errorf("a row of the attempt's own report answers as its rule: %+v", got)
	}
}

// TestTheAttemptUnderThePointerIsTheOneThatFolds. The rows the hit test
// knows are the pane's own and a pointer reports the window's, so a rule
// that has been scrolled has to answer at the row it is on now.
func TestTheAttemptUnderThePointerIsTheOneThatFolds(t *testing.T) {
	m, lines := timeline(t, deepLog())

	y := rowOf(lines, attemptRule(3))
	if y < 0 {
		t.Fatalf("the last attempt's rule was not drawn:\n%s", strings.Join(lines, "\n"))
	}

	want := m.hit(30, y)
	if want.Kind != TargetSeam || want.Pane != 3 {
		t.Fatalf("the last attempt's rule answers as %+v, want the third attempt", want)
	}

	scrolled := scrolledTo(m, lineAt(m)-3)

	moved := rowOf(screenRows(scrolled), attemptRule(3))
	if moved < 0 {
		t.Fatalf("scrolling took the rule off the screen:\n%s", strings.Join(screenRows(scrolled), "\n"))
	}

	if moved == y {
		t.Fatal("scrolling did not move the rule")
	}

	if got := scrolled.hit(30, moved); got != want {
		t.Errorf("after scrolling, the rule answers as %+v, want %+v", got, want)
	}
}

// deepLog is a record too long for one screen whose last attempt begins a
// few rows above the end of it: far enough down to be on the screen, far
// enough up to still be on it after a scroll.
func deepLog() []view.Entry {
	out := append(longLog(), view.Entry{
		At: ago(6 * time.Minute), Kind: "task.started", Attempt: 3,
	})

	for i := range 4 {
		out = append(out, view.Entry{
			At: ago(time.Duration(5-i) * time.Minute), Kind: "phase.started",
			Phase: "gates", Attempt: 3, PhaseN: 2, Engine: "claude", Model: "opus",
		})
	}

	return out
}

// TestOpeningATaskForgetsWhatWasFolded. Folding is the window's and not the
// task's: a reader who shut an attempt on one task and then opened another
// is looking at a record they have never folded anything in.
func TestOpeningATaskForgetsWhatWasFolded(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", wordyLog())
	nextM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 36})
	m = asModel(t, nextM)
	m = showing(t, m, tabTimeline)
	lines := screenRows(m)

	// The entry first: once an attempt is closed its own rule is the first
	// shut arrow on the screen, and the click would land on that instead.
	m = clicked(t, m, m.hit(30, rowOf(lines, foldShut)))
	m = clicked(t, m, m.hit(30, rowOf(screenRows(m), attemptRule(1))))

	if m.shutAttempts == nil || m.opened[tabTimeline] == nil {
		t.Fatal("the window did not remember what the reader folded")
	}

	again := next(t, step(t, step(t, m, "esc"), "enter"),
		logMsg{ID: "ACME-2662", Entries: wordyLog()})

	if again.shutAttempts != nil || again.opened[tabTimeline] != nil {
		t.Errorf("a task opened again is folded as the last one was left: %v %v",
			again.shutAttempts, again.opened[tabTimeline])
	}
}
