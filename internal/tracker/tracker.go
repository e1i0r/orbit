package tracker

// IsTrackerURL reports whether rawURL matches any known tracker provider.
func IsTrackerURL(rawURL string) bool {
	for _, p := range Providers() {
		if p.Match(rawURL) {
			return true
		}
	}

	return false
}
