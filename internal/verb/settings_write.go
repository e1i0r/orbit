package verb

// Writing one setting, and putting one back.
//
// Apart from settings.go because that file is the table — what every setting
// is called, what it means, what it accepts — and this is what happens when
// somebody names one. The two were one file until the table outgrew it.

import (
	"errors"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// Fresh is what one setting reads as when nobody has chosen anything, and
// nothing for a name this table does not have.
//
// It is the window's way of asking the same question `settings clear`
// answers, so that the screen and the command line cannot disagree about
// what a setting comes as. The screen writes the value back through the
// setter it already has rather than through a door of its own: putting a
// setting back is choosing the value it shipped with, and everything that
// happens when somebody chooses one — the theme repainting, the catalogue
// reloading — has to happen then too.
func Fresh(key string) string {
	shipped := store.Shipped()

	for _, s := range settingTable() {
		if s.Name == key {
			return s.Value(shipped)
		}
	}

	return ""
}

// Kept is every setting there is, as a surface shows one: what it is called,
// what it holds now, and what it means.
//
// It is a door because the window draws the same table the terminal prints,
// and used to draw it from a list written out by hand. Six settings were
// added to this package and to no screen — among them whether Orbit is
// allowed to interrupt you, and which account may command it over a chat —
// so the window quietly showed seven of thirteen. A table read from the
// declaration cannot fall behind it.
//
// It carries none of the closures: a surface is shown the values, not the
// validators. Writing goes back through Choose, which is where they live.
func Kept(p *words.Printer, cfg store.Settings) []Setting {
	all := make([]Setting, 0, len(settingTable()))
	for _, one := range settingTable() {
		all = append(all, Setting{Name: one.Name, Value: one.Value(cfg), About: one.About(p)})
	}

	return grouped(p, all)
}

// Shipped is every setting at the value Orbit ships it with, which is the
// table before anybody has chosen anything.
//
// A door because a surface that has to stand in for the settings file — the
// window's own fixtures, a preview — should stand in for the real table and
// not a list of rows somebody typed out. A fixture with its own table is a
// fixture that passes while the screen it stands for has gone stale, which
// is the exact failure this pair of doors exists to end.
func Shipped(p *words.Printer) []Setting { return Kept(p, store.Shipped()) }

// Choose writes one setting by name, through the validator declared beside
// it, and answers the form of the value worth showing.
//
// The window had its own copy of one of those validators — the same two
// checks on the unread cap, in the same words, out of the same catalogue —
// and none of the other twelve. So a number the command line refused, the
// window wrote, and the only setting both agreed about was the one somebody
// had remembered to write twice.
func Choose(p *words.Printer, cfg *store.Settings, key, value string) (string, error) {
	return assign(p, cfg, key, value)
}

// settingKeys is every key set accepts, in the order a refusal lists them.
func settingKeys() []string {
	out := make([]string, 0, len(settingTable()))
	for _, s := range settingTable() {
		out = append(out, s.Name)
	}

	return out
}

// assign writes one value into the settings and gives back the form of it
// that is worth printing.
//
// A key nothing recognises is refused and named. That is the opposite of
// what task.take does with a control word it does not know, and the reason
// is the same asymmetry read the other way round: nothing is running, there
// is a person at the terminal to tell, and silently doing nothing to a
// setting somebody believes they changed is the worst of the three outcomes.
func assign(p *words.Printer, cfg *store.Settings, key, value string) (string, error) {
	for _, s := range settingTable() {
		if s.Name == key {
			return s.Set(p, cfg, value)
		}
	}

	return "", errors.New(p.T("set.no_such_setting", "{key} is not a setting; the keys are {keys}",
		words.Arg{Name: "key", Value: key}, words.Arg{Name: "keys", Value: strings.Join(settingKeys(), ", ")}))
}

// blank puts one setting back to what Orbit ships and says what it reads as
// now, alongside what it read as before.
//
// Both, because the two together are the whole answer and either alone is
// half of one. "unread-cap is back to 5" is what somebody wanted to hear;
// "it was already 5" is what stops them wondering whether the command did
// anything.
//
// A key nothing recognises is refused and named, for the reason assign
// refuses one: nothing is running, there is a person at the terminal, and
// silently doing nothing to a setting somebody believes they cleared is the
// worst of the three outcomes.
func blank(p *words.Printer, cfg *store.Settings, key string) (was, now string, err error) {
	for _, s := range settingTable() {
		if s.Name != key {
			continue
		}

		was = s.Value(*cfg)
		s.Clear(cfg)

		return was, s.Value(*cfg), nil
	}

	return "", "", errors.New(p.T("set.no_such_setting", "{key} is not a setting; the keys are {keys}",
		words.Arg{Name: "key", Value: key}, words.Arg{Name: "keys", Value: strings.Join(settingKeys(), ", ")}))
}
