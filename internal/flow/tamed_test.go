package flow

// A flow file holding what a terminal reads as an instruction.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAFlowFileCannotDriveTheTerminal. Every word of a flow is drawn, and
// an engine writes one of these whenever the designer is asked to draft it.
// A name or a prompt with an escape sequence in it reached the screen as
// the sequence: ESC[2J clears it, a colour paints the rows after it, and
// the reader has no way to tell which flow did that to them.
func TestAFlowFileCannotDriveTheTerminal(t *testing.T) {
	dir := t.TempDir()

	body := `{
		"name":"sneaky\u001b[31m",
		"description":"clears \u001b[2J the screen",
		"phases":[{
			"name":"impl\u0007","engine":"zeta\u001b]0;stolen\u0007","model":"m\u001b[1;1H",
			"prompt":"red \u001b[31mprompt\u001b[0m",
			"gates":[{"name":"g\u001b[2J","command":"make check\u001b[31m"}]
		}]}`

	if err := os.WriteFile(filepath.Join(dir, "sneaky.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write the flow: %v", err)
	}

	f, err := Load(filepath.Join(dir, "sneaky.json"))
	if err != nil {
		t.Fatalf("load the flow: %v", err)
	}

	said := f.Name + f.Description
	for _, p := range f.Phases {
		said += p.Name + p.Engine + p.Model + p.Prompt

		for _, g := range p.Gates {
			said += g.Name + g.Command
		}
	}

	if strings.ContainsAny(said, "\x1b\x07") {
		t.Errorf("a flow loaded from a file carries control characters: %q", said)
	}

	// And the words themselves are all still there.
	for _, want := range []string{"sneaky", "clears", "impl", "zeta", "red prompt", "make check"} {
		if !strings.Contains(said, want) {
			t.Errorf("taming the flow lost %q: %q", want, said)
		}
	}
}
