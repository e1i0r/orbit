package knowledge

// Where a rule came from, and where it stands.
//
// Two vocabularies and neither is a field a caller invents. A rule with no
// source does not get in — a sentence in the agent's context that nobody can
// trace is indistinguishable from one the model made up, which is the whole
// point of keeping this outside the model.

// Source is where a rule came from, and there are four because there are
// four ways Orbit finds anything out. None of them is "the model thought so".
type Source int

const (
	// unsourced is the zero value, and it is not a source. A rule built by
	// somebody who forgot to say where it came from would otherwise pass as
	// having been read off the code, which is the one source nobody has to
	// justify — the mistake would look like the most trustworthy answer.
	unsourced Source = iota
	// FromCode is read off the map: "ledger only appends" because Write
	// inserts and nothing updates. It is regenerated rather than stored.
	FromCode
	// Human is somebody saying it, at a gate or in the supervisor.
	Human
	// FromRecord is a lesson: a gate rejected something, or an attempt
	// failed, and what happened became a rule with the scope of what the
	// task touched. This is the one that grows without anybody writing.
	FromRecord
	// FromProduction is an incident. Nothing reads these yet; the source
	// exists so the shape does not have to change when something does.
	FromProduction
	// FromDocs is read out of what the project already says about itself:
	// the CONTRIBUTING, the README, the docs, and the notes each engine
	// keeps in its own file.
	//
	// Its own source and not Human, even though a person wrote every word
	// of those. What a person typed into the supervisor they meant, now,
	// about this; what a CONTRIBUTING says is what somebody meant two years
	// ago and may have stopped meaning — and a reader deciding whether to
	// keep a rule is owed that difference. Ref carries the file and the
	// line, so they can go and look.
	FromDocs
	// FromHistory is read off what the repository has actually done: the
	// commits, and what travels with what.
	//
	// Apart from FromDocs because they answer different questions and
	// disagree often. A document says what somebody wanted; the history
	// says what the team kept doing. A rule backed by the second is one a
	// reader can trust without going to look, and Ref carries the count
	// that backs it.
	FromHistory
	// FromGates is read off what the checkout already refuses work over:
	// the commands its own pull requests have to pass.
	//
	// Apart from FromDocs, which is the other thing a repository says about
	// itself, because the two are trusted differently. A document says what
	// somebody wanted, and may have stopped meaning it the week after; a
	// workflow is running today, on every change, and a team that stopped
	// meaning it would have deleted it. It is the one source that arrives
	// with the command behind it, and Ref carries the file it was read out
	// of so a reader can go and look at the thing that is already running.
	FromGates
)

// State is where a rule stands: whether it applies, and whether somebody
// stopped it applying.
//
// Three and not two. A rule used to be said or not said, and that one switch
// is what forced switching a rule off when what was needed was something
// else entirely — "skip this one while we get the coverage up" is not
// disagreeing with it.
type State int

const (
	// Active is confirmed and working. It is the zero value, so a rule
	// somebody wrote by hand with a header of two lines applies, which is
	// what they meant by writing it.
	Active State = iota
	// Paused is stopped by a person, with a reason written down. It is not
	// a switch: the reason is what they will read when they come back, and
	// the only thing that will tell them whether it made sense.
	Paused
	// Off is somebody deciding against it. It stays and stops being told:
	// disagreeing with a rule and losing the record that it existed are
	// different things.
	Off
)
