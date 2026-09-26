package cli

import (
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/lowly"
)

// TestOpenCommandOpensClineInItsTerminal. A bare argument is a prompt cline
// runs once and exits on, so the window would suspend itself for a session
// that had already ended; -i is its terminal, with that sentence first.
func TestOpenCommandOpensClineInItsTerminal(t *testing.T) {
	cmd, err := openCommand("cline", "", "look at PAY-1")
	if err != nil {
		t.Fatalf("openCommand: %v", err)
	}

	want := []string{"-i", "look at PAY-1"}
	if got := lowly.Behind(cmd)[1:]; !slices.Equal(got, want) {
		t.Errorf("args = %v, want %v", got, want)
	}
}
