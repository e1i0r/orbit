package cli

// The two doors the supervisor screen writes through: what Orbit knows about
// the code, and a line on one task.
//
// They are here rather than in internal/ui because the window writes nothing
// — it says what the operator meant and something with the state root does
// it. See the layer table in internal/arch.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// learnPort writes down a fact the operator stated.
//
// The source is always Human here, because that is what this door is: a
// person saying something. A fact typed as a sentence brings no check, so
// one asked to stop is still written asking to — the store keeps what was
// meant, and knowledge.Fact.Action is what decides that without a check it
// only warns. The window says so when it confirms.
func learnPort(s *store.Store) func(stops bool, scope, repoPath, phrase string) error {
	return func(stops bool, scope, repoPath, phrase string) error {
		f := knowledge.Fact{
			Scope:  factScope(scope, repoPath),
			Source: knowledge.Human,
			Phrase: phrase,
			Stops:  stops,
			At:     time.Now().UTC(),
		}

		where, err := knowledge.NewStore(s.Root()).Save(f)
		if err != nil {
			return err
		}

		logger.Info("cli/learn", "wrote down %q at %q", phrase, where)

		return nil
	}
}

// factScope turns what the window said into a scope.
//
// An empty scope with no repository to fall back on becomes a general fact
// rather than a refusal: the operator said something true about their work
// and the widest scope is the honest place for it when nothing narrower is
// known. The window's own sentence says which way it went.
func factScope(scope, repoPath string) knowledge.Scope {
	switch {
	case scope == "general", scope == "" && repoPath == "":
		return knowledge.Scope{Kind: knowledge.General}
	case scope != "":
		return knowledge.Scope{Kind: knowledge.Language, Lang: scope}
	default:
		return knowledge.Scope{Kind: knowledge.Repo, Repo: repoPath}
	}
}

// notePort puts a line in one task's notes, found by its id alone.
//
// The id is the whole of what the window has: somebody typing `@ORB-115` in
// the supervisor is looking at a row, not at a path. The board is what turns
// one into the other, the same way every other tool that takes an id does.
func notePort(r *board.Reader, s *store.Store) func(id, text string) error {
	return func(id, text string) error {
		b, _, err := r.Refresh()
		if err != nil {
			return err
		}

		for _, row := range b.Tasks {
			if row.ID != id {
				continue
			}

			opened, openErr := repo.Open(row.RepoPath)
			if openErr != nil {
				return openErr
			}

			t, loadErr := task.Load(s, opened, id)
			if loadErr != nil {
				return loadErr
			}

			return task.Note(s, t, text)
		}

		return fmt.Errorf("no task %s on the board", id)
	}
}

// knowsPort is what the supervisor screen draws down its side: everything
// Orbit has learned that would reach a phase started in this repository.
//
// The repository is the one the window was opened over, resolved once when
// the port is built: the side is about where somebody is working, and the
// task under the cursor moving does not change that.
func knowsPort(s *store.Store, repoPath string) func() []knowledge.Fact {
	return func() []knowledge.Fact {
		facts, err := knowledge.NewStore(s.Root()).Load(repoPath)
		if err != nil {
			logger.Error("cli/learn", "what orbit knows was not read: %v", err)

			return nil
		}

		return knowledge.InScope(facts)
	}
}
