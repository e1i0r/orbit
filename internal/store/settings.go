package store

// Settings is the user's persisted configuration, one JSON file at the root
// of the state tree. Every field's Go zero value is a working setting in
// its own right — UnreadCap: 0 means no cap — so a settings file that will
// not parse yields those defaults rather than failing, for the same reason
// a broken translation catalogue yields English: a reader that cannot read
// still has to hand back something the rest of the program can use.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultUnreadCap is what a store that has never saved settings starts
// with. It is deliberately not the zero value: UnreadCap: 0 means "no cap",
// and that has to stay something a user chooses, never a fact about never
// having chosen at all.
const defaultUnreadCap = 5

// defaultFlow is which pipeline a task is written against when the user has
// never chosen one. The word is also written down in internal/flow, as
// flow.Default, because this package imports nothing of Orbit's — that
// absence is what keeps the on-disk layout from depending on anything above
// it. Two copies of a word drift, so a test in internal/task, which imports
// both, is what holds them together.
const defaultFlow = "task"

// Settings is the user's persisted configuration.
type Settings struct {
	Language  string `json:"language,omitempty"`
	Autopilot bool   `json:"autopilot,omitempty"`
	UnreadCap int    `json:"unreadCap,omitempty"`
	Engine    string `json:"engine,omitempty"`
	Model     string `json:"model,omitempty"`
	Flow      string `json:"flow,omitempty"`
	Theme     string `json:"theme,omitempty"`

	// Notify is whether Orbit interrupts you when a run stops and needs
	// somebody.
	//
	// Off by default, because a program that starts interrupting a person
	// the day they install it has made a decision that was theirs. On, it
	// reaches whatever is configured: the desktop, and a chat if one is set
	// up. One switch and not one per channel — the question is whether you
	// want to be told, and where is a fact about what you have set up.
	Notify bool `json:"notify,omitempty"`

	// ChatID is the one account `orbit chat` answers over a service, as
	// that service numbers its conversations.
	//
	// One and not a list, because this is the whole of the gate: a channel
	// anybody can join is a channel anybody can cancel a run from, and a
	// list is a thing that grows by accident. Empty means the chat answers
	// nobody — which is what a machine that has not been told yet has to
	// do.
	//
	// The token is not here. It is a secret and it lives in the
	// environment; an id is a number somebody should be able to read back
	// out of `orbit settings`.
	ChatID string `json:"chatId,omitempty"`

	// BudgetTask is the most one task may spend, in dollars, and zero is
	// no budget at all — the working zero every field of this file has.
	//
	// Dollars and not tokens because dollars are what a person decides in:
	// a cap in tokens is a cap whose meaning changes with the model. It is
	// enforced between phases, and only for engines that charge per token
	// — under a subscription the money left the account in advance, and a
	// share of it attributed to one run is arithmetic on a charge nobody
	// made.
	BudgetTask float64 `json:"budgetTask,omitempty"`
	// BudgetWorkspace is the most the tasks on this board may have spent
	// before nothing new starts on its own. Zero is no budget.
	BudgetWorkspace float64 `json:"budgetWorkspace,omitempty"`
	// QuotaFloor is the percentage of a subscription engine's window that
	// has to be left for the queue to pick up another task. Zero is no
	// floor. It is the same brake as BudgetWorkspace in the other unit:
	// money for an engine that charges, the window for one that does not.
	QuotaFloor int `json:"quotaFloor,omitempty"`

	// Decisions is whether the decision engine answers for the supervisor,
	// and how far it is trusted: "off", "shadow" or "on". Empty is off,
	// the working zero every field here has.
	//
	// Shadow is the one worth explaining. It asks the same question at the
	// same moment and writes the answer into the record beside what
	// actually happened, and acts on nothing. A week of that is the only
	// honest way to find out whether a machine's judgement matches yours
	// on your own work, and it costs a fraction of a cent to find out.
	//
	// The key itself is not here. It is a secret and it lives in the
	// environment, which is the rule ChatID's comment states above: this
	// table is printed to a terminal, to a screen and into a chat.
	Decisions string `json:"decisions,omitempty"`
	// DecisionFloor is how sure the decision engine has to be before what
	// it says is acted on, as a percentage. Below it the run waits for a
	// person exactly as it does now. Zero is not a floor of nothing — it
	// is nobody having chosen, and DecisionBar reads it as seventy.
	DecisionFloor int `json:"decisionFloor,omitempty"`
	// RunTimeout is how long a run started by the window or by the queue
	// may take before it is stopped, written the way a person writes a
	// duration: 45m, 2h. Empty is no timeout, which is the working zero
	// every field here has and was the only behaviour there was.
	//
	// A setting and not a default, because there is no honest default: a
	// phase that reads a repository is seconds and one that writes a
	// migration is an hour, and a number Orbit picked would stop the
	// second kind for a reason nobody chose. What it is for is the other
	// end — an engine that has wedged on a network read holds a worktree
	// and a slot until somebody notices, and noticing is the part a person
	// asleep cannot do. `orbit run -timeout` is the same stop for a run
	// started by hand, and this is what the flag becomes when the window
	// starts one.
	RunTimeout string `json:"runTimeout,omitempty"`

	// CheckRecord makes every command ask SQLite whether the record is
	// still readable before it does anything. It is off by default because
	// the answer costs a full read of the file; it is a setting at all
	// because damage found on the day it happens is damage there is still a
	// backup for.
	CheckRecord bool `json:"checkRecord,omitempty"`
}

