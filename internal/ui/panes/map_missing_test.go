package panes

// A checkout that is not there any more.

import (
	"strings"
	"testing"
)

// TestAMapWithNoCheckoutSaysSo. Before this, the pane drew git's own words —
// "the repository could not be read: git ls-files -z: chdir
// /Users/…/worktrees/5a14f7401345/FRA-71: no such file or directory" — over
// a task whose worktree had simply been cleaned up. Nothing was wrong, and
// nothing a reader could do about it was in the sentence.
func TestAMapWithNoCheckoutSaysSo(t *testing.T) {
	e := world(t, nil)
	e.Shape = Shape{Read: true, Missing: true}

	drawn := strings.Join(Map(e), "\n")
	if !strings.Contains(drawn, "no checkout") {
		t.Errorf("the map says %q, want it to say the checkout is gone", drawn)
	}

	for _, unwanted := range []string{"could not be read", "chdir", "ls-files"} {
		if strings.Contains(drawn, unwanted) {
			t.Errorf("the map says %q, which still carries %q", drawn, unwanted)
		}
	}
}

// TestAMapThatBrokeStillSaysWhy, because a reading that failed for any other
// reason is a fault and the words git chose are the only evidence there is.
func TestAMapThatBrokeStillSaysWhy(t *testing.T) {
	e := world(t, nil)
	e.Shape = Shape{Read: true, Failed: "git ls-files -z: exit status 128"}

	if drawn := strings.Join(Map(e), "\n"); !strings.Contains(drawn, "exit status 128") {
		t.Errorf("the map says %q, want git's own words in it", drawn)
	}
}
