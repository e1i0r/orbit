package queue

// What the queue is doing, for a reader: which runs are going, who is
// waiting and for what, and whether the service is there to move the line.
//
// Only what the queue alone knows. How a task's own record reads is the
// record's to say; this is the line, and why each place in it is waiting.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// The service's two files under the state root: the lock that says it is
// alive, and the pid that says which process it is.
const (
	serviceLockFile = "queue.service.lock"
	servicePIDFile  = "queue.service.pid"
)

// Why a task is waiting.
const (
	// WhySlots is every slot taken.
	WhySlots = "slots"
	// WhyMemory is memory over the ceiling, with a slot free.
	WhyMemory = "memory"
	// WhyNext is nothing holding it: the service starts it on its next look.
	WhyNext = "next"
)

// Going is a run the queue counts against its slots.
type Going struct {
	ID    string
	Since time.Time // when this run began
	Phase string    // the phase it is in, empty before the first
}

// Waiter is a task in line.
type Waiter struct {
	ID    string
	Since time.Time // when it was asked for, which is its place in line
	Why   string    // WhySlots, WhyMemory or WhyNext
}

// Report is the queue as it stands.
type Report struct {
	Max, Ceiling int // max-running and memory-ceiling, as they are read now
	Memory       int // memory in use, as a percentage
	MemoryKnown  bool
	Service      int // the service's pid, and zero when none is running
	Going        []Going
	Waiting      []Waiter
}

// Status reads the queue: the same look a dispatch takes, without starting
// anything.
func Status(s *store.Store) (Report, error) {
	cfg, err := s.Settings()
	if err != nil {
		return Report{}, err
	}

	st, err := look(s)
	if err != nil {
		return Report{}, err
	}

	r := Report{Max: cfg.Running(), Ceiling: cfg.MemoryBar(), Service: servicePID(s)}
	r.Memory, r.MemoryKnown = memoryUsed()

	for _, id := range st.going {
		g, gErr := going(s, id)
		if gErr != nil {
			return Report{}, gErr
		}

		r.Going = append(r.Going, g)
	}

	why := WhyNext

	switch {
	case st.running >= r.Max:
		why = WhySlots
	case r.MemoryKnown && r.Memory >= r.Ceiling:
		why = WhyMemory
	}

	for _, w := range st.waiting {
		r.Waiting = append(r.Waiting, Waiter{ID: w.task.ID, Since: w.since, Why: why})
	}

	return r, nil
}

// going is one run as the record has it: when the attempt began and the
// phase it is in.
func going(s *store.Store, id string) (Going, error) {
	events, err := task.Events(s, task.Task{ID: id})
	if err != nil {
		return Going{}, err
	}

	g := Going{ID: id}

	for _, e := range events {
		switch e.Kind {
		case record.TaskStarted:
			g.Since, g.Phase = e.At, ""
		case record.PhaseStarted:
			g.Phase = e.Phase
		}
	}

	return g, nil
}

// servicePID is the service's pid when one is running, and zero when none
// is. It reads the file and asks whether that process is there, rather than
// trying the lock: a look that took the lock for an instant could make a
// service starting in that instant think another had the queue, and leave.
func servicePID(s *store.Store) int {
	body, err := os.ReadFile(filepath.Join(s.Root(), servicePIDFile))
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil || !processAlive(pid) {
		return 0
	}

	return pid
}
