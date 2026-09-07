package ui

// The verbs about a task, on the menu of the task they are about.
//
// Some had no key and no letter anywhere in the window, so the command line
// was the only place they were listed — and that line is opened on the
// board, where there is no task for them to be about. The rest had a key
// and nothing else, which is a verb only a reader who already knows it can
// find.

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// aTaskWithCommands is one task on the board and the table those three
// commands come from.
func aTaskWithCommands(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{{ID: "ACME-7", Repo: "acme", RepoPath: "/checkouts/acme", Band: view.Done}}
	m.expanded = map[view.Band]bool{view.BandOf(m.board.Tasks[0]): true}
	m.cursor = m.firstTask()
	m.opts.Commands = []Command{
		{Name: "reconcile"},
		{Name: "note", Args: "-repo <dir> <id> <text>", NeedsArgs: true, AboutATask: true},
		{Name: "direct", Args: "-repo <dir> [-restart] <id> <message>", NeedsArgs: true, AboutATask: true},
		{Name: "pr", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "resolve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "merge", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "close-pr", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "approve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true},
		{Name: "permit", Args: "-repo <dir> [-no] <id>", NeedsArgs: true, AboutATask: true},
		{Name: "critical", Args: "-repo <dir> [-off] <id>", NeedsArgs: true, AboutATask: true},
	}

	return m
}

// TestChoosingOneRunsItRatherThanAskingForArguments. A command that needs
// arguments and has none opens the line with its name on it; these have
// theirs, so they run, and the line — which no longer carries them — is
// never reached.
func TestChoosingOneRunsItRatherThanAskingForArguments(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("ACME-7")

	next, cmd := chooseInMenu(t, m, "approve")

	after := asModel(t, next)
	if after.palette.Up() {
		t.Error("approve went to the command line, which is the board's and has no task on it")
	}

	if after.watching == nil || after.watching.name != "approve" {
		t.Fatalf("choosing approve left the watch on %v, want approve running", after.watching)
	}

	if cmd == nil {
		t.Error("approve was watched with nothing to run")
	}
}

// TestAVerbThatTakesAMessageOpensTheBoxToTypeItIn. direct and note are not
// run from the menu the way approve is: there is a sentence to write, and
// the menu has nothing to fill it in with.
func TestAVerbThatTakesAMessageOpensTheBoxToTypeItIn(t *testing.T) {
	for _, verb := range []string{"note", "direct"} {
		m := aTaskWithCommands(t).openMenu("ACME-7")

		next, _ := chooseInMenu(t, m, verb)

		after := asModel(t, next)
		if !after.note.open {
			t.Fatalf("choosing %s left no box to type the message in", verb)
		}

		if after.note.verb != verb {
			t.Errorf("the box was opened for %q, want %q", after.note.verb, verb)
		}

		if after.note.taskID != "ACME-7" {
			t.Errorf("the box is about %q, want ACME-7", after.note.taskID)
		}

		if after.watching != nil {
			t.Errorf("%s was run with no message, and the box was up asking for one", verb)
		}
	}
}

// TestWhatIsTypedGoesToTheVerbTheBoxWasOpenedFor, with the repository and
// the id in front of it and nothing between them and the message.
func TestWhatIsTypedGoesToTheVerbTheBoxWasOpenedFor(t *testing.T) {
	m := aTaskWithCommands(t).openMessage("direct", "ACME-7")
	m.note.text = "stop and ask first"

	var (
		ran  string
		args []string
	)

	m.opts.Do = func(name string, a []string, _ io.Writer) error {
		ran, args = name, a
		return nil
	}

	next, cmd := m.submitNote()
	if cmd == nil {
		t.Fatalf("the box ran nothing and said %q", asModel(t, next).message)
	}

	for _, one := range commandsIn(t, cmd) {
		one()
	}

	if ran != "direct" {
		t.Errorf("the message went to %q, want direct", ran)
	}

	want := []string{"-repo", "/checkouts/acme", "ACME-7", "stop and ask first"}
	if !slices.Equal(args, want) {
		t.Errorf("direct was run with %v, want %v", args, want)
	}
}

// TestTheBoxSaysWhichVerbItIsFor. A directive stops the run that is going
// and a note does not, and the box a reader is typing into is the last place
// that difference can still be read.
func TestTheBoxSaysWhichVerbItIsFor(t *testing.T) {
	m := aTaskWithCommands(t)

	note := m.openMessage("note", "ACME-7").boxWords()
	direct := m.openMessage("direct", "ACME-7").boxWords()

	if note.title == direct.title || note.prompt == direct.prompt {
		t.Errorf("the box calls itself the same thing either way: %q / %q", note.title, direct.title)
	}

	if !strings.Contains(direct.title, "ACME-7") {
		t.Errorf("the box does not say which task it is about: %q", direct.title)
	}
}

// TestStartingARunIsOnTheMenuAsWell. `orbit run` is about a task like every
// other verb here, and the window's answer to it is a dialog rather than a
// command run bare — so the entry sends the key that opens the dialog.
func TestStartingARunIsOnTheMenuAsWell(t *testing.T) {
	m := aTaskWithCommands(t).openMenu("ACME-7")

	next, _ := pointAt(t, m, m.keys.Start.Help().Key).chooseMenu()

	after := asModel(t, next)
	if after.screen != screenStart {
		t.Errorf("choosing it left the window on %v, want the start dialog", after.screen)
	}
}

// TestARunIsStartedFromTheTasksOwnScreenToo. The key was answered on the
// board and nowhere else, so a reader looking at a task that was abandoned —
// which is read here, not on the board — had to go back to start it again.
func TestARunIsStartedFromTheTasksOwnScreenToo(t *testing.T) {
	m := aTaskWithCommands(t)

	m, _ = m.openDetail(m.board.Tasks[0])

	next, _ := m.detailKey(keystroke(m.keys.Start.Help().Key))

	after := asModel(t, next)
	if after.screen != screenStart {
		t.Errorf("the start key on the task's screen left the window on %v, want the start dialog", after.screen)
	}

	if after.start.id != "ACME-7" {
		t.Errorf("the dialog opened on %q, want ACME-7", after.start.id)
	}
}
