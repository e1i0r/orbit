package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// maxStreamLine is the longest single line ParseStream will read.
//
// One line of claude's stream is one message, and a message carrying a whole
// file can be large; a scanner with the default 64KiB limit would stop part
// way through a run and report a truncated stream as a broken one. Four
// mebibytes is the same order as the record's own line limit, which is what
// bounds the event this ends up in, so nothing survives here that the log
// could not hold anyway. It is a constant rather than a borrowed one because
// internal/engine imports nothing of Orbit's.
const maxStreamLine = 4 << 20

// terminalResult is the object claude prints last, and the only one this
// package reads.
//
// Cost is a plain float64 rather than a pointer: a result object that omits
// the number and one that reports zero are the same fact to a reader of the
// record, which Result's own doc comment already states — an engine that
// does not report a cost is a fact about that engine, not a failure.
type terminalResult struct {
	Type      string  `json:"type"`
	Result    string  `json:"result"`
	SessionID string  `json:"session_id"`
	Cost      float64 `json:"total_cost_usd"`
}

// ParseStream reads claude's streaming JSON and returns what the record
// keeps: the human-readable answer, the session id, and what it cost.
//
// It is a function taking an io.Reader rather than a few lines inside
// Claude.Run so that it can be tested against checked-in bytes with no
// binary present — the same reason claudeArgs was split out. A real headless
// run spends real money, and a suite that spends money is a suite nobody
// runs.
//
// Only the terminal result object is read. The stream also carries every
// assistant turn and every tool result, and none of that belongs in an event
// the window will print in a pane: the result field is claude's own summary
// of the run, in prose, which is what a reader of phase.finished wants. If
// two result objects somehow arrive, the last one wins, because the last one
// is the terminal one.
//
// A stream that ends without a result object is an error naming what was
// missing rather than a zero Result. The session id and the cost live
// nowhere else, so a silent empty answer would write "this phase cost
// nothing and can never be resumed" into an append-only log, which is a lie
// that outlives the run that told it.
func ParseStream(r io.Reader) (Result, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxStreamLine)

	var out Result
	var lines int
	var found bool
	for sc.Scan() {
		lines++
		line := bytes.TrimSpace(sc.Bytes())
		// A binary that prints a warning before it starts streaming is not
		// this function's failure to report. The failure it reports is the
		// absence of a result object, and a stream of nothing but noise
		// still hits it.
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var obj terminalResult
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}
		if obj.Type != "result" {
			continue
		}
		found = true
		out = Result{Output: obj.Result, SessionID: obj.SessionID, Cost: obj.Cost}
	}
	if err := sc.Err(); err != nil {
		return Result{}, fmt.Errorf("reading the engine's stream after %d lines: %w", lines, err)
	}
	if !found {
		return Result{}, fmt.Errorf("the engine's stream ended after %d lines with no result object: the session id and the cost are reported only there, so this phase has no answer, nothing to resume from and no price", lines)
	}
	return out, nil
}
