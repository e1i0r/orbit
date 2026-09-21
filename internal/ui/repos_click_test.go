package ui

// A click on the repository list lands on the checkout that is drawn under
// the pointer — including the rows where none is.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// manyRepos is a window on a board with n checkouts, which is more of them
// than a short terminal has room for.
func manyRepos(t *testing.T, n, w, h int) Model {
	t.Helper()

	b := board.Board{ReadAt: fixtureNow, Repos: n}

	for i := range n {
		name := string(rune('a'+i%26)) + strings.Repeat("o", 1+i/26) + "-repo"
		b.RepoList = append(b.RepoList, board.RepoInfo{Name: name, Path: "/r/" + name})
		b.Tasks = append(b.Tasks, view.Task{Repo: name, ID: "T-" + name, Band: view.Running})
	}

	return modelWith(t, words.For("en"), b, w, h, &recorder{}).openRepos()
}

// TestARepositoryIsClickedWhereItIsDrawn, on every row of the body and at
// every scroll.
//
// The window worked the row out for itself — the line, less the four rows
// of the head, plus the offset — and did not know the list stops short of
// the floor. So a click on the blank above the ways out, or on the line of
// keys itself, named the checkout just under the window: the board came
// back filtered to a repository the reader had never seen, which is the
// failure the list was given an offset to stop.
func TestARepositoryIsClickedWhereItIsDrawn(t *testing.T) {
	m := manyRepos(t, 14, 100, 24)

	list := m.collectRepos()
	if len(list) == 0 {
		t.Fatal("the board has no repositories")
	}

	for _, notch := range []int{0, 3, 9, 40} {
		at := m
		for range notch {
			at = at.wheelRepos(1)
		}

		h, w := at.frame.Body.H, at.frame.Body.W
		rows := at.repolistRows(h, w)

		for line := range h {
			drawn := ansi.Strip(rows[line])
			got := at.hitRepos(5, at.frame.Body.Y+line)

			// What is drawn there, if anything, is what the click owes.
			want := ""

			for _, r := range list {
				if strings.Contains(drawn, r.Name+" ") {
					want = r.Name

					break
				}
			}

			switch {
			case want == "" && got.Kind != point.None:
				t.Errorf("wheel %d, row %d is %q and a click there is %q",
					notch, line, drawn, got.ID)
			case want != "" && got.ID != want:
				t.Errorf("wheel %d, row %d draws %q and a click there is %q",
					notch, line, want, got.ID)
			}
		}
	}
}
