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

	if err := os.WriteFile(path, body, fileMode); err != nil {
		return "", fmt.Errorf("write %q: %w", path, err)
	}

	return path, nil
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
