package task

// The dependency gate: a library the task added is a decision, and this is
// where a person makes it.

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// manifests are the files a dependency arrives in, and how a name is read
// out of a line that was added to one.
//
// A table and not a parser per ecosystem. What this has to answer is "did
// something new appear here, and what is it called", and the added lines of
// a diff answer that in every one of these formats — where a resolver would
// have to be written five times, kept in step with five toolchains, and
// would still be asking a question nobody needed the exact answer to. What
// the reader is owed is the name; whether it resolves is their package
// manager's business.
//
// The key is matched against the file's base name, so a manifest anywhere in
// the tree is one this knows: a monorepo keeps a package.json per workspace,
// and only the root one would be found by a path.
var manifests = map[string]*regexp.Regexp{
	// Both forms of a go.mod requirement: the one-line `require x v1`, and
	// the indented line inside a require block.
	"go.mod":           regexp.MustCompile(`^\s*(?:require\s+)?([\w.\-]+(?:/[\w.\-]+)+)\s+v\S+`),
	"package.json":     regexp.MustCompile(`^\s*"(@?[\w.\-/]+)"\s*:\s*"[^"]+"`),
	"requirements.txt": regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*(?:[=<>~!\[]|$)`),
	"pyproject.toml":   regexp.MustCompile(`^\s*"?([A-Za-z0-9_.\-]+)"?\s*[=><~]`),
	"Gemfile":          regexp.MustCompile(`^\s*gem\s+["']([\w.\-]+)["']`),
	"Cargo.toml":       regexp.MustCompile(`^\s*([\w.\-]+)\s*=`),
	"composer.json":    regexp.MustCompile(`^\s*"([\w.\-/]+)"\s*:\s*"[^"]+"`),
	"pubspec.yaml":     regexp.MustCompile(`^\s*([\w.\-]+)\s*:\s*\S`),
	"build.gradle":     regexp.MustCompile(`^\s*\w+\s+["']([\w.\-]+:[\w.\-]+)`),
}

// newDependencies is every library the task has added and nobody has
// approved, by name, in the order a reader meets them.
//
// A flow that allows them is asked nothing. So is a task that changed no
// manifest, which is nearly all of them — the work here is one map lookup
// per changed file until something in the list actually moves.
func newDependencies(s *store.Store, t Task, f flow.Flow) []string {
	if f.AllowNewDependencies {
		return nil
	}

	found := map[string]bool{}

	for _, r := range openedRepos(s, t) {
		wt, err := s.WorktreeDir(r.Path, t.ID)
		if err != nil {
			continue
		}

		for _, c := range changesIn(r, wt) {
			pattern, ok := manifests[path.Base(c.Path)]
			if !ok {
				continue
			}

			lines, err := r.WorktreeAddedLines(wt, c.Path)
			if err != nil {
				continue
			}

			for _, name := range named(pattern, lines) {
				found[name] = true
			}
		}
	}

	return unapproved(s, t, found)
}

// openedRepos is the repositories the task has a checkout of.
func openedRepos(s *store.Store, t Task) []repo.Repo {
	var out []repo.Repo

	for _, p := range reposOf(s, t) {
		r, err := repo.Open(p)
		if err != nil {
			continue
		}

		out = append(out, r)
	}

	return out
}

// changesIn is what one worktree changed, and nothing when it cannot be
// read: a count that could not be taken is not a dependency that was added.
func changesIn(r repo.Repo, wt string) []repo.Change {
	changes, err := r.WorktreeChanges(wt)
	if err != nil {
		return nil
	}

	return changes
}

// named is the dependency names in a manifest's added lines.
//
// Lines that match nothing are skipped in silence, and there are many of
// them: a version bumped, a brace, a comment, the name of a section. The
// gate is about what appeared, and a line this cannot read a name out of has
// not told it that anything did.
func named(pattern *regexp.Regexp, lines []string) []string {
	var out []string

	for _, line := range lines {
		if m := pattern.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}

	return out
}

// unapproved takes out what a reader has already said yes to, and answers
// the rest in a stable order.
//
// Approval is per name and not per run, which is what makes it worth
// writing down: the same dependency added again by a later phase, or by the
// next attempt at the same task, is a question that was already answered.
func unapproved(s *store.Store, t Task, found map[string]bool) []string {
	if len(found) == 0 {
		return nil
	}

	events, err := Events(s, t)
	if err != nil {
		return nil
	}

	for _, e := range events {
		if e.Kind != record.DependencyApproved {
			continue
		}

		for _, name := range strings.Split(e.Data["names"], ",") {
			delete(found, strings.TrimSpace(name))
		}
	}

	out := make([]string, 0, len(found))
	for name := range found {
		out = append(out, name)
	}

	sort.Strings(out)

	return out
}

// stopForDependencies ends a run that reached for something new, and names
// what it reached for.
//
// It stops rather than asking the model to justify itself. A dependency is a
// decision about what this project now carries — its licence, its
// maintenance, its security updates — and the one thing an agent cannot be
// delegated is whether the project wants to carry it.
func stopForDependencies(s *store.Store, t Task, p flow.Phase, names []string) error {
	text := fmt.Sprintf("Added %s, so phase %q did not run. Approve it with `orbit approve %s`.",
		strings.Join(names, ", "), p.Name, t.ID)

	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskNewDependency,
		Text: text,
		Data: map[string]string{
			"names": strings.Join(names, ","),
			"phase": p.Name,
		},
	})

	return fmt.Errorf("task %s: %s", t.ID, text)
}

// Approve writes down that a reader accepted the dependencies a task added,
// so the run it stopped can be started again and go past this gate.
//
// The names are what was pending when they answered, carried on the event
// rather than left to be worked out again: a reader approves what they were
// shown, and a second look that found a different list would be approval
// somebody never gave.
func Approve(s *store.Store, t Task, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("task %s has no dependency waiting to be approved", t.ID)
	}

	return emit(s, t, record.Event{
		Kind: record.DependencyApproved,
		Text: "approved: " + strings.Join(names, ", "),
		Data: map[string]string{"names": strings.Join(names, ",")},
	})
}

// Pending is what a task is waiting to have approved, for the command that
// asks on a reader's behalf.
func Pending(s *store.Store, t Task, f flow.Flow) []string {
	return newDependencies(s, t, f)
}
