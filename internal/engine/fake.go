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
	Events []StreamEvent
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

// CanResume is false, and it is a fact about the fake rather than a stub: it
// reports no session id, so there is nothing to resume. A test that wants the
// window to offer the keyboard says so through the window's own port.
func (f *Fake) CanResume() bool { return false }

// Models returns nil for the fake engine.
func (f *Fake) Models() []Choice { return nil }

// Efforts returns nil for the fake engine.
func (f *Fake) Efforts() []Choice { return nil }

// CanThink returns false for the fake engine.
func (f *Fake) CanThink() bool { return false }

// Run records the request and returns whatever the fake was told to return.
func (f *Fake) Run(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	f.Calls = append(f.Calls, req)
	if req.OnEvent != nil {
		for _, ev := range f.Events {
			req.OnEvent(ev)
		}
	}
	if f.Err != nil {
		return Result{}, f.Err
	}
	return Result{Output: f.Output}, nil
}
