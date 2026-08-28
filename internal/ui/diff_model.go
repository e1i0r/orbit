package ui

import (
	"strings"
)

// diffFile is one file's structured metadata inside a git diff.
type diffFile struct {
	Path      string
	Status    string // "NEW", "MOD", "DEL"
	Added     int
	Deleted   int
	StartLine int
	EndLine   int
	Hunks     []int
	Rationale string
}

// parseDiffFiles parses raw diff lines into structured files and hunks.
func parseDiffFiles(lines []string) []diffFile {
	var (
		files []diffFile
		cur   *diffFile
	)

	for i, s := range lines {
		if strings.HasPrefix(s, "diff --git ") {
			if cur != nil {
				cur.EndLine = i - 1
				files = append(files, *cur)
			}

			cur = &diffFile{
				Status:    "MOD",
				StartLine: i,
				Hunks:     make([]int, 0),
			}
			// Extract path: diff --git a/foo b/bar
			parts := strings.Fields(s)
			if len(parts) >= 4 {
				cur.Path = strings.TrimPrefix(parts[3], "b/")
			}

			continue
		}

		if cur == nil {
			continue
		}

		switch {
		case strings.HasPrefix(s, "new file mode"):
			cur.Status = "NEW"
		case strings.HasPrefix(s, "deleted file mode"):
			cur.Status = "DEL"
		case strings.HasPrefix(s, "+++ b/"):
			cur.Path = strings.TrimPrefix(s, "+++ b/")
		case strings.HasPrefix(s, "+++ /dev/null"):
			cur.Status = "DEL"
		case strings.HasPrefix(s, "@@"):
			cur.Hunks = append(cur.Hunks, i)
		case strings.HasPrefix(s, "+") && !strings.HasPrefix(s, "+++"):
			cur.Added++
		case strings.HasPrefix(s, "-") && !strings.HasPrefix(s, "---"):
			cur.Deleted++
		}
	}

	if cur != nil {
		cur.EndLine = len(lines) - 1
		files = append(files, *cur)
	}

	return files
}

// diffStats computes aggregate metrics across all parsed files.
func diffStats(files []diffFile) (totalAdded, totalDeleted int) {
	for _, f := range files {
		totalAdded += f.Added
		totalDeleted += f.Deleted
	}

	return totalAdded, totalDeleted
}

// formatFileBadge returns a styled status badge for a diff file.
func formatFileBadge(status string) string {
	switch status {
	case "NEW":
		return Paint(OK).Render("[NEW]")
	case "DEL":
		return Paint(Bad).Render("[DELETED]")
	default:
		return Paint(Accent).Render("[MODIFIED]")
	}
}

// fileIcon returns an emoji icon based on file extension.
func fileIcon(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "🔷"
	case strings.HasSuffix(path, ".json"), strings.HasSuffix(path, ".yaml"),
		strings.HasSuffix(path, ".yml"), strings.HasSuffix(path, ".toml"):
		return "⚙️ "
	case strings.HasSuffix(path, ".md"), strings.HasSuffix(path, ".txt"):
		return "📝"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".js"),
		strings.HasSuffix(path, ".tsx"), strings.HasSuffix(path, ".jsx"):
		return "🟨"
	case strings.HasSuffix(path, ".py"):
		return "🐍"
	case strings.HasSuffix(path, ".sh"):
		return "🐚"
	default:
		return "📄"
	}
}

// formatHunkHeader parses hunk header line numbers: @@ -oldStart,oldLen +newStart,newLen @@.
func formatHunkHeader(hunk string) string {
	parts := strings.SplitN(hunk, "@@", 3)
	if len(parts) < 2 {
		return hunk
	}

	tag := strings.TrimSpace(parts[1])

	context := ""
	if len(parts) > 2 {
		context = strings.TrimSpace(parts[2])
	}

	res := " @@ " + tag + " @@"
	if context != "" {
		res += " " + context
	}

	return res
}
