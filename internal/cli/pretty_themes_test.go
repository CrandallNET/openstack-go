package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if got := embeddedThemes.Themes["default"].ProgressBar; len(got) < 1 || len(got) > 2 {
		t.Fatalf("expected default progress_bar length to be 1 or 2, got %#v", got)
	}
}

func TestLoadThemeFromFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "theme.json")
	data := `{
  "schema_version": 1,
  "themes": {
    "default": {
      "label": "1",
      "uuid": "2",
      "name": "3",
      "ip_address": "4",
      "timestamp": "5",
      "number": "6",
      "boolean_true": "2",
      "boolean_false": "1",
      "warning": "3",
      "error": "1",
      "device": "4",
      "flavor": "5",
      "image": "6",
      "volume": "7",
      "na": "8",
      "cell_text": "7",
      "border": "8",
      "header": "7",
      "empty_state": "8",
      "progress_bar": ["8", "7"]
    },
    "user": {
      "label": "7",
      "uuid": "6",
      "name": "5",
      "ip_address": "4",
      "timestamp": "3",
      "number": "2",
      "boolean_true": "7",
      "boolean_false": "1",
      "warning": "3",
      "error": "1",
      "device": "4",
      "flavor": "5",
      "image": "6",
      "volume": "7",
      "na": "8",
      "cell_text": "7",
      "border": "8",
      "header": "7",
      "empty_state": "8",
      "progress_bar": ["#101010", "#20ff20"]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write theme.json: %v", err)
	}
	theme, err := loadThemeFromFile(path, "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.LabelColor != "7" {
		t.Fatalf("expected label color 7, got %q", theme.LabelColor)
	}
	if len(theme.ProgressBarColors) != 2 || theme.ProgressBarColors[0] != "#101010" || theme.ProgressBarColors[1] != "#20ff20" {
		t.Fatalf("unexpected progress colors: %#v", theme.ProgressBarColors)
	}
}

func TestLoadThemeFromFileMissingTheme(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "theme.json")
	data := `{"schema_version":1,"themes":{"default":{"label":"7","uuid":"7","name":"7","ip_address":"7","timestamp":"7","number":"7","boolean_true":"7","boolean_false":"7","warning":"7","error":"7","device":"7","flavor":"7","image":"7","volume":"7","na":"7","cell_text":"7","border":"7","header":"7","empty_state":"7","progress_bar":["7"]}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write theme.json: %v", err)
	}
	_, err := loadThemeFromFile(path, "user")
	if err == nil || !strings.Contains(err.Error(), `theme "user" not found`) {
		t.Fatalf("expected missing user theme error, got %v", err)
	}
}
