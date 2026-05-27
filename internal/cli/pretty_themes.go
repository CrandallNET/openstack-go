package cli

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
)

//go:embed colors/themes.json
var prettyThemesJSON []byte

var embeddedThemes = mustLoadEmbeddedThemes(prettyThemesJSON)

func mustLoadEmbeddedThemes(data []byte) themeStore {
	themes, err := parseEmbeddedThemes(data)
	if err != nil {
		panic(fmt.Sprintf("load embedded pretty themes: %v", err))
	}
	return themes
}

func parseEmbeddedThemes(data []byte) (themeStore, error) {
	var themes themeStore
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&themes); err != nil {
		return themes, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return themes, fmt.Errorf("unexpected content after JSON document")
	}
	if themes.SchemaVersion != 1 {
		return themes, fmt.Errorf("unsupported schema_version %d", themes.SchemaVersion)
	}
	if _, ok := themes.Themes["default"]; !ok {
		return themes, fmt.Errorf("themes must include at least \"default\"")
	}
	return themes, nil
}

type themeStore struct {
	SchemaVersion int                    `json:"schema_version"`
	Themes        map[string]*themeColors `json:"themes"`
}

type themeColors struct {
	Label        string `json:"label"`
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	IPAddress    string `json:"ip_address"`
	Timestamp    string `json:"timestamp"`
	Number       string `json:"number"`
	BooleanTrue  string `json:"boolean_true"`
	BooleanFalse string `json:"boolean_false"`
	Warning      string `json:"warning"`
	Error        string `json:"error"`
	Device       string `json:"device"`
	Flavor       string `json:"flavor"`
	Image        string `json:"image"`
	Volume       string `json:"volume"`
	NA           string `json:"na"`
	CellText     string `json:"cell_text"`
	Border       string `json:"border"`
	Header       string `json:"header"`
	EmptyState   string `json:"empty_state"`
}

// DefaultTheme returns the "default" theme loaded from colors/themes.json.
func DefaultTheme() *Theme {
	return ThemeByName("default")
}

// ThemeByName returns the named theme, falling back to "default" when unknown.
func ThemeByName(name string) *Theme {
	selection := "default"
	if candidate := embeddedThemes.Themes[name]; candidate != nil {
		selection = name
	}
	selected := embeddedThemes.Themes[selection]
	return &Theme{
		LabelColor:        selected.Label,
		UUIDColor:         selected.UUID,
		NameColor:         selected.Name,
		IPAddressColor:    selected.IPAddress,
		TimestampColor:    selected.Timestamp,
		NumberColor:       selected.Number,
		BooleanTrueColor:  selected.BooleanTrue,
		BooleanFalseColor: selected.BooleanFalse,
		WarningColor:      selected.Warning,
		ErrorColor:        selected.Error,
		DeviceColor:       selected.Device,
		FlavorColor:       selected.Flavor,
		ImageColor:        selected.Image,
		VolumeColor:       selected.Volume,
		NAColour:          selected.NA,
		CellTextColor:     selected.CellText,
		BorderColor:       selected.Border,
		HeaderColor:       selected.Header,
		EmptyStateColor:   selected.EmptyState,
	}
}

// Theme holds a set of color definitions for the pretty renderer.
// Each field maps a semantic role to an 8-bit terminal color code.
type Theme struct {
	// Semantic role colors for data cells.
	LabelColor      string // Field labels (key names before the colon)
	UUIDColor       string // UUIDs and ID-like hex fragments
	NameColor       string // Resource names
	IPAddressColor  string // IP addresses and hostnames
	TimestampColor  string // Dates and times
	NumberColor     string // Numeric flavor specs
	BooleanTrueColor  string // Boolean True, healthy/active status
	BooleanFalseColor string // Boolean False
	WarningColor    string // Transitional statuses (BUILD, MIGRATING, CREATING, etc.)
	ErrorColor      string // Error/dead statuses (ERROR, SHELVED, SHUTOFF, FAILED, etc.)
	DeviceColor     string // Block device paths
	FlavorColor     string // Flavor names
	ImageColor      string // Generic image values (OS-specific use brand palette)
	VolumeColor     string // Volume names
	NAColour        string // Not-available placeholder ("N/A")

	// Table structural colors (lipgloss + bubble-table).
	CellTextColor   string // Bubble-table cell text
	BorderColor     string // Bubble-table border foreground
	HeaderColor     string // Bubble-table header foreground
	EmptyStateColor string // "No rows" empty state message
}

// BuildStyles returns a set of lipgloss styles derived from the theme.
// Each style is pre-configured with the corresponding Foreground color.
func (t *Theme) BuildStyles() prettyThemeStyles {
	return prettyThemeStyles{
		LabelStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.LabelColor)).Bold(true),
		UUIDStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color(t.UUIDColor)),
		NameStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color(t.NameColor)),
		IPAddressStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.IPAddressColor)),
		TimestampStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.TimestampColor)),
		NumberStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.NumberColor)),
		BooleanTrueStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.BooleanTrueColor)),
		BooleanFalseStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color(t.BooleanFalseColor)),
		WarningStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.WarningColor)),
		ErrorStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.ErrorColor)),
		DeviceStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.DeviceColor)),
		FlavorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.FlavorColor)),
		ImageStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.ImageColor)),
		VolumeStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.VolumeColor)),
		NAStyle:          lipgloss.NewStyle().Foreground(lipgloss.Color(t.NAColour)),
	}
}

// BuildBubbleTableStyle returns a bubble-table-compatible base style
// using the theme's structural colors.
func (t *Theme) BuildBubbleTableStyle() bubbleStyle {
	return bubbleStyle{
		cellText:   t.CellTextColor,
		border:     t.BorderColor,
		header:     t.HeaderColor,
		emptyState: t.EmptyStateColor,
	}
}

// prettyThemeStyles holds pre-built lipgloss styles for semantic roles.
type prettyThemeStyles struct {
	LabelStyle       lipgloss.Style
	UUIDStyle        lipgloss.Style
	NameStyle        lipgloss.Style
	IPAddressStyle   lipgloss.Style
	TimestampStyle   lipgloss.Style
	NumberStyle      lipgloss.Style
	BooleanTrueStyle   lipgloss.Style
	BooleanFalseStyle  lipgloss.Style
	WarningStyle     lipgloss.Style
	ErrorStyle       lipgloss.Style
	DeviceStyle      lipgloss.Style
	FlavorStyle      lipgloss.Style
	ImageStyle       lipgloss.Style
	VolumeStyle      lipgloss.Style
	NAStyle          lipgloss.Style
}

// bubbleStyle holds the structural colors used by bubble-table rendering.
type bubbleStyle struct {
	cellText   string
	border     string
	header     string
	emptyState string
}
