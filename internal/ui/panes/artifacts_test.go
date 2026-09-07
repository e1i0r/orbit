package panes

// The artifacts pane's two small rules: how a size is spelled, and which
// syntax a file of the task's directory is read in.

import "testing"

// TestFormatBytesNamesItsUnit. A size is read off the disk now, so an empty
// file is nought bytes and says so: the listing this replaces clamped every
// size up to one byte because none of them had been measured.
func TestFormatBytesNamesItsUnit(t *testing.T) {
	for _, c := range []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1 k"},
		{2048, "2 k"},
		{1024*1024 - 1, "1023 k"},
		{3 * 1024 * 1024, "3 M"},
	} {
		if got := formatBytes(c.bytes); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

// TestAFileIsReadInTheSyntaxItsNameNames. What is in these files is a
// document and a word, not somebody's Go, and a well that called the record
// a language would paint half of every line as a keyword of it.
func TestAFileIsReadInTheSyntaxItsNameNames(t *testing.T) {
	for name, want := range map[string]string{
		"events.jsonl": "data",
		"run":          "",
		"control":      "",
		"task.md":      "",
	} {
		if got := fileFamily(name); got != want {
			t.Errorf("fileFamily(%q) = %q, want %q", name, got, want)
		}
	}
}
