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
