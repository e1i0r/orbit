package theme

// CurrentPalette's fallback for a theme name nothing answers to.

import "testing"

func TestCurrentPaletteFallsBackOnAnUnknownTheme(t *testing.T) {
	old := currentThemeName

	t.Cleanup(func() { currentThemeName = old })

	currentThemeName = "not-a-real-theme"

	if got := CurrentPalette(); got != themePalettes["monokai"] {
		t.Errorf("CurrentPalette with an unknown theme name = %+v, want the monokai fallback", got)
	}

	currentThemeName = "nord"

	if got := CurrentPalette(); got != themePalettes["nord"] {
		t.Errorf("CurrentPalette(nord) = %+v, want the nord palette", got)
	}
}
