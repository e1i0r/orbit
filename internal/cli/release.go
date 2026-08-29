package cli

// Installing a release: reading the archive off the network, checking it
// against what the release published, and putting the binary inside it in
// place of the one that is running. `orbit upgrade` decides whether to; this
// is how.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// maxDownload is larger than any orbit archive has been and small enough that
// a server answering with something else cannot exhaust the machine reading
// it. The whole archive is held at once on purpose: a hash cannot be checked
// against a stream that has already been written over the executable.
const maxDownload = 64 << 20

// selfUpdate downloads one release archive, checks it against the hash the
// release published for it, and puts the binary inside it in place of the one
// that is running.
//
// The check is not ceremony. This function overwrites the executable running
// it with bytes that arrived over a network; unchecked, a truncated download
// or a proxy answering with something of its own leaves a reader with no
// working orbit and therefore no way to run `orbit upgrade` again.
func selfUpdate(ctx context.Context, bin, sums asset) error {
	archive, err := download(ctx, bin.URL)
	if err != nil {
		return fmt.Errorf("download %s: %w", bin.Name, err)
	}

	published, err := download(ctx, sums.URL)
	if err != nil {
		return fmt.Errorf("read the checksums of this release: %w", err)
	}

	if err := verify(archive, published, bin.Name); err != nil {
		return err
	}

	newOrbit, err := binaryFrom(archive)
	if err != nil {
		return err
	}

	return replaceExecutable(newOrbit)
}

// download reads one URL whole, and refuses a body that will not end.
func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "orbit/"+Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("responded %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}

	if len(body) > maxDownload {
		return nil, fmt.Errorf("is larger than %d bytes", maxDownload)
	}

	return body, nil
}

// verify is what the release said one archive would hash to, against what
// arrived.
//
// The file is one "hash  name" line per asset — what sha256sum writes and
// what sha256sum -c reads. A name that is not in it is refused rather than
// waved through: an archive nobody published a hash for is an archive nobody
// is standing behind.
func verify(archive, published []byte, name string) error {
	var want string

	for line := range strings.SplitSeq(string(published), "\n") {
		hash, file, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		// sha256sum separates with two spaces and marks a file it read in
		// binary mode with a "*". Neither is part of the name.
		if strings.TrimPrefix(strings.TrimSpace(file), "*") == name {
			want = hash
			break
		}
	}

	if want == "" {
		return fmt.Errorf("this release publishes no checksum for %s", name)
	}

	got := fmt.Sprintf("%x", sha256.Sum256(archive))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("%s hashes to %s and this release published %s", name, got, want)
	}

	return nil
}

// binaryFrom is the orbit inside a release archive.
//
// Regular files only. A tar entry named orbit that is a symlink or a
// directory carries no bytes, and reading it answers an empty file that would
// then be written over the executable — leaving a reader with an orbit that
// does nothing and cannot upgrade itself back.
func binaryFrom(archive []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		if header.Name == "orbit" || strings.HasSuffix(header.Name, "/orbit") {
			return entry(tr, header.Name)
		}
	}

	return nil, fmt.Errorf("executable not found in archive")
}

// entry is one file out of the archive, and no more of it than the ceiling
// the download was already held to.
//
// io.ReadAll(io.LimitReader(tr, maxDownload)) answers maxDownload bytes and
// no error at all when the entry is bigger than that, so an archive over the
// ceiling was cut in the middle and the half of a binary that came out of it
// was written over the running orbit. download refuses a body it cannot hold
// whole; this is the same bytes one layer in, and refuses the same way.
func entry(tr io.Reader, name string) ([]byte, error) {
	bin, err := io.ReadAll(io.LimitReader(tr, maxDownload+1))
	if err != nil {
		return nil, err
	}

	if len(bin) > maxDownload {
		return nil, fmt.Errorf("%s in this archive is larger than %d bytes", name, maxDownload)
	}

	return bin, nil
}

// replaceExecutable puts these bytes in place of the orbit that is running.
//
// The symlink is resolved first because the rename below has to land on the
// file itself: renaming over a symlink replaces the link, and the reader ends
// up with a new orbit at the name they type and the old one still sitting
// wherever the link used to point.
func replaceExecutable(newBytes []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	return replaceFile(execPath, newBytes)
}

// replaceFile writes the new binary beside the old one and moves it on top in
// one step, so that a write cut short halfway never leaves a reader holding
// half an orbit at the name they type.
//
// It is a function of its own for the reason parseBootTime is one: what it
// does is worth a test, and the only path replaceExecutable can hand it is
// the binary running that test.
func replaceFile(execPath string, newBytes []byte) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "orbit-update-*")
	if err != nil {
		return err
	}

	tmpName := tmpFile.Name()

	if err := fill(tmpFile, newBytes); err != nil {
		return errors.Join(err, os.Remove(tmpName))
	}

	if err := os.Rename(tmpName, execPath); err != nil {
		return errors.Join(err, os.Remove(tmpName))
	}

	return nil
}

// fill writes the new binary into the temporary file and closes it, whatever
// happens.
//
// Every one of these used to be a bare tmpFile.Close() beside a discarded
// os.Remove: a full disk reached the reader as the write error alone, and the
// half-written orbit-update-* it left behind was never mentioned. A failure
// to clean up is the reader's to know about, because the file is sitting next
// to their binary.
func fill(f *os.File, newBytes []byte) error {
	if _, err := f.Write(newBytes); err != nil {
		return errors.Join(err, f.Close())
	}

	if err := f.Chmod(0o755); err != nil {
		return errors.Join(err, f.Close())
	}

	return f.Close()
}
