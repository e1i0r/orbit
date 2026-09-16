package settings

// The screen's own suite: what the table says, what a key does to it, and
// what a change writes. It never builds a window — everything this screen
// needs arrives in an Env, which is the whole point of it being a package.

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/words"
)

// file is the settings file as this screen sees one: a name to a value, and
// a way to make any write fail.
//
// A map and not a struct of typed fields, because that is the shape the
// screen now works in — it is handed a table and hands one value back, and
// knows nothing about which field of which struct either ends up in. The
// real validators live in internal/verb, which this package may not name;
// what is checked here is that the screen carries a refusal to the band, not
// which values are refused.
type file struct {
	held   map[string]string
	wrote  []string
	refuse error
}

// kept is the table this file answers with, in the order the vocabulary
// declares it.
//
// The order is written out because a fixture cannot ask internal/verb for
// it, and it matters: the screen splices the two window-kept dials in after
// the model, so a table in another order is a different screen.
var kept = []struct{ name, about string }{
	{"language", "the language orbit speaks"},
	{"autopilot", "whether a run walks its whole flow without stopping"},
	{"unread-cap", "how many finished tasks may sit unread before nothing new starts"},
	{"engine", "the engine a task runs on when it names none"},
	{"model", "the model a phase asks for when it names none"},
	{"flow", "the flow a new task is written against"},
	{"check-record", "whether every command asks SQLite if the record is still readable"},
	{"theme", "the visual color theme for the window"},
	{"notify", "whether Orbit interrupts you when a run stops and needs somebody"},
	{"chat-id", "the one account `orbit chat` answers over a service"},
	{"budget-task", "the most one task may spend in dollars; 0 is no budget"},
	{"budget-workspace", "the most the board may have spent before nothing new starts on its own"},
	{"quota-floor", "how much of an engine's window must be left for the queue to go on"},
}

// Kept is every setting and what this file holds for it.
func (f *file) Kept() []Kept {
	out := make([]Kept, 0, len(kept))
	for _, one := range kept {
		out = append(out, Kept{Name: one.name, Value: f.held[one.name], About: one.about})
	}

	return out
}

// Choose writes one down, or refuses everything.
func (f *file) Choose(key, value string) error {
	if f.refuse != nil {
		return f.refuse
	}

	f.held[key] = value
	f.wrote = append(f.wrote, key+"="+value)

	return nil
}

// Fresh is what a setting comes as, written out here rather than asked of
// internal/verb — this package may not see it, and that is the point of the
// port. What matters to a test of clearing is that the screen writes back
// whatever it is told, not which values Orbit happens to ship.
func (f *file) Fresh(key string) string {
	return map[string]string{
		"autopilot":  "off",
		"unread-cap": "5",
		"flow":       "task",
		"theme":      "frauddi",
	}[key]
}

// env is the world this screen is given in these tests: one engine with two
// models and two efforts, and a settings file nobody else is writing to.
func env(t *testing.T, f *file) Env {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	return Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		Store:   f,
		Kept:    f.Kept,
		Choose:  f.Choose,
		Dials:   Dials{Engine: "zeta", Effort: "brisk", Thinking: "adaptive"},
		Engines: func() []string { return []string{"zeta", "omega"} },
		Models: func(string) (ids, labels []string) {
			return []string{"", "zeta/one"}, []string{"default", "one"}
		},
		Efforts: func(engine string) (ids, labels []string) {
			if engine == "omega" {
				return []string{"slow"}, []string{"slow"}
			}

			return []string{"brisk", "hasty"}, []string{"brisk", "hasty"}
		},
		Flows: func() []string { return []string{"cover", "quick"} },
	}
}

