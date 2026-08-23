package engine

import "context"

// Fake is the engine every test uses.
//
// It exists so that no test can ever spend money or need a network. The
// program this replaces had a test that fired a real paid call, which made
// its own suite unsafe to run.
type Fake struct {
	Output string
	Err    error
	Calls  []Request
}

// The compiler is the right reviewer for this: an engine that stops
// satisfying the interface should fail the build, not a test.
var _ Engine = (*Fake)(nil)

// NewFake returns a fake that answers with a fixed string.
func NewFake(output string) *Fake {
	return &Fake{Output: output}
}

// Name identifies the engine in the record.
func (f *Fake) Name() string { return "fake" }

// Run records the request and returns whatever the fake was told to return.
func (f *Fake) Run(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	f.Calls = append(f.Calls, req)
	if f.Err != nil {
		return Result{}, f.Err
	}
	return Result{Output: f.Output}, nil
}
