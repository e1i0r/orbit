package ui

// render_test.go is the measured render, and the plan expects it to catch
// more real bugs than everything else in this task combined.
//
// It renders the window at a fixed size, strips the escape sequences, splits
// the result into rows, and asserts three things: that there are exactly as
// many rows as the terminal was given, that no row is wider in *cells* than
// the terminal was given, and that the rows that must say something do.
// Nothing here compares against a stored file, so a deliberate change to the
// wording never fails it — only a frame that does not fit does.
//
// The matrix is five boards by four sizes by three languages. The
// pseudolocale is in it because a translated string is longer than its
// English almost everywhere, and a layout measured only in English is a
// layout that has been measured in its narrowest case.

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// renderAt draws one model at one size and returns its rows, with colour
// pinned to the ASCII profile so a palette change can never fail a test and
// a developer whose shell sets NO_COLOR reads the same bytes CI does.
//
// The profile is applied through colorprofile.Writer — the same machinery
// tea.WithColorProfile drives — and then ansi.Strip removes what is left, so
// what is measured is the text a terminal would actually paint cells with.
func renderAt(t *testing.T, m Model, w, h int) []string {
	t.Helper()

	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})

	sized, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}

	var buf bytes.Buffer

	writer := &colorprofile.Writer{Forward: &buf, Profile: colorprofile.Ascii}
	if _, err := writer.WriteString(sized.View().Content); err != nil {
		t.Fatalf("write the frame: %v", err)
	}

	return strings.Split(ansi.Strip(buf.String()), "\n")
}

// boards are the five shapes the matrix runs over. They are functions rather
// than values so that one test cannot hand the next a board it has changed.
func boards() []struct {
	name  string
	board board.Board
} {
	one := fixtureTasks()[3:4]

	sameRepo := fixtureTasks()
	for i := range sameRepo {
		sameRepo[i].Repo = "payments"
	}

	var thirty []view.Task

	for i, t := range fixtureTasks() {
		for n := range 6 {
			t.ID = "ACME-" + string(rune('2'+n)) + "70" + string(rune('0'+i))
			thirty = append(thirty, t)
		}
	}

	return []struct {
		name  string
		board board.Board
	}{
		{"empty", fixtureBoard(nil, 4)},
		{"one task", fixtureBoard(one, 1)},
		{"thirty tasks", fixtureBoard(thirty[:30], 4)},
		{"one repository", fixtureBoard(sameRepo, 1)},
		{"four repositories", fixtureBoard(fixtureTasks(), 4)},
	}
}

// sizes are the terminals the window is measured in. 60x20 is the smallest
// one it accepts at all, and it is the one that finds the bugs.
var sizes = []struct{ w, h int }{{200, 60}, {100, 30}, {80, 24}, {60, 20}}

func TestEveryFrameFitsTheTerminalItWasGiven(t *testing.T) {
	for _, lang := range []string{"en", "es", "qps"} {
		printer := printerFor(t, lang)

		for _, b := range boards() {
			for _, size := range sizes {
				name := lang + "/" + b.name + "/" + strconv.Itoa(size.w) + "x" + strconv.Itoa(size.h)
				t.Run(name, func(t *testing.T) {
					m := modelWith(t, printer, b.board, size.w, size.h, nil)

					rows := renderAt(t, m, size.w, size.h)
					if len(rows) != size.h {
						t.Fatalf("the frame is %d rows, want %d", len(rows), size.h)
					}

					for i, row := range rows {
						if cells := ansi.StringWidth(row); cells > size.w {
							t.Errorf("row %d is %d cells wide, want at most %d: %q", i, cells, size.w, row)
						}
					}

					wantRows(t, m, rows)
				})
			}
		}
	}
}

