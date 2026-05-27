package cli

import "testing"

func TestThemeByNameFallsBackToDefault(t *testing.T) {
	defaultTheme := ThemeByName("default")
	unknownTheme := ThemeByName("unknown-theme")
	if unknownTheme.LabelColor != defaultTheme.LabelColor {
		t.Fatalf("expected unknown theme to fall back to default")
	}
}

func TestEmbeddedThemesIncludeDefaultAndPacman(t *testing.T) {
	if embeddedThemes.Themes["default"] == nil {
		t.Fatalf("expected default theme to be present")
	}
	if embeddedThemes.Themes["pacman"] == nil {
		t.Fatalf("expected pacman theme to be present")
	}
}

