package cli

// The task story as a pull request reads it: the same five fields the
// terminal draws as a tree, drawn here as a diagram.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/task"
)

// storySection is the story in a pull request body, and nothing at all for a
// task that has none.
//
// One datum, two renders. The terminal can draw text and nothing else, and a
// pull request can draw a diagram — so the same five fields become a tree
// there and a flowchart here, and neither is allowed to know something the
// other does not. What must never happen is a second story assembled for
// this side, which is how two accounts of one task come to disagree.
//
// The fields are written under the diagram as well as inside it. Mermaid is
// rendered by GitHub and by almost nothing else: in a terminal `gh pr view`,
// in an email notification, in a client that does not run it, the block is
// eight lines of source, and a reader there is owed the sentences.
func storySection(s *task.Story) string {
	if s == nil {
		return ""
	}

	steps := []struct{ id, label, about string }{
		{"entry", s.Entry, "the way in"},
		{"purpose", s.Purpose, "what it is for"},
		{"symptom", s.Symptom, "what went wrong"},
		{"cause", s.Cause, "why"},
		{"fix", s.Fix, "what was done"},
	}

	var b strings.Builder

	b.WriteString("\n### How it happened\n\n```mermaid\nflowchart TD\n")

	for i, step := range steps {
		fmt.Fprintf(&b, "    %s[%q]\n", step.id, label(step.label))

		if i > 0 {
			fmt.Fprintf(&b, "    %s --> %s\n", steps[i-1].id, step.id)
		}
	}

	b.WriteString("```\n\n")

	for _, step := range steps {
		fmt.Fprintf(&b, "- **%s** — %s\n", step.about, step.label)
	}

	return b.String()
}

// label is a sentence Mermaid can hold inside a quoted node.
//
// A quote inside a quoted label closes it early and turns the rest of the
// sentence into syntax: the diagram fails to render, on GitHub, silently, in
// the one place a reviewer was going to read it. Newlines end the statement
// for the same reason.
func label(text string) string {
	text = strings.ReplaceAll(text, `"`, "'")
	text = strings.ReplaceAll(text, "\n", " ")

	return strings.TrimSpace(text)
}
