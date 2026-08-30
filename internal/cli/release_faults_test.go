package cli

// The ways installing a release goes wrong. Every one of them ends with the
// reader still running the orbit they had, because the alternative is an
// orbit that cannot upgrade itself back.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// published stands a release up: the archive at one URL and, at another, the
// checksums file that vouches for it.
func published(t *testing.T, name string, archive []byte) (asset, asset) {
	t.Helper()

	sums := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := archive
		if strings.HasSuffix(r.URL.Path, "/sums") {
			body = []byte(sums)
		}

		if _, err := w.Write(body); err != nil {
			t.Errorf("serve %s: %v", r.URL.Path, err)
		}
	}))
	t.Cleanup(ts.Close)

	return asset{Name: name, URL: ts.URL + "/archive"}, asset{Name: "checksums.txt", URL: ts.URL + "/sums"}
}

// dead is a server that is not answering, which is what a reader upgrading on
// a train has.
func dead(t *testing.T) string {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close()

	return ts.URL
}

// TestAnArchiveThatHashesRightAndIsNotOneIsRefused. The hash says the bytes
// arrived as published; it says nothing about what is in them. A release
// built wrong is still a release nobody's orbit should be replaced with.
func TestAnArchiveThatHashesRightAndIsNotOneIsRefused(t *testing.T) {
	bin, sums := published(t, "orbit_2.0.0_test.tar.gz", []byte("this was never an archive"))

	err := selfUpdate(t.Context(), words.For("en"), bin, sums)
	if err == nil {
		t.Fatal("an archive that is not one was installed over orbit")
	}

	if strings.Contains(err.Error(), "hashes to") {
		t.Errorf("the hash is what was refused, and it matched: %v", err)
	}
}

// TestAnArchiveThatEndsInTheMiddleIsRefused: the header of the file named
// orbit arrives, its bytes do not. Whatever turned up so far is not an
// answer.
func TestAnArchiveThatEndsInTheMiddleIsRefused(t *testing.T) {
	body := make([]byte, 8<<10)
	if _, err := rand.NewChaCha8([32]byte{}).Read(body); err != nil {
		t.Fatalf("fill a body gzip cannot shrink: %v", err)
	}

	whole := archiveOf(t, "orbit", body)

	cut := whole[:len(whole)/2]
	if _, err := binaryFrom(words.For("en"), cut); err == nil {
		t.Fatal("half an archive answered a whole binary")
	}
}

// TestSomethingThatIsNotAnArchiveAtAllIsRefused is the gzip header failing
// before any of this gets as far as looking for a file inside.
func TestSomethingThatIsNotAnArchiveAtAllIsRefused(t *testing.T) {
	if _, err := binaryFrom(words.For("en"), []byte("not gzip, not tar, not anything")); err == nil {
		t.Fatal("bytes that are not an archive answered a binary")
	}
}

// TestChecksumsThatCannotBeReadStopTheUpdate. An update whose checksums did
// not arrive is an update that cannot be checked, and one that cannot be
// checked is one that does not happen.
func TestChecksumsThatCannotBeReadStopTheUpdate(t *testing.T) {
	bin, sums := published(t, "orbit_2.0.0_test.tar.gz", archiveOf(t, "orbit", []byte("#!/bin/sh\n")))
	sums.URL = dead(t)

	err := selfUpdate(t.Context(), words.For("en"), bin, sums)
	if err == nil {
		t.Fatal("an unchecked archive was installed over orbit")
	}

	if !strings.Contains(err.Error(), "checksums") {
		t.Errorf("the refusal does not say the checksums were the problem: %v", err)
	}
}

// TestAnArchiveThatDidNotArriveStopsTheUpdate.
func TestAnArchiveThatDidNotArriveStopsTheUpdate(t *testing.T) {
	bin, sums := published(t, "orbit_2.0.0_test.tar.gz", nil)
	bin.URL = dead(t)

	err := selfUpdate(t.Context(), words.For("en"), bin, sums)
	if err == nil {
		t.Fatal("an archive that never arrived was installed over orbit")
	}

	if !strings.Contains(err.Error(), bin.Name) {
		t.Errorf("the refusal does not name the archive that failed: %v", err)
	}
}

