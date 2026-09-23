package verb

// The settings: the family that reads them and writes one, and the table of
// what they are called, what they mean, and what may be written into them.
//
// One table and not a switch beside a list of names beside a paragraph of
// help. Every setting is one entry here, so a setting that is added is a
// setting `orbit settings set` prints, refuses wrong values for, and lists
// in its own refusal — rather than one that three of those four know about.

import (
	"errors"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// settings is the family, parent first. The parent keeps its own body: it
// is the reading, and `orbit settings` has printed them for as long as there
// have been any.
func settings() []Verb {
	return []Verb{
		{
			Name: "settings", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.settings", "every setting and what it is set to")
			},
		},
		{
			Name: "clear", Under: "settings",
			About: func(p *words.Printer) string {
				return p.T("verb.clear", "put one setting back to what Orbit ships")
			},
			Takes: []Field{
				{Name: "key", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.clear.key", "which setting")
				}},
			},
		},
		{
			Name: "set", Under: "settings",
			About: func(p *words.Printer) string {
				return p.T("verb.set", "change one of Orbit's own settings")
			},
			Takes: []Field{
				{Name: "key", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.set.key", "which setting")
				}},
				{Name: "value", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.set.value", "what to set it to")
				}},
			},
		},
	}
}

// Setting is one setting as a reader sees it: what it is called, what it
// holds now, and what it means. It is what the settings reading answers
// with, and carries none of the closures below — a surface is shown the
// values, not the validators.
type Setting struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	About string `json:"about"`
	Group string `json:"group"` // the heading it is listed under: settings_groups.go
}

// Rule is one line of the settings file.
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
type Rule struct {
	Name  string
	About func(*words.Printer) string
	Set   func(*words.Printer, *store.Settings, string) (string, error)
	Value func(store.Settings) string
	// Clear puts this one setting back to what Orbit ships, and it is here
	// beside Set for the reason Set is here beside About: what a setting
	// means, what it accepts and what it comes as are one decision, and a
	// list of defaults kept somewhere else is the list that goes stale.
	//
	// It takes no value and answers nothing. What the setting reads as
	// afterwards is Value's to say, which is what keeps clearing and
	// listing from ever disagreeing about the same field.
	//
	// It writes the shipped value rather than taking the line out of the
	// file, and the difference is worth knowing: every field is omitempty,
	// so a zero written on purpose and a field never written are the same
	// bytes on disk. There is no line to remove that would mean anything.
	Clear func(*store.Settings)
}

// settingTable is every setting there is, in the order a refusal lists them.
//
// It is a function and not a package variable for the reason view.Bands is:
// a slice at package scope is state a caller can reorder, and this package
// keeps none.
func settingTable() []Rule {
	return append([]Rule{{
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
		Clear: func(cfg *store.Settings) { cfg.Language = store.Shipped().Language },
	}, {
		Name: "autopilot",
		About: func(p *words.Printer) string {
			return p.T("setting.autopilot", "whether a run walks its whole flow without stopping")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			on, err := onOff(p, "autopilot", value)
			if err != nil {
				return "", err
			}

			cfg.Autopilot = on

			return offOn(on), nil
		},
		Value: func(cfg store.Settings) string { return offOn(cfg.Autopilot) },
		Clear: func(cfg *store.Settings) { cfg.Autopilot = store.Shipped().Autopilot },
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
		Clear: func(cfg *store.Settings) { cfg.UnreadCap = store.Shipped().UnreadCap },
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
		Clear: func(cfg *store.Settings) { cfg.Engine = store.Shipped().Engine },
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
		Clear: func(cfg *store.Settings) { cfg.Model = store.Shipped().Model },
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
		Clear: func(cfg *store.Settings) { cfg.Flow = store.Shipped().Flow },
	}, {
		Name: "check-record",
		About: func(p *words.Printer) string {
			return p.T("setting.check_record", "whether every command asks SQLite if the record is still readable")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			on, err := onOff(p, "check-record", value)
			if err != nil {
				return "", err
			}

			cfg.CheckRecord = on

			return offOn(on), nil
		},
		Value: func(cfg store.Settings) string { return offOn(cfg.CheckRecord) },
		Clear: func(cfg *store.Settings) { cfg.CheckRecord = store.Shipped().CheckRecord },
	}, {
		Name:  "theme",
		About: func(p *words.Printer) string { return p.T("setting.theme", "the visual color theme for the window") },
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			cfg.Theme = value
			return value, nil
		},
		Value: func(cfg store.Settings) string {
			if cfg.Theme == "" {
				// The one the window will actually draw. Spelled here as a
				// second copy of the word, this table printed monokai for a
				// cockpit drawing frauddi.
				return theme.DefaultTheme
			}

			return cfg.Theme
		},
		Clear: func(cfg *store.Settings) { cfg.Theme = store.Shipped().Theme },
	}, {
		Name: "notify",
		About: func(p *words.Printer) string {
			return p.T("setting.notify", "whether Orbit interrupts you when a run stops and needs somebody")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			on, err := onOff(p, "notify", value)
			if err != nil {
				return "", err
			}

			cfg.Notify = on

			return offOn(on), nil
		},
		Value: func(cfg store.Settings) string { return offOn(cfg.Notify) },
		Clear: func(cfg *store.Settings) { cfg.Notify = store.Shipped().Notify },
	}, {
		Name: "chat-id",
		About: func(p *words.Printer) string {
			return p.T("setting.chat_id", "the one account `orbit chat` answers over a service")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			cfg.ChatID = strings.TrimSpace(value)
			return cfg.ChatID, nil
		},
		Value: func(cfg store.Settings) string { return cfg.ChatID },
		Clear: func(cfg *store.Settings) { cfg.ChatID = store.Shipped().ChatID },
	}}, append(budgetSettings(), queueSettings()...)...)
}

// onOff reads a switch the way a person writes one.
//
// "on" and "off" first, because that is what the window's switch is labelled
// and a command line that disagreed with the screen would be two vocabularies
// for one setting. strconv.ParseBool after it, so true/false/1/0 — what
// anybody who has used a config file expects — are not refusals.
func onOff(p *words.Printer, name, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}

	on, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New(p.T("set.on_or_off", "{setting} is on or off, not {value}",
			words.Arg{Name: "setting", Value: name},
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
