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
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/web"
	"github.com/e1i0r/orbit/internal/words"
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

	s, err := store.Open()
	if err != nil {
		return err
	}

	r := board.NewReader(s, dir)
	if err := r.Rescan(); err != nil {
		return fmt.Errorf("%s: %w", p.T("web.rescan", "look for repositories"), err)
	}

	handler := web.New(r, r, dir).Handler()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("%s: %w", p.T("web.listen", "listen on {addr}",
			words.Arg{Name: "addr", Value: *addr}), err)
	}

	fmt.Fprintf(ctx.Out, "%s\n", p.T("web.serving", "orbit is at http://{addr} — press ctrl-c to stop",
		words.Arg{Name: "addr", Value: listener.Addr().String()}))

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
