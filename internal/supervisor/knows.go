package supervisor

// What Orbit knows, put in front of the supervisor before it answers.
//
// It is here and not in supervise.go because reading the facts is a
// different job from asking a model a question: this file walks the state
// root and every repository the record has heard of, and supervise.go only
// wants the sentences that came back.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// standing is everything Orbit has learned, across the state root and every
// repository the record has heard of.
//
// Every repository and not the one somebody happens to be in: this supervisor
// is global — it answers about tasks wherever they are — so a rule it was not
// shown is a rule it can contradict.
//
// A store that cannot be read costs the facts and not the answer. The
// sentences are how a rule is explained; the gate is what enforces one, and
// the gate reads its own copy.
func standing(s *store.Store) []knowledge.Fact {
	ks := knowledge.NewStore(s.Root())

	facts, err := ks.Load("")
	if err != nil {
		logger.Error("supervisor", "what orbit knows was not read: %v", err)

		return nil
	}

	// Repos always finishes its listing: a damaged marker costs that one
	// directory and the rest come back beside the error, which is the
	// contract internal/task/join.go reads it by too. Returning here on any
	// error at all meant one unreadable marker took every healthy
	// repository's rules with it — the opposite of what the paragraph above
	// promises. The one failure that leaves nothing to walk is the repos/
	// directory itself refusing to be listed, and that arrives as a
	// *store.ReposError with no repositories at all.
	repos, err := s.Repos()
	if err != nil {
		logger.Error("supervisor", "the repositories to read facts from: %v", err)
	}

	var listing *store.ReposError
	if errors.As(err, &listing) {
		return knowledge.InScope(facts)
	}

	for _, r := range repos {
		own, err := ks.LoadRepo(r.Path)
		if err != nil {
			logger.Error("supervisor", "what orbit knows about %q: %v", r.Path, err)
			continue
		}

		// LoadRepo and not Load: Load reads the state root before the
		// repository, and those facts are already in hand — asking for them
		// once per repository is a walk and a decode per repository, thrown
		// away here.
		//
		// The check stays because a file's own header outranks where it
		// sits: one under this checkout that calls itself general or about a
		// language belongs to no repository, and is not this repository's to
		// tell.
		for _, f := range own {
			if f.Scope.Repo == r.Path {
				facts = append(facts, f)
			}
		}
	}

	return knowledge.InScope(facts)
}

// alreadyKnown is the standing facts, in front of the supervisor while it
// answers.
//
// Two things need it. It directs tasks and answers questions about the work,
// and without the rules it could say something a gate would refuse an hour
// later. And it is what makes the offer above possible at all: telling
// whether the operator is repeating themselves is comparing meaning, which a
// model does and matching text does not — the same thing said in other words
// is not the same string.
func alreadyKnown(facts []knowledge.Fact) string {
	if len(facts) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n## What Orbit already knows\n\n")

	for _, f := range facts {
		if about := reaches(f.Scope); about != "" {
			fmt.Fprintf(&b, "- (%s) %s\n", about, strings.TrimSpace(f.Phrase))
			continue
		}

		fmt.Fprintf(&b, "- %s\n", strings.TrimSpace(f.Phrase))
	}

	return b.String()
}

// reaches is how far one fact goes, and nothing for a fact that goes
// everywhere.
//
// This supervisor is global, so the sentences arrive from every checkout at
// once and a rule about the ledger reads exactly like a rule about all of
// them. Left unsaid, the model applies "the ledger only appends" to the
// service beside it — which is the one mistake a list of rules can make that
// is worse than having no list.
//
// The window says the same thing in internal/ui/fact, and cannot be shared
// with: nothing under internal/ui is importable from here, and a package of
// its own for two switch statements would be a package to keep in step.
func reaches(sc knowledge.Scope) string {
	switch sc.Kind {
	case knowledge.General:
		return ""
	case knowledge.Language:
		return sc.Lang
	case knowledge.Repo:
		return base(sc.Repo)
	case knowledge.Symbol:
		return base(sc.Repo) + "/" + sc.Path + "#" + sc.Symbol
	default:
		return base(sc.Repo) + "/" + sc.Path
	}
}

// base is a repository by its last segment, which is what a reader calls it
// and what the cockpit prints down the side of the same facts.
func base(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")

	return parts[len(parts)-1]
}
