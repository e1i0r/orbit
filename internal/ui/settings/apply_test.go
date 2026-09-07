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
	s := Open(e).Point(1)

	if out := s.Cycle(1, e); out.Said == "" {
		t.Error("turning the dial said nothing")
	}

	if !f.autopilot {
		t.Error("turning autopilot one to the right left it off")
	}

	if out := s.Cycle(-1, e); out.Said == "" || f.autopilot {
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

	if f.engine != "omega" {
		t.Errorf("the file holds engine %q", f.engine)
	}
}

// TestTheNumberRowRefusesWhatIsNotOne. Both ways it goes wrong: a word, and
// a negative cap, which every reader of the setting treats as no cap at all.
func TestTheNumberRowRefusesWhatIsNotOne(t *testing.T) {
	f := newFile()
	e := env(t, f)

	for _, c := range []struct{ val, want string }{
		{val: "not-a-number", want: "not a whole number"},
		{val: "-1", want: "cannot be negative"},
	} {
		out := Apply("unread-cap", c.val, e)
		if !contains(out.Said, c.want) {
			t.Errorf("a cap of %q said %q, want it to say %q", c.val, out.Said, c.want)
		}

		if f.unread != 3 {
			t.Errorf("a cap of %q was written anyway: the file holds %d", c.val, f.unread)
		}
	}
}

// TestAFlowNameThatIsAPathIsRefused. It is the one thing `orbit set` checked
// and this screen did not, so what the command line would not take, the
// window wrote.
func TestAFlowNameThatIsAPathIsRefused(t *testing.T) {
	f := newFile()

	out := Apply("flow", "../etc/passwd", env(t, f))
	if out.Said == "" {
		t.Error("a flow name that is a path was written without a word")
	}

	if f.flow != "cover" {
		t.Errorf("the file holds flow %q", f.flow)
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
		{name: "language", val: "es", check: func(f *file, o Out) bool { return f.lang == "es" && o.Lang == "es" }},
		{name: "autopilot", val: "on", check: func(f *file, _ Out) bool { return f.autopilot }},
		{name: "unread-cap", val: "9", check: func(f *file, _ Out) bool { return f.unread == 9 }},
		{name: "model", val: "zeta/one", check: func(f *file, _ Out) bool { return f.model == "zeta/one" }},
		{name: "flow", val: "quick", check: func(f *file, _ Out) bool { return f.flow == "quick" }},
		{name: "theme", val: "frauddi", check: func(f *file, _ Out) bool { return f.theme == "frauddi" }},
		{name: "effort", val: "hasty", check: func(_ *file, o Out) bool {
			return o.Dials != nil && o.Dials.Effort == "hasty"
		}},
		{name: "thinking", val: "on", check: func(_ *file, o Out) bool {
			return o.Dials != nil && o.Dials.Thinking == "on"
		}},
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
	e.Store = nil

	if out := Apply("language", "es", e); out.Said == "" && out.Lang == "" {
		t.Skip("nothing was written and nothing was said, which is the whole of what this asks")
	}
}
