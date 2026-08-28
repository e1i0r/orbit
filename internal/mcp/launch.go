package mcp

// The other way a client gets this server: not written into its
// configuration once, but handed to one session on its command line.
//
// The installer is for the clients a reader opens themselves. This is for
// the session Orbit opens for them — the cockpit's `c` — where writing into
// a configuration file would be both wrong and rude: the reader did not ask
// for every future session to have the server, they pressed a key in a
// window that already knows which task they are looking at.

import (
	"encoding/json"
	"fmt"
)

// LaunchConfig is the mcpServers document that gives one session this
// server, as the JSON a client takes on its command line.
//
// binaryPath may be empty, in which case the running executable is named.
// The entry is the installer's own, so a session started from the cockpit
// talks to the same server a client configured by `orbit mcp install` does.
func LaunchConfig(binaryPath string) (string, error) {
	doc := map[string]any{"mcpServers": map[string]any{serverName: entry(binary(binaryPath))}}

	raw, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("encode the mcp configuration for an interactive session: %w", err)
	}

	return string(raw), nil
}
