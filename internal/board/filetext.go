package board

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// fileTextCap is how much of one file a pane is given. It is a screen's
// worth many times over, and it is a bound on what a window can be made to
// hold in memory by a record that grew all day.
const fileTextCap = 64 << 10

// FileText is what one file of a task's directory holds, from the start and
// up to the cap.
//
// The name is a name and not a path: it comes from the listing this same
// port produced, and a name carrying a separator would read whatever it
// pointed at from a directory the window is not entitled to. The check is
// here rather than at the caller because this is the side that opens the
// file.
//
// Four clauses because no three of them are the whole of it: the two dots
// are their own names and carry no separator, the root is its own base, and
// an empty name is the one Base answers "." for.
func (r *Reader) FileText(repoPath, id, name string) (view.FileText, error) {
	if name == "." || name == ".." ||
		strings.ContainsRune(name, filepath.Separator) || name != filepath.Base(name) {
		return view.FileText{}, fmt.Errorf("read %q of task %s: not a file of its own directory", name, id)
	}

	dir, err := r.store.TaskDir(id)
	if err != nil {
		return view.FileText{}, fmt.Errorf("locate the directory of task %s: %w", id, err)
	}

	f, err := os.Open(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		// The file was listed and is gone: the run removed it between the
		// two asks, and an empty read is what is true now.
		return view.FileText{Whole: true}, nil
	}

	if err != nil {
		return view.FileText{}, fmt.Errorf("open %s of task %s: %w", name, id, err)
	}

	defer func() { _ = f.Close() }()

	// One byte past the cap, so that a file exactly the size of the cap is
	// known to be whole rather than guessed to be cut.
	buf := make([]byte, fileTextCap+1)

	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return view.FileText{}, fmt.Errorf("read %s of task %s: %w", name, id, err)
	}

	if n > fileTextCap {
		return view.FileText{Text: string(buf[:fileTextCap])}, nil
	}

	return view.FileText{Text: string(buf[:n]), Whole: true}, nil
}
