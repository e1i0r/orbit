package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestRenderMarkdownRichAndRaw(t *testing.T) {
	md := `# Main Title
## Subtitle
### Section
> Important quote
- [x] Done task
- [ ] Todo task
- Bullet item
1. Numbered item
---
` + "```go\nfmt.Println(\"hello\")\n```"

	// 1. Rich formatting (raw = false)
	rich := renderMarkdown(md, 80, false)
	if len(rich) == 0 {
		t.Fatal("renderMarkdown rich returned empty")
	}

	joined := strings.Join(rich, "\n")
	if !strings.Contains(joined, "Main Title") || !strings.Contains(joined, "Subtitle") {
		t.Errorf("expected headings in rendered markdown, got:\n%s", joined)
	}

	if !strings.Contains(joined, "✔") || !strings.Contains(joined, "☐") {
		t.Errorf("expected checklist symbols in rendered markdown, got:\n%s", joined)
	}

	// 2. Raw formatting (raw = true)
	raw := renderMarkdown(md, 80, true)
	if len(raw) == 0 {
		t.Fatal("renderMarkdown raw returned empty")
	}

	rawJoined := strings.Join(raw, "\n")
	if !strings.Contains(rawJoined, "# Main Title") {
		t.Errorf("expected raw markdown '# Main Title', got:\n%s", rawJoined)
	}
}

func TestToggleMarkdownKeyInDetailView(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "Test task"},
	}
	m, _ = m.openDetail(m.board.Tasks[0])

	if m.rawText {
		t.Error("expected default rawText to be false (formatted by default)")
	}

	// Press 'v' to toggle to raw
	res, _ := m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})

	m = asModel(t, res)
	if !m.rawText {
		t.Error("expected rawText to be true after pressing 'v'")
	}

	// Press 'v' again to toggle back to formatted
	res, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})

	m = asModel(t, res)
	if m.rawText {
		t.Error("expected rawText to be false after pressing 'v' again")
	}
}
