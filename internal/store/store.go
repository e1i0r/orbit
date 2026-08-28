// Package store owns the on-disk layout of everything Orbit remembers.
//
// Nothing else in the program is allowed to build a path by hand. That rule
// is what makes "delete this folder and nothing else breaks" true, and it is
// what keeps Orbit from ever writing inside a repository it does not own.
//
// Computing a path creates nothing. Every method in this file is pure: it
// answers "where would this live?" and stops there. A path method that made
// the directory turned reading into writing — `orbit show` on a mistyped id
// minted an empty task, and `orbit list` then reported it as real. Creation
// is a separate verb, in create.go, called only by the writers that mean it.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// dirMode and fileMode keep the state root to its owner.
//
// This tree holds full checkouts of private repositories, the verbatim text
// of every task, everything the engines printed — and the design puts
// credentials in the same root. Home is 0700 on macOS but commonly 0755 on
// Linux, and $ORBIT_HOME can be pointed anywhere at all, so the mode is
// stated here rather than inherited from wherever the root happens to sit.
const (
	dirMode  os.FileMode = 0o700
	fileMode os.FileMode = 0o600
)

// Store is a rooted view of ~/.orbit. It holds a path and no other state.
type Store struct {
	root string
}

// validateTaskID rejects taskIDs that could escape or traverse the store.
// This package is the only thing standing between a typed id and the filesystem.
//
// Escaping the tree is not the only way an id can be wrong. An id is a name
// three audiences have to agree on — a directory on disk, an argument on a
// command line, and a line of the record read months later — and the rules
// below are the ones that keep those three readings the same. Each is here
// because breaking it produces a task that can be created and then not used:
//
//   - whitespace at either end, or nothing but whitespace: "PAY-1" and
//     "PAY-1 " print identically and are two different directories, and a
//     task named " " cannot be picked out of a listing at all.
//   - a leading dash: every command takes the id as a positional argument,
//     so "-fix" is read as a flag. The task is made and then nothing can
//     open, run or cancel it.
//   - a control character: the id is written into the JSONL record and
//     printed on a terminal. A newline splits one log line into two, and an
//     escape sequence rewrites the screen around it.
func validateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task id cannot be empty")
	}

	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("task id cannot be only whitespace")
	}

	if strings.TrimSpace(taskID) != taskID {
		return fmt.Errorf("task id %q has whitespace at the start or the end", taskID)
	}

	if strings.HasPrefix(taskID, "-") {
		return fmt.Errorf("task id %q starts with a dash, so every command would read it as a flag", taskID)
	}

	if strings.ContainsFunc(taskID, unicode.IsControl) {
		return fmt.Errorf("task id %q contains a control character", taskID)
	}

	if strings.Contains(taskID, "/") || strings.Contains(taskID, string(os.PathSeparator)) {
		return fmt.Errorf("task id %q contains path separator", taskID)
	}

	if taskID == "." || taskID == ".." {
		return fmt.Errorf("task id %q is not allowed", taskID)
	}

	if strings.Contains(taskID, "..") {
		return fmt.Errorf("task id %q contains path traversal", taskID)
	}

	return nil
}

// ValidTaskID is validateTaskID, exported for the one caller outside this
// package that needs the rule and not the write: the window's compose form
// asks the same question `orbit new` ends up asking, and asking it twice
// with two answers is how an id the window accepts arrives at a store that
// refuses it.
func ValidTaskID(taskID string) error {
	return validateTaskID(taskID)
}

// RootPath says where the state root is — $ORBIT_HOME, or ~/.orbit when
// that is unset — and creates nothing.
//
// It is separate from Open because there are callers that need to know where
// the root is without wanting one to exist: repo.Discover skips it while
// walking a tree, and merely listing repositories must not mint a state
// directory.
func RootPath() (string, error) {
	root := os.Getenv("ORBIT_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}

		root = filepath.Join(home, ".orbit")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", root, err)
	}

	return abs, nil
}

// Open resolves the root from $ORBIT_HOME, falling back to ~/.orbit, and
// creates it. The environment variable exists so tests and a second machine
// can point somewhere else without a flag on every command.
func Open() (*Store, error) {
	root, err := RootPath()
	if err != nil {
		return nil, err
	}

	return New(root)
}

