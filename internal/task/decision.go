package task

// A decision is what somebody chose and why, written into the record where
// the work happened.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// decisionAsk is what a plan phase is told to write besides its plan, word
// for word, every time.
//
// Fixed, and not a sentence each flow writes for itself: what comes back has
// to be read by a parser, and a prompt somebody reworded is a section that
// stops parsing on the day they reword it. It is asked of the plan and of
// nothing else — a phase in the middle of implementing does not stop to
// write minutes, and the same decision noted by three phases is one decision
// in the record three times.
//
// It asks for the non-obvious ones and for what was turned down, because
// those are the two a reader six months later cannot reconstruct: what the
// code does is in the code, and what it could have done instead is nowhere
// unless somebody wrote it down.
const decisionAsk = "\n## Decisions\n\n" +
	"End the answer with a `## Decisions` section listing the choices that " +
	"are not obvious from the code itself — a library, a pattern, a tradeoff, " +
	"something you turned down. Leave it out entirely if the plan decided " +
	"nothing of the kind; an empty list is a better answer than an invented one.\n\n" +
	"Write each one as a bullet with these lines under it:\n\n" +
	"```\n" +
	"- id: a-short-slug\n" +
	"  scope: path/one.go, path/two.go\n" +
	"  decision: What was chosen, in one or two sentences.\n" +
	"  rejected: What was turned down, and why.\n" +
	"```\n\n" +
	"`scope` is the paths the decision governs, so that a later change to one " +
	"of them can be checked against it. `rejected` may be left out when " +
	"nothing was.\n"

// decision is one entry of that section, as the record will hold it.
type decision struct {
	ID    string
	Scope string
	Text  string
}

// isPlan reports whether a phase is the one that decides.
//
// By its name, which is the same reading a person makes of a flow file: the
// flows Orbit ships call it `plan` or `1-plan`, and a flow of somebody's own
// that calls its planning phase something else is a flow that has not told
// anybody it plans.
func isPlan(phase string) bool { return strings.Contains(strings.ToLower(phase), "plan") }

// decisionsIn reads the decisions out of a plan's answer.
//
// A bullet opens an entry and the indented lines under it fill it in. An
// entry with no decision line is dropped rather than kept: half an entry is
// worse than none — it puts a line in the record saying a decision was made
// and never says what it was.
//
// The reading is forgiving about everything else. A model that leaves out
// the id gets one made from its own words, a model that leaves out the scope
// gets a decision that governs nothing in particular, and a model that
// writes prose around the section is read for the section.
func decisionsIn(answer string) []decision {
	var (
		out  []decision
		cur  *decision
		best string
	)

	for _, line := range strings.Split(answer, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "##"):
			// A heading ends the section, and opens it when it is the one
			// that was asked for.
			out = append(out, closed(cur)...)
			cur = nil

			if strings.EqualFold(strings.TrimLeft(trimmed, "# "), "decisions") {
				best = "open"
			} else {
				best = ""
			}
		case best != "" && strings.HasPrefix(trimmed, "-"):
			out = append(out, closed(cur)...)
			cur = &decision{}

			fill(cur, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
		case cur != nil:
			fill(cur, trimmed)
		}
	}

	return append(out, closed(cur)...)
}

// fill reads one `key: value` line into the entry it belongs to, and ignores
// a line that is not one.
func fill(d *decision, line string) {
	key, value, found := strings.Cut(line, ":")
	if !found {
		return
	}

	value = strings.TrimSpace(value)

	switch strings.ToLower(strings.TrimSpace(key)) {
	case "id":
		d.ID = slug(value)
	case "scope":
		d.Scope = paths(value)
	case "decision":
		d.Text = value
	case "rejected":
		if value != "" {
			d.Text = strings.TrimSpace(d.Text + "\n\nRejected: " + value)
		}
	}
}

// closed is the entry as it will be recorded, or nothing when it decided
// nothing.
func closed(d *decision) []decision {
	if d == nil || strings.TrimSpace(d.Text) == "" {
		return nil
	}

	if d.ID == "" {
		d.ID = slug(d.Text)
	}

	return []decision{*d}
}

// paths is a scope as one field: the names the model listed, without the
// spaces between them.
func paths(value string) string {
	var out []string

	for _, p := range strings.Split(value, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return strings.Join(out, ",")
}

// slug is a name a later line can point at, made of the first few words when
// the model gave none.
//
// A decision nothing can name is a decision nothing can supersede, which is
// half of what decision.superseded is for.
func slug(text string) string {
	var (
		b     strings.Builder
		words int
		last  rune
	)

	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			last = r
		case last != '-' && b.Len() > 0:
			words++
			if words > 5 {
				return strings.Trim(b.String(), "-")
			}

			b.WriteRune('-')

			last = '-'
		}
	}

	return strings.Trim(b.String(), "-")
}

// noteDecisions writes down what a plan decided.
//
// Best-effort, and said in the log when it fails. What it writes is derived
// from an answer the record already holds whole, so a decision that did not
// land is recoverable from the phase it came out of — where ending the run
// over it would throw away work that is not.
func noteDecisions(s *store.Store, t Task, p flow.Phase, out engine.Result) {
	if !isPlan(p.Name) {
		return
	}

	for _, d := range decisionsIn(out.Output) {
		text, _ := captured(d.Text)
		if err := emit(s, t, record.Event{
			Kind:  record.DecisionMade,
			Phase: p.Name,
			Text:  text,
			Data:  map[string]string{"id": d.ID, "scope": d.Scope},
		}); err != nil {
			logger.Warn("task/run", "%s: the decision %q of phase %q was not written down: %v", t.ID, d.ID, p.Name, err)
		}
	}
}
