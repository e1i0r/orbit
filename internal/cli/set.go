package cli

// orbit set: the settings, as a table of what they are called, what they
// mean, and what may be written into them.
//
// One table and not a switch beside a list of names beside a paragraph of
// help. Every setting is one entry here, so a setting that is added is a
// setting `orbit set` prints, refuses wrong values for, and lists in its own
// refusal — rather than one that three of those four know about.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// Setting is one line of the settings file.
//
// Set is the validator and the assignment together, because they are one
// decision: what a setting will accept is what it means, and a validator
// that lived apart from the field it guards is a validator that will one day
// guard a field nothing writes. It answers with the form worth printing,
// which is not always what was typed — "1" for autopilot is stored as true
// and shown as "on", so the confirmation says what the file now holds rather
// than repeating the argument.
//
// It is handed the printer for the same reason About is: a refusal is a
// sentence a reader reads, and the sentence a setting refuses with belongs
// beside the rule it enforces. Most settings take anything and pass it
// unread, which is why most of these closures never look at it.
type Setting struct {
	Name  string
	About func(*words.Printer) string
	Set   func(*words.Printer, *store.Settings, string) (string, error)
	Value func(store.Settings) string
}

// settingTable is every setting there is, in the order a refusal lists them.
//
// It is a function and not a package variable for the reason view.Bands is:
// a slice at package scope is state a caller can reorder, and this package
// keeps none.
func settingTable() []Setting {
	return []Setting{{
		Name:  "language",
		About: func(p *words.Printer) string { return p.T("setting.language", "the language orbit speaks") },
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			// Not checked against the catalogues Orbit ships. words.For
			// falls back to English for a language it has no catalogue for,
			// and an overlay in $ORBIT_HOME/lang can add one after this
			// line is written — so refusing here would refuse a language
			// that works.
			cfg.Language = value
			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Language },
	}, {
		Name: "autopilot",
		About: func(p *words.Printer) string {
			return p.T("setting.autopilot", "whether a run walks its whole flow without stopping")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			on, err := onOff(p, value)
			if err != nil {
				return "", err
			}

			cfg.Autopilot = on

			return offOn(on), nil
		},
		Value: func(cfg store.Settings) string { return offOn(cfg.Autopilot) },
	}, {
		Name: "unread-cap",
		About: func(p *words.Printer) string {
			return p.T("setting.unread_cap", "how many finished tasks may sit unread before nothing new starts")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			n, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New(p.T("settings.not_a_number", "{val} is not a whole number",
					words.Arg{Name: "val", Value: value}))
			}

			if n < 0 {
				return "", errors.New(p.T("settings.negative_cap",
					"the unread cap cannot be negative; zero is no cap at all"))
			}

			cfg.UnreadCap = n

			return value, nil
		},
		Value: func(cfg store.Settings) string { return strconv.Itoa(cfg.UnreadCap) },
	}, {
		Name: "engine",
		About: func(p *words.Printer) string {
			return p.T("setting.engine", "the engine a task runs on when it names none")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			cfg.Engine = value
			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Engine },
	}, {
		Name: "model",
		About: func(p *words.Printer) string {
			return p.T("setting.model", "the model a phase asks for when it names none")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			cfg.Model = value
			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Model },
	}, {
		Name:  "flow",
		About: func(p *words.Printer) string { return p.T("setting.flow", "the flow a new task is written against") },
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			// Checked for being a name and not for naming anything: a file
			// dropped into $ORBIT_HOME/flows after this line is typed is a
			// flow that works, so refusing a name nothing answers to yet
			// would refuse a setting that is about to be right. What is
			// refused is a name that could never be a flow at all — one
			// that is a path — because that one is a typo in every possible
			// future, and `orbit flows` is the command that says which
			// names there are.
			if err := flow.ValidName(value); err != nil {
				return "", err
			}

			cfg.Flow = value

			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Flow },
	}, {
		Name:  "theme",
		About: func(p *words.Printer) string { return p.T("setting.theme", "the visual color theme for the window") },
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			cfg.Theme = value
			return value, nil
		},
		Value: func(cfg store.Settings) string {
			if cfg.Theme == "" {
				return "monokai"
			}

			return cfg.Theme
		},
	}}
}

// settingKeys is every key set accepts, in the order a refusal lists them.
func settingKeys() []string {
	out := make([]string, 0, len(settingTable()))
	for _, s := range settingTable() {
		out = append(out, s.Name)
	}

	return out
}

// set changes one line of the settings file, or prints them all.
//
// A verb rather than an editor, because the settings file is JSON and the
// two things a reader most wants to change — autopilot and the unread cap —
// are the two a stray comma would take out. Reading the file, changing one
// field and writing it back keeps every other setting exactly as it was,
// which hand-editing does not promise.
//
// There is no -repo flag: settings are the user's and not a repository's,
// and store.Settings reads them from the root of the state tree.
func set(ctx Context, args []string) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		return err
	}

	// No argument at all is a question, not a mistake: it is the only way
	// to see what the settings are without opening the file, and a reader
	// who has forgotten a key's name is asking exactly this. One argument
	// is still a mistake — half a change is not a question.
	if fs.NArg() == 0 {
		cfg, err := s.Settings()
		if err != nil {
			return err
		}

		printSettings(ctx, cfg)

		return nil
	}

	if fs.NArg() < 2 {
		return errors.New(ctx.printer().T("set.needs_key_and_value",
			"set needs a key and a value; the keys are {keys}",
			words.Arg{Name: "keys", Value: strings.Join(settingKeys(), ", ")}))
	}

	key, value := fs.Arg(0), fs.Arg(1)
	// The read, the change and the write are one step. Between them, the
	// window is another process writing the whole of this file back from a
	// copy it read before this line ran: two settings changed at once would
	// be one setting changed and one silently discarded.
	var shown string

	err = s.UpdateSettings(func(cfg *store.Settings) error {
		var err error

		shown, err = assign(ctx.printer(), cfg, key, value)

		return err
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("set.now", "{key} is now {value}",
		words.Arg{Name: "key", Value: key}, words.Arg{Name: "value", Value: shown}))

	return nil
}

// printSettings is every setting, what it is now, and what it means.
func printSettings(ctx Context, cfg store.Settings) {
	p := ctx.printer()

	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)
	for _, s := range settingTable() {
		fmt.Fprintf(w, "  %s\t%s\t%s\n", s.Name, unset(s.Value(cfg)), s.About(p))
	}

	_ = w.Flush() // the writer under it is the one Run was handed
}

// unset is what a setting nobody has chosen prints as. A blank column reads
// as a table that failed to render; a dash reads as an answer.
func unset(value string) string {
	if value == "" {
		return "—"
	}

	return value
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

// onOff reads a switch the way a person writes one.
//
// "on" and "off" first, because that is what the window's switch is labelled
// and a command line that disagreed with the screen would be two vocabularies
// for one setting. strconv.ParseBool after it, so true/false/1/0 — what
// anybody who has used a config file expects — are not refusals.
func onOff(p *words.Printer, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}

	on, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New(p.T("set.autopilot_on_or_off", "autopilot is on or off, not {value}",
			words.Arg{Name: "value", Value: value}))
	}

	return on, nil
}

// offOn is the word the confirmation prints for a switch.
func offOn(on bool) string {
	if on {
		return "on"
	}

	return "off"
}
