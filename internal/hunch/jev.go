package hunch

// Jev, a system-one model: it answers from a closed set of words with a
// probability on each, in about half a second, and it cannot answer
// anything that was not offered.
//
// The key is read from the environment and never from the settings file.
// That is the rule the chat token already keeps: `orbit settings` prints
// its table to a terminal, to a screen and into a chat, and a secret that
// can be printed is a secret that will be. What the settings hold is
// whether this is switched on and how sure it has to be — two numbers a
// reader should be able to read back.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/env"
)

// endpoint is where the question goes. A variable so a test can point it at
// a server that answers the way the real one does, which is how
// internal/tracker tests Linear.
var endpoint = "https://api.typesafe.ai/v1/systemone"

// deadline is how long one decision is given.
//
// Two seconds against an answer that arrives in about six hundred
// milliseconds: what is being bought here is a decision made while nobody
// is watching, and a supervisor that blocks a run for longer than that has
// spent more than the wait it was meant to end. A decision that does not
// arrive is no decision, and the run waits for a person exactly as it did
// before.
const deadline = 2 * time.Second

// model is the one this asks for. Named here rather than settable, because
// which model answers is not a choice a reader can make anything of yet:
// there is one.
const model = "jev-latest"

// Jev is the port over the real service. Ready reports whether this machine
// can use it at all.
type Jev struct{ key string }

// FromEnv is the port this machine can use, and whether it can.
//
// Both answers rather than an error: no key is not a failure, it is a
// machine that has not been given one, and the run it is asked about goes
// on exactly as it did before. The same shape internal/quota uses for an
// allowance nobody can read.
func FromEnv() (Jev, bool) {
	key := env.Read(env.DecisionKey)

	return Jev{key: key}, key != ""
}

// Ready is whether there is a key to ask with.
func (j Jev) Ready() bool { return j.key != "" }

// question is what is asked about a stop, in the provider's own shape.
//
// The three criteria are written out because the words alone do not carry
// the decision: "done" has to mean "the work answers the task", not "the
// run ended". A model given three bare labels invents the difference
// between them, and what it invents is not what the supervisor acts on.
func question() map[string]any {
	return map[string]any{
		"verdict": map[string]any{
			"type": "choice",
			"instructions": "A coding agent has stopped part-way through this task. Read what it was " +
				"asked for and what it says it did. What should the supervisor do?",
			"criteria": map[string]string{
				string(Done): "the work answers the task: let it stand, or let the run carry on",
				string(Again): "it is incomplete or wrong in a way another run could finish: " +
					"send it round again",
				string(Human): "it cannot be settled by another run: the task is ambiguous, the " +
					"approach is wrong, or somebody has to decide something",
			},
		},
	}
}

// answer is the shape that comes back, with only the fields acted on.
type answer struct {
	Model   string `json:"model"`
	Answers struct {
		Verdict struct {
			Choice     string  `json:"choice"`
			Confidence float64 `json:"confidence"`
		} `json:"verdict"`
	} `json:"answers"`
}

// Decide asks about one stop.
//
// An error is a decision that did not happen, and every caller reads that
// the same way: ask a person. Nothing here retries — a run is already
// stopped and a reader is already waiting, and a second two seconds buys a
// second chance at the same outage.
func (j Jev) Decide(ctx context.Context, about Stop) (Verdict, error) {
	if !j.Ready() {
		return Verdict{}, fmt.Errorf("no %s in the environment", env.DecisionKey)
	}

	body, err := json.Marshal(map[string]any{
		"state":     state(about),
		"model":     model,
		"questions": question(),
	})
	if err != nil {
		return Verdict{}, fmt.Errorf("ask about task %s: %w", about.Task, err)
	}

	ctx, stop := context.WithTimeout(ctx, deadline)
	defer stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Verdict{}, fmt.Errorf("ask about task %s: %w", about.Task, err)
	}

	req.Header.Set("Authorization", "Bearer "+j.key)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Verdict{}, fmt.Errorf("ask about task %s: %w", about.Task, err)
	}

	defer func() { _ = res.Body.Close() }() //nolint:errcheck // the answer is already read

	if res.StatusCode != http.StatusOK {
		return Verdict{}, fmt.Errorf("ask about task %s: the decision engine answered %s",
			about.Task, res.Status)
	}

	var got answer
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		return Verdict{}, fmt.Errorf("read the decision about task %s: %w", about.Task, err)
	}

	return Verdict{
		Choice:     Choice(got.Answers.Verdict.Choice),
		Confidence: got.Answers.Verdict.Confidence,
		Model:      got.Model,
	}, nil
}

// state is what is sent, and the whole of it.
//
// Three fields with names a reader would use, because the model is shown
// this and nothing else: no diff, no file paths beyond whatever the run
// itself said, no record kinds. Cut, because a phase that printed a
// megabyte is a phase whose first pages already say whether it worked.
func state(about Stop) map[string]string {
	said := strings.Join(strings.Fields(about.Said), " ")
	if said == "" {
		said = "(it said nothing)"
	}

	return map[string]string{
		"the task":            cut(about.Asked, 2000),
		"the phase":           cut(about.Phase, 120),
		"what the agent said": cut(said, 3000),
	}
}

// cut is a field trimmed to what a decision needs of it, by runes so a cut
// never lands inside a character.
func cut(text string, most int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= most {
		return string(runes)
	}

	return string(runes[:most]) + "…"
}
