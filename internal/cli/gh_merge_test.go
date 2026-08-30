package cli

// Merging a delivered task and closing one that was abandoned. Both are one
// call to gh with a sentence around it, and both were reached by no test
// past the flag parsing.

import (
	"slices"
	"strings"
	"testing"
)

// TestMergingSquashesTheBranchAndDeletesIt. What gh is asked for is the
// whole of what merging means here: one commit on the base branch and no
// branch left behind. A flag dropped from that line is a task delivered as
// a merge commit, or a branch that outlives the work.
func TestMergingSquashesTheBranchAndDeletesIt(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "")

	code, out, errOut := run(t, "merge", "-repo", dir, "PAY-1")
	if code != 0 {
		t.Fatalf("merge exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "orbit/PAY-1") {
		t.Errorf("the reader is not told which branch went in: %q", out)
	}

	asked := flagValue(t, argv, "merge")
	if asked != "orbit/PAY-1" {
		t.Errorf("gh was asked to merge %q", asked)
	}

	given := gaveGh(t, argv)
	for _, flag := range []string{"--squash", "--delete-branch"} {
		if !slices.Contains(given, flag) {
			t.Errorf("gh was not given %s: %v", flag, given)
		}
	}
}

// TestClosingCarriesTheCommentToGh. The sentence left on somebody else's
// repository is written in internal/cli, because internal/repo runs git and
// gh and does not write English. This is the other end of that: the comment
// reaches gh, rather than being written and dropped.
func TestClosingCarriesTheCommentToGh(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "")

	code, out, errOut := run(t, "close-pr", "-repo", dir, "PAY-1")
	if code != 0 {
		t.Fatalf("close-pr exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "orbit/PAY-1") {
		t.Errorf("the reader is not told which branch was closed: %q", out)
	}

	if closed := flagValue(t, argv, "close"); closed != "orbit/PAY-1" {
		t.Errorf("gh was asked to close %q", closed)
	}

	if comment := flagValue(t, argv, "--comment"); comment != closingComment(Context{}) {
		t.Errorf("gh was left the comment %q", comment)
	}
}

// TestWhatGhRefusedReachesTheReader. Each of these wraps gh's failure in a
// sentence of its own, and a wrap that dropped the error under it would
// leave the reader a sentence naming the task and no reason for it.
func TestWhatGhRefusedReachesTheReader(t *testing.T) {
	for _, tc := range []struct {
		command string
		reason  string
	}{
		{"merge", "pull request is not mergeable"},
		{"close-pr", "no pull request found for branch"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			dir := deliverable(t, "make the thing")
			fakeGh(t, "echo '"+tc.reason+"' >&2\nexit 1")

			code, _, errOut := run(t, tc.command, "-repo", dir, "PAY-1")
			if code == 0 {
				t.Fatalf("%s exited 0 against a gh that refused", tc.command)
			}

			for _, want := range []string{"PAY-1", tc.reason} {
				if !strings.Contains(errOut, want) {
					t.Errorf("the refusal does not carry %q: %q", want, errOut)
				}
			}
		})
	}
}
