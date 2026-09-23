package panes

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAVerbStillOutShowsWhatItIsDoing. Elio watched CREATE PR say "in
// progress" for twenty-seven minutes with no way to tell what it was doing.
// The steps its carrier takes are on its node, newest last, and the newest
// is on the band's sentence too.
func TestAVerbStillOutShowsWhatItIsDoing(t *testing.T) {
	e := tree(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(3 * time.Minute)},
		{Kind: "deliver.step", Verb: "CREATE PR", Tool: "Bash", Text: `{"command":"git push -u origin ORB-121"}`},
		{Kind: "deliver.step", Verb: "CREATE PR", Tool: "Bash", Text: `{"command":"gh pr create --fill"}`},
	})

	got := text(rowsOf(Pipeline(e)))
	for _, want := range []string{"git push", "gh pr create"} {
		if !strings.Contains(got, want) {
			t.Errorf("the node of a verb still out does not show %q:\n%s", want, got)
		}
	}

	st, out := Waiting(e)
	if !out {
		t.Fatal("CREATE PR is not out")
	}

	if said := WithDoing("supervisor is working on CREATE PR", st); !strings.Contains(said, "gh pr create") {
		t.Errorf("the band says %q, want the step in hand", said)
	}
}
