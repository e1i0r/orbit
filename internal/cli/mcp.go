package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/mcp"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// runMCP is `orbit mcp`: the server a supervising model talks to, and the
// installer that registers it.
//
// The two live under one command because they are one feature and because
// the argument list is what the installer writes into every client's
// configuration — `orbit mcp` with no flags has to be the server, for ever,
// or an install performed today stops working when the command's default
// changes.
func runMCP(ctx Context, args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	install := fs.Bool("install", false, "register orbit as an MCP server in Claude Desktop, Claude Code and Cursor")
	// The root is the answer to the one thing a client gets wrong. A desktop
	// application spawns this process with a working directory of its own
	// bundle, so the default is not the working directory: it is every
	// repository the state root already has a record of. This flag is for
	// the other case — a tree of checkouts none of which has a task yet.
	root := fs.String("root", "", "look for repositories under this directory instead of the ones orbit already knows")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}
	if *install || fs.Arg(0) == "install" {
		return installMCP(ctx)
	}
	if rest := fs.Arg(0); rest != "" {
		return fmt.Errorf("mcp takes no arguments but %q; `orbit mcp` is the server and `orbit mcp install` registers it", rest)
	}
	return serveMCP(ctx, *root)
}

// serveMCP runs the server on this process's own standard input and output.
//
// ctx.Out is deliberately not used. The tests in this package hand every
// command a buffer, and a JSON-RPC server writing into a buffer nobody reads
// would be a server that appears to work; more importantly, the client is
// reading this process's stdout and nothing else may be written to it. The
// logger goes to a file under the state root, which is what makes that
// possible.
func serveMCP(ctx Context, root string) error {
	s, err := store.Open()
	if err != nil {
		return err
	}
	if logErr := logger.Init(s.LogPath()); logErr == nil {
		defer func() { _ = logger.CloseGlobal() }() //nolint:errcheck // best-effort logger flush on process exit
		logger.Info("cli/mcp", "mcp server started (root=%q, version=%s)", root, Version)
	}
	session := mcp.Session{Root: root, Version: Version}
	if err := mcp.NewServer(os.Stdin, os.Stdout, session).Serve(); err != nil {
		logger.Error("cli/mcp", "mcp server stopped: %v", err)
		return fmt.Errorf("the mcp server: %w", err)
	}
	logger.Info("cli/mcp", "mcp server stopped: the client closed the connection")
	return nil
}

// installMCP registers this binary in every client configuration it finds.
//
// A client that could not be written is reported and does not stop the
// others: somebody with Cursor and no Claude Desktop should not have the
// install fail over an application they have never had.
func installMCP(ctx Context) error {
	p := ctx.Words
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find the home directory the clients keep their configuration in: %w", err)
	}
	fmt.Fprintln(ctx.Out, p.T("mcp.installing", "registering orbit as an mcp server"))
	failed := 0
	for _, res := range mcp.Install("", home) {
		if res.Err != nil {
			failed++
			logger.Error("cli/mcp", "register in %s at %q failed: %v", res.Target, res.Path, res.Err)
			fmt.Fprintf(ctx.Out, "  %s\n", p.T("mcp.install_failed", "{client}: {err}",
				words.Arg{Name: "client", Value: res.Target}, words.Arg{Name: "err", Value: res.Err.Error()}))
			continue
		}
		logger.Info("cli/mcp", "registered in %s at %q", res.Target, res.Path)
		fmt.Fprintf(ctx.Out, "  %s\n", p.T("mcp.install_wrote", "{client} — {path}",
			words.Arg{Name: "client", Value: res.Target}, words.Arg{Name: "path", Value: res.Path}))
	}
	if failed > 0 {
		return fmt.Errorf("%d client configurations could not be written", failed)
	}
	fmt.Fprintln(ctx.Out, p.T("mcp.install_restart", "restart the client to connect; it will run `orbit mcp`"))
	return nil
}