// New roots a store at an explicit path and creates it if it is missing.
//
// This is the one creation the path rules exempt: making ~/.orbit on first
// run is deliberate, and a store you cannot open is not a store.
func New(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", root, err)
	}

	if err := os.MkdirAll(abs, dirMode); err != nil {
		return nil, fmt.Errorf("create %q: %w", abs, err)
	}

	return &Store{root: abs}, nil
}

// Root is the absolute path of the folder holding everything.
func (s *Store) Root() string { return s.root }

// FlowDir is where a user's own flows live: one JSON file per flow, at the
// root of the state tree rather than under any one repository, because a
// flow is no more scoped to a repository than a setting is.
//
// It answers a plain string and not (string, error) because nothing about
// it can fail — it is the root and one fixed segment, with no caller-
// supplied component to resolve or reject. Like every path here it creates
// nothing: a state root with no flows directory is a user who has written
// none, and internal/flow reads that as "no flows of your own" rather than
// as a fault.
func (s *Store) FlowDir() string { return filepath.Join(s.root, "flows") }

// repoKey is the directory name one repository is filed under: SHA-256 of
// its absolute path, truncated to 12 hex characters.
//
// A hash rather than the path because the path contains separators and
// spaces, and because the key must stay one directory deep no matter how
// deeply nested the repository is. It lives in one function because it was
// written out twice and two copies of a key can drift apart. Twelve
// characters is 48 bits, which over a handful of paths will not collide;
// create.go writes the path beside the hash and refuses to file a second
// repository under a key whose marker already names another one, so a
// collision is an error somebody reads rather than two records merged.
func repoKey(abs string) string {
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:])[:12]
}

// RepoDir is where one repository's record lives, keyed by the absolute path
// of the repository itself.
func (s *Store) RepoDir(repoPath string) (string, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", repoPath, err)
	}

	return filepath.Join(s.root, "repos", repoKey(abs)), nil
}

// TasksDir is where every task of one repository lives. It exists so that
// nothing outside this package has to know that "tasks" sits under a repo's
// directory — TaskDir and List both build on this one path.
func (s *Store) TasksDir(repoPath string) (string, error) {
	repoDir, err := s.RepoDir(repoPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(repoDir, "tasks"), nil
}

// TaskDir is where one task's record lives.
func (s *Store) TaskDir(repoPath, taskID string) (string, error) {
	if err := validateTaskID(taskID); err != nil {
		return "", err
	}

	tasksDir, err := s.TasksDir(repoPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(tasksDir, taskID), nil
}

// TaskFilePath is the written task itself — the sentence a human typed.
func (s *Store) TaskFilePath(repoPath, taskID string) (string, error) {
	dir, err := s.TaskDir(repoPath, taskID)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "task.md"), nil
}

// WorktreeDir is where a task's throwaway checkout goes. It is deliberately
// not under the task directory: worktrees are large and disposable, and the
// record is small and kept.
//
// It is also, deliberately, inside the state root. The design weighed
// splitting config, data and worktrees across three directories and settled
// on one folder, on the grounds that an audit of one tree beats an audit of
// three. The consequence must be stated rather than discovered: the engine
// runs here with this directory as its working directory, so the
// append-only record — and the credentials file the design puts in the same
// root — are reachable from it by relative path. Therefore no engine may
// ever be handed a directory permission at or above the state root:
// no --add-dir, no equivalent, at Root() or any ancestor of it.
func (s *Store) WorktreeDir(repoPath, taskID string) (string, error) {
	if err := validateTaskID(taskID); err != nil {
		return "", err
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", repoPath, err)
	}

	return filepath.Join(s.root, "worktrees", repoKey(abs), taskID), nil
}

// EventsPath is the append-only log for one task — the source of truth.
func (s *Store) EventsPath(repoPath, taskID string) (string, error) {
	dir, err := s.TaskDir(repoPath, taskID)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "events.jsonl"), nil
}

// LogDir is where internal diagnostic logs of Orbit live.
func (s *Store) LogDir() string { return filepath.Join(s.root, "logs") }

// LogPath is the file path for Orbit's internal diagnostic log.
func (s *Store) LogPath() string { return filepath.Join(s.root, "logs", "orbit.log") }

// SupervisorLogPath is the global conversation thread and persistent memory of the supervisor.
func (s *Store) SupervisorLogPath() string { return filepath.Join(s.root, "supervisor.jsonl") }
