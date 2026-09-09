package cli

// orbit web: the record in a browser.
//
// Served and not packaged. Orbit runs where the code is, which is often not
// the machine the reader is sitting at, so a browser pointed at the machine
// doing the work is the shape that fits — and a desktop shell would need
// this server underneath it anyway. That makes the shell a later decision
// rather than a fork in the road.

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/web"
	"github.com/e1i0r/orbit/internal/words"
	"github.com/e1i0r/orbit/ui"
)

// webAddr is where the server listens when nobody says otherwise.
//
// Loopback and not every interface: what this serves is somebody's whole
// record — task titles, prompts, diffs — and a default that put it on the
// network would be a decision made for the reader rather than by them.
const webAddr = "127.0.0.1:7777"

// webTimeouts bound one request. A diff of a large worktree is the slowest
// thing here and it is git's own deadline that governs it; these are the
// backstop for a client that stops reading.
const (
	webRead  = 30 * time.Second
	webWrite = 5 * time.Minute
	webIdle  = 2 * time.Minute
)

// rescanning looks for repositories and tasks again, for as long as the
// process lives.
//
// It never stops, and that is the whole of its lifetime: this goroutine
// outlives nothing, because `orbit web` runs until the terminal it was
// started in ends. A failed scan is logged and not fatal — the board that
// was already read is still the right answer, and a server that exited
// because one walk of a directory failed would be a server that exits when
// somebody moves a folder.
func rescanning(r *board.Reader) {
	for range time.Tick(board.RescanEvery) {
		if err := r.Rescan(); err != nil {
			logger.Info("cli/web", "look for repositories again: %v", err)
		}
	}
}

// serveWeb runs the server until the process is stopped.
func serveWeb(ctx Context, args []string) error {
	p := ctx.printer()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	addr := fs.String("addr", webAddr, "the address to listen on")

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	dir, err := oneDirectory(ctx, fs.Args())
	if err != nil {
		return err
	}

	// Resolved, because it is shown: a board headed "." tells the reader
	// nothing about which tree they are looking at, and they may be looking
	// at it from another machine.
	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("%s: %w", p.T("web.resolve", "resolve the directory"), err)
	}

	s, err := store.Open()
	if err != nil {
		return err
	}

	r := board.NewReader(s, dir)
	if err := r.Rescan(); err != nil {
		return fmt.Errorf("%s: %w", p.T("web.rescan", "look for repositories"), err)
	}

	// Refused here rather than at the first request. A binary built without
	// `make ui` carries no window, and a server that started, printed a URL
	// and answered every page with an error is a server that looks broken
	// where it is only incomplete.
	if !ui.Built() {
		return errors.New(p.T("web.no_window",
			"this orbit was built without the window; run `make ui` and build it again"))
	}

	files, err := ui.Files()
	if err != nil {
		return fmt.Errorf("%s: %w", p.T("web.built", "read the window built into orbit"), err)
	}

	// And again on a clock, for as long as the server runs. Refresh only
	// re-reads the tasks the reader already knows, so a task written at a
	// terminal — or by any other process — never reached a browser tab that
	// was already open: the page polled every three seconds and was told
	// the same board every time. The window has ticked this since it was
	// written; the server had it only at startup.
	go rescanning(r)

	ports := webPorts(r, s, newEngines(), dir, ctx.printer())
	ports.Files = files

	handler := web.New(ports).Handler()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("%s: %w", p.T("web.listen", "listen on {addr}",
			words.Arg{Name: "addr", Value: *addr}), err)
	}

	said := p.T("web.serving", "orbit is at http://{addr} — press ctrl-c to stop",
		words.Arg{Name: "addr", Value: listener.Addr().String()})

	fmt.Fprintf(ctx.Out, "%s\n", said)

	logger.Info("cli/web", "orbit web started on %q over %q", listener.Addr().String(), dir)

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: webRead,
		WriteTimeout:      webWrite,
		IdleTimeout:       webIdle,
	}

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("%s: %w", p.T("web.serve", "serve the window"), err)
	}

	return nil
}

// oneDirectory is the directory the board is of: the one argument, or the
// working directory. More than one is a refusal rather than a guess, the way
// `orbit top` refuses it.
func oneDirectory(ctx Context, args []string) (string, error) {
	switch len(args) {
	case 0:
		return ".", nil
	case 1:
		return args[0], nil
	default:
		return "", errors.New(ctx.printer().T("web.one_directory",
			"web watches one directory; pass one, or none for the one you are in"))
	}
}