// wantRows is assertion (iii): the rows that must say something, saying it.
//
// The product name anchors the header in every language because it is a
// name and not a word — it is the one run of English a pseudolocale frame is
// supposed to contain. The bar is anchored on its brackets for the same
// reason: a key's glyph is the same in every language and its description is
// not.
//
// The body is anchored on a task's id at the row the model says it is on,
// worked out from the model rather than searched for: a search finds the id
// wherever it has drifted to, and the drift is the bug. The id is safe to
// anchor on for the same reason the name is — layout never abbreviates the
// id column and no translation touches it.
func wantRows(t *testing.T, m Model, rows []string) {
	t.Helper()

	if !strings.Contains(rows[0], "orbit") {
		t.Errorf("the header row is %q, want it to name the program", rows[0])
	}

	list := m.rows()

	shown := page(m.frame.Body.H-1, len(list), m.offset)
	for i := m.offset; i < len(list) && i-m.offset < shown; i++ {
		if list[i].head || list[i].blank {
			continue
		}

		row := m.frame.Body.Y + 1 + i - m.offset
		if !strings.Contains(rows[row], list[i].task.ID) {
			t.Errorf("row %d is %q, want the first task, %s, on it", row, rows[row], list[i].task.ID)
		}

		break
	}

	band := rows[len(rows)-4]
	if strings.TrimSpace(band) == "" {
		t.Error("the activity band is blank, and a status area that goes blank reads as broken")
	}

	bar := rows[len(rows)-1]
	if !strings.Contains(bar, "[") {
		t.Errorf("the key bar is %q, want it to offer at least one key", bar)
	}
}

// TestAnAppliedFilterKeepsSayingSoAfterTheTypingStops is the band's job once
// the prompt closes. Enter hands the keyboard back and leaves the filter on,
// so every count on the screen is smaller than the board's own and the band
// is the only place that says why. A window that only says it while the
// reader is typing tells a reader who set a filter and came back an hour
// later that four of their tasks have gone missing.
func TestAnAppliedFilterKeepsSayingSoAfterTheTypingStops(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	for _, k := range []string{"/", "a", "p", "p", "enter"} {
		next, _ := m.Update(press(k))

		typed, ok := next.(Model)
		if !ok {
			t.Fatalf("Update returned %T, want ui.Model", next)
		}

		m = typed
	}

	if m.filtering || m.filter != "app" {
		t.Fatalf("filtering=%v filter=%q, want the prompt closed and the filter kept", m.filtering, m.filter)
	}

	rows := renderAt(t, m, 100, 30)
	band := rows[len(rows)-4]
	// Six of the fifteen fixture tasks are in the app repository, two of
	// them in a band the window opens on and four in one it does not: the
	// sentence counts what the filter let through, not what is drawn.
	for _, want := range []string{"/app", "6 of 15 shown", "esc"} {
		if !strings.Contains(band, want) {
			t.Errorf("the band says %q, want %q in it", band, want)
		}
	}
}

// TestATerminalUnderTheMinimumIsRefusedWithTheNumber is the other half of
// the same rule: a window that cannot be drawn says so in one line and says
// how many columns it needed, rather than drawing a broken one.
func TestATerminalUnderTheMinimumIsRefusedWithTheNumber(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	rows := renderAt(t, m, 40, 20)
	if len(rows) != 20 {
		t.Fatalf("the refusal is %d rows, want 20", len(rows))
	}

	joined := strings.Join(rows, "\n")
	for _, want := range []string{"60", "40"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal does not carry %q: %q", want, strings.TrimSpace(joined))
		}
	}

	for i, row := range rows {
		if cells := ansi.StringWidth(row); cells > 40 {
			t.Errorf("row %d of the refusal is %d cells wide, want at most 40", i, cells)
		}
	}
}

// TestTheEmptyStateSaysWhichKindOfEmpty walks the three sentences the plan
// says "empty" actually is. The panes in the program this replaces taught
// this: one word for three situations sends a reader looking for a task
// they are certain they wrote.
func TestTheEmptyStateSaysWhichKindOfEmpty(t *testing.T) {
	printer := words.For("en")
	cases := []struct {
		name  string
		model func(t *testing.T) Model
		want  string
	}{{
		name:  "repositories, and no tasks in them",
		model: func(t *testing.T) Model { return modelWith(t, printer, fixtureBoard(nil, 4), 100, 30, nil) },
		want:  "4 repositories",
	}, {
		name:  "no repositories at all",
		model: func(t *testing.T) Model { return modelWith(t, printer, fixtureBoard(nil, 0), 100, 30, nil) },
		want:  "~/work",
	}, {
		name: "a filter that emptied the list",
		model: func(t *testing.T) Model {
			m := modelWith(t, printer, fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
			m.filtering = true
			m.filter = "nothing matches this"

			return m
		},
		want: "nothing matches this",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := strings.Join(renderAt(t, c.model(t), 100, 30), "\n")
			if !strings.Contains(body, c.want) {
				t.Errorf("the empty state does not mention %q:\n%s", c.want, body)
			}
		})
	}
}
