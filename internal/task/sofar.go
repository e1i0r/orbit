package task

// What the attempt before this one got as far as doing.
//
// A phase that died halfway leaves its work in the worktree, and the worktree
// is never lost. What is lost is the account of it: whoever picks the phase up
// opens a folder of half-finished changes with nothing saying which of them
// were meant, what had already been tried, or why it stopped. The engine that
// wrote them could answer — it has a session — but the engine that arrives
// after a change of engine has none, and that is the whole case for writing it
// down somewhere both of them can read.
//
// Built out of what is already recorded: the files the worktree holds, the
// commands the record says were run, the tool calls the permissions refused,
// and the event the attempt ended on. Not out of a model. A summary written by
// a model is a second thing Orbit spends money on unasked, and the free one is
// enough — "what has been done" is a question the diff answers and cannot be
// wrong about.
//
// Which is the rule the rest of this file is written to. A summary that says
// something was done when it was not is worse than no summary, because the
// reader takes it as given and builds on top of it. So nothing here is
// inferred: every line is a fact off the record or off the disk, and the
// headings claim only what their source can prove. The worktree holds changes
// from every phase that came before, not only from the last attempt, and it
// says so rather than crediting them to the attempt.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// How much of it is told.
//
// A summary that fills half a context window is a summary that spends what it
// came to save. These are the counts at which the reader — a model reading
// after its predecessor — has the shape of what happened: what the tree holds,
// and what was tried on it.
const (
	atMostFiles    = 20
	atMostCommands = 12
	atMostRefused  = 5
)

// soFar is what the attempt before this one at the same phase got as far as
// doing, and nothing at all when this is the first attempt.
//
// Nothing at all is the ordinary case, and it is deliberate. A heading with no
// facts under it is a question the model has to spend a turn answering — did
// the attempt before do nothing, or did this program lose what it did? The
// same reasoning bounds the section to this phase: changes an earlier phase
// made are already accounted for by what that phase answered, and a summary
// that swept them up would be telling the reader twice.
func soFar(s *store.Store, t Task, phase, wt string) string {
	was, ok := lastAttempt(s, t, phase)
	if !ok {
		return ""
	}

	var (
		ran, refused = whatItRan(was)
		files        = nowHeld(t, wt)
	)

	if len(files)+len(ran)+len(refused) == 0 {
		return "\n## The attempt before you\n\n" + howItEnded(was) +
			" It left no trace: no files changed, no commands run.\n"
	}

	var b strings.Builder

	b.WriteString("\n## The attempt before you\n\n")
	b.WriteString(howItEnded(was) + " What follows is read off the record and " +
		"off the worktree — what happened, not what was meant to.\n")

	writeFiles(&b, files)
	writeList(&b, "Commands it ran", ran, atMostCommands)
	// What it was denied is the half the reader would otherwise repeat. An
	// attempt that spent three turns reaching for the network under a
	// posture that forbids it will spend three more, and being told is the
	// only thing that stops it.
	writeList(&b, "What it was not allowed to do", refused, atMostRefused)

	return b.String()
}

// lastAttempt is the stretch of the record that belongs to the previous run of
// this phase: everything between the run that started before this one and this
// one's own start.
//
// This run's start is already written by the time the prompt is built — it is
// emitted before the engine is called — so the last start in the record is
// ours, and the one before it opens the attempt being summarised.
func lastAttempt(s *store.Store, t Task, phase string) ([]record.Event, bool) {
	events, err := Events(s, t)
	if err != nil {
		// The summary is a convenience, and a record that will not be read
		// is a bigger problem than a prompt without a section in it. The run
		// goes ahead with what every run had before this existed.
		logger.Error("task/run", "read what the last attempt at phase %q of task %q did: %v",
			phase, t.ID, err)

		return nil, false
	}

	var starts []int

	for i, e := range events {
		if e.Kind == record.PhaseStarted && e.Phase == phase {
			starts = append(starts, i)
		}
	}

	if len(starts) < 2 {
		return nil, false
	}

	opened, ours := starts[len(starts)-2], starts[len(starts)-1]

	return events[opened+1 : ours], true
}

// endings is how an attempt's last event reads as a sentence.
//
// The word for each is the record's own: phase.ran_out is the ending FRA-111
// gave a name to, and a reader told "it ran out of tokens" knows something
// quite different from one told "it broke" — the first means the work was
// going fine and the clock stopped, the second that something was wrong.
var endings = map[string]string{
	record.PhaseRanOut:    "It ran out of tokens before it could finish.",
	record.PhaseFailed:    "It stopped with an error.",
	record.PhaseCancelled: "It was stopped from outside.",
	record.PhaseRetried:   "What it left behind was refused by a gate.",
	record.PhaseWaiting:   "It stopped at a gate and waited.",
	record.PhaseFinished:  "It finished, and this phase is being run again.",
}

