package engine

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// scanJSONLines hands every JSON object on a stream to onLine.
//
// Blank lines and anything not starting with a brace are skipped: all three
// engines print the odd human sentence on stdout beside their events —
// codex's "Reading additional input from stdin..." arrives before its first
// object — and a line this cannot read is not a reason to stop reading the
// rest.
func scanJSONLines(r io.Reader, onLine func([]byte)) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxStreamLine)

	lines := 0

	for sc.Scan() {
		lines++

		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		onLine(line)
	}

	if err := sc.Err(); err != nil {
		return lines, fmt.Errorf("reading the engine's stream after %d lines: %w", lines, err)
	}

	return lines, nil
}

// emit calls back with one event when anybody is listening.
func emit(onEvent func(StreamEvent), ev StreamEvent) {
	if onEvent != nil {
		onEvent(ev)
	}
}

// firstNonEmpty is the first of its arguments that says something.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}

	return ""
}

// silentStream is what an engine's parser returns when the stream held no
// event it recognised.
//
// A phase that produced nothing and a phase whose event shape moved under us
// look identical from the outside — both end with an empty Result — and only
// one of them is a phase. Saying so is what makes the second one noticed;
// spec.report decides what to do about it, and keeps whatever the engine did
// print in plain text.
func silentStream(engine string, lines int) error {
	return fmt.Errorf(
		"%s's stream ended after %d lines with no event this build recognises: the session id and the cost are reported only there, so this phase has nothing to resume from and no price",
		engine, lines)
}
