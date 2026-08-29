package flow

// Writing a flow down and taking one away — the other half of resolve.go.
//
// A flow of the reader's own is a file in one directory, and that was always
// the whole extension mechanism; until now the only way to add one was a
// text editor, which is fine for a person and impossible for anything that
// reaches Orbit through a tool call. These two functions are the same act
// with the same rules: a name that is a name, a flow that validates, and a
// file whose contents Resolve will read back as the flow that was saved.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// The state root's own modes, spelled again here for the same reason
// internal/engine spells out the permission words: this package imports
// nothing of Orbit's, and a flow directory that ended up group-readable
// because two packages disagreed about a number would be a quiet widening.
const (
	dirMode  os.FileMode = 0o700
	fileMode os.FileMode = 0o600
)

// Decode reads a flow from the JSON of one.
//
// It is decode exported, for the one kind of caller that has a flow document
// in hand and no file to put it in first: the MCP server, which is handed a
// flow as arguments to a tool call. Going through here rather than through
// json.Unmarshal is what makes an invented field an error — the decoder
// refuses unknown ones — so a model that writes "engines" instead of
// "engine" is told, rather than saving a phase with no engine at all.
func Decode(raw []byte, source string) (Flow, error) {
	return decode(raw, source)
}

// Save writes a flow into the reader's own flow directory, replacing
// whatever was there under that name.
//
// The name it is filed under is the flow's own: Resolve refuses a file whose
// name and whose contents disagree, so saving under anything else would
// write a file that cannot be read back.
func Save(src Source, f Flow) (string, error) {
	path, err := UserPath(src, f.Name)
	if err != nil {
		return "", err
	}

	if err := f.Validate(); err != nil {
		return "", err
	}

	body, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode the flow %q: %w", f.Name, err)
	}

	body = append(body, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", dir, err)
	}

	if err := writeAtomically(path, body); err != nil {
		return "", err
	}

	return path, nil
}

// writeAtomically writes a flow into place in one step, by writing it beside
// itself and renaming it over.
//
// os.WriteFile truncates the file and then fills it, so every reader between
// those two is reading a flow that is half there or not there at all. A flow
// is read at the moment a task is about to run on it, and Resolve is
// deliberately hard about what it finds: a file that will not parse is the
// reader's flow failing and is reported as such, rather than falling back to
// the built-in of the same name. So a save interrupted at the wrong instant
// does not lose an edit — it takes the name out of service, while leaving it
// in the list, because the listing looks at file names and not at contents.
//
// The rename is the whole fix and it is one line. It also means the file the
// name points at is replaced rather than written through, so a flow path
// somebody has pointed elsewhere is not a way to have Orbit write outside
// its own directory.
//
// This is internal/mcp's writeAtomically, spelled again for the reason the
// modes above are: this package imports nothing of Orbit's.
func writeAtomically(path string, body []byte) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create a temporary file beside %q: %w", path, err)
	}

	name := tmp.Name()

	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()     //nolint:errcheck // the write already failed; the close cannot add to it
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at

		return fmt.Errorf("write %q: %w", name, err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at

		return fmt.Errorf("close %q: %w", name, err)
	}

	if err := os.Chmod(name, fileMode); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at

		return fmt.Errorf("set the mode of %q: %w", name, err)
	}

	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at

		return fmt.Errorf("move %q into place at %q: %w", name, path, err)
	}

	return nil
}

// Delete removes a flow of the reader's own, and says whether a built-in of
// the same name was underneath it.
//
// That second answer is the one thing a caller cannot work out afterwards
// and would badly want to know: deleting a shadow does not remove the flow,
// it restores the shipped one, so a task written against that name goes on
// running — differently. A built-in on its own cannot be deleted at all,
// because it is inside the binary; saying so is better than reporting a
// success that removed nothing.
func Delete(src Source, name string) (revealed bool, err error) {
	path, err := UserPath(src, name)
	if err != nil {
		return false, err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if slices.Contains(BuiltinNames(), name) {
				return false, fmt.Errorf("the flow %q is built into orbit and cannot be deleted; write your own of that name to change what it does", name)
			}

			return false, fmt.Errorf("there is no flow of your own called %q", name)
		}

		return false, fmt.Errorf("remove %q: %w", path, err)
	}

	return slices.Contains(BuiltinNames(), name), nil
}

// UserPath is the file one name is a flow of the reader's own in, whether or
// not anything is there yet.
func UserPath(src Source, name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}

	dir := dirOf(src)
	if dir == "" {
		return "", fmt.Errorf("there is nowhere to keep a flow: this caller has no flow directory")
	}

	return filepath.Join(dir, name+".json"), nil
}
