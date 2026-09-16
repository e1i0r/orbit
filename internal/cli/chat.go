package cli

// `orbit chat` — the fifth way in, over a terminal.
//
// The chat itself is internal/chat: it reads a message as a verb, runs it,
// and dresses the answer. Which service carries the words is a Channel, and
// this is the plainest one there is — standard input and standard output.
//
// It is here rather than as a test fixture because it is the thing that
// makes the door real before any service is written: the same loop, the same
// gate, the same confirmations, with the messages typed instead of pushed.
// A service adapter that behaves differently from this one is a bug in the
// adapter.

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/chat"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// atTheTerminal is the chat somebody is already sitting in front of.
type atTheTerminal struct {
	in  io.Reader
	out io.Writer
}

func (atTheTerminal) Name() string { return "terminal" }

// Listen reads a line at a time until the input ends or the reader stops it.
func (t atTheTerminal) Listen(ctx context.Context, said func(chat.Message)) error {
	lines := bufio.NewScanner(t.in)
	for lines.Scan() {
		// The reader asked to stop, which is not a failure of the input.
		if err := ctx.Err(); err != nil {
			return nil //nolint:nilerr // deliberate: a reader leaving is how this ends
		}

		// One conversation and one account, because a terminal is one
		// person: the gate below is satisfied by being here at all.
		said(chat.Message{Where: "terminal", Who: "operator", Text: lines.Text()})
	}

	return lines.Err()
}

func (t atTheTerminal) Say(_ context.Context, _, text string) error {
	_, err := fmt.Fprintln(t.out, text)

	return err
}

// chatting runs the desk over the terminal until the input ends.
func chatting(ctx Context, args []string) error {
	p := ctx.printer()

	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	dir, err := oneDirectory(ctx, fs.Args())
	if err != nil {
		return err
	}

	// Stopped by the reader rather than only by the end of input: a chat
	// is something somebody leaves open, and Ctrl-C is how they close it.
	stopping, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintln(ctx.Out, p.T("chat.terminal_open",
		"type a command, or /help for the list. ctrl-c to leave."))

	desk := chat.Open(chat.Env{
		Words: p,
		// A terminal is the one channel where being here is the
		// permission: whoever is typing already has the machine.
		Allowed: func(string) bool { return true },
		World:   worldFor(dir, p),
	}, atTheTerminal{in: os.Stdin, out: ctx.Out})

	return desk.Serve(stopping)
}

// worldFor opens the machine for one message and closes it again.
//
// Fresh each time rather than held: the store has one write lock, and a chat
// left open all afternoon holding it is a chat that stops every run on the
// machine. Opening it costs a file handle and a moment, once per thing
// somebody asks for.
func worldFor(dir string, p *words.Printer) func() (verb.World, func(), error) {
	return func() (verb.World, func(), error) {
		s, err := store.Open()
		if err != nil {
			return nil, func() {}, err
		}

		r := board.NewReader(s, dir)
		if err := r.Rescan(); err != nil {
			//nolint:errcheck // the rescan's failure is what is being reported
			_ = s.Close()

			return nil, func() {}, fmt.Errorf("look for repositories: %w", err)
		}

		//nolint:errcheck // closing a read-only store on the way out
		return newWorld(s, r, p), func() { _ = s.Close() }, nil
	}
}
