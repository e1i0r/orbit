package view

// Reason is the word a row needs beyond its phase, as a key and its
// arguments — never as a sentence. The fold produces keys and the window
// translates them, which is what keeps internal/view out of internal/words
// and keeps a Spanish reader from being handed an English reason folded
// three layers down.
//
// It is set for every task whose row cannot be read from its band and its
// phase alone: one that needs you says why (failed, timed out, abandoned,
// waiting at a gate), one the reader has held says it is held even though it
// is still Running, and one that was cancelled says so even though its band
// is Done. A run that is simply working has its phase to show, and its
// Reason is zero.
type Reason struct {
	Key  string // one of the Reason constants below
	Args []Arg  // the values that key's sentence names
}

// Arg is one named value for the sentence a Reason names.
//
// It is view's own type and not words.Arg on purpose: internal/view may
// import internal/record and nothing else of Orbit's, which is what keeps
// the fold free of anything to do with language. The shape is identical to
// words.Arg, so internal/ui — which imports both — converts field for field.
type Arg struct {
	Name  string
	Value string
}

// The reasons a task can carry, and the arguments each one's sentence uses.
//
// Args always names every placeholder its key's sentence uses, even when the
// record left the value empty. A missing Arg is worse than an empty one:
// internal/words substitutes what it is given and leaves anything it was not
// given alone, so an absent phase would put the placeholder itself on screen.
// That is why a run that failed before it reached a phase has a key of its
// own rather than the same key with the argument left off.
const (
	// ReasonFailed is a run that stopped inside a phase. Args: phase.
	ReasonFailed = "reason.failed"
	// ReasonFailedToStart is a run that never reached a phase — an invalid
	// flow, an engine nobody configured, a worktree that could not be made.
	// It names no phase because the record has none to name. Args: none.
	ReasonFailedToStart = "reason.failed_to_start"
	// ReasonGate is a phase stopped and waiting because the flow asked it
	// to. Nothing moves until the reader answers. Args: phase.
	ReasonGate = "reason.gate"
	// ReasonHeld is a phase stopped because the reader asked it to. The task
	// is still Running — it still holds a worktree and still holds a slot —
	// and this is the word that says so. Args: phase.
	ReasonHeld = "reason.held"
	// ReasonCancelled is a run the reader stopped. Its band is Done and this
	// is how the row says which kind of done. Args: none.
	ReasonCancelled = "reason.cancelled"
	// ReasonTimedOut is a run that outlived the deadline it was given.
	// Nobody chose it, so unlike a cancellation it needs you. Args: none.
	ReasonTimedOut = "reason.timed_out"
	// ReasonAbandoned is a run whose process is gone and whose log never got
	// a terminal event — a SIGKILL, or a machine that went down. Args: none.
	ReasonAbandoned = "reason.abandoned"
)
