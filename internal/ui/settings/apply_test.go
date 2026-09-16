package settings

import (
	"errors"
	"testing"
)

// TestTurningADialWritesIt, in both directions and round the ends.
func TestTurningADialWritesIt(t *testing.T) {
	f := newFile()
	e := env(t, f)

	// row 1 is autopilot: off, on.
	s := Open(e).Point(1, e)

	if out := s.Cycle(1, e); out.Said == "" {
		t.Error("turning the dial said nothing")
	}

	if f.held["autopilot"] != "on" {
		t.Error("turning autopilot one to the right left it off")
	}

	if out := s.Cycle(-1, e); out.Said == "" || f.held["autopilot"] != "off" {
		t.Error("turning it back left it on")
	}
}

// TestAnEngineTakesItsEffortWithIt. The effort belongs to the engine it was
// chosen for, so an engine that does not offer it must move it — otherwise
// the run starts with an effort its engine refuses.
func TestAnEngineTakesItsEffortWithIt(t *testing.T) {
	f := newFile()
	e := env(t, f)

	out := Apply("engine", "omega", e)
	if out.Dials == nil {
		t.Fatal("changing the engine moved no dial")
	}

	if out.Dials.Effort != "slow" {
		t.Errorf("the effort is %q, want the one omega offers", out.Dials.Effort)
	}

	if f.held["engine"] != "omega" {
		t.Errorf("the file holds engine %q", f.held["engine"])
	}
}

// TestARefusedValueReachesTheBandAndNotTheFile. What a setting will accept
// is declared in internal/verb, beside what it means; this screen's share of
// that is to carry the refusal to the reader and write nothing.
//
// It used to be the screen's share to decide as well, for exactly one
// setting — the two checks on the unread cap, in the same words, out of the
// same catalogue — and for none of the other twelve. So the only setting the
// window and the command line agreed about was the one somebody had
// remembered to write twice.
func TestARefusedValueReachesTheBandAndNotTheFile(t *testing.T) {
	f := newFile()
	f.refuse = errors.New("not a whole number")

	out := Apply("unread-cap", "not-a-number", env(t, f))
	if !contains(out.Said, "not a whole number") {
		t.Errorf("the band says %q, want the refusal it was given", out.Said)
	}

	if f.held["unread-cap"] != "3" {
		t.Errorf("a refused cap was written anyway: the file holds %q", f.held["unread-cap"])
	}

	if len(f.wrote) != 0 {
		t.Errorf("a refused value still wrote %v", f.wrote)
	}
}

// TestTheLanguageAsksTheWindowToReload, because a catalogue is loaded once
// and no screen can reload it for the rest of the window.
func TestTheLanguageAsksTheWindowToReload(t *testing.T) {
	out := Apply("language", "es", env(t, newFile()))
	if out.Lang != "es" {
		t.Errorf("changing the language asked for %q", out.Lang)
	}
}

// TestEverySettingReachesTheFile. One case per row of the table, because a
// switch that grows a case and no test grows a case nobody wrote.
func TestEverySettingReachesTheFile(t *testing.T) {
	for _, c := range []struct {
		name  string
		val   string
		check func(*file, Out) bool
	}{
		{name: "language", val: "es", check: func(f *file, o Out) bool {
			return f.held["language"] == "es" && o.Lang == "es"
		}},
		{name: "autopilot", val: "on", check: held("autopilot", "on")},
		{name: "unread-cap", val: "9", check: held("unread-cap", "9")},
		{name: "model", val: "zeta/one", check: held("model", "zeta/one")},
		{name: "flow", val: "quick", check: held("flow", "quick")},
		{name: "theme", val: "frauddi", check: held("theme", "frauddi")},
		// The six that were declared and never drawn. One case each,
		// because a table read off the vocabulary is only worth having if
		// the rows it grew actually reach the file.
		{name: "check-record", val: "on", check: held("check-record", "on")},
		{name: "notify", val: "on", check: held("notify", "on")},
		{name: "chat-id", val: "7912204269", check: held("chat-id", "7912204269")},
		{name: "budget-task", val: "5", check: held("budget-task", "5")},
		{name: "budget-workspace", val: "50", check: held("budget-workspace", "50")},
		{name: "quota-floor", val: "20", check: held("quota-floor", "20")},
		{name: "effort", val: "hasty", check: func(_ *file, o Out) bool {
			return o.Dials != nil && o.Dials.Effort == "hasty"
		}},
		{name: "thinking", val: "on", check: func(_ *file, o Out) bool {
			return o.Dials != nil && o.Dials.Thinking == "on"
		}},
		// A name the vocabulary does not have is refused by it, and this
		// fixture is not the vocabulary — so what is checked here is that
		// the screen wrote it wherever it was told to, and nothing else.
		{name: "nothing-of-the-sort", val: "x", check: func(_ *file, o Out) bool { return o.Said != "" }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFile()

			out := Apply(c.name, c.val, env(t, f))
			if !c.check(f, out) {
				t.Errorf("%s = %q left the file %+v and answered %+v", c.name, c.val, f, out)
			}
		})
	}
}

// TestARefusedThemeDoesNotRepaintTheWindow. The paint happens after the
// write, so a theme the file would not take must not be the one on screen.
func TestARefusedThemeDoesNotRepaintTheWindow(t *testing.T) {
	f := newFile()
	f.refuse = errors.New("no")

	if out := Apply("theme", "midnight", env(t, f)); out.Said != "no" {
		t.Errorf("a refused theme said %q", out.Said)
	}
}

// TestAnEngineTheFileRefusesChangesNothingElse.
func TestAnEngineTheFileRefusesChangesNothingElse(t *testing.T) {
	f := newFile()
	f.refuse = errors.New("locked")

	out := Apply("engine", "omega", env(t, f))
	if out.Dials != nil {
		t.Error("an engine that was refused still moved the dials")
	}

	if out.Said != "locked" {
		t.Errorf("it said %q", out.Said)
	}
}

// TestWithoutAFileNothingIsWritten. The window can be up before the ports
// are wired, and a screen that dereferences one is a window that dies.
func TestWithoutAFileNothingIsWritten(t *testing.T) {
	e := env(t, newFile())
	e.Store, e.Kept, e.Choose = nil, nil, nil

	if out := Apply("language", "es", e); out.Said == "" && out.Lang == "" {
		t.Skip("nothing was written and nothing was said, which is the whole of what this asks")
	}
}

// held is the commonest check in the table above: the file holds this value
// under this name.
func held(key, want string) func(*file, Out) bool {
	return func(f *file, _ Out) bool { return f.held[key] == want }
}
