package repo

// The readings taken off a real history.
//
// customs_test.go measures what the two readings say about a list of
// commits handed to them; this is the half that goes to git — how the
// history is asked for, what a repository too young to read answers, and
// whether a commit subject's shape is read off the same log the file names
// come from.
//
// Both of the functions here read zero. A reading that asked git the wrong
// question would come back empty, and empty is also what a young repository
// answers: the two would be indistinguishable without this.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// worked is a checkout with that many commits in it, each one touching the
// files the caller names and carrying the subject they give.
//
// The subjects are a function of the commit's number so a test can ask for
// "all of them conventional" or "half of them" without writing out twenty
// lines.
func worked(t *testing.T, n int, subject func(int) string, files func(int) []string) string {
	t.Helper()

	dir := t.TempDir()

	git := func(args ...string) {
		t.Helper()

		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull,
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	git("init", "-q", "-b", "main")

	for i := range n {
		for _, name := range files(i) {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatalf("make room for %s: %v", name, err)
			}

			body := strings.Repeat("x", i+1)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}

		git("add", "-A")
		git("commit", "-q", "-m", subject(i))
	}

	return dir
}

// held is the reading of that kind, and whether there was one at all.
func held(customs []Custom, kind string) (Custom, bool) {
	for _, one := range customs {
		if one.Kind == kind {
			return one, true
		}
	}

	return Custom{}, false
}

// TestARepositoryTooYoungToReadSaysHowYoungRatherThanNothing.
//
// A repository somebody started last week has no history to read, and that
// is an answer rather than an empty list — the count comes back so the
// caller can say "the last 4 commits" instead of implying it looked at
// forty and found nothing.
func TestARepositoryTooYoungToReadSaysHowYoungRatherThanNothing(t *testing.T) {
	dir := worked(t, 4,
		func(i int) string { return "fix: change " + strings.Repeat("x", i+1) },
		func(int) []string { return []string{"internal/db/query.go"} },
	)

	one, err := Open(dir)
	if err != nil {
		t.Fatalf("open the checkout: %v", err)
	}

	customs, commits, err := one.Customs()
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if commits != 4 {
		t.Errorf("it read %d commits, want the four there are", commits)
	}

	if len(customs) != 0 {
		t.Errorf("a repository of four commits offered %+v", customs)
	}
}

// TestAHistoryLongEnoughIsReadFromGitAndNotGuessed. Both readings come off
// one history: the folders come from --name-only and the shape from the
// subjects, and a reading that asked git the wrong question comes back empty
// — which is also what a young repository answers.
func TestAHistoryLongEnoughIsReadFromGitAndNotGuessed(t *testing.T) {
	dir := worked(t, 22,
		func(i int) string { return "fix: change " + strings.Repeat("x", i+1) },
		func(int) []string { return []string{"internal/db/query.go", "internal/db/query_test.go"} },
	)

	one, err := Open(dir)
	if err != nil {
		t.Fatalf("open the checkout: %v", err)
	}

	customs, commits, err := one.Customs()
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if commits != 22 {
		t.Errorf("it read %d commits, want the twenty-two there are", commits)
	}

	tests, there := held(customs, TestsTravel)
	if !there {
		t.Fatalf("a folder whose every change brought a test was not read: %+v", customs)
	}

	// The top folder and not the leaf: a reading is about a part of the
	// project somebody could name out loud, and "internal/db/query" is a
	// file rather than a place a rule is filed under.
	if tests.Where != "internal" || !tests.Holds() {
		t.Errorf("the reading is %+v, want internal holding", tests)
	}

	shape, there := held(customs, MessagesKeepAShape)
	if !there {
		t.Fatalf("the shape of the subjects was not read: %+v", customs)
	}

	if !shape.Holds() {
		t.Errorf("every subject was written the same way and the reading is %+v", shape)
	}
}

// TestAProjectWithNoHabitIsSaidToHaveNone. A measurement taken and found not
// to hold is the half a document cannot give: it is what says a written rule
// is no longer true, so it comes back rather than being filtered out here.
func TestAProjectWithNoHabitIsSaidToHaveNone(t *testing.T) {
	dir := worked(t, 22,
		func(i int) string {
			if i%2 == 0 {
				return "fix: change " + strings.Repeat("x", i+1)
			}

			return "changed " + strings.Repeat("x", i+1)
		},
		func(i int) []string {
			if i%4 == 0 {
				return []string{"ui/draw.go", "ui/draw_test.go"}
			}

			return []string{"ui/draw.go"}
		},
	)

	one, err := Open(dir)
	if err != nil {
		t.Fatalf("open the checkout: %v", err)
	}

	customs, _, err := one.Customs()
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	for _, reading := range customs {
		if reading.Holds() {
			t.Errorf("a habit kept half the time reads as held: %+v", reading)
		}

		if reading.Of == 0 {
			t.Errorf("a reading was taken over nothing: %+v", reading)
		}
	}
}

// TestAMergeIsGitsSentenceAndNotTheTeams, so it says nothing about the form
// they keep — counted, it would drag every shape below the line on any
// project that merges pull requests.
func TestAMergeIsGitsSentenceAndNotTheTeams(t *testing.T) {
	kept := 0

	for _, subject := range []string{
		"Merge pull request #12 from e1i0r/fix",
		"Merge branch 'main' into fix",
	} {
		if conventional(subject) || ticketed(subject) {
			kept++
		}
	}

	if kept != 0 {
		t.Errorf("%d of git's own sentences read as a shape the team keeps", kept)
	}
}

// TestAHistoryGitWillNotAnswerForIsAFailureAndNotAnEmptyReading. A directory
// that is not a checkout has no history, and answering "no habits" about it
// would be this program saying something it does not know.
func TestAHistoryGitWillNotAnswerForIsAFailureAndNotAnEmptyReading(t *testing.T) {
	_, _, err := Repo{Path: filepath.Join(t.TempDir(), "not-a-checkout"), Name: "nowhere"}.Customs()
	if err == nil {
		t.Error("a directory that is not a checkout answered with a reading")
	}
}
