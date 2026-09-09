package panes

// Everything ever said about a task, in the order it was said.
//
// It is the conversation and not the log: the notes pane shows what a person
// wrote, the timeline shows every event, and this shows the words — a
// person's and each engine's, whichever program they were typed in.
//
// The reason it is a pane of its own is the reason the file behind it
// exists. A task can be walked by claude until its quota runs out, carried
// on by codex, and finished by claude again; what each of them said is the
// only account of why the work is where it is, and it lives in the record
// rather than inside any of them. This is where a reader sees that, and
// `orbit history` is the same reading as a file for the next engine.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// History is the conversation, and beside it which turn each row that folds
// stands for — laid out in one pass for the reason the timeline is.
func History(e Env) ([]string, map[int]int) {
	p := e.Words

	if e.Failed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.Failed)}, nil
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(
			p.T("history.title", "Everything said about this task")),
		"  " + theme.Paint(theme.Dim).Render(p.T("history.subtitle",
			"a person's words and each engine's, whichever program they were typed in")),
		"",
	}

	turns := historyTurns(e)
	if len(turns) == 0 {
		return append(out, "  "+theme.Paint(theme.Dim).Render(p.T("history.empty",
			"nothing has been said about it yet"))), nil
	}

	heads := map[int]int{}
	measure := max(e.Frame.Body.W-4, 20)

	for i, turn := range turns {
		heads[len(out)] = i
		out = append(out, historyRows(turn, measure)...)
	}

	return out, heads
}

// historyTurn is one thing somebody said.
type historyTurn struct {
	by   string
	at   string
	role theme.Role
	text string
}

// historyTurns is the conversation out of the record, in the order it
// happened — the same reading the file behind `orbit history` makes, so a
// reader looking at this pane and an engine handed that file are looking at
// one conversation.
func historyTurns(e Env) []historyTurn {
	var out []historyTurn

	for _, entry := range e.Entries {
		text := strings.TrimSpace(entry.Said())
		if text == "" {
			continue
		}

		switch entry.What() {
		case view.EntryNoted, view.EntryDialogue:
			out = append(out, historyTurn{
				by: spokeBy(entry), at: stamp(entry), role: theme.Live, text: text,
			})
		case view.EntryFinished, view.EntryFailed:
			out = append(out, historyTurn{
				by: spokeBy(entry), at: stamp(entry), role: theme.Accent, text: text,
			})
		default:
			// Tool calls, gates and the rest are what happened rather than
			// what was said. The timeline is where those belong.
		}
	}

	return out
}

// spokeBy is who said it: the person when they did, the engine when one did,
// and the phase itself for what a run wrote.
func spokeBy(e view.Entry) string {
	switch {
	case e.By != "":
		return e.By
	case e.Engine != "":
		return e.Engine
	case e.Phase != "":
		return e.Phase
	}

	return "orbit"
}

// stamp is when, and nothing for an entry whose clock never answered.
func stamp(e view.Entry) string {
	if e.At.IsZero() {
		return ""
	}

	return e.At.Format("Jan 2 15:04")
}

// historyRows is one turn drawn: who and when on its own line, and what they
// said under it.
func historyRows(turn historyTurn, measure int) []string {
	head := "  " + theme.Paint(turn.role).Bold(true).Render(turn.by)
	if turn.at != "" {
		head += " " + theme.Paint(theme.Dim).Render(turn.at)
	}

	out := []string{head}

	for _, line := range strings.Split(markdown.Plain(turn.text), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		for _, wrapped := range cells.Lines(line, measure) {
			out = append(out, "  "+theme.Paint(theme.Dim).Render(wrapped))
		}
	}

	return append(out, "")
}