// newFile is a settings file with something in every row, so that a test
// that changes one can tell it apart from a zero value.
func newFile() *file {
	return &file{held: map[string]string{
		"language": "en", "autopilot": "off", "unread-cap": "3",
		"engine": "zeta", "model": "zeta/one", "flow": "cover",
		"check-record": "off", "theme": "frauddi", "notify": "off",
		"chat-id": "", "budget-task": "0", "budget-workspace": "0", "quota-floor": "0",
	}}
}

// TestOpenReadsTheFlowsOnce. The dial is asked for when the screen comes up,
// not while it draws: Rows runs on every keystroke and every mouse move, and
// a directory read in there is one per frame.
func TestOpenReadsTheFlowsOnce(t *testing.T) {
	reads := 0
	e := env(t, newFile())
	e.Flows = func() []string {
		reads++

		return []string{"cover"}
	}

	s := Open(e)

	for range 5 {
		s.Rows(e)
	}

	if reads != 1 {
		t.Errorf("the flows were read %d times, want once", reads)
	}
}

// TestTheTableIsTheFileAndTheBuild. Nothing on this screen is written here:
// the engines and their models are the build's, the flows are the reader's,
// and the values are the file's.
func TestTheTableIsTheFileAndTheBuild(t *testing.T) {
	f := newFile()
	f.held["autopilot"] = "on"

	e := env(t, f)
	rows := map[string]Row{}

	for _, r := range Open(e).Rows(e) {
		rows[r.Key] = r
	}

	for _, c := range []struct{ key, want string }{
		{key: "language", want: "en"},
		{key: "autopilot", want: "on"},
		{key: "unread-cap", want: "3"},
		{key: "engine", want: "zeta"},
		{key: "model", want: "zeta/one"},
		{key: "effort", want: "brisk"},
		{key: "thinking", want: "adaptive"},
		{key: "flow", want: "cover"},
	} {
		if got := rows[c.key].Val; got != c.want {
			t.Errorf("the %s row reads %q, want %q", c.key, got, c.want)
		}
	}

	if got := rows["flow"].Options; len(got) != 2 || got[0] != "cover" {
		t.Errorf("the flow dial offers %v, want what the reader has", got)
	}

	if got := rows["model"].Label(0); got != "default" {
		t.Errorf("the model dial draws %q for the empty id, want its label", got)
	}
}

// TestAValueTheBuildDoesNotOfferFallsToTheFirst. An effort left over from
// another engine would otherwise be drawn as chosen while no pill is lit.
func TestAValueTheBuildDoesNotOfferFallsToTheFirst(t *testing.T) {
	e := env(t, newFile())
	e.Dials.Effort = "xhigh"

	for _, r := range Open(e).Rows(e) {
		if r.Key == "effort" && r.Val != "brisk" {
			t.Errorf("the effort row reads %q, want the first the engine offers", r.Val)
		}
	}
}

// TestWithoutAFileThereIsNoTable, and the only key that answers is the one
// that leaves: a screen with nothing on it must still let go of the keyboard.
func TestWithoutAFileThereIsNoTable(t *testing.T) {
	e := env(t, nil)
	e.Store, e.Kept, e.Choose = nil, nil, nil

	s := Open(e)
	if rows := s.Rows(e); rows != nil {
		t.Errorf("a screen with no settings file drew %d rows", len(rows))
	}

	if _, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e); !out.Close {
		t.Error("esc did not leave a screen with nothing on it")
	}
}

// TestARefusedWriteSaysSoInsteadOfSayingItIsSet. The value is already drawn
// by the time the write runs, so a failure that says nothing is a window
// showing a setting that is not in the file.
func TestARefusedWriteSaysSoInsteadOfSayingItIsSet(t *testing.T) {
	f := newFile()
	f.refuse = errors.New("the settings file is locked by another orbit")

	out := Apply("language", "es", env(t, f))
	if out.Said != "the settings file is locked by another orbit" {
		t.Errorf("a refused write said %q", out.Said)
	}

	if out.Lang != "" {
		t.Error("a refused language change still asked the window to reload its words")
	}
}
