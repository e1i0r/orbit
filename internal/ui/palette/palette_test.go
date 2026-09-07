package palette

// The ':' line's own arithmetic: which candidate a typed prefix keeps, where
// the pointer's rows land, and what the line itself draws.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/words"
)

// three is a small table: one command with a usage and a description, one
// bare, and one the window refuses.
func three() []Command {
	return []Command{
		{Name: "new", Args: "<repo> <title>", About: "write a task"},
		{Name: "settings"},
		{Name: "nope", Refused: true, Because: "not from here"},
	}
}

// world is the Env: the words, the keys, a body of five rows, and that
// table.
func world(t *testing.T) Env {
	t.Helper()

	frame, err := layout.Fit(100, 30)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	return Env{
		Words:    words.For("en"),
		Keys:     keymap.New(words.For("en")),
		Frame:    frame,
		Commands: three(),
	}
}

// TestSettingsIsReachedByEveryWordAReaderCallsIt. The screen is called
// settings in one language and ajustes in the other, and a reader types
// whichever one they think in.
func TestSettingsIsReachedByEveryWordAReaderCallsIt(t *testing.T) {
	e := world(t)

	for _, typed := range []string{"conf", "config", "set", "ajus", "ajustes"} {
		s := OpenWith(typed)

		var found bool

		for _, c := range s.candidates(e.Commands) {
			if c.Name == "settings" {
				found = true
			}
		}

		if !found {
			t.Errorf("%q does not reach settings", typed)
		}
	}

	if len(OpenWith("zzz").candidates(e.Commands)) != 0 {
		t.Error("a prefix nothing starts with matched something")
	}
}

// TestOnlyTheFirstWordIsThePrefix. What follows it is the command's own
// arguments: matching against the whole line meant that the moment a space
// was typed nothing matched, so ⏎ on "cancel PAY-11" did nothing at all and
// the line sat there looking broken.
func TestOnlyTheFirstWordIsThePrefix(t *testing.T) {
	e := world(t)

	got := OpenWith("new PAY-11 something").candidates(e.Commands)
	if len(got) != 1 || got[0].Name != "new" {
		t.Errorf("a command with its arguments typed after it matched %+v", got)
	}
}

// TestAVerbAboutOneTaskIsNotOnThisLine. The line is opened on the board,
// where there is no task for such a verb to be about.
func TestAVerbAboutOneTaskIsNotOnThisLine(t *testing.T) {
	e := world(t)
	e.Commands = append(e.Commands, Command{Name: "pr", AboutATask: true})

	for _, c := range Open().candidates(e.Commands) {
		if c.Name == "pr" {
			t.Error("a verb about one task is offered on the board's own line")
		}
	}
}

// TestTheListFollowsTheSelectionRatherThanTheReaderFollowingTheList.
func TestTheListFollowsTheSelectionRatherThanTheReaderFollowingTheList(t *testing.T) {
	e := world(t)
	e.Frame.Body.H = 5

	for _, c := range []struct {
		what        string
		h, sel, off int
		want        int
	}{
		{"no room at all", 0, 0, 4, 0},
		{"the selection above the window", 5, 1, 6, 1},
		{"the selection below the window", 5, 10, 0, 6},
		{"already on screen", 5, 2, 1, 1},
		{"a window taller than the selection", 1000, 5, -2000, 0},
	} {
		e.Frame.Body.H = c.h

		s := State{sel: c.sel, offset: c.off}
		if got := s.ensureVisible(e).offset; got != c.want {
			t.Errorf("%s: offset %d, want %d", c.what, got, c.want)
		}
	}
}

// TestTabCompletesWhatWasStarted, and never jumps sideways: the list is
// already prefix-filtered, so the selection always completes what was typed.
func TestTabCompletesWhatWasStarted(t *testing.T) {
	e := world(t)

	after, _ := OpenWith("n").Key(tea.KeyPressMsg{Code: tea.KeyTab}, e)
	if after.Typed() != "new" {
		t.Errorf("tab left %q on the line, want new", after.Typed())
	}

	if after.sel != 0 || after.offset != 0 {
		t.Errorf("tab left sel=%d offset=%d, want the list back at the top", after.sel, after.offset)
	}

	// Nothing selected: the line is left exactly as it was.
	if none, _ := OpenWith("zzz-no-match").Key(tea.KeyPressMsg{Code: tea.KeyTab}, e); none.Typed() != "zzz-no-match" {
		t.Errorf("tab with nothing selected changed the line to %q", none.Typed())
	}
}