// Shipped is the configuration Orbit comes with: every field's working
// default, for a machine where nobody has chosen anything.
//
// One place and not three literals. A reader that answers the defaults for a
// file that is not there, a reader that answers them for a file that will
// not parse, and whatever puts one setting back to how it came all have to
// agree about what "how it came" is — and three copies of a struct literal
// agree only until somebody changes one.
func Shipped() Settings {
	return Settings{UnreadCap: defaultUnreadCap, Flow: defaultFlow}
}

// The three things the decision engine may be doing, as the settings file
// spells them.
const (
	// DecisionsOff is no decision engine: the supervisor is what it was.
	DecisionsOff = "off"
	// DecisionsShadow asks and writes down the answer, and acts on nothing.
	DecisionsShadow = "shadow"
	// DecisionsOn acts on what it is sure enough about.
	DecisionsOn = "on"
	// defaultDecisionFloor is how sure it has to be when nobody has said.
	defaultDecisionFloor = 70
)

// Deciding is what the decision engine is allowed to do here. A file that
// says nothing, or says something this build does not know, reads as off:
// the safe half of this decision is the one that changes nothing.
func (s Settings) Deciding() string {
	switch strings.TrimSpace(s.Decisions) {
	case DecisionsShadow:
		return DecisionsShadow
	case DecisionsOn:
		return DecisionsOn
	}

	return DecisionsOff
}

// DecisionBar is how sure it has to be before Orbit acts on it, as a
// percentage, and seventy where nobody has chosen.
func (s Settings) DecisionBar() int {
	if s.DecisionFloor <= 0 || s.DecisionFloor > 100 {
		return defaultDecisionFloor
	}

	return s.DecisionFloor
}

// Timeout is RunTimeout as a duration, and zero for a run nothing stops.
//
// Anything that will not parse is zero as well: the value is refused where
// it is typed, so a file holding one was edited by hand, and a run that
// refused to start over it would be a settings file that stops work rather
// than shaping it.
func (s Settings) Timeout() time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(s.RunTimeout))
	if err != nil || d <= 0 {
		return 0
	}

	return d
}

// settingsPath is the one file settings live in, at the root of the state
// tree rather than under any one repository: settings are not scoped to a
// repository.
func (s *Store) settingsPath() string {
	return filepath.Join(s.root, "settings.json")
}

