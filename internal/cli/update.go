package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/words"
)

const defaultRepo = "e1i0r/orbit"

var updateEndpoint = ""

type releaseInfo struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func update(ctx Context, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	check := fs.Bool("check", false, "check for updates without installing")
	force := fs.Bool("force", false, "force update even if up to date")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	p := ctx.Words
	fmt.Fprintf(ctx.Out, "%s\n", p.T("update.checking", "checking for updates..."))

	reqCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rel, err := fetchRelease(reqCtx)
	if err != nil {
		return fmt.Errorf("fetch update: %w", err)
	}

	latestVer := strings.TrimPrefix(rel.TagName, "v")
	curVer := strings.TrimPrefix(Version, "v")

	if !*force && curVer != "dev" && curVer == latestVer {
		fmt.Fprintf(ctx.Out, "%s\n", p.T("update.already_latest",
			"orbit is already on the latest version ({version})",
			updateArg("version", rel.TagName)))
		return nil
	}

	if *check {
		fmt.Fprintf(ctx.Out, "%s\n", p.T("update.available",
			"new version available: {latest} (current: {current})",
			updateArg("latest", rel.TagName), updateArg("current", Version)))
		return nil
	}

	fmt.Fprintf(ctx.Out, "%s\n", p.T("update.installing",
		"updating orbit from {current} to {latest}...",
		updateArg("current", Version), updateArg("latest", rel.TagName)))

	// Find matching binary asset
	assetURL := findAssetURL(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if assetURL != "" {
		if err := selfUpdate(reqCtx, assetURL); err == nil {
			fmt.Fprintf(ctx.Out, "%s\n", p.T("update.success",
				"successfully updated orbit to {version}!",
				updateArg("version", rel.TagName)))
			return nil
		}
	}

	// Fallback to go install if asset not found or binary replacement failed
	if err := goInstall(reqCtx); err != nil {
		return fmt.Errorf("update orbit: %w", err)
	}

	fmt.Fprintf(ctx.Out, "%s\n", p.T("update.success",
		"successfully updated orbit to {version}!",
		updateArg("version", rel.TagName)))
	return nil
}

func updateArg(k, v string) words.Arg {
	return words.Arg{Name: k, Value: v}
}

func fetchRelease(ctx context.Context) (releaseInfo, error) {
	url := updateEndpoint
	if url == "" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", defaultRepo)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("User-Agent", "orbit/"+Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("github api responded %d", resp.StatusCode)
	}

	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return releaseInfo{}, err
	}
	return rel, nil
}

func findAssetURL(assets []asset, goos, goarch string) string {
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) &&
			strings.HasSuffix(name, ".tar.gz") {
			return a.URL
		}
	}
	return ""
}

func selfUpdate(ctx context.Context, downloadURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download responded %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Name == "orbit" || strings.HasSuffix(header.Name, "/orbit") {
			binBytes, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			return replaceExecutable(binBytes)
		}
	}
	return fmt.Errorf("executable not found in archive")
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
	if err := tmpFile.Chmod(0755); err != nil {
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

func goInstall(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "install", "github.com/"+defaultRepo+"/cmd/orbit@latest")
	return cmd.Run()
}
