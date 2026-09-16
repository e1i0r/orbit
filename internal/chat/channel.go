package chat

// What a chat is, as this package needs it.
//
// A port, the way an engine is one. Which service carries the words is an
// adapter of its own — Telegram, Slack, a terminal — and the whole point of
// the seam is that a second one is a file rather than a migration.
//
// The interface is two methods because a chat is two things: somebody says
// something, and Orbit says something back. Everything about tokens,
// long-polling, retries and rate limits belongs to whoever implements it.

import "context"

// A Message is one thing somebody said, and where they said it.
type Message struct {
	// Where is the conversation it arrived in, in whatever the service
	// calls one. It is carried back on the answer so a channel that holds
	// more than one conversation knows which to reply to.
	Where string
	// Who is the account that sent it, as the service names it. It is what
	// the gate below is checked against, and it is written down when a
	// message is turned away.
	Who  string
	Text string
}

// A Channel is a chat Orbit can be reached through.
type Channel interface {
	// Listen hands over every message until the context is done.
	//
	// It blocks, and it is the adapter's job to keep going: a service that
	// is unreachable for a minute is a minute of no messages, never an
	// error that ends the loop. Orbit is a program somebody leaves running.
	Listen(ctx context.Context, said func(Message)) error
	// Say answers into one conversation. An answer that cannot be
	// delivered is logged and dropped: a chat that is down must not be
	// able to stop a run, which is the same rule a gate follows.
	Say(ctx context.Context, where, text string) error
	// Name is what this channel is called, for the log and for the record.
	Name() string
}

// A Command is one thing a chat can be asked for, as a service's own menu
// wants it: a name with no spaces in it, and a line saying what it does.
type Command struct {
	Name  string
	About string
}

// An Announcing channel can show the reader what can be asked for.
//
// Optional, like Working, and for the same reason: a terminal has no menu to
// put one in. What makes it worth a method rather than a message is that a
// service's own menu appears as the reader types, which is the difference
// between a command list somebody has to remember to ask for and one that is
// simply there.
type Announcing interface {
	// Announce replaces whatever menu the channel was showing. It is called
	// once, when the desk opens, because what can be asked for changes only
	// when Orbit itself does.
	Announce(ctx context.Context, all []Command) error
}

// A Working channel can show that an answer is coming.
//
// Optional, and asked for by type rather than declared in Channel, because
// not every service has one: a terminal has nowhere to put it, and a channel
// that had to implement an empty method to say so would be a channel
// pretending.
//
// It matters more here than in most places. A verb answers in a moment; a
// sentence said to the supervisor runs a model, and forty seconds of silence
// on a phone is indistinguishable from a bot that is not running. The
// alternative — a message saying "working on it" — leaves a line in the
// conversation for ever about something that has since finished.
type Working interface {
	// Working says an answer is coming. It is called again while the wait
	// lasts, because the services that have this show it for a few seconds
	// and then stop.
	Working(ctx context.Context, where string) error
}
