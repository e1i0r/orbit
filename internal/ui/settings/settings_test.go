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

// file is the settings file, in memory, with a way to make any write fail.
type file struct {
	lang      string
	autopilot bool
	unread    int
	engine    string
	model     string
	flow      string
	theme     string
	refuse    error
}

func (f *file) Language() string { return f.lang }
func (f *file) Autopilot() bool  { return f.autopilot }
func (f *file) UnreadCap() int   { return f.unread }
func (f *file) Engine() string   { return f.engine }
func (f *file) Model() string    { return f.model }
func (f *file) Flow() string     { return f.flow }
func (f *file) Theme() string    { return f.theme }

func (f *file) SetLanguage(v string) error { f.lang = v; return f.refuse }
func (f *file) SetAutopilot(v bool) error  { f.autopilot = v; return f.refuse }
func (f *file) SetUnreadCap(v int) error   { f.unread = v; return f.refuse }
func (f *file) SetEngine(v string) error   { f.engine = v; return f.refuse }
func (f *file) SetModel(v string) error    { f.model = v; return f.refuse }
func (f *file) SetFlow(v string) error     { f.flow = v; return f.refuse }
func (f *file) SetTheme(v string) error    { f.theme = v; return f.refuse }

// env is the world this screen is given in these tests: one engine with two
// models and two efforts, and a settings file nobody else is writing to.
func env(t *testing.T, f *file) Env {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	return Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		Store:   f,
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

// newFile is a settings file with something in every field, so that a test
// that changes one can tell it apart from a zero value.
func newFile() *file {
	return &file{lang: "en", unread: 3, engine: "zeta", model: "zeta/one", flow: "cover", theme: "frauddi"}
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
	f.autopilot = true

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
	e.Store = nil

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
