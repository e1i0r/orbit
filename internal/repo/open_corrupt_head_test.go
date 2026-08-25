package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCorruptGitHead(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "corrupt_head")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repoDir, "init", "-q", "-b", "main")
	gitCmd(t, repoDir, "config", "user.email", "test@orbit.local")
	gitCmd(t, repoDir, "config", "user.name", "Orbit Tester")
	gitCmd(t, repoDir, "commit", "-q", "--allow-empty", "-m", "init")

	// Writing an invalid ref in .git/HEAD keeps .git valid for --show-toplevel but fails --abbrev-ref HEAD
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("ref: refs/heads/invalid..branch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(repoDir); err == nil {
		t.Error("expected error opening repo with invalid HEAD ref")
	}
}

func TestOpenFromSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "my_project")
	if err := os.MkdirAll(filepath.Join(repoDir, "pkg", "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repoDir, "init", "-q", "-b", "main")
	gitCmd(t, repoDir, "config", "user.email", "test@orbit.local")
	gitCmd(t, repoDir, "config", "user.name", "Orbit Tester")
	gitCmd(t, repoDir, "commit", "-q", "--allow-empty", "-m", "init")

	r, err := Open(filepath.Join(repoDir, "pkg", "sub"))
	if err != nil {
		t.Fatalf("Open from subfolder failed: %v", err)
	}
	if r.Name != "my_project" {
		t.Errorf("expected r.Name == 'my_project', got %q", r.Name)
	}
}
