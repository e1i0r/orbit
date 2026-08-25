// Package engine wraps the programs that actually write code — claude, codex,
// opencode — behind three verbs.
//
// The interface is small on purpose. Every difference between the engines
// belongs on the screen, stated, rather than hidden behind a shim that
// pretends they are the same: an engine that cannot resume a session should
// grey out the button and say why.
package engine

import "context"

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

// Result is what came back. SessionID and Cost are empty when an engine does
// not report them, which is a fact about that engine and not a failure.
type Result struct {
	Output    string
	SessionID string
	Cost      float64
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
}
