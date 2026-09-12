package knowledge

// Where the facts live, which is two places for one reason: whether they
// travel.
//
// A fact about a repository goes inside it, under `.orbit/knowledge/`, so it
// moves with the push — whoever clones the project gets what Orbit learned
// about it, and a rule that is about to start steering the agent arrives in a
// diff somebody reviews rather than appearing on one machine in silence.
//
// A fact about everything, or about a language, belongs to no checkout. There
// is no repository to put it in that would not be picked at random, so it
// lives in the state root, and the price is paid knowingly: it does not
// travel and nobody else sees it.
//
// From out here that is one store. Load answers both.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// dirName is what the directory is called under both roots.
	dirName = "knowledge"
	// ext is the extension, and Markdown is the choice: these are read in a
	// pull request by a person deciding whether the agent should be told
	// this, and a diff of prose is a thing a person can judge.
	ext = ".md"
	// generalDir and langDir are where the two rootless kinds are filed.
	generalDir = "general"
	langDir    = "lang"

	dirMode  = 0o755
	fileMode = 0o644
)

// A Store reads and writes facts across both roots.
type Store struct {
	state string
}

// NewStore opens the store over a state root. The repositories are named per
// call rather than held, because which ones matter is the caller's question
// and it changes with the task.
func NewStore(stateRoot string) *Store {
	return &Store{state: stateRoot}
}

// Save writes one fact down and answers where it went.
func (s *Store) Save(f Fact) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	path := filepath.Join(s.dirFor(f.Scope), fileName(f))
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return "", fmt.Errorf("make room for the fact at %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(encode(f)), fileMode); err != nil {
		return "", fmt.Errorf("write the fact at %q: %w", path, err)
	}

	return path, nil
}

// Replace writes a changed fact and takes the one it replaces away.
//
// Saving alone is not enough whenever the change moves the file. A fact with
// no reference is filed under a slug of its own sentence and one with a scope
// is filed under its path, so editing either writes somewhere new — and the
// copy nobody meant to keep would go on being told and go on refusing work.
//
// The old one is removed after the new one is written, so a failure in the
// middle leaves two facts rather than none: a duplicate is visible in the
// screen that lists them, and a fact that vanished is not.
func (s *Store) Replace(was, now Fact) (string, error) {
	where, err := s.Save(now)
	if err != nil {
		return "", err
	}

	before := filepath.Join(s.dirFor(was.Scope), fileName(was))
	if before == where {
		return where, nil
	}

	if err := os.Remove(before); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return where, fmt.Errorf("remove the fact it replaces at %q: %w", before, err)
	}

	return where, nil
}

// Load is everything known while working in one repository: the general
// facts, the ones of every language, and the repository's own.
//
// A directory that is not there is a repository nobody has written anything
// about yet, which is what every repository starts as and is not a failure.
func (s *Store) Load(repo string) ([]Fact, error) {
	facts, err := s.read(filepath.Join(s.state, dirName), "")

	if repo == "" {
		return facts, err
	}

	own, ownErr := s.read(filepath.Join(repo, ".orbit", dirName), repo)

	return append(facts, own...), errors.Join(err, ownErr)
}

// LoadRepo is one checkout's own facts, and none of the state root's.
//
// Load answers with both, which is what a phase wants: it works in one
// repository and everything known reaches it. A caller walking every
// repository the record has heard of wants the other shape — it already
// holds the state root's facts, and Load would hand them back once per
// repository, to be walked, decoded and thrown away N times over.
//
// A directory that is not there is a repository nobody has written anything
// about yet, the same as in Load.
func (s *Store) LoadRepo(repo string) ([]Fact, error) {
	// Guarded as Load guards it: joined onto an empty repository the path is
	// the relative `.orbit/knowledge`, and the walk would read whatever the
	// process happens to be standing in.
	if repo == "" {
		return nil, nil
	}

	return s.read(filepath.Join(repo, ".orbit", dirName), repo)
}

// read walks one root. repo is the checkout the facts belong to, and empty
// for the state root, where they belong to none.
func (s *Store) read(root, repo string) ([]Fact, error) {
	var (
		facts  []Fact
		failed []error
	)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ext {
			return nil
		}

		body, readErr := os.ReadFile(path)
		if readErr != nil {
			failed = append(failed, fmt.Errorf("read the fact at %q: %w", path, readErr))

			return nil
		}

		f, decErr := decode(string(body), rel(root, path), repo)
		if decErr != nil {
			failed = append(failed, fmt.Errorf("the fact at %q: %w", path, decErr))

			return nil
		}

		facts = append(facts, f)

		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		// Nothing written down here yet, which is where everything starts.
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read the facts under %q: %w", root, err)
	}

	// A damaged file costs itself and no more. These are files a person is
	// invited to write by hand, and aborting the walk on the first one meant
	// a single typo left every repository's rules unread — the same shape
	// store.Repos already answers a damaged marker with.
	return facts, errors.Join(failed...)
}

// dirFor is the directory a scope files its facts in.
//
// The path kinds mirror the code: a fact about `backend/ledger` sits in
// `.orbit/knowledge/backend/ledger/`, so finding what is known about a
// directory is looking in the directory of the same name.
func (s *Store) dirFor(sc Scope) string {
	stateRoot := filepath.Join(s.state, dirName)

	switch sc.Kind {
	case General:
		return filepath.Join(stateRoot, generalDir)
	case Language:
		return filepath.Join(stateRoot, langDir, sc.Lang)
	case Repo:
		return filepath.Join(sc.Repo, ".orbit", dirName)
	case Dir:
		return filepath.Join(sc.Repo, ".orbit", dirName, filepath.FromSlash(sc.Path))
	case File, Symbol:
		return filepath.Join(sc.Repo, ".orbit", dirName, filepath.FromSlash(filepath.Dir(sc.Path)))
	default:
		return stateRoot
	}
}

// fileName is what one fact is called on disk: what it came out of and what
// it says, or only what it says when it came out of nothing named.
//
// Both halves, because neither is enough on its own. The reference is what a
// reader recognises — PAY-1 beside the code says where to go and read what
// happened — and one task can teach more than one thing, so a name that was
// only the reference kept the last of them and dropped the rest without
// saying so.
func fileName(f Fact) string {
	if f.Ref == "" {
		return slug(f.Phrase) + ext
	}

	return f.Ref + "-" + slug(f.Phrase) + ext
}

// slug is a sentence turned into a file name: lowercase words joined by
// dashes, cut at a length that still reads as the sentence it came from.
func slug(phrase string) string {
	const words = 6

	kept := make([]rune, 0, len(phrase))
	for _, r := range strings.ToLower(phrase) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			kept = append(kept, r)
		case len(kept) > 0 && kept[len(kept)-1] != '-':
			kept = append(kept, '-')
		}
	}

	parts := strings.Split(strings.Trim(string(kept), "-"), "-")
	if len(parts) > words {
		parts = parts[:words]
	}

	if name := strings.Join(parts, "-"); name != "" {
		return name
	}

	return "fact"
}

// rel is a fact's path under its root, in slash form, which is what the
// scope is read from.
func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}

	return filepath.ToSlash(r)
}
