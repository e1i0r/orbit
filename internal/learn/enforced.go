package learn

// The rules a checkout is already keeping, whether anybody wrote them down
// or not.
//
// This is the one source that brings a rule with its gate. The others bring
// a sentence and leave the command to whoever keeps it — which is right,
// because inventing a gate is a decision — but here the command exists and
// is already refusing work: a project whose pull requests go red when `go
// vet` does has been keeping that rule for years, and the only thing missing
// was Orbit knowing about it.
//
// It costs nothing and needs no model. A workflow either parses or it does
// not, so there is no sentence to check against a citation and nothing to
// pay for — which is why it can run on its own where the cold reading has to
// be asked for.
//
// What it does not do is keep them. That a repository runs something is not
// the same as wanting Orbit to send work back over it, and the difference is
// a person's to say: these wait in the tray like everything else.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/store"
)

// FromTheGates is what the tray says these came from.
const FromTheGates = "what this repo enforces"

// Enforced offers a rule for each thing the checkout already refuses work
// over, and answers with the ones it had not offered before.
//
// The command is the name. A gate somebody has already been asked about is
// not asked about again however many workflows run it and however often this
// is read — which is what makes it safe to run on every board rather than
// only when somebody asks.
func Enforced(s *store.Store, repo string, gates []Gate) ([]Said, error) {
	if repo == "" {
		return nil, fmt.Errorf("say which checkout to read")
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	var out []Said

	for i, gate := range gates {
		mark := "gate:" + repo + ":" + gate.Command

		answered, err := d.Answered(mark)
		if err != nil {
			return out, err
		}

		if answered {
			continue
		}

		// A second of separation, because the instant is the row's name and
		// two offered in the same read would otherwise collide — the second
		// would be dropped in silence, which is what the insert does with a
		// name it already has.
		out = append(out, Said{
			At:    time.Now().UTC().Add(time.Duration(i) * time.Second),
			Text:  sentence(gate),
			By:    FromTheGates,
			Repo:  repo,
			Gate:  gate.Command,
			Habit: mark,
		})
	}

	for _, one := range out {
		if err := Propose(s, one); err != nil {
			return out, err
		}
	}

	return out, nil
}

// Gate is one thing a checkout already refuses work over, as internal/repo
// read it. It is redeclared here rather than imported for the reason every
// shape in this package is: what crosses into learn is data.
type Gate struct {
	Command string
	Where   string
}

// sentence is how a gate reads as a rule.
//
// The command said back, because that is the whole of it: there is no
// sentence behind one of these that is truer than the command itself, and a
// paraphrase would be Orbit putting words in somebody's mouth about a thing
// it read off a file.
func sentence(g Gate) string {
	return g.Command + " has to pass"
}
