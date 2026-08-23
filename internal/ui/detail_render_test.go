package ui

// detail_render_test.go is the measured render extended to the task view,
// the three frames the task is specified by, and the two functions the diff
// tab's editor is built out of.
//
// The measured render is the same three assertions render_test.go makes —
// the right number of rows, no row wider in cells than the terminal, and the
// rows that must say something saying it — run over each of the three tabs
// at each size in each language. The assertion this task adds is the last
// test in the file: a diff line far wider than the pane must scroll inside
// it, and must never widen the frame around it.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/words"
)

// tabNames are the three tabs by the name a subtest reports them under.
var tabNames = []struct {
	name string
	tab  tab
}{{"log", tabLog}, {"diff", tabDiff}, {"evidence", tabEvidence}}

func TestEveryTaskViewFrameFitsTheTerminalItWasGiven(t *testing.T) {
	for _, lang := range []string{"en", "es", "qps"} {
		printer := printerFor(t, lang)
		for _, which := range tabNames {
			for _, size := range sizes {
				name := lang + "/" + which.name + "/" + strconv.Itoa(size.w) + "x" + strconv.Itoa(size.h)
				t.Run(name, func(t *testing.T) {
					m, _ := openIn(t, printer, "ACME-2662", fixtureEntries(), fixtureDiff)
					rows := renderAt(t, showing(t, m, which.tab), size.w, size.h)
					if len(rows) != size.h {
						t.Fatalf("the frame is %d rows, want %d", len(rows), size.h)
					}
					for i, row := range rows {
						if cells := ansi.StringWidth(row); cells > size.w {
							t.Errorf("row %d is %d cells wide, want at most %d: %q", i, cells, size.w, row)
						}
					}
					wantTaskRows(t, rows)
				})
			}
		}
	}
}

// wantTaskRows is the third assertion for the task view: the heading names
// the task the pane is open on, the band still says something, and the key
// bar still offers a key.
//
// The id is the anchor for the reason wantRows gives — no translation
// touches it and no layout abbreviates it — and it is the one fact that
// tells a reader whose changes they are looking at.
func wantTaskRows(t *testing.T, rows []string) {
	t.Helper()
	frame := strings.Join(rows, "\n")
	if !strings.Contains(frame, "ACME-2662") {
		t.Errorf("no row names the task the view is open on:\n%s", frame)
	}
	if strings.TrimSpace(rows[len(rows)-2]) == "" {
		t.Error("the activity band is blank under the task view")
	}
	if !strings.Contains(rows[len(rows)-1], "[") {
		t.Errorf("the key bar is %q, want it to offer at least one key", rows[len(rows)-1])
	}
}

// TestTheTaskViewIsTheScreenItWasSpecifiedAs is the three frames, one per
// tab, at the size the plan draws its screens at.
func TestTheTaskViewIsTheScreenItWasSpecifiedAs(t *testing.T) {
	for _, which := range tabNames {
		t.Run(which.name, func(t *testing.T) {
			m, _ := openIn(t, printerFor(t, "en"), "ACME-2662", fixtureEntries(), fixtureDiff)
			golden(t, "detail-"+which.name+"-100x30-en", renderAt(t, showing(t, m, which.tab), 100, 30))
		})
	}
}

// TestNoEnglishSurvivesThePseudolocaleInTheTaskView is the mechanical half
// of reading the pseudolocale frame, for the words this screen writes
// itself. A phase name, an engine, a model, an id and a session are data and
// stay as they are; everything below is the window talking.
func TestNoEnglishSurvivesThePseudolocaleInTheTaskView(t *testing.T) {
	for _, which := range tabNames {
		t.Run(which.name, func(t *testing.T) {
			m, _ := openIn(t, printerFor(t, "qps"), "ACME-2662", fixtureEntries(), fixtureDiff)
			frame := strings.Join(renderAt(t, showing(t, m, which.tab), 100, 30), "\n")
			// "diff" is not on this list and cannot be: git's own output
			// begins `diff --git`, which is data the pane quotes rather
			// than a word the window writes. The tab's name is covered by
			// "evidence" and "log" beside it, which appear nowhere in the
			// fixture's data.
			for _, english := range []string{
				"log", "evidence", "attempt", "scrolls", "following", "written down",
				"started", "finished", "failed", "waiting", "abandoned", "let go",
				"cost", "session", "kept", "bytes", "no changes", "printed nothing",
			} {
				if strings.Contains(frame, english) {
					t.Errorf("the pseudolocale frame still says %q, so that string never went through T:\n%s", english, frame)
				}
			}
		})
	}
}

// TestALongDiffLineNeverWidensTheFrame is the assertion this task adds to
// the measured render, and it is the one the pane exists to make true.
//
// A diff of a minified file arrives as a single line of several thousand
// cells. Wrapping it would push the rest of the hunk off the bottom of the
// screen; letting it through would push the frame past the terminal and make
// the whole window wrap, which in the program this replaces was how a wide
// diff took the layout apart. The pane cuts instead, and the reader reaches
// the rest of the line with ← and →.
func TestALongDiffLineNeverWidensTheFrame(t *testing.T) {
	for _, size := range sizes {
		t.Run(strconv.Itoa(size.w)+"x"+strconv.Itoa(size.h), func(t *testing.T) {
			m, _ := openIn(t, words.For("en"), "ACME-2662", fixtureEntries(), wideDiff())
			rows := renderAt(t, showing(t, m, tabDiff), size.w, size.h)
			for i, row := range rows {
				if cells := ansi.StringWidth(row); cells > size.w {
					t.Fatalf("row %d is %d cells wide against a terminal of %d — the diff widened the frame", i, cells, size.w)
				}
			}
			if len(rows) != size.h {
				t.Fatalf("the frame is %d rows, want %d", len(rows), size.h)
			}
		})
	}
}

