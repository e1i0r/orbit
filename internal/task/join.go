package task

// A repository joins a task by being worked in.
//
// There is no step where the scope of a task is declared and no list anybody
// confirms. A repository is on the task because Orbit opened a worktree of
// it for that task, in whichever phase that happened — the first, or the
// fourth, or one a reader started by hand afterwards. The record of what was
// worked in is the scope, so there is no second account of it to keep in
// agreement with the first.

import (
	"fmt"
	"os"
	"slices"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// Join opens a worktree of r for the task and returns where it is.
//
// The checkout comes first and the event after it: a repository has joined
// when there is somewhere to work in it, so a worktree that could not be
// made leaves the task exactly as it was rather than joined to a repository
// it cannot reach.
//
// A worktree that is already there is reused rather than remade, which is
// what makes a second phase, a retry and a reader running the command twice
// all mean the same thing. The event is appended once for the same reason:
// repo.joined says a repository is part of this task's work, and saying it
// again on the fourth phase adds nothing a reader wants.
func Join(s *store.Store, t Task, r repo.Repo) (string, error) {
	joined, err := s.TaskRepos(t.ID)
	if err != nil {
		return "", err
	}

	wt, err := s.CreateWorktreeParent(r.Path, t.ID)
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(wt); statErr != nil {
		if err := r.AddWorktree(wt, "orbit/"+t.ID); err != nil {
			return "", err
		}
	}

	if slices.Contains(joined, r.Path) {
		return wt, nil
	}

	// CreateTaskDir and not JoinRepo: a repository joining is also the
	// state root learning it exists and the task gaining the directory its
	// own files live in, and those three have one caller between them
	// precisely so that a repository can never be half joined.
	if _, err := s.CreateTaskDir(r.Path, t.ID); err != nil {
		return "", err
	}

	if err := emit(s, t, record.Event{
		Kind: record.RepoJoined,
		Data: map[string]string{"repo": r.Name, "path": r.Path},
	}); err != nil {
		return "", err
	}

	return wt, nil
}

// Joinable is the repository of the workspace that answers to a name, or a
// refusal listing the names that would have worked.
//
// The name is matched and never interpreted: a model that asks for a
// repository nobody has checked out is told so plainly, because the
// alternative is Orbit deciding which repository somebody meant and joining
// the wrong one — a mistake that shows up as work committed in a project
// that was not part of the task.
func Joinable(in repo.Repo, name string) (repo.Repo, error) {
	found, err := repo.Siblings(in)
	if err != nil {
		return repo.Repo{}, err
	}

	names := make([]string, 0, len(found))

	for _, r := range found {
		if r.Name == name {
			return r, nil
		}

		names = append(names, r.Name)
	}

	if len(names) == 0 {
		return repo.Repo{}, fmt.Errorf("no repository called %q in %s, which holds none",
			name, repo.Workspace(in))
	}

	return repo.Repo{}, fmt.Errorf("no repository called %q in %s, which holds %v",
		name, repo.Workspace(in), names)
}

// IDEnv is where a running phase is told which task it is running.
//
// The alternative is the prompt naming the id and the model repeating it
// back on the command line, which is a transcription the model has to get
// right and a reader has to notice when it does not. A task acting on a
// neighbouring task's record is not a mistake worth leaving room for.
const IDEnv = "ORBIT_TASK"

// childEnv is what a phase's process is told about the run it belongs to.
func childEnv(t Task) []string {
	return []string{
		IDEnv + "=" + t.ID,
		repo.WorkspaceEnv + "=" + repo.Workspace(t.Repo),
	}
}
