package ui

// The settings file every test in this package is handed, in memory.
//
// A file of its own because the 300-line ceiling is a real ceiling and
// because it is one thing: the window's settings port, answered out of
// internal/verb's own table rather than out of fields kept here. A fixture
// with a table of its own keeps passing while the screen it stands for has
// fallen behind — which is how six settings came to be declared and never
// drawn.

import (
	"strconv"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// settingsFile is the settings file in memory, with a way to make any write
// fail.
//
// It holds internal/verb's own table rather than a set of fields of its own.
// A fixture with its own table is one that keeps passing while the screen it
// stands for has fallen behind, and one that takes values the real settings
// would refuse — both of which had already happened by the time this was
// written.
type settingsFile struct {
	tbl  *verb.Table
	fail error
}

// newSettingsFile is the table as Orbit ships it.
func newSettingsFile() *settingsFile { return &settingsFile{tbl: verb.NewTable()} }

// settingsWith is the three a test usually cares about, written through the
// real table so that what a fixture holds is what the file would have held.
func settingsWith(autopilot bool, lang string, unread int) *settingsFile {
	s := newSettingsFile()
	for key, val := range map[string]string{
		"autopilot":  offOn(autopilot),
		"language":   lang,
		"unread-cap": strconv.Itoa(unread),
	} {
		if _, err := s.tbl.Choose(words.For("en"), key, val); err != nil {
			panic(err)
		}
	}

	return s
}

// settingsFailing is a file that refuses every write, which is how a test
// reaches the sentence a locked settings file answers with.
func settingsFailing(why error) *settingsFile {
	s := newSettingsFile()
	s.fail = why

	return s
}

// table is the one held, built on first use so that the zero value of this
// struct is still a working port — several tests write &settingsFile{} and
// one of them only wants the refusal.
func (s *settingsFile) table() *verb.Table {
	if s.tbl == nil {
		s.tbl = verb.NewTable()
	}

	return s.tbl
}

// Kept is every setting and what this file holds for it, and Choose writes
// one by name through the validator declared beside it.
func (s *settingsFile) Kept(p *words.Printer) []verb.Setting { return s.table().Kept(p) }

func (s *settingsFile) Choose(p *words.Printer, key, value string) (string, error) {
	if s.fail != nil {
		return "", s.fail
	}

	return s.table().Choose(p, key, value)
}

// Fresh is what internal/verb declares a setting comes as, which is what the
// real adapter answers: the fixture goes through the same table rather than
// keeping a second set of defaults nobody would update.
func (s *settingsFile) Fresh(key string) string { return s.table().Fresh(key) }

// The typed getters the header and the board ask, read off the same table.
// Nothing here keeps a second copy of a value, so a write through Choose is
// visible to every one of them at once — which is what the real adapter
// does, and what a fixture with its own fields could not.
func (s *settingsFile) Autopilot() bool  { return s.table().Value("autopilot") == "on" }
func (s *settingsFile) Language() string { return s.table().Value("language") }
func (s *settingsFile) UnreadCap() int   { return number(s.table().Value("unread-cap")) }
func (s *settingsFile) Engine() string   { return s.table().Value("engine") }
func (s *settingsFile) Model() string    { return s.table().Value("model") }
func (s *settingsFile) Flow() string     { return s.table().Value("flow") }
func (s *settingsFile) Theme() string    { return s.table().Value("theme") }

func (s *settingsFile) BudgetWorkspace() float64 { return money(s.table().Value("budget-workspace")) }
func (s *settingsFile) QuotaFloor() int          { return number(s.table().Value("quota-floor")) }

// The setters, which every one of them can refuse: the settings file has a
// lock, and a second orbit holding it makes any of these say so after
// waiting two seconds.
func (s *settingsFile) SetAutopilot(v bool) error  { return s.set("autopilot", offOn(v)) }
func (s *settingsFile) SetLanguage(v string) error { return s.set("language", v) }
func (s *settingsFile) SetUnreadCap(v int) error   { return s.set("unread-cap", strconv.Itoa(v)) }
func (s *settingsFile) SetEngine(v string) error   { return s.set("engine", v) }
func (s *settingsFile) SetModel(v string) error    { return s.set("model", v) }
func (s *settingsFile) SetFlow(v string) error     { return s.set("flow", v) }
func (s *settingsFile) SetTheme(v string) error    { return s.set("theme", v) }

// set is the one path every typed setter takes.
func (s *settingsFile) set(key, value string) error {
	_, err := s.Choose(words.For("en"), key, value)

	return err
}

// flip puts the autopilot switch where a test needs it, through the same
// door everything else writes through.
func (s *settingsFile) flip(t *testing.T, on bool) {
	t.Helper()

	if err := s.SetAutopilot(on); err != nil {
		t.Fatalf("the fixture refused the autopilot switch: %v", err)
	}
}

// put writes one setting by name, for a test that wants a value the typed
// setters do not reach: the budget and the quota floor are chosen with
// `orbit set` and read by the window, so the port has no writer for either.
func (s *settingsFile) put(t *testing.T, key, value string) {
	t.Helper()

	if _, err := s.Choose(words.For("en"), key, value); err != nil {
		t.Fatalf("the fixture refused %s = %s: %v", key, value, err)
	}
}

// offOn is a switch as the settings table spells one.
func offOn(on bool) string {
	if on {
		return "on"
	}

	return "off"
}

// number and money read a value back out of the table, answering zero for
// what will not parse — which cannot happen, because the table only ever
// holds what its own validators wrote.
func number(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}

	return n
}

func money(v string) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}

	return f
}
