package chat

// The loop: a message arrives, a verb runs, an answer goes back.
//
// Everything a chat needs that is not the service itself. The service is a
// Channel; what it carries is decided here, once, so that a second service
// inherits the gate, the confirmation and the wording rather than growing
// its own.

import (
	"context"
	"strings"
	"sync"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what the loop needs of the machine.
type Env struct {
	// World is what a verb reaches the machine through, built fresh for
	// each message: a store held open for days across a chat is a lock
	// held for days.
	World func() (verb.World, func(), error)
	Words *words.Printer
	// Allowed says whether an account may ask for anything at all.
	//
	// A chat anybody can join is a chat anybody can cancel a run from, so
	// this is not optional and it is not a list: nil turns every message
	// away, which is the safe answer for a build that forgot to say.
	Allowed func(who string) bool
}

// A Desk is one channel, served.
type Desk struct {
	env Env
	to  Channel

	// pending is the irreversible thing each conversation has been asked
	// to confirm. Free text can be misread and a merge cannot be taken
	// back, so the second message is the one that acts.
	mu      sync.Mutex
	pending map[string]Asked
}

// Open is a desk over one channel.
func Open(e Env, to Channel) *Desk {
	return &Desk{env: e, to: to, pending: map[string]Asked{}}
}

// Serve hands every message to the verbs until the context is done.
func (d *Desk) Serve(ctx context.Context) error {
	return d.to.Listen(ctx, func(m Message) { d.heard(ctx, m) })
}

// heard is one message, answered.
func (d *Desk) heard(ctx context.Context, m Message) {
	// Before anything is parsed. A message from somebody else is not a
	// command with a bad argument; it is not a command at all, and reading
	// it far enough to say why would be reading it.
	if d.env.Allowed == nil || !d.env.Allowed(m.Who) {
		logger.Warn("chat", "%s: a message from %q was turned away", d.to.Name(), m.Who)

		return
	}

	if answer := d.answer(ctx, m); answer != "" {
		if err := d.to.Say(ctx, m.Where, answer); err != nil {
			// Logged and dropped. A chat that is down must not be able to
			// stop anything, which is the rule a gate follows too.
			logger.Error("chat", "%s: answer to %q not delivered: %v", d.to.Name(), m.Where, err)
		}
	}
}

// answer is what one message is worth saying back, and nothing for a message
// that asks nothing.
func (d *Desk) answer(ctx context.Context, m Message) string {
	p := d.env.Words

	if help(m.Text) {
		return Help(p)
	}

	if yes, said := d.confirmed(ctx, m); said != "" || yes {
		return said
	}

	asked, isCommand, err := Read(m.Text, p)
	if err != nil {
		return err.Error()
	}

	if !isCommand {
		// A chat is a place people also talk. Answering every stray
		// sentence with a usage message is how a channel gets muted.
		return ""
	}

	if v, ok := verb.One(asked.Verb); ok && cannot[v.Path()] != "" {
		return p.T("chat.not_here", "{verb} is not something a chat can do: {why}",
			words.Arg{Name: "verb", Value: v.Path()},
			words.Arg{Name: "why", Value: cannot[v.Path()]})
	}

	if takesBack(asked.Verb) {
		return d.hold(m.Where, asked, p)
	}

	return d.run(ctx, asked, p)
}

// run asks for the verb and dresses what it answered.
func (d *Desk) run(ctx context.Context, asked Asked, p *words.Printer) string {
	w, done, err := d.env.World()
	if err != nil {
		return err.Error()
	}

	defer done()

	out, err := verb.Run(ctx, w, asked.Verb, asked.In)
	if err != nil {
		return err.Error()
	}

	return Reply(out, p)
}

// hold puts an irreversible ask aside and asks for a second message.
//
// The second message and not a word on the first, because free text can be
// misread and the cost is not symmetrical: a confirmation costs one line, a
// merge that nobody meant is in the branch other people work from.
func (d *Desk) hold(where string, asked Asked, p *words.Printer) string {
	d.mu.Lock()
	d.pending[where] = asked
	d.mu.Unlock()

	return p.T("chat.confirm", "{verb} cannot be taken back. Send /yes to go ahead.",
		words.Arg{Name: "verb", Value: asked.Verb})
}

// confirmed is the answer to a conversation that has something waiting.
//
// Anything other than /yes clears it. A person who asked to merge and then
// typed something else has changed the subject, and a confirmation that
// survives the change of subject is a confirmation that fires later.
func (d *Desk) confirmed(ctx context.Context, m Message) (bool, string) {
	d.mu.Lock()

	asked, waiting := d.pending[m.Where]
	if waiting {
		delete(d.pending, m.Where)
	}

	d.mu.Unlock()

	if !waiting {
		return false, ""
	}

	if strings.TrimSpace(strings.ToLower(m.Text)) != "/yes" {
		// Not swallowed: the message is still a message, and whatever it
		// asks for is answered on the next pass.
		return false, ""
	}

	return true, d.run(ctx, asked, d.env.Words)
}

// takesBack says whether a verb does something that cannot be undone.
//
// Outward is the declaration's own word for it — everything else Orbit does
// can be undone by asking again, and a pull request is on somebody's GitHub
// the moment it opens. Deleting a task is the one that leaves this machine's
// disk instead of this machine, and it belongs beside them.
func takesBack(name string) bool {
	if name == "task delete" {
		return true
	}

	v, ok := verb.One(name)

	return ok && v.Outward
}

// help is whether the message asks what can be asked.
//
// Not a verb: there is no `orbit help` in the declaration, and a chat is the
// one way in where the list has to be a message rather than a man page.
func help(text string) bool {
	switch strings.TrimSpace(strings.ToLower(text)) {
	case "/help", "/start", "help", "?":
		return true
	}

	return false
}
