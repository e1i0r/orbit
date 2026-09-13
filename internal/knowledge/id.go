package knowledge

// What a rule is called, for as long as it exists.

import (
	"crypto/rand"
	"encoding/hex"
)

// idBytes is how long a rule's name is: four bytes, written as eight hex
// characters.
//
// Short enough to read out loud and to sit in a header without crowding the
// sentence under it, and wide enough that a person with thousands of rules
// across every repository they have will not see two of them collide. It is
// not a hash of anything — a name derived from the sentence would change
// when the sentence did, which is the whole thing this exists to survive.
const idBytes = 4

// Name is a name nothing else has, for a caller that needs to know what a
// rule will be called before it is written.
//
// Save coins one for a fact that arrives without, which is what every other
// caller wants. This is for the one that has to write down what happened to
// the rule in the same breath as writing the rule: it cannot read the name
// back out of a path, and reading the file again to find out would be a
// second answer to a question that was already settled.
func Name() string { return coin() }

// coin is a name nothing else has.
//
// crypto/rand and not a counter: a counter needs somewhere to keep the last
// number, and the only place to keep it is a file in a checkout that two
// machines can both be holding.
//
// It cannot fail in a way worth handling. rand.Read on every platform Orbit
// runs on either fills the buffer or the process is already in no state to
// write files, and Go's own docs say to treat an error here as fatal — so an
// error would have to become a fact that could not be written for a reason
// nobody could act on.
func coin() string {
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		panic("no randomness on this machine to name a rule with: " + err.Error())
	}

	return hex.EncodeToString(b)
}
