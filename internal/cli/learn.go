package cli

// The two doors the supervisor screen writes through: what Orbit knows about
// the code, and a line on one task.
//
// They are here rather than in internal/ui because the window writes nothing
// — it says what the operator meant and something with the state root does
// it. See the layer table in internal/arch.

import (
	"errors"
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// learnPort writes down a fact the operator stated.
//
// The source is always Human here, because that is what this door is: a
// person saying something. A fact typed as a sentence brings no check, so
// one asked to stop is still written asking to — the store keeps what was
// meant, and knowledge.Rule.Action is what decides that without a check it
// only warns. The window says so when it confirms.
func learnPort(s *store.Store) func(stops bool, scope, repoPath, phrase string) error {
	return func(stops bool, scope, repoPath, phrase string) error {
		f := knowledge.Rule{
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
func knowsPort(s *store.Store, repoPath string) func() []knowledge.Rule {
	return func() []knowledge.Rule {
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
func knowsAllPort(r *board.Reader, s *store.Store) func() []knowledge.Rule {
	return func() []knowledge.Rule {
		ks := knowledge.NewStore(s.Root())

		// Beside the error: Load answers with the facts it could read, so a
		// file with a typo in its header costs that file and leaves the
		// screen listing everything else.
		facts, err := ks.Load("")
		if err != nil {
			logger.Error("cli/learn", "what orbit knows was not read in full: %v", err)
		}

		b, _, err := r.Refresh()
		if err != nil {
			logger.Error("cli/learn", "the repositories to read facts from: %v", err)

			return knowledge.Every(facts)
		}

		for _, repo := range b.RepoList {
			// LoadRepo and not Load: the state root's facts are already in
			// hand, and Load would walk and decode them again for every
			// checkout on the board only to have them filtered back out.
			own, err := ks.LoadRepo(repo.Path)
			if err != nil {
				logger.Error("cli/learn", "what orbit knows about %s: %v", repo.Name, err)
			}

			facts = append(facts, onlyOf(repo.Path, own)...)
		}

		// Every and not InScope: this is the screen where a fact is turned
		// back on, and one that hid what was off could not.
		return knowledge.Every(facts)
	}
}

// onlyOf keeps the facts that belong to one checkout.
//
// LoadRepo already reads only that checkout's directory, but a file's own
// header outranks where it sits: one filed under this repository that calls
// itself general or about a language belongs to no repository, and listing it
// here would repeat it once per checkout.
func onlyOf(repoPath string, facts []knowledge.Rule) []knowledge.Rule {
	kept := make([]knowledge.Rule, 0, len(facts))

	for _, f := range facts {
		if f.Scope.Repo == repoPath {
			kept = append(kept, f)
		}
	}

	return kept
}

// turnFactPort switches a fact off, or on again.
//
// Replace and not Save, even though nothing about the fact moves. The file a
// person wrote by hand is called whatever they called it, and Save writes to
// the name the fact's own fields produce — which for one of those is a
// second copy, still read, still told, still refusing work.
//
// was and now are the same fact because that is all this gesture has: the
// screen hands over the fact with Off already flipped. What it was before is
// worked out from that, which is the one case where there is nothing to
// work out.
func turnFactPort(s *store.Store) func(knowledge.Rule) error {
	return func(f knowledge.Rule) error {
		where, err := knowledge.NewStore(s.Root()).Replace(f, f)
		if err != nil {
			return err
		}

		if err := learn.Happened(s, learn.Turn{
			Rule: f.ID, What: learn.Stood(f.State), By: learn.Operator, Was: f.Why,
		}); err != nil {
			return err
		}

		logger.Info("cli/learn", "turned %q at %q, state=%v", f.Phrase, where, f.State)

		return nil
	}
}

// replaceFactPort writes a corrected fact and takes away the one it replaces.
//
// Replace and not Save, because correcting the sentence moves the file: a
// fact with no reference is filed under a slug of what it says. Saving alone
// would leave the old copy behind, still told and still refusing work.
func replaceFactPort(s *store.Store) func(was, now knowledge.Rule, where learn.Turn) error {
	return func(was, now knowledge.Rule, where learn.Turn) error {
		ks := knowledge.NewStore(s.Root())

		at, err := ks.Replace(was, now)
		if err != nil {
			return err
		}

		// Read back rather than assumed: a fact somebody wrote by hand had
		// no name until this write, and what happened to it has to be
		// written under the name it actually got.
		if now.ID == "" {
			now.ID = was.ID
		}

		if err := learn.Changed(s, was, now, where); err != nil {
			return err
		}

		logger.Info("cli/learn", "replaced %q with %q at %q", was.Phrase, now.Phrase, at)

		return nil
	}
}

// windowReplacePort is the same write, for a screen that has no run to name.
//
// The knowledge screen is opened over the board and not over a task, so a
// correction taken there is about the rule and about nothing else. The
// command line's own pause can say which task it was in the way at, because
// the reader typed it.
func windowReplacePort(s *store.Store) func(was, now knowledge.Rule) error {
	return func(was, now knowledge.Rule) error {
		return replaceFactPort(s)(was, now, learn.Turn{By: learn.Operator})
	}
}

// ruleStoryPort is what one rule has put you through, in the same sentences
// the command line tells it in.
//
// The same sentences and not a second set. What a rule has cost you is one
// reading, and two surfaces telling it two ways would be two accounts of the
// evidence a person is about to decide on.
//
// A failure answers with nothing rather than with an error: the review is a
// screen, and a screen that refused to draw because a table could not be read
// would be the worse answer.
// forgetRulePort takes a rule off the disk, and refuses one the record has
// anything to say about.
//
// The refusal comes back as the sentence the reader sees, said here because
// this is where the catalogue is: internal/learn decides whether a rule can
// go and carries what it did in a typed error, and the words are this
// layer's.
func forgetRulePort(s *store.Store, p *words.Printer) func(knowledge.Rule) error {
	return func(f knowledge.Rule) error {
		err := learn.Forget(s, f, learn.Operator)

		var did learn.DidSomethingError
		if errors.As(err, &did) {
			return errors.New(p.T("knowledge.forget_did",
				"this one has a history: it was {what}. Switch it off instead — it stops applying, "+
					"and what it put you through stays readable",
				words.Arg{Name: "what", Value: did.What}))
		}

		return err
	}
}

func ruleStoryPort(s *store.Store, p *words.Printer) func(knowledge.Rule) []string {
	return func(f knowledge.Rule) []string {
		turns, err := learn.History(s, f.ID)
		if err != nil {
			logger.Error("cli/learn", "what happened to rule %s was not read back: %v", f.ID, err)

			return nil
		}

		return verb.Story(p, f, turns)
	}
}
