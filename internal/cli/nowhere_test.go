package cli

// Writing a task from a directory that is not a repository.
//
// -repo has defaulted to "." since the first version, which made the current
// directory the answer to a question the command had to have an answer to. A
// task does not need one now, so the default is allowed to come back empty —
// and a -repo the reader typed is still a -repo that has to open, because a
// path somebody got wrong is a mistake and not a decision.

import (
	"strings"
	"testing"
)

// TestATaskWrittenWhereThereIsNoRepositoryIsStillWrittenDown.
func TestATaskWrittenWhereThereIsNoRepositoryIsStillWrittenDown(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Chdir(t.TempDir())

	code, out, errOut := run(t, "new", "-id", "ACME-1", "--", "find out which service owns the retry")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}

	if !strings.Contains(out, "no repository") {
		t.Errorf("the line does not say the task is against none:\n%s", out)
	}

	code, out, errOut = run(t, "show", "ACME-1")
	if code != 0 {
		t.Fatalf("show exit %d: %s", code, errOut)
	}

	if !strings.Contains(out, "which service owns the retry") {
		t.Errorf("the task that was written down cannot be read back:\n%s", out)
	}
}

// TestARepositoryTypedOutStillHasToOpen. The default is a guess about where
// the reader is; -repo is a thing they said. A task quietly written against
// nothing because the path had a typo in it is a task filed somewhere its
// author will not look.
func TestARepositoryTypedOutStillHasToOpen(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	missing := t.TempDir() + "/not-a-checkout"

	code, _, errOut := run(t, "new", "-repo", missing, "-id", "ACME-2", "--", "write the importer")
	if code == 0 {
		t.Fatal("a -repo that is not a repository was accepted")
	}

	if !strings.Contains(errOut, "not-a-checkout") {
		t.Errorf("the refusal does not name the path that was typed:\n%s", errOut)
	}
}

// TestAnIDNothingAnswersToIsRefusedWithoutNamingARepository. The refusal
// named the repository the reader was standing in, which for a reader
// standing outside one is a sentence that ends in "in " — and the
// repository was never why the id found nothing.
func TestAnIDNothingAnswersToIsRefusedWithoutNamingARepository(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Chdir(t.TempDir())

	code, _, errOut := run(t, "show", "ACME-404")
	if code == 0 {
		t.Fatal("show on an id nothing answers to exited 0")
	}

	if !strings.Contains(errOut, "nothing recorded for ACME-404") {
		t.Errorf("the refusal does not say the id found nothing:\n%s", errOut)
	}

	if strings.Contains(errOut, "ACME-404 in") {
		t.Errorf("the refusal names a repository the task does not have:\n%s", errOut)
	}
}
