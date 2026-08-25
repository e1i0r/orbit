package tracker

// Package tracker provides extensible parsing for issue tracker URLs (Linear,
// Jira, GitHub, GitLab) and formats standardized prompts with MCP instructions.

// IsTrackerURL reports whether rawURL matches any known tracker provider.
func IsTrackerURL(rawURL string) bool {
	for _, p := range registry {
		if p.Match(rawURL) {
			return true
		}
	}
	return false
}
