package ui

// Reading JSON an engine wrote by hand.
//
// A model asked for JSON writes a prompt with line breaks in it and leaves
// them raw inside the string, which is not JSON — the decoder says "invalid
// character '\n' in string literal" and the reader is left with a refusal
// and a wall of text they cannot fix. Since what is wrong is exactly one
// thing, and mending it changes no valid document, it is mended here rather
// than handed back.

// mendJSON escapes the control characters an engine left raw inside a string.
//
// Outside a string every one of them is whitespace and stays as it is. A
// well-formed document cannot hold them inside one, so this is a no-op on
// anything a machine wrote.
func mendJSON(raw []byte) []byte {
	out := make([]byte, 0, len(raw)+16)

	var (
		inString bool
		escaped  bool
	)

	for _, b := range raw {
		switch {
		case escaped:
			escaped = false
		case b == '\\' && inString:
			escaped = true
		case b == '"':
			inString = !inString
		case inString:
			if mended, ok := escapeIn(b); ok {
				out = append(out, mended...)
				continue
			}
		}

		out = append(out, b)
	}

	return out
}

// escapeIn is the two-character form of a control character, for the ones
// that turn up in a prompt somebody dictated.
func escapeIn(b byte) (string, bool) {
	switch b {
	case '\n':
		return `\n`, true
	case '\r':
		return `\r`, true
	case '\t':
		return `\t`, true
	}

	return "", false
}
