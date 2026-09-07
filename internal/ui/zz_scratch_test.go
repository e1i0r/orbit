package ui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/markdown"
)

func TestScratchLinkWrap(t *testing.T) {
	src := "Pull request status\n\n- PR #97: [ORB-102: Add a helper function `FormatBytes(b int64) string` in internal/ui/bytes.go](https://github.com/e1i0r/orbit/pull/97)\n- State: MERGED into main\n"
	for _, l := range markdown.Render(src, 60, false) {
		fmt.Printf("%q\n", ansi.Strip(l))
	}
}
