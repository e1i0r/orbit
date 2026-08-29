package mcp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// bandSlugs names the four bands for a caller that is a program.
//
// view.Band.String() exists for a test failure and says "needs you", with a
// space. A tool argument has to be a token a model can pass straight back,
// so the two spellings are kept apart here rather than one of them being
// bent to serve both — the first version of this package declared
// "needs_you" in the schema and then filtered on String(), so the band
// filter silently matched nothing at all.
var bandSlugs = map[view.Band]string{
	view.ToDo:     "todo",
	view.NeedsYou: "needs_you",
	view.Running:  "running",
	view.Done:     "done",
}

// bandSlug names one band, or says the number when the value is not one of
// the four — an invention would be worse than an obviously wrong token.
func bandSlug(b view.Band) string {
	if s, ok := bandSlugs[b]; ok {
		return s
	}

	return b.String()
}

// bandNames is every slug a band filter accepts, in the order the bands are
// drawn, so the schema and a refusal list them the same way.
func bandNames() []string {
	bands := view.Bands()

	names := make([]string, 0, len(bands))
	for _, b := range bands {
		names = append(names, bandSlug(b))
	}

	return names
}

// parseBand turns a filter argument back into a band.
func parseBand(slug string) (view.Band, error) {
	for b, s := range bandSlugs {
		if strings.EqualFold(s, slug) {
			return b, nil
		}
	}

	return 0, fmt.Errorf("%q is not a band; the bands are %s", slug, strings.Join(bandNames(), ", "))
}

// findTask locates one task by id across every repository on the board.
//
// The repository is optional and is a disambiguator rather than a
// requirement: a model that has just read a row off orbit_list_tasks has the
// id and nothing else, and making it also supply a path it was never given
// is how a tool call becomes a guess. It only has to be supplied when two
// repositories hold a task under the same id.
func findTask(b board.Board, id, repoHint string) (view.Task, error) {
	if id == "" {
		return view.Task{}, fmt.Errorf("this tool needs task_id")
	}

	var matches []view.Task

	for _, t := range b.Tasks {
		if t.ID != id {
			continue
		}

		if repoHint != "" && !sameRepo(t, repoHint) {
			continue
		}

		matches = append(matches, t)
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return view.Task{}, fmt.Errorf("no task %q on the board%s", id, inRepo(repoHint))
	default:
		where := make([]string, 0, len(matches))
		for _, t := range matches {
			where = append(where, t.Repo)
		}

		return view.Task{}, fmt.Errorf("task %q is in %s; say which with repo", id, strings.Join(where, ", "))
	}
}

// sameRepo answers whether a hint picks out this task's repository. A hint is
// either the name the rows carry or the path a caller read off orbit_list_repos,
// and both are accepted because a model has been shown both.
func sameRepo(t view.Task, hint string) bool {
	return strings.EqualFold(t.Repo, hint) || t.RepoPath == hint
}

// inRepo is the trailing clause of a not-found message, and empty when the
// caller named no repository.
func inRepo(hint string) string {
	if hint == "" {
		return ""
	}

	return fmt.Sprintf(" in %q", hint)
}

// openTaskRepo opens the repository a board row belongs to.
//
// It goes through the row's RepoPath rather than through the caller's
// argument: the path on the row is the one the record was folded from, so a
// tool cannot act on a different checkout than the one it just reported.
func openTaskRepo(t view.Task) (repo.Repo, error) {
	r, err := repo.Open(t.RepoPath)
	if err != nil {
		return repo.Repo{}, fmt.Errorf("open the repository of task %s at %q: %w", t.ID, t.RepoPath, err)
	}

	return r, nil
}

