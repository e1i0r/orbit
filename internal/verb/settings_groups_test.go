package verb

import (
	"testing"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// TestEverySettingIsInExactlyOneGroup. A setting in no group is listed last
// under no heading, and one in two would be listed twice.
func TestEverySettingIsInExactlyOneGroup(t *testing.T) {
	seen := map[string]int{}

	for _, g := range settingGroups() {
		for _, name := range g.members {
			seen[name]++
		}
	}

	for _, key := range settingKeys() {
		if seen[key] != 1 {
			t.Errorf("%q is in %d groups, want exactly one", key, seen[key])
		}

		delete(seen, key)
	}

	for name := range seen {
		t.Errorf("a group names %q, which is not a setting", name)
	}
}

// TestSettingsAreListedByGroup: the queue's two limits side by side under
// Queue, and the decision engine's two under Decisions.
func TestSettingsAreListedByGroup(t *testing.T) {
	all := Kept(words.For("en"), store.Shipped())

	if all[0].Name != "max-running" || all[0].Group != "Queue" {
		t.Errorf("the list starts with %s under %q, want max-running under Queue",
			all[0].Name, all[0].Group)
	}

	for i, s := range all {
		if s.Name == "decisions" && (i+1 >= len(all) || all[i+1].Name != "decision-floor") {
			t.Error("decision-floor does not follow decisions")
		}
	}
}
