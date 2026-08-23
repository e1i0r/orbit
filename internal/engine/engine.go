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
