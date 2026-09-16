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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/chat"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
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
	where := fs.String("on", "terminal", "the chat to be reached through: terminal, or telegram")

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

	e := chat.Env{Words: p, World: worldFor(dir, p), Answers: supervising()}

	to, err := reachedThrough(ctx, *where, &e)
	if err != nil {
		return err
	}

	return chat.Open(e, to).Serve(stopping)
}

// tokenEnv is where the bot's token is read from.
//
// The environment and not the settings file: it is a secret, and anybody
// holding it can read and write as the bot. What does live in settings is
// the id of the one conversation it answers, which is a number and not a
// key.
const tokenEnv = "ORBIT_TELEGRAM_TOKEN"

// reachedThrough is the channel the reader asked for, and the gate that goes
// with it.
func reachedThrough(ctx Context, where string, e *chat.Env) (chat.Channel, error) {
	p := ctx.printer()

	switch where {
	case "terminal":
		fmt.Fprintln(ctx.Out, p.T("chat.terminal_open",
			"type a command, or /help for the list. ctrl-c to leave."))

		// A terminal is the one channel where being here is the
		// permission: whoever is typing already has the machine.
		e.Allowed = func(string) bool { return true }

		return atTheTerminal{in: os.Stdin, out: ctx.Out}, nil
	case "telegram":
		token := os.Getenv(tokenEnv)
		if token == "" {
			return nil, errors.New(p.T("chat.no_token",
				"{env} is not set; @BotFather gives you one when you make a bot",
				words.Arg{Name: "env", Value: tokenEnv}))
		}

		who, err := allowedChat(ctx, e)
		if err != nil {
			return nil, err
		}

		fmt.Fprintln(ctx.Out, who)

		return chat.Bot(token), nil
	}

	return nil, fmt.Errorf("%s", p.T("chat.no_such_channel",
		"{name} is not a chat orbit knows; it knows terminal and telegram",
		words.Arg{Name: "name", Value: where}))
}

// allowedChat fills the gate from the settings, and says what it found.
//
// A machine that has not been told who its reader is answers nobody — and
// tells the first person who writes what their own id is, which is the one
// message worth answering a stranger with. Without it, setting this up means
// reading an HTTP API by hand to find a number.
func allowedChat(ctx Context, e *chat.Env) (string, error) {
	p := ctx.printer()

	s, err := store.Open()
	if err != nil {
		return "", err
	}

	defer s.Close() //nolint:errcheck // read once, on the way in

	cfg, err := s.Settings()
	if err != nil {
		return "", err
	}

	if cfg.ChatID == "" {
		e.Allowed = func(string) bool { return false }
		e.Stranger = func(m chat.Message) string {
			return p.T("chat.your_id",
				"nobody is allowed to command this Orbit yet. Your chat id is {id} — "+
					"run `orbit settings set chat-id {id}` and start me again.",
				words.Arg{Name: "id", Value: m.Who})
		}

		return p.T("chat.waiting_for_id",
			"no chat-id is set: write to the bot and it will tell you yours."), nil
	}

	e.Allowed = func(who string) bool { return who == cfg.ChatID }

	return p.T("chat.listening", "listening, and answering {id} only",
		words.Arg{Name: "id", Value: cfg.ChatID}), nil
}

// theThread is the conversation a chat holds with the supervisor.
//
// One and not one per message, so a chat is a conversation rather than a
// series of strangers: the supervisor is handed what was said before it, the
// same way the window's thread hands it its own.
const theThread = "chat"

// supervising is the supervisor, asked a sentence and answering one.
//
// The engine is the one the settings name, because a chat has no dial to
// turn: the window picks an engine per conversation and a phone has nowhere
// to show the choice, so the standing one is the honest answer.
//
// It spends money — one model run per sentence somebody types — and that is
// the whole reason it is here rather than inside internal/chat: a build with
// no engine, or a reader who has not chosen one, gets a chat that writes the
// line down and says so.
func supervising() func(context.Context, string) (string, error) {
	return func(ctx context.Context, said string) (string, error) {
		s, err := store.Open()
		if err != nil {
			return "", err
		}

		defer s.Close() //nolint:errcheck // read and write, closed on the way out

		cfg, err := s.Settings()
		if err != nil {
			return "", err
		}

		eng, err := engineNamed(newEngines(), cfg.Engine)
		if err != nil {
			return "", err
		}

		// And the model the settings name. A chat has no dial to turn, so
		// the standing choice is the only honest one — and until now it was
		// a setting that reached nothing at all.
		return supervisor.SuperviseIn(ctx, s, eng, cfg.Model, theThread, said)
	}
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
