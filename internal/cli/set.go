package cli

// orbit set: the settings, as a table of what they are called, what they
// mean, and what may be written into them.
//
// One table and not a switch beside a list of names beside a paragraph of
// help. Every setting is one entry here, so a setting that is added is a
// setting `orbit set` prints, refuses wrong values for, and lists in its own
// refusal — rather than one that three of those four know about.

import (
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
type Setting struct {
	Name  string
	About func(*words.Printer) string
	Set   func(*store.Settings, string) (string, error)
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
		Set: func(cfg *store.Settings, value string) (string, error) {
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
		Set: func(cfg *store.Settings, value string) (string, error) {
			on, err := onOff(value)
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
		Set: func(cfg *store.Settings, value string) (string, error) {
			n, err := strconv.Atoi(value)
			if err != nil {
				return "", fmt.Errorf("unread-cap wants a whole number, not %q", value)
			}

			if n < 0 {
				return "", fmt.Errorf("unread-cap cannot be negative; zero is no cap at all")
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
		Set: func(cfg *store.Settings, value string) (string, error) {
			cfg.Engine = value
			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Engine },
	}, {
		Name: "model",
		About: func(p *words.Printer) string {
			return p.T("setting.model", "the model a phase asks for when it names none")
		},
		Set: func(cfg *store.Settings, value string) (string, error) {
			cfg.Model = value
			return value, nil
		},
		Value: func(cfg store.Settings) string { return cfg.Model },
	}, {
		Name:  "flow",
		About: func(p *words.Printer) string { return p.T("setting.flow", "the flow a new task is written against") },
		Set: func(cfg *store.Settings, value string) (string, error) {
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
		Set: func(cfg *store.Settings, value string) (string, error) {
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

	cfg, err := s.Settings()
	if err != nil {
		return err
	}
	// No argument at all is a question, not a mistake: it is the only way
	// to see what the settings are without opening the file, and a reader
	// who has forgotten a key's name is asking exactly this. One argument
	// is still a mistake — half a change is not a question.
	if fs.NArg() == 0 {
		printSettings(ctx, cfg)
		return nil
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("set needs a key and a value; the keys are %s", strings.Join(settingKeys(), ", "))
	}

	key, value := fs.Arg(0), fs.Arg(1)

	shown, err := assign(&cfg, key, value)
	if err != nil {
		return err
	}

	if err := s.SaveSettings(cfg); err != nil {
		return err
	}

	fmt.Fprintf(ctx.Out, "%s is now %s\n", key, shown)

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
func assign(cfg *store.Settings, key, value string) (string, error) {
	for _, s := range settingTable() {
		if s.Name == key {
			return s.Set(cfg, value)
		}
	}

	return "", fmt.Errorf("%q is not a setting; the keys are %s", key, strings.Join(settingKeys(), ", "))
}

// onOff reads a switch the way a person writes one.
//
// "on" and "off" first, because that is what the window's switch is labelled
// and a command line that disagreed with the screen would be two vocabularies
// for one setting. strconv.ParseBool after it, so true/false/1/0 — what
// anybody who has used a config file expects — are not refusals.
func onOff(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}

	on, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("autopilot is on or off, not %q", value)
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