// howItEnded is the sentence for the event the attempt ended on, and a plain
// admission when the record does not say.
//
// It does not say when the record holds no terminal event, which happens to an
// attempt whose process was killed outright. Guessing there would be inventing
// the one fact this section exists to get right.
func howItEnded(was []record.Event) string {
	for i := len(was) - 1; i >= 0; i-- {
		if line, named := endings[was[i].Kind]; named {
			return line
		}
	}

	return "It did not get to the end, and the record does not say what stopped it."
}

// nowHeld is what the worktree has that the branch it was cut from does not.
//
// The one account of the state that cannot be wrong. A tool call says a file
// was written; the diff says what is in it now — an attempt that edited a file
// and put it back changed nothing, and a list built from the calls would say
// it had.
func nowHeld(t Task, wt string) []repo.Change {
	if wt == "" || t.Repo.Path == "" {
		return nil
	}

	changes, err := t.Repo.WorktreeChanges(wt)
	if err != nil {
		logger.Error("task/run", "count what the worktree of task %q holds: %v", t.ID, err)
		return nil
	}

	return changes
}

// whatItRan is the commands the attempt ran and the tools it was refused,
// oldest first and each said once.
//
// Once because a model that ran the same test nine times ran it for a reason,
// and nine identical lines say nothing the first one did not — they only spend
// the room the files needed.
func whatItRan(was []record.Event) (ran, refused []string) {
	var (
		sawRan     = map[string]bool{}
		sawRefused = map[string]bool{}
	)

	for _, e := range was {
		switch e.Kind {
		case record.PhaseToolCall:
			if one := command(e); one != "" && !sawRan[one] {
				sawRan[one] = true

				ran = append(ran, one)
			}
		case record.PhaseRefused:
			if one := strings.TrimSpace(e.Data["tool"]); one != "" && !sawRefused[one] {
				sawRefused[one] = true

				refused = append(refused, one)
			}
		}
	}

	return ran, refused
}

// runsACommand are the tools that run something, under the names the engines
// give them.
//
// Named rather than guessed at, because the worth of one of these lines is
// that it is a command somebody could type again. A tool that reads a file is
// not one, and a list of every file the attempt opened would be a list of
// nothing.
var runsACommand = map[string]bool{
	"bash": true, "shell": true, "run": true, "execute": true, "terminal": true,
}

// command is what a tool call ran, and nothing for a call that ran nothing.
func command(e record.Event) string {
	if !runsACommand[strings.ToLower(strings.TrimSpace(e.Data["tool"]))] {
		return ""
	}

	// The first line of it. The arguments are whatever the engine wrote and
	// may be a whole script; what the reader wants is the shape of what was
	// tried, and a heredoc pasted into a prompt is the room the files needed.
	one := strings.TrimSpace(e.Text)
	if cut := strings.IndexByte(one, '\n'); cut >= 0 {
		one = strings.TrimSpace(one[:cut]) + " …"
	}

	if len(one) > lineWidth {
		one = one[:lineWidth] + "…"
	}

	return one
}

// lineWidth is how much of one command is shown: enough to recognise it by,
// short enough that twelve of them are still a list.
const lineWidth = 120

// writeFiles says what the tree holds, under a heading that credits it to the
// tree and not to the attempt — the changes are every phase's, and which of
// them belong to the last attempt is not something the diff knows.
func writeFiles(b *strings.Builder, files []repo.Change) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(b, "\n### What the worktree holds now (%s, this phase "+
		"and every one before it)\n\n", many(len(files), "file"))

	for i, c := range files {
		if i == atMostFiles {
			fmt.Fprintf(b, "- … and %d more\n", len(files)-atMostFiles)

			return
		}

		if c.Binary() {
			fmt.Fprintf(b, "- `%s` (binary)\n", c.Path)

			continue
		}

		fmt.Fprintf(b, "- `%s` +%d −%d\n", c.Path, c.Added, c.Deleted)
	}
}

// writeList is one heading and the first few of what is under it, with a line
// saying how many were left out — because "and 30 more" is itself a fact about
// the attempt.
func writeList(b *strings.Builder, head string, all []string, most int) {
	if len(all) == 0 {
		return
	}

	fmt.Fprintf(b, "\n### %s (%d)\n\n", head, len(all))

	for i, one := range all {
		if i == most {
			fmt.Fprintf(b, "- … and %d more\n", len(all)-most)

			return
		}

		fmt.Fprintf(b, "- `%s`\n", one)
	}
}

// many is a count and the word for what it counts, in the number the count
// calls for. A prompt is read by something that reads English, and "1 files"
// is the kind of seam that makes a reader wonder what else was assembled
// without being looked at.
func many(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}

	return fmt.Sprintf("%d %ss", n, one)
}
