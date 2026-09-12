package learn

// Noticing that a sentence was a rule.
//
// By shape and not by meaning. What somebody types into the supervisor is
// already the answer — it is their sentence, about their code, in the words
// they chose — so a model asked to interpret it could only make it worse,
// and would put a call between a person saying something and Orbit hearing
// it.
//
// The list is short and hand-written, and it is meant to be argued with: a
// shape that turns out to catch nothing comes off, and one that keeps being
// missed goes on. Every one of them is a way of saying "from now on", which
// is the only thing being looked for.

import "strings"

// openings are the ways a rule starts, in English first and Spanish after.
//
// Two languages because the supervisor is talked to in whichever one the
// person thinks in, and a reader whose rules are only noticed in English is
// a reader Orbit does not listen to.
var courtesies = []string{
	"please ", "ok ", "okay ", "hey ", "por favor ", "dale ", "che ",
}

var openings = []string{
	"always ", "never ", "don't ", "do not ", "make sure ", "from now on ",
	"i want ", "i don't want ", "i do not want ", "you must ", "you should ",
	"siempre ", "nunca ", "no quiero ", "quiero que ", "asegurate ",
	"asegúrate ", "de ahora en ", "tenés que ", "tienes que ", "hay que ",
}

// courtesies are what people put in front of a rule without changing it.
//
// One of these is taken off and the openings are tried again, which is the
// whole of what "in the middle of a sentence" needs to mean. Looking for the
// words anywhere at all was the first shape and it read "we never found out
// why the fuzz test hangs" as an order — the same word, in the past, about
// us instead of to the agent.

// aRule is how long a sentence may be and still be read as one.
//
// A rule is short: it is the thing somebody would have typed into a config
// file if there had been one. Past this it is a paragraph — a briefing, a
// question, an explanation — and reading those as rules would fill the tray
// with everything anybody ever said.
const aRule = 240

// aboutTheFuture says whether a sentence is somebody laying down a rule.
//
// It answers about the sentence and never about the person: a line it says
// yes to is offered and not applied, so being wrong here costs a row in a
// tray somebody ignores.
func aboutTheFuture(said string) bool {
	said = strings.ToLower(strings.TrimSpace(said))
	if said == "" || len(said) > aRule {
		return false
	}

	// A question is not a rule however it opens. "should I always run the
	// tests first?" is somebody asking, and answering it by writing down
	// what they asked is the wrong way round.
	if strings.HasSuffix(said, "?") || strings.HasSuffix(said, "¿") {
		return false
	}

	for _, courtesy := range courtesies {
		if after, found := strings.CutPrefix(said, courtesy); found {
			said = after

			break
		}
	}

	for _, opening := range openings {
		if strings.HasPrefix(said, opening) {
			return true
		}
	}

	return false
}
