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
	if err != nil {
		return nil, err
	}

	if repo == "" {
		return facts, nil
	}

	own, err := s.read(filepath.Join(repo, ".orbit", dirName), repo)
	if err != nil {
		return nil, err
	}

	return append(facts, own...), nil
}

// read walks one root. repo is the checkout the facts belong to, and empty
// for the state root, where they belong to none.
func (s *Store) read(root, repo string) ([]Fact, error) {
	var facts []Fact

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ext {
			return nil
		}

		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read the fact at %q: %w", path, readErr)
		}

		f, decErr := decode(string(body), rel(root, path), repo)
		if decErr != nil {
			return fmt.Errorf("the fact at %q: %w", path, decErr)
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

	return facts, nil
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

// fileName is what one fact is called on disk: the thing it came out of when
// it has one, and a slug of its own sentence when it does not.
//
// The reference is preferred because it is what a reader recognises — REF-9
// beside the code it is about says where to go and read the argument.
func fileName(f Fact) string {
	name := f.Ref
	if name == "" {
		name = slug(f.Phrase)
	}

	return name + ext
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
