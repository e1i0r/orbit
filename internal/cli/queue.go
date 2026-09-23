package cli

// The queue's service, as orbit starts it: see internal/queue/service.go.

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/e1i0r/orbit/internal/queue"
	"github.com/e1i0r/orbit/internal/store"
)

// serveQueue runs the queue's service until nothing is waiting, or until it
// is told to stop. Nobody types it: a start that leaves a task waiting runs
// it, so what it has to say goes to the log, and its exit code is all a
// reader of the process table sees.
func serveQueue(errOut io.Writer) int {
	s, err := store.Open()
	if err != nil {
		fmt.Fprintf(errOut, "orbit: %v\n", err)

		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := queue.Serve(ctx, s); err != nil {
		fmt.Fprintf(errOut, "orbit: the queue: %v\n", err)

		return 1
	}

	return 0
}
