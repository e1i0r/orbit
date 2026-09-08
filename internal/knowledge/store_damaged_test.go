package knowledge

// What a store answers when one of the files under it is damaged: the walk
// carries on, because the files are written by hand and one of them being
// wrong is ordinary.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestATypoCostsItsOwnFileAndNoOther. The files are written by hand, so one
// of them will be wrong sooner or later; aborting the walk on the first bad
// header meant a single typo left every sound rule unread.
func TestATypoCostsItsOwnFileAndNoOther(t *testing.T) {
	state, repo := roots(t)

	dir := filepath.Join(state, "knowledge", "general")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	sound := "---\nscope: general\nsource: human\nref: SOUND\n---\n\nThe ledger only appends.\n"
	if err := os.WriteFile(filepath.Join(dir, "ledger.md"), []byte(sound), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "typo.md"), []byte("---\nsource: humman\n---\n\nMistyped.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := NewStore(state).Load(repo)
	if err == nil {
		t.Error("Load: the damaged file was not reported")
	}

	if len(got) != 1 {
		t.Fatalf("loaded %d facts, want the sound one beside the error", len(got))
	}

	if got[0].Phrase != "The ledger only appends." {
		t.Errorf("the sentence came back as %q", got[0].Phrase)
	}
}

// TestLoadRepoRefusesAnEmptyRepository, which would otherwise join to the
// relative `.orbit/knowledge` and read whatever the process is standing in.
func TestLoadRepoRefusesAnEmptyRepository(t *testing.T) {
	state, _ := roots(t)

	got, err := NewStore(state).LoadRepo("")
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}

	if got != nil {
		t.Errorf("LoadRepo(\"\") answered with %d facts, want none", len(got))
	}
}
