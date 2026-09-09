package ui

// The worktree's fingerprint: enough to know whether the diff on screen is
// still the diff on disk, and cheap enough to ask every two seconds.
//
// The diff is re-read on a clock while a task is open, so that a run writing
// in the worktree is watched rather than photographed. Re-reading it is not
// free: git writes the whole text out and the window styles every line of
// it, which on a generated file is the freeze #133 was about. Most of those
// re-reads change nothing at all — an engine that is thinking moves no
// files, and it thinks for most of a phase.
//
// So the clock asks this first. `git status --porcelain` names what changed
// without writing a byte of diff, and a stat of each of those paths catches
// what the names cannot: a line rewritten in place leaves the same file in
// the same list, and only its mtime says it moved. The two hashed together
// are the fingerprint; an unchanged fingerprint means the text in hand is
// still the truth.
//
// It never claims sameness on a failure. Nothing readable, git refusing, a
// deadline — all answer the empty string, which is "ask git properly", not
// "nothing changed".

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// worktreePrint is the fingerprint of a worktree's changes, or "" when it
// could not be taken.
func worktreePrint(dir string) string {
	out, err := runGitDiff(dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(out)

	for _, path := range statusPaths(out) {
		info, err := os.Stat(filepath.Join(dir, path))
		if err != nil {
			// A path git named and the filesystem does not have is itself a
			// fact about this moment — a file deleted between the two
			// calls — and it belongs in the fingerprint rather than
			// cancelling it.
			fmt.Fprintf(&b, "\n%s gone", path)

			continue
		}

		fmt.Fprintf(&b, "\n%s %d %d", path, info.Size(), info.ModTime().UnixNano())
	}

	sum := sha256.Sum256([]byte(b.String()))

	return hex.EncodeToString(sum[:])
}

// statusPaths is the files a porcelain status named.
//
// Two columns, a space, then the path — and for a rename, `old -> new`, of
// which the new name is the file on disk to stat. A quoted path is left as
// git wrote it: os.Stat will fail on it, which the caller already reads as a
// fact about the file rather than as an error.
func statusPaths(out string) []string {
	var paths []string

	for _, line := range strings.Split(out, "\n") {
		// The status code, then the path. Cut and not a slice by column:
		// the terminal counts cells, not bytes, and this package does not
		// index into strings.
		_, path, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}

		path = strings.TrimSpace(path)
		if _, after, renamed := strings.Cut(path, " -> "); renamed {
			path = after
		}

		if path != "" {
			paths = append(paths, path)
		}
	}

	return paths
}
