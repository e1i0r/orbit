package mcp

// The repositories Orbit knows about: listing them, looking at one, telling
// Orbit about a new one, and taking one off the books.
//
// Until now a repository appeared in the listing as a side effect of the
// first task written against it, and left it never. That is workable from a
// terminal, where the answer to "which repositories?" is the directory you
// are standing in — and useless to a server whose caller is not standing
// anywhere.

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

func (sn Session) listRepos() CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	return reply(map[string]any{
		"repos": reposOf(sb.board),
		"roots": sn.describeRoots(),
	})
}

// inspectRepo answers one repository from three sides: what Orbit has
// written down about it, what the checkout on disk says, and how its tasks
// stand on the board.
//
// They are separate because they fail separately, and the state where they
// disagree is the one this tool is most needed in. A repository whose
// directory has been deleted still has a record — that is exactly what
// orbit_forget_repo is for — and a tool that refused to describe it because
// git could not be asked, or because the board could not be folded over a
// directory that is gone, would hide the thing the caller is cleaning up.
func (sn Session) inspectRepo(args map[string]any) CallToolResult {
	hint := strings.TrimSpace(stringArg(args, "repo"))
	if hint == "" {
		return refuse(fmt.Errorf("this tool needs repo"))
	}

	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	path, err := sn.repoPath(s, hint)
	if err != nil {
		return refuse(err)
	}

	answer := map[string]any{"name": filepath.Base(path), "path": path}
	if r, err := repo.Open(path); err == nil {
		answer["name"] = r.Name
		answer["branch"] = r.Base
		answer["remote"] = r.Remote
	} else {
		answer["checkout_error"] = err.Error()
	}

	b, boardErr := sn.board(s)
	if boardErr == nil {
		maps.Copy(answer, repoTally(b, path))
		return reply(answer)
	}
	// No board is not no repository. The record is still readable one task at
	// a time, and a count with the reason for its missing neighbours is worth
	// more than a refusal.
	answer["board_error"] = boardErr.Error()

	ids, err := task.List(s, repo.Repo{Path: path, Name: filepath.Base(path)})
	if err != nil {
		answer["tasks_error"] = err.Error()
		return reply(answer)
	}

	answer["tasks"] = len(ids)

	return reply(answer)
}

// addRepo records a repository in the state root.
//
// It goes through repo.Open first, which resolves a path inside a checkout
// to the checkout itself and refuses a directory that is not one at all. A
// caller registering ~/src/orbit/internal gets ~/src/orbit recorded, which
// is the same repository the walk would have found and the same key every
// task under it is already filed against.
func (sn Session) addRepo(args map[string]any) CallToolResult {
	path := strings.TrimSpace(stringArg(args, "path"))
	if path == "" {
		return refuse(fmt.Errorf("this tool needs path"))
	}

	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	r, err := repo.Open(path)
	if err != nil {
		return refuse(err)
	}

	if _, err := s.RegisterRepo(r.Path); err != nil {
		return refuse(fmt.Errorf("record the repository at %q: %w", r.Path, err))
	}

	ids, err := task.List(s, r)
	if err != nil {
		return refuse(fmt.Errorf("list the tasks already in %s: %w", r.Name, err))
	}

	return reply(map[string]any{
		"name":    r.Name,
		"path":    r.Path,
		"branch":  r.Base,
		"remote":  r.Remote,
		"tasks":   len(ids),
		"message": fmt.Sprintf("orbit knows %s at %s; it is listed by orbit_list_repos and searched even when this server is started somewhere else", r.Name, r.Path),
	})
}