// Settings reads the persisted configuration.
//
// A file that has never been saved and a file that will not parse are
// answered the same way: the defaults. Only a real I/O error — not "there
// is nothing here yet" and not "what is here is not JSON" — is returned.
func (s *Store) Settings() (Settings, error) {
	path := s.settingsPath()

	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Shipped(), nil
	}

	if err != nil {
		return Settings{}, fmt.Errorf("read %q: %w", path, err)
	}

	var cfg Settings
	if err := json.Unmarshal(body, &cfg); err != nil {
		// A settings file that will not parse yields the defaults, not a
		// failure — the same reasoning a broken catalogue answers with
		// English rather than an error.
		return Shipped(), nil //nolint:nilerr // deliberate: unparseable settings yield the defaults
	}

	return cfg, nil
}

// SaveSettings writes the configuration, replacing whatever was there.
//
// A file that is there but will not parse is moved aside first rather than
// overwritten. Settings answers an unparseable file with the defaults —
// deliberately, so a reader always has something usable — and every setter
// above this one is a read-modify-write. Without this, one switch flipped on
// screen would replace a whole configuration with the defaults plus that
// switch, and the engine, model and theme somebody chose would be gone with
// nothing anywhere to say they had ever been set.
//
// Moving it aside is not repair and does not pretend to be. Whatever was in
// there is left where a person can read it, and the write goes ahead, so a
// file nobody can parse cannot lock the settings screen either.
func (s *Store) SaveSettings(cfg Settings) error {
	path := s.settingsPath()
	if err := keepUnreadable(path); err != nil {
		return err
	}

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	return WriteAtomically(path, body)
}

// UpdateSettings reads the settings, hands them to change, and writes them
// back, with nobody else able to do the same in between.
//
// The three steps have to be one step. Every setter over this file is a
// read-modify-write of the whole of it — that is what keeps a switch flipped
// on screen from erasing an engine chosen in a terminal — and two of them at
// once is two processes reading the same file, changing different fields,
// and each writing back a copy made before the other's change existed.
// Whichever writes second wins, and nothing anywhere says so: both report
// success, and the setting that lost is simply not there the next time
// somebody looks. `orbit set engine codex` typed while the window is open is
// not a rare arrangement; it is how the two are meant to be used.
//
// The write itself was already atomic. That is a different promise: it says
// a reader never sees half a file, not that a writer never loses a field.
func (s *Store) UpdateSettings(change func(*Settings) error) error {
	release, err := s.lockSettings()
	if err != nil {
		return err
	}
	// The lock is given back whatever happened, and what giving it back had
	// to say is not thrown away: a lock file that could not be removed is
	// the next change's two-second wait and then its refusal.
	return errors.Join(s.changeSettings(change), release())
}

// changeSettings is the read, the change and the write, with the lock
// already held.
func (s *Store) changeSettings(change func(*Settings) error) error {
	cfg, err := s.Settings()
	if err != nil {
		return err
	}

	if err := change(&cfg); err != nil {
		return err
	}

	return s.SaveSettings(cfg)
}

// unreadableSuffix is what an unparseable settings file is renamed with. It
// is a suffix rather than a directory so the two sit side by side: whoever
// opens the state root to find out why their engine reset sees both files in
// the same listing.
const unreadableSuffix = ".unreadable"

// keepUnreadable moves a settings file that will not parse out of the way,
// and leaves every other file exactly where it is.
//
// The parse it does is the same one Settings does — unmarshalling into
// Settings, not merely checking the JSON is well formed — because the two
// have to agree on what "unreadable" means. That takes in a file which is
// valid JSON with a string where the unread cap goes: json.Unmarshal refuses
// it, so it is moved aside like any other file that will not parse, and the
// write that follows starts from the defaults.
//
// A file that cannot be read at all is not this function's problem: the
// write that follows will hit the same fault and report it in its own terms.
func keepUnreadable(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil //nolint:nilerr // a file that is absent or unreadable is the write's problem, not this one's
	}

	var probe Settings
	if json.Unmarshal(body, &probe) == nil {
		return nil
	}

	aside := path + unreadableSuffix
	if err := os.Rename(path, aside); err != nil {
		return fmt.Errorf("move the unreadable %q aside to %q: %w", path, aside, err)
	}

	return nil
}