// TestTheLineDrawsThePlaceholderUntilSomethingIsTyped.
func TestTheLineDrawsThePlaceholderUntilSomethingIsTyped(t *testing.T) {
	e := world(t)

	if line := Open().Line(40, e); !strings.Contains(line, "type a command") {
		t.Errorf("an empty line reads %q, want the placeholder", line)
	}

	if line := OpenWith("set").Line(40, e); !strings.Contains(line, "set") {
		t.Errorf("the line reads %q, want what was typed", line)
	}
}

// TestARefusalReplacesTheDescriptionRatherThanJoiningIt: the reason is the
// part a reader acts on, and a line carrying both is a line truncated to
// lose whichever mattered.
func TestARefusalReplacesTheDescriptionRatherThanJoiningIt(t *testing.T) {
	cmds := three()

	plain := row(cmds[0], false, 60)
	if !strings.Contains(plain, "new") || !strings.Contains(plain, "<repo>") {
		t.Errorf("the row for new reads %q, want the name and its usage", plain)
	}

	if refused := row(cmds[2], false, 60); !strings.Contains(refused, "not from here") {
		t.Errorf("the row for a refused command reads %q, want the reason", refused)
	}

	if row(cmds[0], true, 60) == plain {
		t.Error("the selected row is drawn the same as an unselected one")
	}
}

// TestThePointerReadsTheSameRowsTheDrawingDrew.
func TestThePointerReadsTheSameRowsTheDrawingDrew(t *testing.T) {
	e := world(t)
	s := Open()

	if got := s.Hit(0, 9999, e); got.Kind != point.None {
		t.Errorf("a click far past the body = %+v, want nothing", got)
	}

	got := s.Hit(0, e.Frame.Body.Y, e)
	if got.Kind != point.Command || got.Key != "new" {
		t.Errorf("a click on the first row = %+v, want the first command", got)
	}

	if past := OpenWith("new").Hit(0, e.Frame.Body.Y+3, e); past.Kind != point.None {
		t.Errorf("a click past the filtered list = %+v, want nothing", past)
	}
}

// TestNothingMatchingSaysSoRatherThanDrawingAnEmptyBody.
func TestNothingMatchingSaysSoRatherThanDrawingAnEmptyBody(t *testing.T) {
	e := world(t)

	drawn := strings.Join(OpenWith("zzz").View(10, 80, e), "\n")
	if !strings.Contains(drawn, "no command starts with") {
		t.Errorf("a line nothing matches drew:\n%s", drawn)
	}
}

// TestACommandThatNeedsArgumentsKeepsTheLineUpAndSaysWhatIsMissing. Closing
// it to print the same sentence into a pane is where this used to end: an
// answer with nowhere to type what it asks for.
func TestACommandThatNeedsArgumentsKeepsTheLineUpAndSaysWhatIsMissing(t *testing.T) {
	e := world(t)
	e.Commands = []Command{{Name: "export", Args: "<dir>", NeedsArgs: true}}

	bare, out := OpenWith("export ").Run(e)
	if out.Run != "" || !bare.Up() {
		t.Errorf("a bare export answered run=%q with the line up=%v", out.Run, bare.Up())
	}

	if !strings.Contains(out.Said, "export") {
		t.Errorf("it said %q, want it to name the command and what it wants", out.Said)
	}

	ran, out := OpenWith("export /tmp/out").Run(e)
	if out.Run != "export" || !out.Leave || ran.Up() {
		t.Errorf("export with its argument answered %+v with the line up=%v", out, ran.Up())
	}

	if got := Args(out.Line); len(got) != 1 || got[0] != "/tmp/out" {
		t.Errorf("the arguments are %v, want the one that was typed", got)
	}
}

// TestAClickTakesTwoPresses: the first selects, the second runs. It is the
// board's own rule, and it is what keeps a stray click off a command.
func TestAClickTakesTwoPresses(t *testing.T) {
	e := world(t)

	first, out := Open().Choose("settings", e)
	if out.Run != "" {
		t.Errorf("the first click ran %q", out.Run)
	}

	if first.sel != 1 {
		t.Errorf("the first click left the selection on %d, want the row it landed on", first.sel)
	}

	if _, out := first.Choose("settings", e); out.Run != "settings" {
		t.Errorf("the second click on the same row ran %q", out.Run)
	}

	// A name the list does not hold changes nothing.
	if _, out := first.Choose("no-such-command", e); out.Run != "" || out.Leave {
		t.Errorf("clicking a command that is not on the list answered %+v", out)
	}
}