// TestAUrlThatIsNotOneIsNotFetched is the request that cannot even be built,
// which is the shape a mistyped updateEndpoint has.
func TestAUrlThatIsNotOneIsNotFetched(t *testing.T) {
	if _, err := download(t.Context(), words.For("en"), "://not a url"); err == nil {
		t.Error("a URL that cannot be parsed was downloaded")
	}

	old := updateEndpoint
	updateEndpoint = "://not a url"

	t.Cleanup(func() { updateEndpoint = old })

	if _, err := fetchRelease(t.Context(), words.For("en")); err == nil {
		t.Error("a URL that cannot be parsed answered a release")
	}
}

// TestAReleaseThatWasNotAnsweredIsNotInstalled covers what github can say
// instead of a release: nothing at all, a status, or a body that is not the
// json this expects. None of them may reach the reader as an empty
// releaseInfo they have to tell apart themselves.
func TestAReleaseThatWasNotAnsweredIsNotInstalled(t *testing.T) {
	for _, c := range []struct {
		name string
		url  func(*testing.T) string
	}{
		{"a server that is not answering", dead},
		{"a status instead of a release", func(t *testing.T) string {
			t.Helper()

			// github answers its refusals in json too, so the body decodes
			// cleanly into a release with no tag and no assets. What makes
			// this one not a release is the status and nothing else.
			return serving(t, func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
			})
		}},
		{"a body that is not a release", func(t *testing.T) string {
			t.Helper()

			return serving(t, func(w http.ResponseWriter) { fmt.Fprint(w, "<html>maintenance</html>") })
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			old := updateEndpoint
			updateEndpoint = c.url(t)

			t.Cleanup(func() { updateEndpoint = old })

			if _, err := fetchRelease(t.Context(), words.For("en")); err == nil {
				t.Error("this was taken for a release")
			}
		})
	}
}

func serving(t *testing.T, write func(http.ResponseWriter)) string {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { write(w) }))
	t.Cleanup(ts.Close)

	return ts.URL
}

// TestAReleaseWithNothingForThisMachineSaysWhich. Publishing nothing for a
// machine is not a crisis — go install builds for the machine it is on — but
// the reason has to reach the reader, or the fallback looks like the plan.
func TestAReleaseWithNothingForThisMachineSaysWhich(t *testing.T) {
	err := installRelease(t.Context(), words.For("en"), releaseInfo{
		TagName: "v2.0.0",
		Assets:  []asset{{Name: "orbit_2.0.0_plan9_mips.tar.gz"}, {Name: "checksums.txt"}},
	})
	if err == nil {
		t.Fatal("a release with no build for this machine reported an install")
	}

	if !strings.Contains(err.Error(), "v2.0.0") {
		t.Errorf("the refusal does not name the release: %v", err)
	}
}

// TestAReleaseWithNoChecksumsIsNotInstalled. There is nothing to check the
// archive against, and an archive nobody checked is not written over a
// binary.
func TestAReleaseWithNoChecksumsIsNotInstalled(t *testing.T) {
	name := fmt.Sprintf("orbit_2.0.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	err := installRelease(t.Context(), words.For("en"), releaseInfo{TagName: "v2.0.0", Assets: []asset{{Name: name}}})
	if err == nil {
		t.Fatal("a release that published no checksums reported an install")
	}

	if !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "checksums.txt") {
		t.Errorf("the refusal does not say which archive had no checksums to check it: %v", err)
	}
}

// archiveOf is one file in a tar.gz, the shape a release publishes.
func archiveOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	if err := tw.WriteHeader(&tar.Header{
		Name: name, Typeflag: tar.TypeReg, Size: int64(len(body)), Mode: 0o755,
	}); err != nil {
		t.Fatalf("write the header of %s: %v", name, err)
	}

	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close the tar: %v", err)
	}

	if err := gzw.Close(); err != nil {
		t.Fatalf("close the gzip: %v", err)
	}

	return buf.Bytes()
}
