// Package engine wraps the programs that actually write code — claude, codex,
// opencode — behind three verbs.
//
// The interface is small on purpose. Every difference between the engines
// belongs on the screen, stated, rather than hidden behind a shim that
// pretends they are the same: an engine that cannot resume a session should
// grey out the button and say why.
package engine

import (
	"context"
	"time"
)

// Request is everything an engine needs for one phase.
type Request struct {
	Prompt   string
	Model    string
	Effort   string
	Thinking string
	Dir      string

	// Permissions is what this phase is allowed to touch, in the closed
	// vocabulary permission.go defines. An empty list is not "no opinion":
	// it is the most restrictive posture the engine can state, which is the
	// distinction the whole of permission.go exists to make. Whatever is
	// here is passed to the engine and written into the record, so a run's
	// posture is recoverable from the log rather than from whichever flow
	// file was on disk that day.
	Permissions []string

	// Resume is a session id an engine may be asked to carry on from,
	// empty for a fresh run. It is written and nothing sets it yet: the
	// window's gesture for taking the keyboard is a later task, and the
	// field exists now because the session id that feeds it only started
	// being captured in this one. An engine that cannot resume says so
	// rather than quietly starting over.
	Resume string

	// Env is what the phase's process is told about the run it belongs to,
	// as NAME=value, on top of the environment Orbit itself was started
	// with. It is how a command the engine runs — `orbit join`, in the
	// worktree, with no arguments — knows which task it is acting on
	// without the prompt having to spell an id the model would then have to
	// copy correctly.
	Env []string

	// OnEvent receives streaming events in real time as the engine emits them.
	OnEvent func(StreamEvent)
}

// StreamEvent is a real-time event emitted during engine execution.
type StreamEvent struct {
	Type     string // "thought", "tool_call", "refusal", "result"
	Thought  string
	ToolCall StreamToolCall
	Refusal  StreamRefusal
	Cost     float64
}

// StreamRefusal is a tool call denied by permissions.
type StreamRefusal struct {
	Tool  string
	Input string
}

// StreamToolCall is a tool call invoked by the model.
type StreamToolCall struct {
	Name string
	Args string
}

// Usage is what a phase spent, in tokens, as the engine counted them.
//
// The cache pair is here beside the other two because it is the number that
// moves the bill. A cached read is a fraction of the price of the same tokens
// sent fresh, and a cache that stops being hit fails silently — nothing
// errors, the run works, and the cost per turn quietly multiplies. Read as
// zero across a long session it says something early in the context is
// changing between turns; there is no way to notice that from a total.
//
// Zero is "the engine did not say", not "none": the three CLIs count in
// three vocabularies and not all of them report every field.
type Usage struct {
	// Input is the prompt as it was sent, not counting the part of it that
	// came back from the cache. That is claude's spelling and the one the
	// other two are normalised to, so that a total across a task's phases
	// adds tokens of one kind.
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
}

// Any is whether the engine said anything at all about what it spent.
func (u Usage) Any() bool {
	return u.Input != 0 || u.Output != 0 || u.CacheRead != 0 || u.CacheWrite != 0
}

// Result is what came back. SessionID and Cost are empty when an engine does
// not report them, which is a fact about that engine and not a failure.
type Result struct {
	Output    string
	SessionID string
	Cost      float64
	Usage     Usage
	Thoughts  []string
	Refusals  []StreamRefusal
	ToolCalls []StreamToolCall
}

// Choice is one selectable value for an engine dial (model or effort).
// An empty ID means "default" (whatever the CLI/engine itself defaults to).
type Choice struct {
	ID    string
	Label string
}

// Engine is one program that can be asked to do a phase.
//
// Declared here because this is where it is consumed; the concrete engines
// satisfy it without importing anything of their own.
type Engine interface {
	Name() string
	Run(ctx context.Context, req Request) (Result, error)

	// Locate is the program this engine runs, as an absolute path, or a
	// refusal saying this machine cannot run it.
	//
	// The engine screen asks it to decide whether to draw dials or setup
	// steps, and Run asks it to decide what to execute, so the two cannot
	// disagree. A screen running exec.LookPath on the engine's name and a
	// Run running exec.Command on the same bare name would draw an engine
	// installed somewhere PATH does not mention as "[setup required]" to a
	// reader who has it open in the next window.
	Locate() (string, error)

	// CanResume is whether this engine can carry on a session it started
	// before, which is the difference the package comment above says
	// belongs on the screen rather than behind a shim.
	//
	// It is a method and not a field on Request because it is a fact about
	// the program, not about one phase, and because the window has to be
	// able to ask it before there is a request at all: the gesture that
	// takes the keyboard is offered or greyed out by this answer, and a
	// greyed-out key that says why is the whole of what an honest
	// difference between two engines looks like.
	CanResume() bool

	// Models returns the choices this engine supports for its model dial.
	// The zero-value choice has ID "" and Label "default".
	Models() []Choice

	// Efforts returns the choices this engine supports for its effort dial.
	// An engine with no effort switch returns an empty slice.
	Efforts() []Choice

	// CanThink returns whether this engine supports an extended thinking mode.
	CanThink() bool

	// Transcript is what was said in an interactive session opened in dir
	// after since — the reader's prompts and the engine's answers, oldest
	// first, and no tool call.
	//
	// It is on this interface rather than behind a name check because
	// every engine keeps one and no two keep it alike: claude a directory
	// of JSON lines per working directory, codex a file per session under
	// the day it ran, opencode rows in a database. Which of those a
	// session left behind is a fact about the program, like Locate and
	// CanResume, and the compiler is the right reviewer for a new engine
	// that has not answered the question.
	//
	// An engine whose transcript nobody has mapped yet answers with
	// nothing rather than with a guess: the session is not read back, and
	// the record is left as it was.
	Transcript(dir string, since time.Time) ([]Turn, error)
}
