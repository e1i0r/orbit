package verb

// The repository as a tree, with what a task changed marked on it.
//
// One answer and not one per level. A map is clicked around rather than
// read, and a level that had to be fetched is a level that arrives after the
// reader has already decided where to look. The whole shape is small — it is
// paths and counts, no contents — so it goes over at once and every gesture
// after that is local.

import (
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
)

// Cell is one node of the tree: a directory or a file.
//
// Changed and Lines are what the task did under it, summed upward, so a
// directory says how much of the work is inside it without anybody opening
// it. That is the whole point of the reading: the shape first, the lines
// after.
type Cell struct {
	Name string `json:"name"`
	// Path is from the root of the checkout, which is what every other
	// reading of a change is keyed by — the diff, the impact, the file.
	Path string `json:"path"`
	// Cells is what is inside, and empty for a file. A directory with
	// nothing in it does not reach here: git tracks files, not folders.
	Cells []Cell `json:"cells,omitempty"`
	// Changed is how many files under this one the task touched, and one
	// for a file it touched itself. Zero is a cell the task did not reach.
	Changed int `json:"changed,omitempty"`
	// Lines is what it wrote in them, added and deleted together. It is the
	// weight a map draws with: a directory of five files barely edited and
	// one file rewritten are different things and read the same by count.
	Lines int `json:"lines,omitempty"`
}

// mapped is the checkout as a tree, with the task's change marked on it.
func mapped(w World, in In) (Out, error) {
	t, dir, err := checkout(w, in)
	if err != nil {
		return Out{}, err
	}

	if dir == "" {
		return Out{Said: t.ID + " has no checkout to map"}, nil
	}

	one, err := repo.Open(t.Repo.Path)
	if err != nil {
		return Out{}, err
	}

	files, err := one.WorktreeFiles(dir)
	if err != nil {
		return Out{}, err
	}

	// Beside the error rather than instead of the map: a change git will
	// not describe costs the marks, and a repository drawn with nothing lit
	// is still the answer to "what is in here".
	changes, _ := one.WorktreeChanges(dir) //nolint:errcheck // see above

	root := Grow(files, changes)

	return Out{Said: drawn(root), Saw: root}, nil
}

// scaffolding drops what is not the system: the dot-directories and
// dot-files that hold editor state, CI, linters and agent settings.
//
// They are in the repository and they are not what somebody opens a map of
// it to look at. A root of eight directories reads at a glance; the same
// root with .github, .vscode, .idea and four more does not, and the reader
// pays that price on every level.
//
// It is the one thing here that is a judgement rather than a reading, so it
// is small and it is in one place: a leading dot on the first segment, and
// nothing else. A change inside one of them is still in the diff, still in
// the impact, still in `orbit show` — it is this drawing that leaves it out.
func scaffolding(files []string) []string {
	out := make([]string, 0, len(files))

	for _, path := range files {
		if !strings.HasPrefix(path, ".") {
			out = append(out, path)
		}
	}

	return out
}

// weigh is how much the task wrote in each file it touched.
func weigh(changes []repo.Change) map[string]int {
	out := make(map[string]int, len(changes))
	for _, c := range changes {
		// A binary file counts as touched and weighs nothing: git counted
		// no lines because there are none, and carrying its -1 through
		// would make one image outweigh a rewritten package.
		out[c.Path] = max(c.Added, 0) + max(c.Deleted, 0)
	}

	return out
}

// Grow is the checkout as a tree, with a change marked on it.
//
// Exported because the window draws the same map the browser does, and two
// packages building a tree out of the same paths is two chances for them to
// disagree about what is in a repository. It takes the readings rather than
// taking them itself: where a checkout is and how to ask git are the
// caller's, and this is the rule about what the answers mean.
func Grow(files []string, changes []repo.Change) Cell {
	return grow(scaffolding(files), weigh(changes))
}

// grow builds the tree out of the paths, and sums the change upward.
func grow(files []string, weight map[string]int) Cell {
	root := Cell{Name: "", Path: ""}

	for _, path := range files {
		lines, touched := weight[path]
		put(&root, strings.Split(path, "/"), path, lines, touched)
	}

	tidy(&root)

	return root
}

// put walks the segments of one path, making the cells it passes through.
func put(at *Cell, segments []string, path string, lines int, touched bool) {
	if touched {
		at.Changed++
		at.Lines += lines
	}

	if len(segments) == 0 {
		return
	}

	name := segments[0]

	for i := range at.Cells {
		if at.Cells[i].Name == name {
			put(&at.Cells[i], segments[1:], path, lines, touched)

			return
		}
	}

	// The path of a directory is the path of the file that made it, cut at
	// this depth: that is what every other reading keys a directory by, and
	// working it out from the segments in hand cannot drift from it.
	cut := strings.Join(strings.Split(path, "/")[:depthOf(at, path, name)], "/")
	at.Cells = append(at.Cells, Cell{Name: name, Path: cut})
	put(&at.Cells[len(at.Cells)-1], segments[1:], path, lines, touched)
}

// depthOf is how many segments of path this new cell stands at.
func depthOf(at *Cell, path, name string) int {
	if at.Path == "" {
		return 1
	}

	return len(strings.Split(at.Path, "/")) + 1
}

// tidy sorts each level: directories first, then by name.
//
// Directories first because they are where the reader goes next, and a
// stable order because a map whose cells move between two readings is a map
// nobody can learn the shape of.
func tidy(at *Cell) {
	sort.SliceStable(at.Cells, func(i, j int) bool {
		a, b := at.Cells[i], at.Cells[j]
		if (len(a.Cells) > 0) != (len(b.Cells) > 0) {
			return len(a.Cells) > 0
		}

		return a.Name < b.Name
	})

	for i := range at.Cells {
		tidy(&at.Cells[i])
	}
}

// drawn is the tree as a terminal reads it: what is at the top, and what the
// task reached.
func drawn(root Cell) string {
	var b strings.Builder

	for _, c := range root.Cells {
		mark := "   "
		if c.Changed > 0 {
			mark = " ● "
		}

		b.WriteString(mark + c.Name + "\n")
	}

	return strings.TrimRight(b.String(), "\n")
}