// pickRepo resolves the repository a new task is written against.
//
// An empty hint is answered by the board when it holds exactly one
// repository, and refused when it holds several: writing a task into
// whichever repository happened to sort first is the kind of helpfulness
// that files work against the wrong project.
//
// Whatever it resolves to is checked against the session's root, once, at
// the end. The branches below reach the disk by three different routes and
// arguing each of them safe separately is how the one that was not — a path
// Orbit has never seen, opened straight off the filesystem — stayed unsafe.
func (sn Session) pickRepo(b board.Board, hint string) (repo.Repo, error) {
	r, err := chooseRepo(b, hint)
	if err != nil {
		return repo.Repo{}, err
	}

	if err := sn.within(r.Path); err != nil {
		return repo.Repo{}, err
	}

	return r, nil
}

// chooseRepo is the choice itself, without the confinement.
func chooseRepo(b board.Board, hint string) (repo.Repo, error) {
	if hint != "" {
		for _, r := range b.RepoList {
			if strings.EqualFold(r.Name, hint) || r.Path == hint {
				return repo.Open(r.Path)
			}
		}
		// A path Orbit has never seen is still a repository, and refusing
		// the first task against a fresh checkout would make this tool
		// useless for exactly the moment it is most wanted.
		r, err := repo.Open(hint)
		if err != nil {
			return repo.Repo{}, fmt.Errorf("no repository named or at %q: %w", hint, err)
		}

		return r, nil
	}

	switch len(b.RepoList) {
	case 1:
		return repo.Open(b.RepoList[0].Path)
	case 0:
		return repo.Repo{}, fmt.Errorf("orbit knows no repositories yet; say which one with repo")
	default:
		return repo.Repo{}, fmt.Errorf("orbit knows %d repositories; say which one with repo, one of %s", len(b.RepoList), strings.Join(repoNames(b), ", "))
	}
}

// repoNames is every repository on the board, named, for a refusal that
// tells the caller what would have worked.
func repoNames(b board.Board) []string {
	names := make([]string, 0, len(b.RepoList))
	for _, r := range b.RepoList {
		names = append(names, r.Name)
	}

	sort.Strings(names)

	return names
}

// nextTaskID mints an id for a task nobody named.
//
// The shape is the repository's name upper-cased and a number, which is what
// a reader typing `orbit new -id` writes by hand, and the number is one past
// the highest this repository already carries in that shape. It is checked
// against store.ValidTaskID before it is returned, so a repository whose
// name is not a legal id fragment is refused here rather than at the write.
func nextTaskID(s *store.Store, r repo.Repo) (string, error) {
	prefix := idPrefix(r.Name)

	existing, err := task.List(s, r)
	if err != nil {
		return "", fmt.Errorf("list the tasks already in %s: %w", r.Name, err)
	}

	highest := 0

	for _, id := range existing {
		n, ok := suffixNumber(id, prefix)
		if ok && n > highest {
			highest = n
		}
	}

	id := fmt.Sprintf("%s-%d", prefix, highest+1)
	if err := store.ValidTaskID(id); err != nil {
		return "", fmt.Errorf("an id built from repository %q is not usable: %w", r.Name, err)
	}

	return id, nil
}

// idPrefix turns a repository name into the leading fragment of an id: upper
// case, and everything that is not a letter or a digit dropped. A name with
// nothing usable in it falls back to TASK, which is a poor prefix and a
// legal one.
func idPrefix(name string) string {
	var b strings.Builder

	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return "TASK"
	}

	return b.String()
}

// suffixNumber reads the number off an id this package would have minted,
// and says so when the id is not one.
//
// The digits are counted by strconv rather than by multiplying through them,
// which is what this did and is where it went wrong: a directory named
// ORB-99999999999999999999 ran an int past its width and came back as some
// unrelated number, so nextTaskID took that for the highest id in the
// repository and minted its successor. Out of range is not a number this
// package minted, and the answer to that is no.
func suffixNumber(id, prefix string) (int, bool) {
	rest, ok := strings.CutPrefix(id, prefix+"-")
	if !ok {
		return 0, false
	}
	// strconv accepts a sign and this must not: ORB--1 and ORB-+1 are not
	// ids this package has ever written.
	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}

	return n, true
}
