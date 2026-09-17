package verb

// Taking a rule off the disk.

import (
	"errors"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// forgotten removes a rule nobody meant to write, and refuses one that has
// been through anything.
//
// The two answers are different on purpose. A rule that did something is not
// refused because removing it is hard; it is refused because switching it
// off is the thing that was actually wanted — it stops applying and what it
// put you through stays readable — and a refusal that does not say so leaves
// somebody looking for a flag to force it with.
func forgotten(w World, in In) (Out, error) {
	f, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	p := w.Words()

	if err := w.Forget(f); err != nil {
		var did learn.DidSomethingError
		if errors.As(err, &did) {
			return Out{}, errors.New(p.T("verb.rules.forget.did",
				"{rule} has a history: it was {what}. Switch it off instead — it stops applying, "+
					"and what it put you through stays readable",
				words.Arg{Name: "rule", Value: f.ID},
				words.Arg{Name: "what", Value: did.What}))
		}

		return Out{}, err
	}

	return Out{Said: p.T("verb.rules.forgotten",
		"{rule} is gone. It had done nothing, so there was nothing to lose",
		words.Arg{Name: "rule", Value: f.ID})}, nil
}
