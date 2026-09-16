package panes

// The prompt a phase was given, word for word.
//
// The one thing Orbit sends and the only one that used to be invisible. Every
// other side of a run can be read back — the diff, the turns, the tool calls,
// what the engine answered — so a phase that came out strange could be
// examined from every direction except the one that caused it.
//
// Word for word and not summarised. A pane that drew the headings would be
// drawing what this program believes it sent, and the reason to look at all
// is that the two might differ.
//
// The last attempt first, because a phase that ran three times ran three
// times for a reason and the reader is almost always asking about the run
// they just watched. The ones before it are under it, in the order they
// happened.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Prompt is every prompt this run was given, newest first, and beside it
// which one each row that folds stands for.
func promptPane(e Env) ([]string, map[int]int) {
	p := e.Words
	if e.Failed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.Failed)}, nil
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(
			p.T("prompt.title", "What each phase was asked")),
		"  " + theme.Paint(theme.Dim).Render(p.T("prompt.subtitle",
			"the prompt as the engine received it, word for word")),
		"",
	}

	asked := askedOf(e)
	if len(asked) == 0 {
		return append(out, "  "+theme.Paint(theme.Dim).Render(p.T("prompt.none",
			"no phase of this run has been given a prompt yet"))), nil
	}

	heads, measure := map[int]int{}, max(e.Frame.Body.W-4, 20)

	for i := len(asked) - 1; i >= 0; i-- {
		heads[len(out)] = i
		out = append(out, promptRows(p, asked[i], measure)...)
	}

	return out, heads
}

// askedOf is every prompt in the record, oldest first.
func askedOf(e Env) []view.Entry {
	var out []view.Entry

	for _, entry := range e.Entries {
		if entry.What() == view.EntryAsked {
			out = append(out, entry)
		}
	}

	return out
}

// promptRows is one prompt: which phase, who was given it, how big it was,
// and then the thing itself.
func promptRows(p *words.Printer, one view.Entry, measure int) []string {
	body := strings.Split(strings.TrimRight(one.Said(), "\n"), "\n")

	head := fmt.Sprintf("  %s  %s  %s",
		theme.Paint(theme.Accent).Bold(true).Render(nameOr(one.Phase, "?")),
		theme.Paint(theme.Dim).Render(one.Engine),
		theme.Paint(theme.Dim).Render(size(p, len(body), one.Truncated())),
	)

	out := []string{head, ""}

	// Kept as it was written, wrapped and not cut. A prompt is markdown with
	// fenced blocks in it, and a pane that clipped every line to the measure
	// would be showing something the engine was never handed.
	for _, line := range body {
		if line == "" {
			out = append(out, "")

			continue
		}

		for _, wrapped := range cells.WrapKeeping(line, measure) {
			out = append(out, "    "+theme.Paint(theme.Dim).Render(wrapped))
		}
	}

	return append(out, "")
}

// size is how long the prompt was, and whether the record kept all of it.
//
// Lines and not bytes, because the reader is about to scroll through them
// and a figure in kilobytes tells them nothing about how far that is.
func size(p *words.Printer, lines int, cut bool) string {
	line := p.T("prompt.size", "{n} lines", about("n", strconv.Itoa(lines)))
	if cut {
		line += " · " + p.T("prompt.truncated", "the record kept less than was sent")
	}

	return line
}

// nameOr is a value, or a stand-in when the record has none.
func nameOr(name, fallback string) string {
	if name == "" {
		return fallback
	}

	return name
}