// TestTheDiffKnowsWhichFileALineBelongsTo walks the pair of functions o is
// built from. They are the only arithmetic in this screen, and getting them
// wrong opens the right file at the wrong line — which is worse than not
// opening it, because a reader believes what the editor shows them.
func TestTheDiffKnowsWhichFileALineBelongsTo(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(fixtureDiff, "\n"), "\n")
	cases := []struct {
		name string
		at   int
		file string
		line int
		ok   bool
	}{
		{"the first line of the hunk is the hunk's own start", 5, "retry.go", 28, true},
		{"a context line counts towards the new file", 6, "retry.go", 29, true},
		{"an added line is where it was added", 9, "retry.go", 32, true},
		{"the furniture above the first hunk is that file at its top", 1, "retry.go", 1, true},
		{"the line the pane opens on is the first file below it", 0, "retry.go", 1, true},
		{"a row past the end of the diff is not a file", 99, "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, line, ok := fileAt(lines, c.at)
			if ok != c.ok || file != c.file || line != c.line {
				t.Errorf("fileAt(%d) = %q, %d, %v; want %q, %d, %v", c.at, file, line, ok, c.file, c.line, c.ok)
			}
		})
	}
	// The removal rule needs a diff that removes something, and the fixture
	// above only adds. A line that was taken out is not in the new file at
	// all, so the line under the one that follows it has not moved.
	t.Run("a removed line does not advance the new file's count", func(t *testing.T) {
		wide := strings.Split(strings.TrimSuffix(wideDiff(), "\n"), "\n")
		if file, line, ok := fileAt(wide, 5); file != "bundle.js" || line != 1 || !ok {
			t.Errorf("fileAt(5) = %q, %d, %v; want %q, 1, true", file, line, ok, "bundle.js")
		}
	})
}

// TestOOpensTheEditorInTheWorktreeAndNeverRunsIt builds the command and
// stops. The Cmd Update returns is tea.ExecProcess, which suspends the
// window and hands the terminal over; a test that executed it would launch
// whatever $EDITOR happens to be on the machine running the suite. What is
// worth asserting is what the command says, and that is all this reads.
func TestOOpensTheEditorInTheWorktreeAndNeverRunsIt(t *testing.T) {
	t.Setenv("EDITOR", "vi")
	m, _ := openIn(t, words.For("en"), "ACME-2662", fixtureEntries(), tallDiff())
	m = showing(t, m, tabDiff)
	// Five rows down the hunk, so that the line asserted below is one the
	// pane scrolled to rather than the one it happened to open on. The
	// arithmetic is the point: a command that always said +1 would pass a
	// test taken at the top of the diff.
	for range 8 {
		m = step(t, m, "down")
	}
	cmd, err := m.editorFor()
	if err != nil {
		t.Fatalf("build the editor command: %v", err)
	}
	if cmd.Dir != "/w/ACME-2662" {
		t.Errorf("the editor would run in %q, want the task's worktree", cmd.Dir)
	}
	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{"+31", "retry.go"} {
		if !strings.Contains(args, want) {
			t.Errorf("the editor command is %q, want it to mention %q", args, want)
		}
	}
}

// TestTheEditorRefusesWithoutOne is the other half: $EDITOR unset is a
// sentence in the band and not a silent nothing, because a key that appears
// to do nothing is indistinguishable from a key that is broken.
func TestTheEditorRefusesWithoutOne(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	m, _ := openDetail(t, "ACME-2662")
	after, cmd := showing(t, m, tabDiff).Update(press("o"))
	got, ok := after.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", after)
	}
	if cmd != nil {
		t.Error("o ran something with no editor configured")
	}
	wantBand(t, got, "EDITOR")
}

// TestTheMoreLineSaysWhatToDoAboutIt walks the row under the pane through
// the states it has.
//
// It is the one row on this screen whose whole job is to say that what is on
// screen is not all of it, and a row that says the same thing in every state
// is a row that stops being read. The third state the brief names — how to
// reach a pane that is not taking keystrokes — is not here because nothing
// in this window can put it in that state; moreLine's own comment says so.
func TestTheMoreLineSaysWhatToDoAboutIt(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", longLog())
	tail := paneText(t, m)
	wantIn(t, tail, "following")
	wantIn(t, tail, m.keys.Up.Help().Key)

	let := paneText(t, step(t, m, "up"))
	wantIn(t, let, "scrolls")
	if strings.Contains(let, "following") {
		t.Errorf("the frame still claims to be following after the reader scrolled up:\n%s", let)
	}

	// A pane whose content fits says nothing at all, which is what the
	// three goldens above are drawn with: the row is there and it is blank.
	short := renderAt(t, showing(t, m, tabDiff), 100, 30)
	if line := short[len(short)-4]; strings.TrimSpace(line) != "" {
		t.Errorf("the more line is %q over a diff that fits, want it left blank", line)
	}
}
