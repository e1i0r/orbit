package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
)

// set changes one line of the settings file.
//
// A verb rather than an editor, because the settings file is JSON and the
// two things a reader most wants to change — autopilot and the unread cap —
// are the two a stray comma would take out. Reading the file, changing one
// field and writing it back keeps every other setting exactly as it was,
// which hand-editing does not promise.
//
// There is no -repo flag: settings are the user's and not a repository's,
// and store.Settings reads them from the root of the state tree.
func set(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := parse(fs, args, out); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("set needs a key and a value; the keys are %s", strings.Join(settingKeys(), ", "))
	}
	key, value := fs.Arg(0), fs.Arg(1)

	s, err := store.Open()
	if err != nil {
		return err
	}
	cfg, err := s.Settings()
	if err != nil {
		return err
	}
	shown, err := assign(&cfg, key, value)
	if err != nil {
		return err
	}
	if err := s.SaveSettings(cfg); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s is now %s\n", key, shown)
	return nil
}

// settingKeys is every key set accepts, in the order a refusal lists them.
//
// It is a function and not a package variable for the reason view.Bands is:
// a slice at package scope is state a caller can reorder, and this package
// keeps none. assign below switches on the same strings, and
// TestEverySettingKeyCanBeSet is what keeps the two lists honest.
func settingKeys() []string {
	return []string{"language", "autopilot", "unread-cap", "engine", "model", "flow"}
}

// assign writes one value into the settings and gives back the form of it
// that is worth printing — which is not always what was typed: "1" for
// autopilot is stored as true and shown as "on", so the confirmation says
// what the file now holds rather than repeating the argument.
//
// A key nothing recognises is refused and named. That is the opposite of
// what task.take does with a control word it does not know, and the reason
// is the same asymmetry read the other way round: nothing is running, there
// is a person at the terminal to tell, and silently doing nothing to a
// setting somebody believes they changed is the worst of the three outcomes.
func assign(cfg *store.Settings, key, value string) (string, error) {
	switch key {
	case "language":
		// Not checked against the catalogues Orbit ships. words.For falls
		// back to English for a language it has no catalogue for, and an
		// overlay in $ORBIT_HOME/lang can add one after this line is
		// written — so refusing here would refuse a language that works.
		cfg.Language = value
		return value, nil
	case "autopilot":
		on, err := onOff(value)
		if err != nil {
			return "", err
		}
		cfg.Autopilot = on
		return offOn(on), nil
	case "unread-cap":
		n, err := strconv.Atoi(value)
		if err != nil {
			return "", fmt.Errorf("unread-cap wants a whole number, not %q", value)
		}
		if n < 0 {
			return "", fmt.Errorf("unread-cap cannot be negative; zero is no cap at all")
		}
		cfg.UnreadCap = n
		return value, nil
	case "engine":
		cfg.Engine = value
		return value, nil
	case "model":
		cfg.Model = value
		return value, nil
	case "flow":
		// Checked for being a name and not for naming anything: a file
		// dropped into $ORBIT_HOME/flows after this line is typed is a flow
		// that works, so refusing a name nothing answers to yet would
		// refuse a setting that is about to be right. What is refused is a
		// name that could never be a flow at all — one that is a path —
		// because that one is a typo in every possible future, and `orbit
		// flows` is the command that says which names there are.
		if err := flow.ValidName(value); err != nil {
			return "", err
		}
		cfg.Flow = value
		return value, nil
	default:
		return "", fmt.Errorf("%q is not a setting; the keys are %s", key, strings.Join(settingKeys(), ", "))
	}
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
