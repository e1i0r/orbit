package cli

import (
	"context"
	"testing"
	"time"
)

// TestTheSignalHandlerComesOffWhenTheRunIsCancelled is the second Ctrl-C.
// The first one cancels the run; the handler must then come off, so that the
// second one — pressed because the unwind is taking longer than the person
// pressing it expected — reaches the process with its default disposition
// instead of being swallowed.
//
// It is asserted through the seam rather than by signalling anything: the
// real thing would mean spawning orbit and interrupting it, and this suite
// does not spawn orbit.
func TestTheSignalHandlerComesOffWhenTheRunIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Two deep, so a stop called twice cannot block or panic the test: what
	// is under test is when it is called, not how often.
	stopped := make(chan struct{}, 2)

	restoreOnCancel(ctx, func() { stopped <- struct{}{} })

	select {
	case <-stopped:
		t.Fatal("the handler came off while the run was still going: the first Ctrl-C would kill the run instead of letting it write down that it stopped")
	case <-time.After(20 * time.Millisecond):
	}

	cancel()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Error("the handler was still installed after the run was cancelled: a second Ctrl-C during the unwind would be swallowed, leaving kill -9 as the only way out")
	}
}
