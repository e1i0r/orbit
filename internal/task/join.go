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

	"github.com/e1i0r/orbit/internal/logger"
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

// Joinable is the repository that answers to a name, or a refusal listing
// the names that would have worked.
//
// The name is matched and never interpreted: a model that asks for a
// repository nobody has checked out is told so plainly, because the
// alternative is Orbit deciding which repository somebody meant and joining
// the wrong one — a mistake that shows up as work committed in a project
// that was not part of the task.
func Joinable(s *store.Store, in repo.Repo, name string) (repo.Repo, error) {
	found, err := reachable(s, in)
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
		return repo.Repo{}, fmt.Errorf("no repository called %q in %s, which holds none", name, among(in))
	}

	return repo.Repo{}, fmt.Errorf("no repository called %q in %s, which holds %v", name, among(in), names)
}

// among is what the refusal calls the place it looked in.
//
// A workspace has a directory to name. The repositories Orbit has a record
// of are not a directory and must not be given one: a refusal naming "" or
// "." would send whoever reads it to look in the wrong place for a name that
// was never going to be there.
func among(in repo.Repo) string {
	if where := repo.Workspace(in); where != "" {
		return where
	}

	return "the repositories orbit knows"
}

// reachable is every repository a task may join.
//
// A task that has one takes its neighbours: the workspace is the parent of
// the checkout it was written against, and a walk of it is the candidate set
// every phase has been shown since a repository could join a task at all.
//
// A task with none has no such parent, and $ORBIT_WORKSPACE is the first
// answer — a reader who has named a workspace has answered this question
// whether or not the task has a repository. Failing that, the candidates are
// the repositories the state root has a record of, which is every checkout
// Orbit has been run in. It is a different list read from a different place,
// and it is the only one there is: a task that starts nowhere either gets
// offered these names or is told there is nowhere to go.
func reachable(s *store.Store, in repo.Repo) ([]repo.Repo, error) {
	found, err := repo.Siblings(in)
	if err != nil {
		return nil, err
	}

	if in.Path != "" || len(found) > 0 {
		return found, nil
	}

	return recorded(s)
}

// recorded opens every repository the state root knows about.
//
// A checkout that has been moved or deleted since is left out rather than
// failing the whole list: it is one name a phase is not offered, and the
// other four are worth having. store.Repos reports damage of its own the
// same way — it always finishes the listing — and both faults are logged
// rather than returned, because a candidate set that came back short is not
// a reason to stop a run.
func recorded(s *store.Store) ([]repo.Repo, error) {
	refs, err := s.Repos()
	if err != nil {
		logger.Error("task/join", "read the repositories orbit knows: %v", err)
	}

	found := make([]repo.Repo, 0, len(refs))

	for _, ref := range refs {
		one, openErr := repo.Open(ref.Path)
		if openErr != nil {
			logger.Error("task/join", "open the known repository %q: %v", ref.Path, openErr)
			continue
		}

		found = append(found, one)
	}

	return found, nil
}

// IDEnv is where a running phase is told which task it is running.
//
// The alternative is the prompt naming the id and the model repeating it
// back on the command line, which is a transcription the model has to get
// right and a reader has to notice when it does not. A task acting on a
// neighbouring task's record is not a mistake worth leaving room for.
const IDEnv = "ORBIT_TASK"

// childEnv is what a phase's process is told about the run it belongs to.
//
// The workspace is left unset when there is none to name. A task with no
// repository has no parent directory to offer, and $ORBIT_WORKSPACE set to
// the empty string is not the same as unset: `orbit join` reads it first and
// would take that empty answer as the workspace, walk nothing, and refuse
// every name — instead of falling through to the repositories Orbit knows,
// which for such a task is the whole of where it can go.
func childEnv(t Task) []string {
	env := []string{IDEnv + "=" + t.ID}

	if where := repo.Workspace(t.Repo); where != "" {
		env = append(env, repo.WorkspaceEnv+"="+where)
	}

	return env
}
