package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestListFromTheWorkspaceListsEverything. Orbit is opened on the directory
// above the repositories, so a reader running `orbit list` there is asking
// about the workspace — and being told "not a repository" is an answer about
// git in front of a question about tasks.
func TestListFromTheWorkspaceListsEverything(t *testing.T) {
	root, _ := workspace(t)

	initRepo(t, filepath.Join(root, "app"))

	for repoName, id := range map[string]string{"payments": "ACME-1", "app": "ACME-2"} {
		if code, _, errOut := run(t, "new", "-repo", filepath.Join(root, repoName), "-id", id, "x"); code != 0 {
			t.Fatalf("new %s exited %d: %s", id, code, errOut)
		}
	}

	code, out, errOut := run(t, "list")
	if code != 0 {
		t.Fatalf("list from the workspace exited %d: %s", code, errOut)
	}

	for _, id := range []string{"ACME-1", "ACME-2"} {
		if !strings.Contains(out, id) {
			t.Errorf("the listing leaves out %s:\n%s", id, out)
		}
	}
}

// TestListNamesARepositoryWhenAskedTo.
func TestListNamesARepositoryWhenAskedTo(t *testing.T) {
	root, _ := workspace(t)

	if code, _, errOut := run(t, "new", "-repo", filepath.Join(root, "payments"), "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	initRepo(t, filepath.Join(root, "app"))

	code, out, errOut := run(t, "list", "-repo", filepath.Join(root, "app"))
	if code != 0 {
		t.Fatalf("list of another repository exited %d: %s", code, errOut)
	}

	if strings.Contains(out, "ACME-1") {
		t.Errorf("a task of payments was listed under app:\n%s", out)
	}
}
