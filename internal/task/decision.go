package task

// A decision is what somebody chose and why, written into the record where
// the work happened.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

		// The copy beside the code, after the event and never instead of
		// it. A file that could not be written leaves a decision that is
		// still in the record; an event that was never written would leave
		// a file claiming to be a copy of nothing.
		if err := fileDecision(s, t, p, d); err != nil {
			logger.Warn("task/run", "%s: the decision %q was not written into the repository: %v", t.ID, d.ID, err)
		}
	}
}

// OrbitDir is the directory Orbit keeps its own files in inside a
// repository, and the one part of a task's worktree that is not the task's
// work.
//
// It is named here rather than spelled in three places because two gates
// have to skip it: a diff budget that counted Orbit's own writing would be
// Orbit refusing a change it made itself, and a scope check would report the
// decision file as a file the plan never named — which is true, and is not
// the reader's problem.
const OrbitDir = ".orbit"

// fileDecision writes the decision beside the code it governs.
//
// The event is the decision's home and this is a copy: the record is what
// Orbit reads, and the file is what survives outside it. A reader who has
// the repository and not the state root — somebody reviewing the pull
// request, somebody who cloned it a year later — is who the copy is for, and
// it travels in the same commit as the change that decided it.
//
// The path is the decision's id, so a plan that ran twice rewrites its own
// file rather than leaving the repository two files arguing with each other.
func fileDecision(s *store.Store, t Task, p flow.Phase, d decision) error {
	wt, ok := worktreeFor(s, t, d)
	if !ok {
		// A task with no checkout has nowhere to put a copy, and that is
		// not a failure: the decision is in the record, which is where it
		// lives. The copy arrives when the task joins a repository.
		return nil
	}

	dir := filepath.Join(wt, OrbitDir, "decisions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("make %q: %w", dir, err)
	}

	return os.WriteFile(filepath.Join(dir, d.ID+".md"), []byte(decisionFile(t, p, d)), 0o600)
}

// worktreeFor is the checkout a decision's copy belongs in: the repository
// holding the first path it says it governs, and the one the task is being
// worked in when it governs nothing that can be found.
//
// One copy and not one per repository. A decision written into three
// checkouts is three files that will disagree the first time one of them is
// edited, and the scope is what says which of the three the reader will look
// in.
func worktreeFor(s *store.Store, t Task, d decision) (string, bool) {
	var fallback string

	for _, r := range openedRepos(s, t) {
		wt, err := s.WorktreeDir(r.Path, t.ID)
		if err != nil {
			continue
		}

		if fallback == "" {
			fallback = wt
		}

		for _, scope := range strings.Split(d.Scope, ",") {
			if scope = strings.TrimSpace(scope); scope == "" {
				continue
			}

			if _, err := os.Stat(filepath.Join(wt, scope)); err == nil {
				return wt, true
			}
		}
	}

	return fallback, fallback != ""
}

// decisionFile is the copy as a reader meets it: the facts that place it
// first, then what was decided.
//
// Plain lines and not a serialisation format. It is read by people and by
// grep, and the day something reads it back it will be reading a copy — the
// record is where a program asks.
func decisionFile(t Task, p flow.Phase, d decision) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", d.ID)
	fmt.Fprintf(&b, "task: %s\n", t.ID)
	fmt.Fprintf(&b, "phase: %s\n", p.Name)
	fmt.Fprintf(&b, "at: %s\n", time.Now().UTC().Format(time.RFC3339))

	if d.Scope != "" {
		fmt.Fprintf(&b, "scope: %s\n", strings.ReplaceAll(d.Scope, ",", ", "))
	}

	fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(d.Text))

	return b.String()
}
