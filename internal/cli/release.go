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

	return extract(archive)
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

// extract puts the orbit inside a release archive in place of the one that is
// running.
func extract(archive []byte) error {
	bin, err := binaryFrom(archive)
	if err != nil {
		return err
	}

	return replaceExecutable(bin)
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
			return io.ReadAll(io.LimitReader(tr, maxDownload))
		}
	}

	return nil, fmt.Errorf("executable not found in archive")
}

func replaceExecutable(newBytes []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(execPath)

	tmpFile, err := os.CreateTemp(dir, "orbit-update-*")
	if err != nil {
		return err
	}

	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(newBytes); err != nil {
		tmpFile.Close()

		_ = os.Remove(tmpName)

		return err
	}

	if err := tmpFile.Chmod(0o755); err != nil {
		tmpFile.Close()

		_ = os.Remove(tmpName)

		return err
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, execPath)
}