// forgetRepo removes a repository's record.
//
// It resolves through the state root rather than through the board, because
// the repository being forgotten is routinely one that is no longer on disk,
// and a resolver that had to open the checkout would refuse exactly the case
// this tool is for.
func (sn Session) forgetRepo(args map[string]any) CallToolResult {
	hint := strings.TrimSpace(stringArg(args, "repo"))
	if hint == "" {
		return refuse(fmt.Errorf("this tool needs repo"))
	}

	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	ref, err := knownRepo(s, hint)
	if err != nil {
		return refuse(err)
	}

	ids, err := task.List(s, repo.Repo{Path: ref.Path, Name: filepath.Base(ref.Path)})
	if err != nil {
		return refuse(fmt.Errorf("list the tasks of %q: %w", ref.Path, err))
	}

	if len(ids) > 0 && !boolArg(args, "delete_tasks") {
		return refuse(fmt.Errorf("%q holds %d tasks (%s), and forgetting it deletes their whole record — the only account of what those runs did; pass delete_tasks to do it anyway", ref.Path, len(ids), strings.Join(ids, ", ")))
	}

	dir, err := s.ForgetRepo(ref.Path)
	if err != nil {
		return refuse(err)
	}

	return reply(map[string]any{
		"path":          ref.Path,
		"removed":       dir,
		"tasks_deleted": len(ids),
		"message":       fmt.Sprintf("orbit no longer has a record of %s; the checkout itself and any worktrees under the state root were left where they are", ref.Path),
	})
}

// repoPath resolves a name or a path to the repository it means: the state
// root's own listing first, then the board, then the disk.
//
// The record comes first because it is the only source that survives the
// checkout being deleted, and the board is second because a repository can
// be registered and outside every directory this session walks.
func (sn Session) repoPath(s *store.Store, hint string) (string, error) {
	if ref, err := knownRepo(s, hint); err == nil {
		return ref.Path, nil
	}

	if b, err := sn.board(s); err == nil {
		for _, r := range b.RepoList {
			if strings.EqualFold(r.Name, hint) || r.Path == hint {
				return r.Path, nil
			}
		}
	}

	r, err := repo.Open(hint)
	if err != nil {
		return "", fmt.Errorf("no repository named or at %q: %w", hint, err)
	}

	return r.Path, nil
}

// knownRepo finds a repository in the state root's own listing, by path or
// by the name its directory has.
func knownRepo(s *store.Store, hint string) (store.RepoRef, error) {
	refs, err := s.Repos()
	if len(refs) == 0 && err != nil {
		return store.RepoRef{}, fmt.Errorf("list the repositories orbit knows: %w", err)
	}

	abs, absErr := filepath.Abs(hint)

	var named []store.RepoRef

	for _, ref := range refs {
		if ref.Path == hint || (absErr == nil && ref.Path == abs) {
			return ref, nil
		}

		if strings.EqualFold(filepath.Base(ref.Path), hint) {
			named = append(named, ref)
		}
	}

	switch len(named) {
	case 1:
		return named[0], nil
	case 0:
		return store.RepoRef{}, fmt.Errorf("orbit has no record of a repository named or at %q", hint)
	default:
		return store.RepoRef{}, fmt.Errorf("orbit knows %d repositories called %q; say which one by path: %s", len(named), hint, strings.Join(pathsOf(named), ", "))
	}
}

func pathsOf(refs []store.RepoRef) []string {
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.Path)
	}

	return paths
}

// repoTally is how one repository's tasks stand, counted the same way the
// board counts the whole of them.
func repoTally(b board.Board, path string) map[string]any {
	counts := map[string]int{}
	for _, band := range view.Bands() {
		counts[bandSlug(band)] = 0
	}

	tally := map[string]any{}
	tasks, unread, spend := 0, 0, 0.0

	for _, t := range b.Tasks {
		if t.RepoPath != path {
			continue
		}

		tasks++
		counts[bandSlug(t.Band)]++
		spend += t.Cost
	}

	for _, t := range board.Unreads(b) {
		if t.RepoPath == path {
			unread++
		}
	}

	tally["tasks"] = tasks
	tally["counts"] = counts
	tally["unread"] = unread
	tally["spend"] = spend

	return tally
}
