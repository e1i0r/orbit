package chat

// Reading a chat message as a request for a verb.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

func en() *words.Printer { return words.For("en") }

// TestAMessageIsTheSameWordsAsACommandLine is the whole claim of this
// package: somebody who learned the vocabulary at a terminal types the same
// thing into a phone and means it.
func TestAMessageIsTheSameWordsAsACommandLine(t *testing.T) {
	for _, c := range []struct {
		said string
		verb string
		task string
		args map[string]string
	}{
		{said: "/board", verb: "board"},
		{said: "/board list", verb: "board list"},
		{said: "/task show ACME-3", verb: "task show", task: "ACME-3"},
		{
			said: "/task start ACME-3 -engine codex",
			verb: "task start", task: "ACME-3",
			args: map[string]string{"engine": "codex"},
		},
		{
			// A field of words takes the rest of the line, quoted or not,
			// exactly as it does at a terminal.
			said: `/board new -id ACME-12 "the webhook retries on 5xx"`,
			verb: "board new",
			args: map[string]string{"id": "ACME-12", "text": "the webhook retries on 5xx"},
		},
		{
			said: "/task note ACME-3 it needs a test",
			verb: "task note", task: "ACME-3",
			args: map[string]string{"text": "it needs a test"},
		},
		{
			// A switch with nothing after it is the switch turned on.
			said: "/task direct ACME-3 -restart use the retry helper",
			verb: "task direct", task: "ACME-3",
			args: map[string]string{"restart": "true", "text": "use the retry helper"},
		},
	} {
		t.Run(c.said, func(t *testing.T) {
			got, isCommand, err := Read(c.said, en())
			if err != nil {
				t.Fatalf("Read: %v", err)
			}

			if !isCommand {
				t.Fatal("a line starting with a slash was not read as a command")
			}

			if got.Verb != c.verb {
				t.Errorf("it asks for %q, want %q", got.Verb, c.verb)
			}

			if got.In.Task != c.task {
				t.Errorf("it is about task %q, want %q", got.In.Task, c.task)
			}

			for name, want := range c.args {
				if got.In.Args[name] != want {
					t.Errorf("%s is %q, want %q", name, got.In.Args[name], want)
				}
			}

			// Only what was typed reaches the verb. A declared field is not
			// an answer, and a verb that cannot tell empty from absent
			// throws away the thing it was asked to change.
			if len(got.In.Args) != len(c.args) {
				t.Errorf("it passes %v, want only %v", got.In.Args, c.args)
			}
		})
	}
}

// TestProseIsNotACommandAndNotAMistake. A chat is a place people also talk,
// and a line with no slash is neither a verb nor an error.
func TestProseIsNotACommandAndNotAMistake(t *testing.T) {
	got, isCommand, err := Read("dale con codex", en())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if isCommand {
		t.Errorf("prose was read as a command: %+v", got)
	}
}

// TestWhatItRefuses, and that every refusal says what would have worked.
func TestWhatItRefuses(t *testing.T) {
	for _, c := range []struct{ name, said, want string }{
		{"a verb nothing answers to", "/nope ACME-3", "/help"},
		{"a slash and nothing else", "/", "asks for nothing"},
		{"a task verb with no task", "/task show", "none was named"},
		{"a quote nobody closed", `/board new -id A "open`, "never closed"},
		{"more than the verb takes", "/board list extra words", "takes nothing after"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, isCommand, err := Read(c.said, en())
			if err == nil {
				t.Fatal("it was accepted")
			}

			if !isCommand {
				t.Error("a slash line was not read as a command")
			}

			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal is %q, want it to mention %q", err, c.want)
			}
		})
	}
}

// TestAFamilysChildIsBothItsWords. Read the other way round, `/task start`
// runs the parent's listing and the reader never finds out why.
func TestAFamilysChildIsBothItsWords(t *testing.T) {
	got, _, err := Read("/rules keep 2", en())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.Verb != "rules keep" {
		t.Errorf("it asks for %q, want rules keep", got.Verb)
	}
}

// TestTheRecordSaysItCameFromAChat. Four ways in say where they were asked;
// a fifth that said nothing would leave a run six months later with no
// account of what started it.
func TestTheRecordSaysItCameFromAChat(t *testing.T) {
	got, _, err := Read("/board", en())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.In.Door != door || got.In.By != "operator" {
		t.Errorf("it was asked by %q through %q", got.In.By, got.In.Door)
	}
}

// TestTheMenuSpellingIsAcceptedBack. A service will not take a space in a
// command, so what a reader taps arrives as one word — and the tapped
// command and the typed one have to be the same thing.
func TestTheMenuSpellingIsAcceptedBack(t *testing.T) {
	for _, one := range Menu(en()) {
		got, isCommand, err := Read("/"+one.Name+shapeFor(t, one.Name), en())
		if err != nil {
			t.Errorf("/%s is offered in the menu and refused when tapped: %v", one.Name, err)
			continue
		}

		if !isCommand {
			t.Errorf("/%s was not read as a command", one.Name)
			continue
		}

		if want := strings.ReplaceAll(one.Name, "_", " "); got.Verb != want {
			t.Errorf("/%s asked for %q, want %q", one.Name, got.Verb, want)
		}
	}
}

// shapeFor is enough of a line to satisfy what a verb must have, so that the
// test above is about the spelling of the name and not about its arguments.
func shapeFor(t *testing.T, name string) string {
	t.Helper()

	v, ok := verb.One(strings.ReplaceAll(name, "_", " "))
	if !ok {
		t.Fatalf("%q is in the menu and not in the declaration", name)
	}

	line := ""
	if v.OnTask {
		line += " ACME-1"
	}

	for _, f := range v.Takes {
		if f.Needed {
			line += " x"
		}
	}

	return line
}
