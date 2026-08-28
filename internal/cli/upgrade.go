package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
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

func upgrade(ctx Context, args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	check := fs.Bool("check", false, "check for updates without installing")

	force := fs.Bool("force", false, "force update even if up to date")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	p := ctx.Words
	fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.checking", "checking for updates..."))

	reqCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rel, err := fetchRelease(reqCtx)
	if err != nil {
		return fmt.Errorf("fetch update: %w", err)
	}

	latestVer := strings.TrimPrefix(rel.TagName, "v")
	curVer := strings.TrimPrefix(Version, "v")

	if !*force && curVer != "dev" && curVer == latestVer {
		fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.already_latest",
			"orbit is already on the latest version ({version})",
			updateArg("version", rel.TagName)))

		return nil
	}

	if *check {
		fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.available",
			"new version available: {latest} (current: {current})",
			updateArg("latest", rel.TagName), updateArg("current", Version)))

		return nil
	}

	fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.installing",
		"updating orbit from {current} to {latest}...",
		updateArg("current", Version), updateArg("latest", rel.TagName)))

	// The release archive first, and — if it could not be used — the reason,
	// out loud. `go install` is a real second way, but it needs a Go
	// toolchain and it writes into GOBIN rather than over this binary, so a
	// reader whose archive quietly failed ends up with a newer orbit
	// somewhere else on their $PATH and the old one still in front of it.
	// That sentence used to be dropped by an `if err == nil`, and the fallback
	// then reported success for having updated a different file.
	failed := installRelease(reqCtx, rel)
	if failed == nil {
		fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.success",
			"successfully updated orbit to {version}!",
			updateArg("version", rel.TagName)))

		return nil
	}

	fmt.Fprintf(ctx.Err, "%s\n", p.T("upgrade.fell_back",
		"could not install the release archive ({reason}); trying go install instead",
		updateArg("reason", failed.Error())))

	if err := goInstall(reqCtx); err != nil {
		return fmt.Errorf("update orbit: %w; the release archive was not installed either: %w", err, failed)
	}

	fmt.Fprintf(ctx.Out, "%s\n", p.T("upgrade.success",
		"successfully updated orbit to {version}!",
		updateArg("version", rel.TagName)))

	return nil
}

// installRelease puts the binary this release publishes for this machine over
// the one that is running.
//
// Both halves can say no, and both refusals are the caller's to report: a
// release with nothing for this machine, and a release whose archive does not
// hash to what it said it would.
func installRelease(ctx context.Context, rel releaseInfo) error {
	bin, ok := findAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return fmt.Errorf("release %s publishes no %s %s archive", rel.TagName, runtime.GOOS, runtime.GOARCH)
	}

	sums, ok := findChecksums(rel.Assets)
	if !ok {
		return fmt.Errorf("release %s publishes no checksums.txt, so %s cannot be checked", rel.TagName, bin.Name)
	}

	return selfUpdate(ctx, bin, sums)
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

// findAsset is the archive a release publishes for one operating system on
// one architecture.
//
// The name is matched on "_goos_goarch." rather than on the two words
// appearing anywhere in it, which is what this did: strings.Contains(name,
// "arm") is true of orbit_1.2.3_linux_arm64.tar.gz, so a 32-bit machine
// downloaded the 64-bit build, wrote it over the running orbit and left the
// reader with an executable their kernel refuses — and no orbit to run
// `orbit upgrade` with. Publishing nothing for a machine is not a problem;
// go install builds for the machine it is on.
func findAsset(assets []asset, goos, goarch string) (asset, bool) {
	want := fmt.Sprintf("_%s_%s.", strings.ToLower(goos), strings.ToLower(goarch))
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, want) && strings.HasSuffix(name, ".tar.gz") {
			return a, true
		}
	}

	return asset{}, false
}

// findChecksums is the file a release publishes its hashes in. There is one
// per release rather than one per archive, which is why it is looked for by
// name and not beside the archive.
func findChecksums(assets []asset) (asset, bool) {
	for _, a := range assets {
		if strings.EqualFold(a.Name, "checksums.txt") {
			return a, true
		}
	}

	return asset{}, false
}

// goInstall builds the latest release on this machine, which is the way out
// for a release that published nothing this machine can run.
//
// What go said comes back with it. This was cmd.Run(), so a missing toolchain,
// a module proxy that would not answer and a compile error all reached the
// reader as the same three words: exit status 1.
func goInstall(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "install", "github.com/"+defaultRepo+"/cmd/orbit@latest")

	out, err := cmd.CombinedOutput()
	if err != nil {
		if said := strings.TrimSpace(string(out)); said != "" {
			return fmt.Errorf("go install: %w: %s", err, said)
		}

		return fmt.Errorf("go install: %w", err)
	}

	return nil
}
