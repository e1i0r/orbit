package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func createFakeArchive(t *testing.T, binaryContent string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	body := []byte(binaryContent)

	hdr := &tar.Header{
		Name: "orbit",
		Mode: 0o755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}

	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestUpdateCheckAlreadyLatest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := releaseInfo{
			TagName: "v1.0.0",
		}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer ts.Close()

	oldEndpoint := updateEndpoint
	updateEndpoint = ts.URL
	oldVersion := Version
	Version = "1.0.0"

	t.Cleanup(func() {
		updateEndpoint = oldEndpoint
		Version = oldVersion
	})

	t.Setenv("ORBIT_HOME", t.TempDir())

	code, out, errOut := run(t, "upgrade")
	if code != 0 {
		t.Fatalf("upgrade exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "already on the latest version") &&
		!strings.Contains(out, "última versión") {
		t.Errorf("expected up to date message, got:\n%s", out)
	}
}

func TestUpdateCheckAvailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := releaseInfo{
			TagName: "v2.0.0",
		}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer ts.Close()

	oldEndpoint := updateEndpoint
	updateEndpoint = ts.URL
	oldVersion := Version
	Version = "1.0.0"

	t.Cleanup(func() {
		updateEndpoint = oldEndpoint
		Version = oldVersion
	})

	t.Setenv("ORBIT_HOME", t.TempDir())

	code, out, errOut := run(t, "upgrade", "-check")
	if code != 0 {
		t.Fatalf("upgrade -check exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "v2.0.0") {
		t.Errorf("expected new version v2.0.0 in output, got:\n%s", out)
	}
}

// TestWhyTheReleaseCouldNotBeInstalledIsSaidOutLoud.
//
// An upgrade has two ways of working and they are not the same thing: the
// release archive is written over this binary, while `go install` builds into
// GOBIN and leaves this one where it is. So a reader whose archive failed and
// whose fallback worked has a newer orbit somewhere on their $PATH, the old
// one still in front of it, and a message saying they were updated. The
// reason the first way failed used to be dropped by an `if err == nil`; here
// it is a 404 on the download, and it has to reach them.
//
// go is taken off the PATH so the fallback fails at once rather than building
// orbit from the network in the middle of a test.
func TestWhyTheReleaseCouldNotBeInstalledIsSaidOutLoud(t *testing.T) {
	downloads := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no such release asset", http.StatusNotFound)
	}))
	defer downloads.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rel := releaseInfo{
			TagName: "v2.0.0",
			Assets: []asset{
				{
					Name: fmt.Sprintf("orbit_2.0.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					URL:  downloads.URL,
				},
				{Name: "checksums.txt", URL: downloads.URL},
			},
		}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer api.Close()

	oldEndpoint, oldVersion := updateEndpoint, Version
	updateEndpoint, Version = api.URL, "1.0.0"

	t.Cleanup(func() { updateEndpoint, Version = oldEndpoint, oldVersion })

	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	code, out, errOut := run(t, "upgrade")
	if code == 0 {
		t.Fatalf("an upgrade that installed nothing exited 0:\n%s\n%s", out, errOut)
	}

	if !strings.Contains(errOut, "404") {
		t.Errorf("the reason the archive could not be installed was not reported:\n%s", errOut)
	}

	if !strings.Contains(errOut, "go install") {
		t.Errorf("the reason the fallback failed was not reported:\n%s", errOut)
	}

	if strings.Contains(out, "successfully updated") {
		t.Errorf("an upgrade that installed nothing reported success:\n%s", out)
	}
}

func TestUpdateHelpAndUnknownFlag(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	code, out, errOut := run(t, "upgrade", "-h")
	if code != 0 {
		t.Errorf("upgrade -h exited %d, want 0: %s", code, errOut)
	}

	if !strings.Contains(out, "orbit upgrade") {
		t.Errorf("upgrade -h does not describe command:\n%s", out)
	}

	code, _, errOut = run(t, "upgrade", "-bogus")
	if code == 0 {
		t.Error("upgrade -bogus exited 0, want refusal")
	}

	if !strings.Contains(errOut, "-bogus") {
		t.Errorf("error did not name unknown flag:\n%s", errOut)
	}
}

// TestWhatGoSaidComesBackWithIt. The fallback shells out, and this was
// cmd.Run(): a missing toolchain, a proxy that would not answer and a compile
// error all reached the reader as the same three words, "exit status 1", with
// the sentence that would have told them which one on a pipe nobody read.
func TestWhatGoSaidComesBackWithIt(t *testing.T) {
	dir := t.TempDir()

	script := "#!/bin/sh\necho 'go: module lookup disabled by GOPROXY=off' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o755); err != nil {
		t.Fatalf("write the go this test stands in for: %v", err)
	}

	t.Setenv("PATH", dir)

	err := goInstall(t.Context())
	if err == nil {
		t.Fatal("a go install that failed came back as a success")
	}

	if !strings.Contains(err.Error(), "module lookup disabled") {
		t.Errorf("what go said did not come back with the failure: %v", err)
	}
}
