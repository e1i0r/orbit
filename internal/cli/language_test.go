package cli

// The terminal in the reader's language.
//
// `orbit set language es` used to change the window and nothing else: every
// sentence the commands printed was written where it was printed, in English,
// so a reader who chose Spanish got a Spanish cockpit and an English shell.
// These tests are the sweep — each one runs a command under es and refuses
// the English it used to answer with.

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// speaking is a workspace whose reader chose this language.
func speaking(t *testing.T, language string) (orbitHome, dir string) {
	t.Helper()

	root, orbitHome := workspace(t)

	if code, _, errOut := run(t, "set", "language", language); code != 0 {
		t.Fatalf("set language %s exited %d: %s", language, code, errOut)
	}

	return orbitHome, filepath.Join(root, "payments")
}

// TestEveryCommandThatNeedsATaskIDRefusesInTheReadersLanguage. Eight commands
// refuse the same way and now say so through one key, so the sweep is over
// the commands rather than over the sentence: a command left behind still
// answers in English while the other seven have moved.
func TestEveryCommandThatNeedsATaskIDRefusesInTheReadersLanguage(t *testing.T) {
	_, dir := speaking(t, "es")

	for _, command := range []string{"cancel", "direct", "note", "pause", "read", "resume", "run", "show"} {
		t.Run(command, func(t *testing.T) {
			code, _, errOut := run(t, command, "-repo", dir)
			if code == 0 {
				t.Fatalf("%s with no id exited 0", command)
			}

			if strings.Contains(errOut, "needs the id of a task") {
				t.Errorf("a reader who chose Spanish is refused in English: %q", errOut)
			}

			// The command's own name is typed rather than read, so it is
			// the one thing in the sentence that stays as it was.
			if !strings.Contains(errOut, command) {
				t.Errorf("the refusal does not name the command that refused: %q", errOut)
			}
		})
	}
}

// TestTheEverydayVerbsSpeakTheReadersLanguage walks one workspace the way a
// reader does — write a task, note it, look at it, stop it — and asks of each
// answer only that it is not the English sentence it used to be.
//
// The steps run in order against the same workspace, because half of them
// need what the step before wrote.
func TestTheEverydayVerbsSpeakTheReadersLanguage(t *testing.T) {
	orbitHome, dir := speaking(t, "es")
	elsewhere := t.TempDir()

	for _, step := range []struct {
		args    []string
		english string
		before  func(*testing.T)
	}{
		{args: []string{"list", "-repo", dir}, english: "no tasks against"},
		{args: []string{"new", "-repo", dir}, english: "needs -id"},
		{args: []string{"new", "-repo", dir, "-id", "PAY-1"}, english: "written out after the flags"},
		{args: []string{"new", "-repo", dir, "-id", "PAY-1", "make the thing"}, english: "written against"},
		{args: []string{"note", "-repo", dir, "PAY-1"}, english: "needs text for task"},
		{args: []string{"note", "-repo", dir, "PAY-1", "look at the tests"}, english: "note recorded for"},
		{args: []string{"read", "-repo", dir, "PAY-1"}, english: "marked read"},
		{args: []string{"pause", "-repo", dir, "PAY-1"}, english: "a run in flight"},
		{args: []string{"direct", "-repo", dir, "PAY-1"}, english: "needs a message for task"},
		{args: []string{"direct", "-repo", dir, "PAY-1", "look again"}, english: "redirected"},
		{args: []string{"show", "-repo", dir, "PAY-404"}, english: "nothing recorded for"},
		// Stopping is asked of a run, so there has to be one: without a
		// live marker cancel refuses instead, and the refusal it gives is
		// task's rather than this layer's.
		{
			args:    []string{"cancel", "-repo", dir, "PAY-1"},
			english: "asked to stop",
			before:  func(t *testing.T) { plantLiveMarker(t, orbitHome) },
		},
		{args: []string{"reconcile", "-repo", dir}, english: "every run is accounted for"},
		{args: []string{"supervisor"}, english: "supervisor thread is empty"},
		{args: []string{"repos", elsewhere}, english: "no repositories under"},
		{args: []string{"top", elsewhere, dir}, english: "takes one directory"},
		{args: []string{"top", filepath.Join(elsewhere, "nowhere")}, english: "look at"},
		{
			args:    []string{"top", filepath.Join(elsewhere, "notes.txt")},
			english: "is not a directory",
			before: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(elsewhere, "notes.txt"), []byte("a file\n"), 0o600); err != nil {
					t.Fatalf("write the file top is pointed at: %v", err)
				}
			},
		},
	} {
		t.Run(strings.Join(step.args, " "), func(t *testing.T) {
			if step.before != nil {
				step.before(t)
			}

			_, out, errOut := run(t, step.args...)

			said := out + errOut
			if strings.TrimSpace(said) == "" {
				t.Fatalf("%v said nothing at all", step.args)
			}

			if strings.Contains(said, step.english) {
				t.Errorf("a reader who chose Spanish is told %q", said)
			}
		})
	}
}

// TestTheUpgradeRefusalsAreInTheReadersLanguage. Upgrading is the one command
// whose refusals are written four layers down — the archive, its hash, the
// file inside it — and every one of those layers reaches the reader whole.
func TestTheUpgradeRefusalsAreInTheReadersLanguage(t *testing.T) {
	es := words.For("es")
	forThisMachine := fmt.Sprintf("orbit_2.0.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	for _, tc := range []struct {
		name    string
		refuse  func(*testing.T) error
		english string
	}{
		{"a release with nothing for this machine", func(*testing.T) error {
			return installRelease(t.Context(), es, releaseInfo{
				TagName: "v2.0.0",
				Assets:  []asset{{Name: "orbit_2.0.0_plan9_mips.tar.gz"}, {Name: "checksums.txt"}},
			})
		}, "publishes no"},
		{"a release with no checksums", func(*testing.T) error {
			return installRelease(t.Context(), es, releaseInfo{
				TagName: "v2.0.0", Assets: []asset{{Name: forThisMachine}},
			})
		}, "cannot be checked"},
		{"an archive nobody published a hash for", func(*testing.T) error {
			return verify(es, []byte("bytes"), []byte("d0  other.tar.gz\n"), "orbit.tar.gz")
		}, "publishes no checksum"},
		{"an archive that is not the one published", func(*testing.T) error {
			return verify(es, []byte("bytes"), []byte("d0  orbit.tar.gz\n"), "orbit.tar.gz")
		}, "hashes to"},
		{"an archive with no orbit in it", func(t *testing.T) error {
			_, err := binaryFrom(es, archiveOf(t, "README.md", []byte("read me")))

			return err
		}, "not found in archive"},
		{"a server that answered with a status", func(t *testing.T) error {
			url := serving(t, func(w http.ResponseWriter) { w.WriteHeader(http.StatusTeapot) })
			_, err := download(t.Context(), es, url)

			return err
		}, "responded"},
		{"github answering with a status", func(t *testing.T) error {
			old := updateEndpoint
			updateEndpoint = serving(t, func(w http.ResponseWriter) { w.WriteHeader(http.StatusForbidden) })

			t.Cleanup(func() { updateEndpoint = old })

			_, err := fetchRelease(t.Context(), es)

			return err
		}, "api responded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.refuse(t)
			if err == nil {
				t.Fatal("this was not refused at all")
			}

			if strings.Contains(err.Error(), tc.english) {
				t.Errorf("a reader who chose Spanish is refused in English: %v", err)
			}
		})
	}
}
