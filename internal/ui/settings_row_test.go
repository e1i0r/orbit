package ui

import "testing"

// settingRow is where a setting is in the screen's list, by name: the list
// is in groups, and a test that counted from the top would be counting the
// grouping rather than the setting.
func settingRow(t *testing.T, m Model, key string) int {
	t.Helper()

	for i, r := range m.settingRowsList() {
		if r.Key == key {
			return i
		}
	}

	t.Fatalf("no row for %q", key)

	return -1
}
