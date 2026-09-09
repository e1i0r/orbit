package verb

// The settings table, and the promise each entry makes.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// The synopsis promises a set of keys and assign switches on them. This is
// what keeps the two lists from drifting: a key added to one and not the
// other fails here rather than at a reader's terminal.
func TestEverySettingKeyCanBeSet(t *testing.T) {
	values := map[string]string{
		"language":     "es",
		"autopilot":    "on",
		"unread-cap":   "3",
		"engine":       "claude",
		"model":        "sonnet",
		"flow":         "careful",
		"theme":        "tokyo-night",
		"check-record": "on",

		"budget-task":      "1.50",
		"budget-workspace": "20",
		"quota-floor":      "15",
	}
	for _, key := range settingKeys() {
		value, ok := values[key]
		if !ok {
			t.Fatalf("%q is offered as a key and this test has no value for it", key)
		}

		var cfg store.Settings
		if _, err := assign(words.For("en"), &cfg, key, value); err != nil {
			t.Errorf("set %s %s: %v", key, value, err)
		}
	}
}

// A refusal that does not say what would have worked leaves the reader
// guessing.
func TestARefusedKeyListsTheKeysThereAre(t *testing.T) {
	var cfg store.Settings

	_, err := assign(words.For("en"), &cfg, "colour", "blue")
	if err == nil {
		t.Fatal("a key nothing recognises was accepted")
	}

	for _, key := range settingKeys() {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("the refusal does not offer %q:\n%s", key, err)
		}
	}
}

// TestFlowMarkAnswersEmptyForAnUnclassifiedOrigin covers the default case,
// which flow.List never produces: an unmarkable name gets a blank column
// rather than a panic in a listing.
func TestFlowMarkAnswersEmptyForAnUnclassifiedOrigin(t *testing.T) {
	if got := FlowMark(words.For("en"), flow.OriginUnknown); got != "" {
		t.Errorf("FlowMark(OriginUnknown) = %q, want empty", got)
	}
}
