package queue

// The queue's service: a process of its own that starts waiting runs as
// slots free up, and goes away when nothing is waiting.
//
// It is not installed and nobody starts it by hand. A start that leaves a
// task waiting starts the service, and the service lives for as long as the
// queue has something in it. It is a separate process and not a goroutine
// in the window because the queue has to keep moving with the window
// closed, the same reason a run is a process of its own.
//
// One at a time on a machine: the service holds a lock for as long as it
// runs, and a second one that finds it taken leaves at once. So starting
// it is cheap and safe, and every start that leaves something waiting does.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// ServiceArg is the first argument that makes orbit the queue's service. It
// is not a command a reader types, so it is not in the table.
const ServiceArg = "__queue"

// queueTick is how often the service looks for a free slot. A run takes
// minutes, so two seconds is soon enough to be unnoticeable and rare enough
// to cost nothing.
const queueTick = 2 * time.Second

// errBusy is a lock somebody else holds.
var errBusy = errors.New("held by another process")

// Serve is the service: start what there is room for, wait, and again,
// until nothing is waiting or ctx ends. A service that finds another one
// already running returns at once and says nothing: that one has the queue.
func Serve(ctx context.Context, s *store.Store) error {
	unlock, err := lockFile(filepath.Join(s.Root(), serviceLockFile), false)
	if errors.Is(err, errBusy) {
		return nil
	}

	if err != nil {
		return err
	}

	defer unlock()

	// Its pid is written down for `orbit queue` to name. The lock is what
	// says it is alive; the file only says which process it is.
	pidPath := filepath.Join(s.Root(), servicePIDFile)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		logger.Warn("queue", "the service's pid could not be written down: %v", err)
	}

	defer func() {
		if err := os.Remove(pidPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Warn("queue", "the service's pid could not be taken back: %v", err)
		}
	}()

	logger.Info("queue", "the queue's service started as process %d", os.Getpid())

	for {
		_, left, err := Dispatch(s)
		if err != nil {
			logger.Error("queue", "the queue could not be read: %v", err)
		} else if left == 0 {
			logger.Info("queue", "nothing is waiting; the queue's service stops")

			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(queueTick):
		}
	}
}

// serviceProcess starts the service in a process of its own, in a group of
// its own so a closing terminal does not take it, and reaps it when it
// ends. If one is already running, the new one sees the lock and leaves.
func serviceProcess(s *store.Store) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, ServiceArg)

	cmd.Env = append(os.Environ(), "ORBIT_HOME="+s.Root())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		_ = cmd.Wait() //nolint:errcheck // the service's outcome is its log
	}()

	return nil
}
