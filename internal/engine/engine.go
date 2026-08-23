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
	Prompt string
	Model  string
	Dir    string

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
}

// Result is what came back. SessionID and Cost are empty when an engine does
// not report them, which is a fact about that engine and not a failure.
type Result struct {
	Output    string
	SessionID string
	Cost      float64
}

// Engine is one program that can be asked to do a phase.
//
// Declared here because this is where it is consumed; the concrete engines
// satisfy it without importing anything of their own.
type Engine interface {
	Name() string
	Run(ctx context.Context, req Request) (Result, error)
}
