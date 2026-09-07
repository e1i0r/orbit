package cells

// Cells, lines and the small answers about a value that every screen needs.
//
// The window measures in cells and not in bytes: a terminal draws a wide
// rune in two columns and a combining mark in none, so a cut made on the
// byte count leaves a row that is the wrong width and, half the time, a
// broken escape sequence with it. Everything here counts what a reader can
// see.

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Fit cuts one line to w cells, counting cells and never bytes.
//
// The tail is an ellipsis rather than nothing, because a line that was cut
// and a line that happened to end there are two different facts and a reader
// deciding whether to widen the terminal needs to tell them apart.
func Fit(text string, w int) string {
	if w <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= w {
		return text
	}

	return ansi.Truncate(text, w, "…")
}

// Fill pads a region out to the number of rows it was given, and cuts it to
// that number if a caller overshot. Every region goes through it, so the
// claim "the frame is exactly h rows" holds by construction rather than by
// four separate arguments.
func Fill(lines []string, h int) []string {
	for len(lines) < h {
		lines = append(lines, "")
	}

	if len(lines) > h {
		return lines[:max(h, 0)]
	}

	return lines
}

// PadRight fills a string out to a width in cells, and leaves alone anything
// already that wide or wider.
func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}

	return s + strings.Repeat(" ", width-w)
}

// Lines breaks a paragraph into the rows it is drawn as, at word boundaries
// where it can and mid-word where a word is wider than the room.
func Lines(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}

	var res []string

	var words []string
	for _, w := range strings.Fields(text) {
		words = append(words, chop(w, maxW)...)
	}

	if len(words) == 0 {
		return []string{""}
	}

	var cur strings.Builder

	for _, w := range words {
		switch {
		case cur.Len() == 0:
			cur.WriteString(w)
		case lipgloss.Width(cur.String())+1+lipgloss.Width(w) <= maxW:
			cur.WriteString(" " + w)
		default:
			res = append(res, cur.String())
			cur.Reset()
			cur.WriteString(w)
		}
	}

	if cur.Len() > 0 {
		res = append(res, cur.String())
	}

	return res
}

// OrDef is a value, or what to show when there is none.
func OrDef(s, def string) string {
	if s == "" {
		return def
	}

	return s
}

// First is the head of a list, or "" when there is none.
func First(list []string) string {
	if len(list) == 0 {
		return ""
	}

	return list[0]
}

// TrimLastRune removes the last character of a line being typed, counting
// runes and never bytes: backspacing "café" a byte at a time leaves an
// invalid string on screen, which is the same mistake as measuring a column
// in bytes. The palette backspaces through here too.
func TrimLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}

	return string(runes[:len(runes)-1])
}

// DialLabel is what the option at i on a dial is drawn as: the label beside
// it when there is one, and the option itself when there is not.
//
// It takes two slices rather than a slice of pairs because the ids are what
// every dial in this package already holds, compares and stores, and the
// labels are only ever read at the moment of drawing.
func DialLabel(ids, labels []string, i int) string {
	if i < 0 || i >= len(ids) {
		return ""
	}

	if i < len(labels) {
		return labels[i]
	}

	return ids[i]
}

// chop breaks a word wider than the measure into pieces of it.
//
// A word with no space in it is a path, a URL, or the JSON a tool call was
// made with, and there is nowhere in it to wrap. Left whole it comes back
// from the wrap longer than it was wrapped to, and whoever draws it cuts it
// to the measure — so the tail of the word is on no line at all, and a fold
// that counts lines is told there is nothing to open.
func chop(word string, maxW int) []string {
	if lipgloss.Width(word) <= maxW {
		return []string{word}
	}

	return strings.Split(ansi.Hardwrap(word, maxW, false), "\n")
}

// WrapKeeping breaks a paragraph into drawn rows and keeps the line breaks
// the writer put there: a list of checks, one per line, is a list and not a
// run of words.
func WrapKeeping(text string, maxLen int) []string {
	if strings.TrimSpace(text) == "" || maxLen <= 0 {
		return nil
	}

	var lines []string

	// A line the writer ended is a line: these fields hold paragraphs now,
	// and wrapping them as one run of words would join a list of checks
	// into a sentence.
	for _, para := range strings.Split(text, "\n") {
		lines = append(lines, wrapParagraph(para, maxLen)...)
	}

	return lines
}

// wrapOne folds one paragraph, which has no newlines left in it.
func wrapParagraph(text string, maxLen int) []string {
	wordsList := strings.Fields(text)
	if len(wordsList) == 0 {
		return []string{""}
	}

	var lines []string

	curr := ""

	for _, wd := range wordsList {
		switch {
		case curr == "":
			curr = wd
		case lipgloss.Width(curr)+1+lipgloss.Width(wd) <= maxLen:
			curr += " " + wd
		default:
			lines = append(lines, curr)
			curr = wd
		}
	}

	if curr != "" {
		lines = append(lines, curr)
	}

	return lines
}

// NextOption is the option one turn of a dial away, in either direction,
// coming round at both ends.
func NextOption(options []string, current string, delta int) string {
	if len(options) == 0 {
		return current
	}

	idx := 0

	for i, opt := range options {
		if opt == current {
			idx = i
			break
		}
	}

	nextIdx := (idx + delta) % len(options)
	if nextIdx < 0 {
		nextIdx += len(options)
	}

	return options[nextIdx]
}

// Spread puts one thing at the left of a line and another at the right, with
// at least one space between them, and gives the right-hand one up entirely
// when they will not both fit.
//
// Dropping it whole is the point. A right-hand hint truncated to "unread cap
// reac…" costs the reader the number, which was the only part of it worth
// the cells.
func Spread(left, right string, w int) string {
	if right == "" {
		return Fit(left, w)
	}

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return Fit(left, w)
	}

	return left + strings.Repeat(" ", gap) + right
}
