package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestUpdateDownloadArchive(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		archive := createFakeArchive(t, "#!/bin/sh\necho updated\n")

		w.Header().Set("Content-Type", "application/gzip")

		if _, err := w.Write(archive); err != nil {
			t.Errorf("write archive: %v", err)
		}
	}))
	defer downloadServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := releaseInfo{
			TagName: "v2.0.0",
			Assets: []asset{
				{
					Name: "orbit_2.0.0_darwin_arm64.tar.gz",
					URL:  downloadServer.URL,
				},
				{
					Name: "orbit_2.0.0_linux_amd64.tar.gz",
					URL:  downloadServer.URL,
				},
			},
		}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer apiServer.Close()

	oldEndpoint := updateEndpoint
	updateEndpoint = apiServer.URL

	t.Cleanup(func() { updateEndpoint = oldEndpoint })

	// Test findAssetURL logic
	url := findAssetURL([]asset{
		{Name: "orbit_2.0.0_darwin_arm64.tar.gz", URL: "https://example.com/darwin"},
	}, "darwin", "arm64")
	if url != "https://example.com/darwin" {
		t.Errorf("findAssetURL = %q, want darwin asset", url)
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
