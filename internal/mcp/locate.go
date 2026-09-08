package mcp

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
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

// findTask locates one task by id, and the id is the whole of what it takes.
//
// It is enough now in a way it was not: an id names one task in the whole
// state root, whatever number of repositories that task is worked in. Two
// rows under one id is a picture the board can no longer draw, so there is
// nothing here to disambiguate and nothing for a caller to supply beyond the
// id it read off orbit_list_tasks.
func findTask(b board.Board, id string) (view.Task, error) {
	if id == "" {
		return view.Task{}, fmt.Errorf("this tool needs task_id")
	}

	for _, t := range b.Tasks {
		if t.ID == id {
			return t, nil
		}
	}

	return view.Task{}, fmt.Errorf("no task %q on the board", id)
}

// sameRepo answers whether a hint picks out one of the repositories the task
// is worked in. A hint is either the name the rows carry or the path a caller
// read off orbit_list_repos, and both are accepted because a model has been
// shown both.
//
// Every repository the task joined and not only the one it is filed under: a
// task that was written in payments and worked in ledger is in ledger to
// anybody who watched it happen, and a tool that answered "no such task"
// there would be denying its own report.
func sameRepo(b board.Board, t view.Task, hint string) bool {
	// By path first and whole: the row carries every checkout it joined by
	// path, and a caller who read one off orbit_list_repos means that
	// checkout and not another of the same name. Matching the name it
	// resolves to answered for the wrong one of two payments.
	if slices.Contains(t.RepoPaths, hint) || t.RepoPath == hint {
		return true
	}

	// A hint that is a checkout Orbit knows has had its whole answer above.
	// Falling through to the name it resolves to let the other payments
	// answer for this task, which is the one thing asking by path is for.
	if slices.ContainsFunc(b.RepoList, func(r board.RepoInfo) bool { return r.Path == hint }) {
		return false
	}

	name := repoNamed(b, hint)

	for _, joined := range t.Repos {
		if strings.EqualFold(joined, name) {
			return true
		}
	}

	return strings.EqualFold(t.Repo, name)
}

// repoNamed is a hint as a repository name.
//
// The rows carry names and a caller may have a path, so the path is turned
// into the name the board knows it by. A path the board has never heard of
// is left as it is: it will match nothing, which is the true answer, and it
// is the caller's own words that come back in the message.
func repoNamed(b board.Board, hint string) string {
	for _, r := range b.RepoList {
		if r.Path == hint {
			return r.Name
		}
	}

	return hint
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
		var named []board.RepoInfo

		for _, r := range b.RepoList {
			if r.Path == hint {
				return repo.Open(r.Path)
			}

			if strings.EqualFold(r.Name, hint) {
				named = append(named, r)
			}
		}

		// Refused rather than guessed, the way knownRepo refuses the same
		// question: taking the first of two checkouts called payments meant
		// this tool reported success while working in the other one.
		if len(named) > 1 {
			return repo.Repo{}, fmt.Errorf("orbit knows %d repositories called %q; say which one by path: %s",
				len(named), hint, strings.Join(repoPathsOf(named), ", "))
		}

		if len(named) == 1 {
			return repo.Open(named[0].Path)
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

// repoPathsOf is the paths of the repositories a name did not tell apart.
func repoPathsOf(repos []board.RepoInfo) []string {
	paths := make([]string, 0, len(repos))
	for _, r := range repos {
		paths = append(paths, r.Path)
	}

	sort.Strings(paths)

	return paths
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
