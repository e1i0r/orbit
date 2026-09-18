package task

// The lines the account of a previous attempt is made of: a count and the
// word for what it counts, a command shown by its shape, and the files the
// tree holds now.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
)

// TestACountAndTheWordForWhatItCounts.
//
// A prompt is read by something that reads English, and "1 files" is the
// kind of seam that makes a reader wonder what else was assembled without
// being looked at.
func TestACountAndTheWordForWhatItCounts(t *testing.T) {
	for _, one := range []struct {
		n    int
		want string
	}{
		{0, "0 files"},
		{1, "1 file"},
		{2, "2 files"},
		{12, "12 files"},
	} {
		if got := many(one.n, "file"); got != one.want {
			t.Errorf("many(%d) = %q, want %q", one.n, got, one.want)
		}
	}
}

// TestACommandIsShownByItsShapeAndNotItsWholeScript.
//
// The arguments are whatever the engine wrote and may be a whole script.
// What the reader wants is the shape of what was tried, and a heredoc pasted
// into a prompt is the room the files needed — so it is the first line, and
// no more of it than a command can be recognised by.
func TestACommandIsShownByItsShapeAndNotItsWholeScript(t *testing.T) {
	ran := func(tool, args string) record.Event {
		return record.Event{Kind: record.PhaseToolCall, Text: args, Data: map[string]string{"tool": tool}}
	}

	if got := command(ran("Bash", "go build ./...")); got != "go build ./..." {
		t.Errorf("a command came back as %q", got)
	}

	// A tool that reads a file is not a command somebody could type again.
	if got := command(ran("Read", "internal/task/run.go")); got != "" {
		t.Errorf("a file that was read came back as the command %q", got)
	}

	// The first line, and a mark saying there was more.
	script := command(ran("Bash", "cat <<'EOF' > f.txt\nline one\nline two\nEOF"))
	if script != "cat <<'EOF' > f.txt …" {
		t.Errorf("a script came back as %q", script)
	}

	// A single line exactly as wide as is shown is shown whole: a mark
	// saying something was cut, over nothing, is a reader going to look for
	// the rest of a command that is all there.
	exact := strings.Repeat("a", lineWidth)
	if got := command(ran("Bash", exact)); got != exact {
		t.Errorf("a command of exactly %d characters came back as %q", lineWidth, got)
	}

	over := command(ran("Bash", strings.Repeat("a", lineWidth+1)))
	if !strings.HasSuffix(over, "…") || len(over) != lineWidth+len("…") {
		t.Errorf("a command one character too long came back as %d characters", len(over))
	}
}

// TestTheFilesAreCappedAndTheRestAreCounted.
//
// "… and 30 more" is itself a fact about the attempt, and a prompt listing
// four hundred paths is one where the files are the prompt. The cap has to
// bite once: a line after every file but the twentieth is a list with
// thirty counts in it and nothing to count.
func TestTheFilesAreCappedAndTheRestAreCounted(t *testing.T) {
	var all []repo.Change
	for i := range atMostFiles + 5 {
		all = append(all, repo.Change{Path: fmt.Sprintf("internal/f%02d.go", i), Added: 1})
	}

	var b strings.Builder

	writeFiles(&b, all)

	listing := b.String()
	if n := strings.Count(listing, "… and "); n != 1 {
		t.Errorf("the listing counts what was left out %d times:\n%s", n, listing)
	}

	if !strings.Contains(listing, "… and 5 more") {
		t.Errorf("the listing does not say how many were left out:\n%s", listing)
	}

	// The first twenty are there and the twenty-first is not.
	if !strings.Contains(listing, "internal/f19.go") {
		t.Errorf("the last file inside the cap is missing:\n%s", listing)
	}

	if strings.Contains(listing, "internal/f20.go") {
		t.Errorf("a file past the cap was listed anyway:\n%s", listing)
	}

	// And a tree with nothing in it writes no heading at all.
	var empty strings.Builder

	writeFiles(&empty, nil)

	if empty.String() != "" {
		t.Errorf("a tree holding nothing wrote %q", empty.String())
	}
}
