package ui

import "fmt"

// FormatLatency paints a latency by how much of it a person would notice:
// fast disappears in green, slow admits itself in yellow, and past half a
// second turns red because that is roughly where waiting starts to feel
// like waiting. It is pure layout — no clock, no terminal, just ms in and
// a styled string out.
func FormatLatency(ms int64) string {
	text := fmt.Sprintf("%dms", ms)
	switch {
	case ms < 100:
		return Paint(OK).Render(text)
	case ms < 500:
		return Paint(Warn).Render(text)
	default:
		return Paint(Bad).Render(text)
	}
}
