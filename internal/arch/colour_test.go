package arch

// A colour is named in one place, and the screens ask for the job.
//
// internal/ui/theme is where the hexes live: tokens.go says so for paper and
// text, badge.go for the pills. A hex written at a call site is a colour no
// theme can reach — the window went from frauddi to dracula around a Save
// button that stayed the same green, and the only way to find that green was
// to grep nine files under internal/ui.
//
// Test files are exempt: a golden that asserts what was drawn names the
// colour it expects, and that is the point of it.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// theOnlyPlaceForHexes is the package allowed to spell one.
const theOnlyPlaceForHexes = "internal/ui/theme/"

// hexColour is a colour written as a hex literal, in either length.
var hexColour = regexp.MustCompile(`"#(?:[0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})"`)

// TestColoursLiveInTheTheme.
func TestColoursLiveInTheTheme(t *testing.T) {
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, theOnlyPlaceForHexes) {
			continue
		}

		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		for _, found := range hexColour.FindAllString(string(body), -1) {
			t.Errorf("%s spells the colour %s — name it in internal/ui/theme and ask for the job here",
				path, found)
		}
	}
}
