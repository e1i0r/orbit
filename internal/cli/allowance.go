package cli

// What an engine has left to spend, as a run asks it.
//
// internal/task decides when a task should change hands and internal/quota
// knows who has anything left, and neither can see the other: one is about
// walking a flow and the other about a proxy, a file of rollouts and an API
// key in the environment. This is the sentence between them, and it is short
// on purpose — the only thing a relay needs is whether it is worth handing
// work to an engine, and when to come back if it is not.

import (
	"time"

	"github.com/e1i0r/orbit/internal/quota"
	"github.com/e1i0r/orbit/internal/task"
)

// allowancePort answers what one engine has left.
//
// A nil meter yields nil, which internal/task reads as every engine being
// free — the same answer this gives for an engine nothing can read, and the
// right one: "nobody can see opencode's quota" and "opencode has none left"
// are different sentences, and only one of them is a reason not to try. Three
// of the four engines on this machine have never reported a number, and a
// relay that treated silence as empty would refuse to hand work to any of
// them.
//
// The workspace's quota floor is deliberately not applied. That floor exists
// to stop the queue picking up *new* work while an allowance runs low; a
// relay is finishing work already begun, and stranding a task at 90% of a
// window to protect a task nobody has written yet is the floor doing the
// opposite of its job.
func allowancePort(m *quota.Meter) task.Allowance {
	if m == nil {
		return nil
	}

	return func(name string) task.Spare {
		// wait is true: this is a command that reads the number once and
		// then acts on it, with no later frame for a background fetch to
		// arrive in. A relay that guessed because the cache was cold would
		// hand the work to the wrong engine and be unable to say why.
		reading := m.Read(name, true)

		if !reading.Sourced || reading.Mode.Spends() {
			// Nothing to read. An engine paid per token has no window to
			// run out of — what stops it is a card, which is not something
			// this can see — and an engine with no source has no window
			// anybody can see. Both are free to try.
			return task.Spare{Free: true}
		}

		return fromWindows(reading.Windows)
	}
}

// fromWindows is what an engine's windows say about handing it work: free
// while any of them has room, and otherwise how long until the first one
// comes back.
//
// Any of them rather than all, because a provider that reports an hourly
// window and a weekly one refuses a request when either is full. The hour is
// usually the one that fills and usually the one that comes back first,
// which is why the soonest reset is the answer to "until when".
//
// An engine with no windows at all is free. It has not answered yet — a cold
// cache, a source in backoff — and a reading that has not arrived is not a
// reading of nothing left.
func fromWindows(windows []quota.Window) task.Spare {
	var (
		spent bool
		back  time.Duration
	)

	for _, w := range windows {
		if w.Pct < full {
			continue
		}

		spent = true

		if w.ResetsIn > 0 && (back == 0 || w.ResetsIn < back) {
			back = w.ResetsIn
		}
	}

	if !spent {
		return task.Spare{Free: true}
	}

	// back stays zero when the provider said a window is full and did not
	// say when it returns. That is not "it is back now": it is spent with no
	// hour to give, and what reads this says so rather than naming one.
	return task.Spare{Back: back}
}

// full is the share of a window at which a provider stops answering.
const full = 100
