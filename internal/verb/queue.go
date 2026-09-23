package verb

// orbit queue: the line, and one task's way through it.
//
// Two readings. With no task, what the queue is doing: the runs going, who
// is waiting and for what, and whether its service is there. With a task,
// its whole way through: when it was queued, when it started, each phase,
// how it ended, and the last things it did. Nothing about the queue runs
// where nobody can see it.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/queue"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// lastSteps is how many of a task's latest tool calls its way through the
// queue ends on.
const lastSteps = 5

// queued answers orbit queue, with or without a task.
func queued(w World, in In) (Out, error) {
	report, err := queue.Status(w.Store())
	if err != nil {
		return Out{}, err
	}

	if id := in.Arg("task"); id != "" {
		return trip(w, in, id, report)
	}

	return Out{Said: line(w.Words(), report, time.Now()), Saw: report}, nil
}

// line is the queue written out for a terminal.
func line(p *words.Printer, r queue.Report, now time.Time) string {
	var b strings.Builder

	memory := ""
	if r.MemoryKnown {
		memory = p.T("queue.memory_now", " (now {n}%)", words.Arg{Name: "n", Value: strconv.Itoa(r.Memory)})
	}

	b.WriteString(p.T("queue.head", "Queue: {max} at a time, memory under {ceiling}%",
		words.Arg{Name: "max", Value: strconv.Itoa(r.Max)},
		words.Arg{Name: "ceiling", Value: strconv.Itoa(r.Ceiling)}) + memory + "\n")

	if r.Service > 0 {
		b.WriteString(p.T("queue.service_on", "Service: running (process {pid})",
			words.Arg{Name: "pid", Value: strconv.Itoa(r.Service)}) + "\n")
	} else {
		b.WriteString(p.T("queue.service_off", "Service: not running; it starts when something waits") + "\n")
	}

	fmt.Fprintf(&b, "\n%s (%d/%d)\n", p.T("queue.going", "Running"), len(r.Going), r.Max)

	for _, g := range r.Going {
		fmt.Fprintf(&b, "  %-10s %-6s %s\n", g.ID, ago(now, g.Since), g.Phase)
	}

	fmt.Fprintf(&b, "\n%s (%d)\n", p.T("queue.waiting", "Waiting"), len(r.Waiting))

	for i, wt := range r.Waiting {
		fmt.Fprintf(&b, "  %d. %-10s %-6s %s\n", i+1, wt.ID, ago(now, wt.Since), why(p, wt.Why))
	}

	return strings.TrimRight(b.String(), "\n")
}

// why is a waiting task's reason, in words.
func why(p *words.Printer, key string) string {
	switch key {
	case queue.WhySlots:
		return p.T("queue.why_slots", "no free slot")
	case queue.WhyMemory:
		return p.T("queue.why_memory", "memory over the ceiling")
	default:
		return p.T("queue.why_next", "starts on the service's next look")
	}
}

// trip is one task's way through the queue, read off its record.
func trip(w World, in In, id string, r queue.Report) (Out, error) {
	t, err := found(w, In{Task: id, Repo: in.Repo})
	if err != nil {
		return Out{}, err
	}

	events, err := task.Events(w.Store(), t)
	if err != nil {
		return Out{}, err
	}

	p := w.Words()

	var (
		b     strings.Builder
		steps []string
	)

	b.WriteString(t.ID + "\n")

	for _, e := range events {
		if said := milestone(p, e, r, t.ID); said != "" {
			fmt.Fprintf(&b, "  %s  %s\n", e.At.Local().Format("15:04:05"), said)
		}

		if e.Kind == record.PhaseToolCall || e.Kind == record.DeliverStep {
			steps = append(steps, e.At.Local().Format("15:04:05")+"  "+view.ToolLine(e.Data["tool"], e.Text))
		}
	}

	if len(steps) > 0 {
		b.WriteString("\n" + p.T("queue.last_steps", "last steps:") + "\n")

		for _, s := range steps[max(len(steps)-lastSteps, 0):] {
			b.WriteString("  " + s + "\n")
		}
	}

	return Out{Said: strings.TrimRight(b.String(), "\n")}, nil
}

// milestone is one line of a task's way through the queue, and empty for
// an event that is not one.
func milestone(p *words.Printer, e record.Event, r queue.Report, id string) string {
	switch e.Kind {
	case record.TaskQueued:
		return p.T("queue.trip_queued", "queued") + place(p, r, id)
	case record.TaskStarted:
		return p.T("queue.trip_started", "started")
	case record.PhaseStarted:
		return e.Phase
	case record.PhaseFinished:
		return e.Phase + " → " + p.T("queue.trip_finished", "finished")
	case record.PhaseFailed:
		return e.Phase + " → " + p.T("queue.trip_failed", "failed") + ": " + firstLineOf(e.Data["error"])
	case record.TaskFinished:
		return p.T("queue.trip_done", "done")
	case record.TaskFailed:
		return p.T("queue.trip_task_failed", "failed") + ": " + firstLineOf(e.Text)
	case record.TaskCancelled:
		return p.T("queue.trip_cancelled", "cancelled")
	case record.TaskRequeued:
		return p.T("queue.trip_requeued", "back in To Do")
	}

	return ""
}

// place is where a task still waiting stands, and why, and empty for one
// that is not waiting any more.
func place(p *words.Printer, r queue.Report, id string) string {
	for i, wt := range r.Waiting {
		if wt.ID == id {
			return p.T("queue.trip_place", " · {n} in line, {why}",
				words.Arg{Name: "n", Value: strconv.Itoa(i + 1)},
				words.Arg{Name: "why", Value: why(p, wt.Why)})
		}
	}

	return ""
}

// ago is how long since, short, and a dash where the record had no time.
func ago(now, since time.Time) string {
	if since.IsZero() {
		return "-"
	}

	return now.Sub(since).Round(time.Second).String()
}

// firstLineOf is the first line of a longer text, without the colours an
// engine's own terminal output carries: opencode's errors arrive with them.
func firstLineOf(text string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(colours.ReplaceAllString(text, "")), "\n")

	return first
}

// colours is a terminal colour sequence.
var colours = regexp.MustCompile(`\x1b\[[0-9;]*m`)
