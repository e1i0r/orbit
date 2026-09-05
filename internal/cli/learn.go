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

// knowsAllPort is everything Orbit has learned, for the screen that lists it
// whole: the state root's own facts and every repository on the board.
//
// The board is asked rather than a fixed list, so a repository that gains its
// first fact appears the moment the screen is opened again.
func knowsAllPort(r *board.Reader, s *store.Store) func() []knowledge.Fact {
	return func() []knowledge.Fact {
		ks := knowledge.NewStore(s.Root())

		facts, err := ks.Load("")
		if err != nil {
			logger.Error("cli/learn", "what orbit knows was not read: %v", err)

			return nil
		}

		b, _, err := r.Refresh()
		if err != nil {
			logger.Error("cli/learn", "the repositories to read facts from: %v", err)

			return knowledge.Every(facts)
		}

		for _, repo := range b.RepoList {
			own, err := ks.Load(repo.Path)
			if err != nil {
				logger.Error("cli/learn", "what orbit knows about %s: %v", repo.Name, err)
				continue
			}

			facts = append(facts, onlyOf(repo.Path, own)...)
		}

		// Every and not InScope: this is the screen where a fact is turned
		// back on, and one that hid what was off could not.
		return knowledge.Every(facts)
	}
}

// onlyOf keeps the facts that belong to one checkout. Load answers the state
// root's facts as well, and adding those once per repository would list the
// general ones as many times as there are checkouts.
func onlyOf(repoPath string, facts []knowledge.Fact) []knowledge.Fact {
	kept := make([]knowledge.Fact, 0, len(facts))

	for _, f := range facts {
		if f.Scope.Repo == repoPath {
			kept = append(kept, f)
		}
	}

	return kept
}

// turnFactPort switches a fact off, or on again.
//
// Nothing about it moves, so Save writes over the file the fact is already
// in: the path is made from the scope and the reference, and neither changed.
func turnFactPort(s *store.Store) func(knowledge.Fact) error {
	return func(f knowledge.Fact) error {
		where, err := knowledge.NewStore(s.Root()).Save(f)
		if err != nil {
			return err
		}

		logger.Info("cli/learn", "turned %q at %q, off=%v", f.Phrase, where, f.Off)

		return nil
	}
}

// replaceFactPort writes a corrected fact and takes away the one it replaces.
//
// Replace and not Save, because correcting the sentence moves the file: a
// fact with no reference is filed under a slug of what it says. Saving alone
// would leave the old copy behind, still told and still refusing work.
func replaceFactPort(s *store.Store) func(was, now knowledge.Fact) error {
	return func(was, now knowledge.Fact) error {
		where, err := knowledge.NewStore(s.Root()).Replace(was, now)
		if err != nil {
			return err
		}

		logger.Info("cli/learn", "replaced %q with %q at %q", was.Phrase, now.Phrase, where)

		return nil
	}
}
