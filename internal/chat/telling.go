package chat

// The other direction: what Orbit says without being asked.
//
// Apart from serve.go because it is the opposite half. Everything there
// happens because somebody wrote a message; nothing here does, and that is
// the whole difference between a chat you have to remember to open and one
// that reaches you.

import (
	"context"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// looking is how often the record is asked what has happened.
//
// Seconds and not milliseconds: what is being waited for is a run that has
// stopped, and nobody reaching for a phone can tell four seconds from one.
// It is one query per tick — the record answers what was written after a row
// across every task at once — and on most ticks the answer is nothing.
const looking = 4 * time.Second

// telling pushes what the record says has happened, for as long as the desk
// is open.
//
// The whole of the outward half. It is a poll and not a hook because a run is
// its own process: it can be killed, and a notification that depended on the
// dying process to send it is the notification that never arrives for the
// task that most needed one.
func (d *Desk) telling(ctx context.Context) {
	if d.env.Watch == nil || (d.env.Tells == "" && d.env.Also == nil) {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(looking):
		}

		news, err := d.env.Watch(ctx)
		if err != nil {
			logger.Warn("chat", "%s: read what has happened: %v", d.to.Name(), err)

			continue
		}

		for _, one := range news {
			said := News(one, d.env.Words)
			if said == "" {
				continue
			}

			if d.env.Tells != "" {
				d.send(ctx, d.env.Tells, said)
			}

			if d.env.Also != nil {
				d.env.Also(said)
			}
		}
	}
}

// heard is one message, answered.
func (d *Desk) heard(ctx context.Context, m Message) {
	// Before anything is parsed. A message from somebody else is not a
	// command with a bad argument; it is not a command at all, and reading
	// it far enough to say why would be reading it.
	if d.env.Allowed == nil || !d.env.Allowed(m.Who) {
		logger.Warn("chat", "%s: a message from %q was turned away", d.to.Name(), m.Who)

		if d.env.Stranger != nil {
			if said := d.env.Stranger(m); said != "" {
				d.send(ctx, m.Where, said)
			}
		}

		return
	}

	stop := d.waiting(ctx, m.Where)
	answer := d.answer(ctx, m)

	stop()

	if answer != "" {
		d.send(ctx, m.Where, answer)
	}
}

// send answers, and gives up rather than failing.
//
// Logged and dropped. A chat that is down must not be able to stop anything,
// which is the rule a gate follows too.
func (d *Desk) send(ctx context.Context, where, text string) {
	if err := d.to.Say(ctx, where, text); err != nil {
		logger.Error("chat", "%s: answer to %q not delivered: %v", d.to.Name(), where, err)
	}
}
