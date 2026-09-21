package hunch

// What the port does with what the service answers.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/env"
)

// asked is what the stand-in service was sent, filled when the request
// arrives rather than when the server is set up.
type asked struct {
	auth string
	sent map[string]any
}

// answering is a stand-in for the service that answers with the body given,
// and remembers what it was asked.
func answering(t *testing.T, status int, body string) *asked {
	t.Helper()

	got := &asked{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.auth = r.Header.Get("Authorization")

		if err := json.NewDecoder(r.Body).Decode(&got.sent); err != nil {
			t.Errorf("read what was asked: %v", err)
		}

		w.WriteHeader(status)

		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("answer: %v", err)
		}
	}))

	t.Cleanup(srv.Close)

	was := endpoint
	endpoint = srv.URL

	t.Cleanup(func() { endpoint = was })

	return got
}

// TestAVerdictIsReadBackWithHowSureItWas.
func TestAVerdictIsReadBackWithHowSureItWas(t *testing.T) {
	answering(t, http.StatusOK, `{"model":"jev-1.13.0","answers":{"verdict":
		{"type":"choice","choice":"done","confidence":0.83,
		 "probabilities":{"done":0.83,"again":0.15,"human":0.02}}},
		"usage":{"input_tokens":412,"output_tokens":30}}`)

	verdict, err := Jev{key: "k"}.Decide(context.Background(), Stop{
		Task: "ACME-1", Asked: "make it idempotent", Phase: "review", Said: "make check is green",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}

	if verdict.Choice != Done || verdict.Confidence != 0.83 || verdict.Model != "jev-1.13.0" {
		t.Errorf("the verdict is %+v, want done at 0.83 from jev-1.13.0", verdict)
	}

	if !verdict.Acts(70) || verdict.Acts(90) {
		t.Errorf("a verdict at 0.83 acts under a floor of 90 or does not under 70: %+v", verdict)
	}
}

// TestWhatIsSentIsTheTaskAndWhatItSaid, and nothing else: no record kinds,
// no diff, no paths of Orbit's own.
func TestWhatIsSentIsTheTaskAndWhatItSaid(t *testing.T) {
	got := answering(t, http.StatusOK,
		`{"model":"m","answers":{"verdict":{"choice":"again","confidence":0.5}}}`)

	if _, err := (Jev{key: "k"}).Decide(context.Background(), Stop{
		Task: "ACME-2", Asked: "make it idempotent", Phase: "implement", Said: "it broke",
	}); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	state, ok := got.sent["state"].(map[string]any)
	if !ok {
		t.Fatalf("what was sent is %T, want the three fields of a stop", got.sent["state"])
	}

	if got.auth != "Bearer k" {
		t.Errorf("the key was sent as %q", got.auth)
	}

	if len(state) != 3 || state["the task"] != "make it idempotent" ||
		state["what the agent said"] != "it broke" {
		t.Errorf("what was sent is %v", state)
	}

	// The three words it may answer with, written out where the model can
	// read what each of them means.
	questions, ok := got.sent["questions"].(map[string]any)
	if !ok {
		t.Fatalf("no questions were asked: %v", got.sent)
	}

	verdict, ok := questions["verdict"].(map[string]any)
	if !ok {
		t.Fatalf("the verdict was not asked for: %v", questions)
	}

	criteria, ok := verdict["criteria"].(map[string]any)
	if !ok {
		t.Fatalf("the verdict was asked with no criteria: %v", verdict)
	}

	for _, word := range []Choice{Done, Again, Human} {
		if _, offered := criteria[string(word)]; !offered {
			t.Errorf("%q was not offered as an answer: %v", word, criteria)
		}
	}
}

// TestAPhaseThatSaidNothingSaysSo. An empty state reads to a model as a
// missing field rather than as a run that printed nothing, and those are
// different facts about the work.
func TestAPhaseThatSaidNothingSaysSo(t *testing.T) {
	got := answering(t, http.StatusOK,
		`{"model":"m","answers":{"verdict":{"choice":"human","confidence":0.6}}}`)

	if _, err := (Jev{key: "k"}).Decide(context.Background(), Stop{Task: "ACME-3"}); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	state, ok := got.sent["state"].(map[string]any)
	if !ok {
		t.Fatalf("nothing was sent: %v", got.sent)
	}

	if state["what the agent said"] != "(it said nothing)" {
		t.Errorf("a silent phase was sent as %q", state["what the agent said"])
	}
}

// TestAServiceThatRefusesIsNoDecision, named so a reader of the log knows
// which of the two it was.
func TestAServiceThatRefusesIsNoDecision(t *testing.T) {
	answering(t, http.StatusUnauthorized, `{"error":"bad key"}`)

	_, err := Jev{key: "k"}.Decide(context.Background(), Stop{Task: "ACME-4"})
	if err == nil {
		t.Fatal("a refused request answered as though it were a decision")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("the error is %q, want it to name what the service answered", err)
	}
}

// TestNoKeyIsNoPort. A machine that was never given a key is not a machine
// with a broken decision engine: it is one that has none, and every caller
// reads that the same way.
func TestNoKeyIsNoPort(t *testing.T) {
	t.Setenv(env.DecisionKey, "")

	port, ready := FromEnv()
	if ready || port.Ready() {
		t.Error("a machine with no key answered that it was ready")
	}

	if _, err := port.Decide(context.Background(), Stop{Task: "ACME-5"}); err == nil {
		t.Error("a port with no key answered a decision")
	}

	t.Setenv(env.DecisionKey, "  k  ")

	if port, ready = FromEnv(); !ready || !port.Ready() {
		t.Error("a key with spaces around it read as no key")
	}
}

// TestOffDecidesNothing, which is what a nil port and an unset machine both
// come down to.
func TestOffDecidesNothing(t *testing.T) {
	verdict, err := Off{}.Decide(context.Background(), Stop{Task: "ACME-6"})
	if err != nil {
		t.Fatalf("Off answered an error: %v", err)
	}

	if verdict.Choice.Known() || verdict.Acts(0) {
		t.Errorf("Off answered %+v, want nothing to act on", verdict)
	}
}
