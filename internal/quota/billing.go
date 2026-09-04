package quota

// How each engine is paid for, where its quota is read from, and the answer
// a caller gets when it asks about one.

import "os"

// Mode is how an engine is paid for.
type Mode int

const (
	// Unstated is an engine nobody has written a row for in FromEnv. It is
	// the zero value deliberately: a build that grows a fourth engine and
	// forgets it here says nothing about how it is paid, rather than
	// claiming whichever mode happened to be written first.
	Unstated Mode = iota
	// Subscription is a fixed price paid in advance, which buys a window of
	// use rather than an amount of it. What is left is a fraction and an
	// hour, and there is no per-run figure of it that anybody is charged.
	Subscription
	// PerToken is metered: every run adds to a bill that arrives later, so
	// money is the honest unit and the number is real.
	PerToken
)

// Spends is whether money is the unit this engine's use is spoken in.
//
// This one method is the decision the package comment describes, and it is
// the whole of what a header, a budget or a stats screen has to ask before
// printing a dollar sign. Only PerToken spends: under a subscription the
// money left the account once, in advance, and a share of it attributed to
// one run is arithmetic on a charge nobody made.
func (m Mode) Spends() bool { return m == PerToken }

// Reading is everything there is to say about one engine's quota right now.
type Reading struct {
	// Engine is the name that was asked about, carried back so a caller
	// holding several readings cannot mistake one engine's for another's.
	Engine string

	// Mode is how this engine is paid for, and with it whether the money
	// its runs report means anything.
	Mode Mode

	// Windows is what the source answered, empty when it has not answered
	// yet — the first look of a cold cache, or a source in backoff.
	Windows []Window

	// Sourced is whether there is anywhere to read this engine's windows
	// from at all. It is what keeps "nobody can see opencode's quota" from
	// being drawn as "opencode has none left": with no source there is
	// nothing to wait for and nothing to show, and a reader is owed that
	// sentence rather than an empty corner of the screen.
	Sourced bool
}

// Meter is every engine Orbit can run: how each is paid for, and the source
// its windows come from.
type Meter struct{ engines []metered }

// metered is one engine's row of that table.
type metered struct {
	name string
	mode Mode
	from *Client // nil when this engine has no quota source
}

// FromEnv builds the meter this machine describes.
//
// Two facts are read out of the environment rather than assumed, because
// both are the reader's own arrangement and neither is visible any other
// way. An engine driven with an API key in the environment is billed per
// token by whoever issued that key; the same engine with no key is being
// used under the subscription its login carries. And a base URL is where a
// proxy that answers GET /quota would be if there is one — the engines
// pointed at nothing are the engines with no source, which is a state this
// package reports rather than hides.
//
// agy is a subscription with no source and no variable either, from the
// other direction: it is signed into with a Google account and there is no
// key that would make it metered, so what a run of it costs is a share of a
// monthly price nothing here can read.
//
// opencode is per token with no source and no variable to look at: it drives
// whichever provider its own configuration names, on the reader's key, so
// there is no single endpoint that could answer for it and no arrangement in
// which its use is not metered by somebody.
//
// The names are the names a record writes, and internal/cli holds the two
// tables together with a test — this package cannot import the engines, and
// an engine named here but nowhere else would be a row nobody ever reads.
func FromEnv() *Meter {
	return &Meter{engines: []metered{
		{name: "agy", mode: Subscription},
		{name: "claude", mode: keyed("ANTHROPIC_API_KEY"), from: New(os.Getenv("ANTHROPIC_BASE_URL"))},
		{name: "codex", mode: keyed("OPENAI_API_KEY"), from: New(os.Getenv("OPENAI_BASE_URL"))},
		{name: "opencode", mode: PerToken},
	}}
}

// keyed is the mode of an engine that can be either, decided by whether a
// key for its provider is in the environment.
func keyed(env string) Mode {
	if os.Getenv(env) != "" {
		return PerToken
	}

	return Subscription
}

// Read is one engine's quota, as much of it as there is to say.
//
// wait is Client.Quota's: true for a command that draws one frame and then
// exits, which has no later frame for a background fetch to arrive in.
//
// An engine this table has never heard of comes back Unstated, unsourced and
// empty, which is the same answer as a nil meter's. Both are "nothing is
// known about this name", and neither is a failure worth an error: the
// caller is a status bar, and what it does with either is say so.
func (m *Meter) Read(name string, wait bool) Reading {
	if m == nil {
		return Reading{Engine: name}
	}

	for _, e := range m.engines {
		if e.name != name {
			continue
		}

		return Reading{
			Engine:  name,
			Mode:    e.mode,
			Windows: e.from.Quota(wait),
			Sourced: e.from != nil,
		}
	}

	return Reading{Engine: name}
}

// Mode is how one engine is paid for, without asking its source anything.
//
// The mode is the half of a Reading that costs nothing to answer, and it is
// the half a caller with a column of money to print needs: no window has to
// be fetched to know that a subscription has no dollars in it.
func (m *Meter) Mode(name string) Mode {
	if m == nil {
		return Unstated
	}

	for _, e := range m.engines {
		if e.name == name {
			return e.mode
		}
	}

	return Unstated
}
