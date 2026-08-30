package cli

// Installing a release, in the four steps upgrade takes and cannot take back:
// choosing the archive, reading it, checking it, and opening it.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestAMachineIsGivenItsOwnBuild is the collision a substring match has:
// "arm" is inside "arm64", so a 32-bit machine asking for its build is handed
// the 64-bit one, writes it over the running orbit, and is left with a file
// its kernel refuses and no orbit to upgrade with.
func TestAMachineIsGivenItsOwnBuild(t *testing.T) {
	assets := []asset{
		{Name: "orbit_2.0.0_linux_arm64.tar.gz", URL: "https://example.com/arm64"},
		{Name: "orbit_2.0.0_linux_arm.tar.gz", URL: "https://example.com/arm"},
		{Name: "orbit_2.0.0_darwin_arm64.tar.gz", URL: "https://example.com/darwin"},
		{Name: "checksums.txt", URL: "https://example.com/sums"},
	}

	for _, c := range []struct{ goos, goarch, want string }{
		{"linux", "arm", "https://example.com/arm"},
		{"linux", "arm64", "https://example.com/arm64"},
		{"darwin", "arm64", "https://example.com/darwin"},
	} {
		got, ok := findAsset(assets, c.goos, c.goarch)
		if !ok {
			t.Errorf("%s %s was given no build at all", c.goos, c.goarch)
			continue
		}

		if got.URL != c.want {
			t.Errorf("%s %s was given %s (%s), want %s", c.goos, c.goarch, got.Name, got.URL, c.want)
		}
	}

	if got, ok := findAsset(assets, "windows", "amd64"); ok {
		t.Errorf("a machine this release publishes nothing for was given %s", got.Name)
	}
}

func TestTheChecksumFileIsFoundByName(t *testing.T) {
	assets := []asset{
		{Name: "orbit_2.0.0_linux_amd64.tar.gz", URL: "https://example.com/bin"},
		{Name: "CHECKSUMS.txt", URL: "https://example.com/sums"},
	}

	got, ok := findChecksums(assets)
	if !ok || got.URL != "https://example.com/sums" {
		t.Errorf("findChecksums = %v, %v; want the checksums asset", got, ok)
	}

	if _, ok := findChecksums(assets[:1]); ok {
		t.Error("a release with no checksums.txt was said to have one")
	}
}

// TestAnArchiveIsOnlyInstalledIfItIsTheOneThatWasPublished. These bytes are
// about to become the reader's orbit; the one thing that can still be said
// about them at this point is whether they are what the release said they
// would be.
func TestAnArchiveIsOnlyInstalledIfItIsTheOneThatWasPublished(t *testing.T) {
	archive := createFakeArchive(t, "#!/bin/sh\necho updated\n")
	sum := fmt.Sprintf("%x", sha256.Sum256(archive))
	name := "orbit_2.0.0_linux_amd64.tar.gz"

	published := fmt.Sprintf("%s  orbit_2.0.0_darwin_arm64.tar.gz\n%s  %s\n", strings.Repeat("0", 64), sum, name)
	if err := verify(words.For("en"), archive, []byte(published), name); err != nil {
		t.Errorf("the archive that was published was refused: %v", err)
	}

	err := verify(words.For("en"), []byte("something else entirely"), []byte(published), name)
	if err == nil {
		t.Fatal("an archive that hashes to something else was installed")
	}

	if !strings.Contains(err.Error(), sum) {
		t.Errorf("the refusal does not say what was expected: %v", err)
	}

	if err := verify(words.For("en"), archive, []byte(published), "orbit_2.0.0_linux_arm.tar.gz"); err == nil {
		t.Error("an archive nobody published a checksum for was installed anyway")
	}
}

// TestASumWrittenTheWayShaWritesItIsRead: sha256sum -c reads two spaces and a
// "*" before a name it read in binary mode, and goreleaser has published both
// shapes. Neither is part of the file's name.
func TestASumWrittenTheWayShaWritesItIsRead(t *testing.T) {
	archive := []byte("an archive")
	sum := fmt.Sprintf("%x", sha256.Sum256(archive))

	for _, published := range []string{
		sum + "  orbit.tar.gz\n",
		sum + " *orbit.tar.gz\n",
		strings.ToUpper(sum) + "  orbit.tar.gz",
	} {
		if err := verify(words.For("en"), archive, []byte(published), "orbit.tar.gz"); err != nil {
			t.Errorf("verify(%q) = %v, want it read", published, err)
		}
	}
}

// TestOnlyARealFileIsTakenForTheBinary. A tar entry named orbit that is a
// symlink carries no bytes: reading it answers an empty file, and installing
// that leaves an orbit that does nothing and cannot upgrade itself back.
func TestOnlyARealFileIsTakenForTheBinary(t *testing.T) {
	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, h := range []*tar.Header{
		{Name: "orbit", Typeflag: tar.TypeSymlink, Linkname: "/bin/sh", Mode: 0o777},
		{Name: "README.md", Typeflag: tar.TypeReg, Size: 3, Mode: 0o644},
	} {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatalf("write the header of %s: %v", h.Name, err)
		}

		if h.Size > 0 {
			if _, err := tw.Write([]byte("hi\n")); err != nil {
				t.Fatalf("write %s: %v", h.Name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close the tar: %v", err)
	}

	if err := gzw.Close(); err != nil {
		t.Fatalf("close the gzip: %v", err)
	}

	got, err := binaryFrom(words.For("en"), buf.Bytes())
	if err == nil {
		t.Fatalf("an archive with no orbit in it answered %d bytes to install", len(got))
	}
}

// TestTheArchiveComesBackWhole is the reading half against a real server: the
// bytes that arrive are the bytes that were served, and the binary inside
// them is the one that was packed.
func TestTheArchiveComesBackWhole(t *testing.T) {
	body := "#!/bin/sh\necho updated\n"
	archive := createFakeArchive(t, body)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(archive); err != nil {
			t.Errorf("serve the archive: %v", err)
		}
	}))
	defer ts.Close()

	got, err := download(t.Context(), words.For("en"), ts.URL)
	if err != nil {
		t.Fatalf("download: %v", err)
	}

	if !bytes.Equal(got, archive) {
		t.Fatalf("downloaded %d bytes, served %d", len(got), len(archive))
	}

	bin, err := binaryFrom(words.For("en"), got)
	if err != nil {
		t.Fatalf("take orbit out of the archive: %v", err)
	}

	if string(bin) != body {
		t.Errorf("the binary inside the archive is %q, want %q", bin, body)
	}
}

func TestADownloadThatDidNotHappenSaysWhy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := download(t.Context(), words.For("en"), ts.URL)
	if err == nil {
		t.Fatal("a 404 came back as an archive")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("the failure does not say what the server answered: %v", err)
	}
}

// TestADownloadThatWillNotEndIsStopped: this is read into memory whole, so a
// server answering with an endless body would otherwise take the machine down
// with it.
func TestADownloadThatWillNotEndIsStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		block := bytes.Repeat([]byte("x"), 1<<20)
		for range (maxDownload >> 20) + 2 {
			if _, err := w.Write(block); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	_, err := download(ctx, words.For("en"), ts.URL)
	if err == nil {
		t.Fatal("a body larger than the limit was read whole")
	}

	if !strings.Contains(err.Error(), "larger than") {
		t.Errorf("the refusal does not say the body was too large: %v", err)
	}
}
