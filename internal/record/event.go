// Package record is the append-only log of what happened, and the only thing
// in Orbit that is authoritative. Every view — the command line today, the
// window tomorrow — is derived from it and can be thrown away.
package record

import "time"

// Event is one thing that happened, on one line of JSON.
//
// The shape is deliberately flat and small. A record that needs a schema
// migration to stay readable is not a record you can still read in a year
// with `cat`.
type Event struct {
	At    time.Time         `json:"at"`
	Kind  string            `json:"kind"`
	Phase string            `json:"phase,omitempty"`
	Text  string            `json:"text,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}
