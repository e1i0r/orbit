package task

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
)

// TestWhatAReviewerSaidBecomesAnEventAndThenAPrompt. A comment pasted into a
// prompt is gone when the run is; a comment in the record outlives the run
// that answers it, which is what a reader six months later is reading.
func TestWhatAReviewerSaidBecomesAnEventAndThenAPrompt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-33", "answer the review", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	n, err := Review(s, tk, r, []repo.Comment{
		{Author: "elio", Path: "internal/task/run.go", Line: 42, Body: "this needs a test"},
		{Author: "elio", Body: "and the name is wrong"},
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if n != 2 {
		t.Errorf("Review recorded %d comments, want 2", n)
	}

	said := unansweredReviews(s, tk)
	if len(said) != 2 {
		t.Fatalf("the next phase would be told %d comments, want 2: %v", len(said), said)
	}

	if !strings.Contains(said[0], "elio") || !strings.Contains(said[0], "internal/task/run.go:42") {
		t.Errorf("a comment does not say who asked or where: %q", said[0])
	}

	if !strings.Contains(said[1], "the pull request") {
		t.Errorf("a comment about the whole pull request does not say so: %q", said[1])
	}
}

// TestAPhaseThatRanHasBeenTold. A comment repeated into every phase of a
// flow is a comment three phases answer separately.
func TestAPhaseThatRanHasBeenTold(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-34", "answer once", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := Review(s, tk, r, []repo.Comment{{Author: "elio", Body: "fix it"}}); err != nil {
		t.Fatalf("Review: %v", err)
	}

	if err := emit(s, tk, record.Event{Kind: record.PhaseStarted, Phase: "fix"}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if said := unansweredReviews(s, tk); len(said) != 0 {
		t.Errorf("a phase that already ran would be told again: %v", said)
	}
}

// TestAnEmptyReviewIsNotAComment. An approval with no words is the most
// common review there is, and listing it would put "nothing to do" in front
// of the things there are to do.
func TestAnEmptyReviewIsNotAComment(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-35", "approved silently", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := Review(s, tk, r, []repo.Comment{{Author: "elio", Body: "   "}}); err != nil {
		t.Fatalf("Review: %v", err)
	}

	if said := unansweredReviews(s, tk); len(said) != 0 {
		t.Errorf("an empty review became something to answer: %v", said)
	}
}

// TestACommentCarriesItsLinkWhenItHasOneAndNoneWhenItDoesNot.
//
// The link is how somebody gets from the row back to the thread it came
// from, and half of these arrive without one — a review left on the pull
// request itself, a comment from a channel that has no permalink. A field
// holding the empty string reads as a link that is there and goes nowhere.
func TestACommentCarriesItsLinkWhenItHasOneAndNoneWhenItDoesNot(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-34", "answer the review", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const link = "https://github.com/e1i0r/orbit/pull/1#discussion_r1"

	if _, err := Review(s, tk, r, []repo.Comment{
		{Author: "elio", Body: "this needs a test", URL: link},
		{Author: "elio", Body: "and the name is wrong"},
	}); err != nil {
		t.Fatalf("Review: %v", err)
	}

	var said []record.Event

	for _, e := range mustEvents(t, s, tk) {
		if e.Kind == record.ReviewComment {
			said = append(said, e)
		}
	}

	if len(said) != 2 {
		t.Fatalf("the record holds %d comments, want 2", len(said))
	}

	if said[0].Data["url"] != link {
		t.Errorf("a comment with a link carries %q", said[0].Data["url"])
	}

	if got, there := said[1].Data["url"]; there {
		t.Errorf("a comment with no link carries url=%q", got)
	}
}
