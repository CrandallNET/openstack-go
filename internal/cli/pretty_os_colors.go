package cli

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
)

type prettyImageValue string

func (value prettyImageValue) PrettyString() string {
	return string(value)
}

func (value prettyImageValue) PrettySemanticRole() string {
	return "image"
}

type prettyOSImageColorDefinition struct {
	Name    string   `json:"name"`
	Hex     string   `json:"hex"`
	Sample  string   `json:"sample"`
	Matches []string `json:"matches"`
	Sources []string `json:"sources"`
}

type prettyOSImageColorPalette struct {
	SchemaVersion      int                            `json:"schema_version"`
	ContrastBackground string                         `json:"contrast_background"`
	MinimumContrast    float64                        `json:"minimum_contrast"`
	Definitions        []prettyOSImageColorDefinition `json:"definitions"`
}

//go:embed colors/os_colors.json
var prettyOSImageColorsJSON []byte

var prettyOSImageColorPaletteData = mustLoadPrettyOSImageColorPalette(prettyOSImageColorsJSON)

var (
	prettyOSImageColorContrastBackground = prettyOSImageColorPaletteData.ContrastBackground
	prettyOSImageColorMinimumContrast    = prettyOSImageColorPaletteData.MinimumContrast
	prettyOSImageColorDefinitions        = prettyOSImageColorPaletteData.Definitions
)

func mustLoadPrettyOSImageColorPalette(data []byte) prettyOSImageColorPalette {
	palette, err := parsePrettyOSImageColorPalette(data)
	if err != nil {
		panic(fmt.Sprintf("load embedded pretty OS color palette: %v", err))
	}
	return palette
}

func parsePrettyOSImageColorPalette(data []byte) (prettyOSImageColorPalette, error) {
	var palette prettyOSImageColorPalette
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&palette); err != nil {
		return palette, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return palette, fmt.Errorf("unexpected content after JSON document")
	}
	if palette.SchemaVersion != 1 {
		return palette, fmt.Errorf("unsupported schema_version %d", palette.SchemaVersion)
	}
	palette.ContrastBackground = prettyNormalizeHexColor(palette.ContrastBackground)
	if _, _, _, ok := prettyHexRGB(palette.ContrastBackground); !ok {
		return palette, fmt.Errorf("invalid contrast_background %q", palette.ContrastBackground)
	}
	if palette.MinimumContrast <= 0 {
		return palette, fmt.Errorf("minimum_contrast must be positive")
	}
	if len(palette.Definitions) == 0 {
		return palette, fmt.Errorf("definitions must not be empty")
	}
	seenNames := make(map[string]struct{}, len(palette.Definitions))
	for index := range palette.Definitions {
		definition := &palette.Definitions[index]
		definition.Name = strings.TrimSpace(definition.Name)
		definition.Hex = prettyNormalizeHexColor(definition.Hex)
		definition.Sample = strings.TrimSpace(definition.Sample)
		if definition.Name == "" {
			return palette, fmt.Errorf("definitions[%d].name must not be empty", index)
		}
		nameKey := strings.ToLower(definition.Name)
		if _, ok := seenNames[nameKey]; ok {
			return palette, fmt.Errorf("duplicate OS color definition name %q", definition.Name)
		}
		seenNames[nameKey] = struct{}{}
		if _, _, _, ok := prettyHexRGB(definition.Hex); !ok {
			return palette, fmt.Errorf("invalid hex color %q for %s", definition.Hex, definition.Name)
		}
		if definition.Sample == "" {
			return palette, fmt.Errorf("definition %s sample must not be empty", definition.Name)
		}
		if len(definition.Matches) == 0 {
			return palette, fmt.Errorf("definition %s matches must not be empty", definition.Name)
		}
		for matchIndex, match := range definition.Matches {
			match = strings.TrimSpace(match)
			if match == "" {
				return palette, fmt.Errorf("definition %s matches[%d] must not be empty", definition.Name, matchIndex)
			}
			definition.Matches[matchIndex] = match
		}
		for sourceIndex, source := range definition.Sources {
			definition.Sources[sourceIndex] = strings.TrimSpace(source)
		}
	}
	return palette, nil
}

func prettyNormalizeHexColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return color
	}
	color = strings.TrimPrefix(color, "#")
	return "#" + strings.ToUpper(color)
}

func prettyOSImageColorByName(name string) (string, bool) {
	for _, definition := range prettyOSImageColorDefinitions {
		if strings.EqualFold(definition.Name, name) {
			return definition.Hex, true
		}
	}
	return "", false
}

func prettyOSImageColorForText(text string) (string, bool) {
	normalized := prettyNormalizeOSImageText(text)
	if normalized == "" {
		return "", false
	}
	for _, definition := range prettyOSImageColorDefinitions {
		for _, match := range definition.Matches {
			if prettyOSImageTextMatches(normalized, match) {
				return definition.Hex, true
			}
		}
	}
	return "", false
}

func prettyOSImageStyleForText(text string) (lipgloss.Style, bool) {
	color, ok := prettyOSImageColorForText(text)
	if !ok {
		return lipgloss.Style{}, false
	}
	return prettyStyleForColor(color), true
}

func prettyStyleForColor(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func prettyOSImageColorContrastRatio(color string) (float64, bool) {
	return prettyContrastRatio(color, prettyOSImageColorContrastBackground)
}

func prettyContrastRatio(foreground string, background string) (float64, bool) {
	foregroundLuminance, ok := prettyRelativeLuminance(foreground)
	if !ok {
		return 0, false
	}
	backgroundLuminance, ok := prettyRelativeLuminance(background)
	if !ok {
		return 0, false
	}
	light := math.Max(foregroundLuminance, backgroundLuminance)
	dark := math.Min(foregroundLuminance, backgroundLuminance)
	return (light + 0.05) / (dark + 0.05), true
}

func prettyRelativeLuminance(color string) (float64, bool) {
	red, green, blue, ok := prettyHexRGB(color)
	if !ok {
		return 0, false
	}
	return 0.2126*prettyLinearRGB(red) + 0.7152*prettyLinearRGB(green) + 0.0722*prettyLinearRGB(blue), true
}

func prettyHexRGB(color string) (float64, float64, float64, bool) {
	color = strings.TrimPrefix(strings.TrimSpace(color), "#")
	if len(color) != 6 {
		return 0, 0, 0, false
	}
	red, err := strconv.ParseUint(color[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	green, err := strconv.ParseUint(color[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	blue, err := strconv.ParseUint(color[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return float64(red) / 255, float64(green) / 255, float64(blue) / 255, true
}

func prettyLinearRGB(value float64) float64 {
	if value <= 0.03928 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func prettyNormalizeOSImageText(text string) string {
	var builder strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func prettyOSImageTextMatches(normalized string, match string) bool {
	needle := prettyNormalizeOSImageText(match)
	if needle == "" {
		return false
	}
	if len(needle) <= 4 && !strings.Contains(needle, " ") {
		for _, token := range strings.Fields(normalized) {
			if token == needle {
				return true
			}
			if strings.HasPrefix(token, needle) && len(token) > len(needle) && unicode.IsDigit(rune(token[len(needle)])) {
				return true
			}
		}
		return false
	}
	return strings.Contains(normalized, needle)
}

func RenderPrettyOSColorTest(stdout io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	renderOpts := *opts
	renderOpts.Format = "pretty"

	rows := make([]outputRow, 0, len(prettyOSImageColorDefinitions))
	for _, definition := range prettyOSImageColorDefinitions {
		contrast := "unknown"
		if ratio, ok := prettyOSImageColorContrastRatio(definition.Hex); ok {
			contrast = fmt.Sprintf("%.2f:1", ratio)
		}
		rows = append(rows, outputRow{
			"OS":            prettyImageValue(definition.Name),
			"Hex":           definition.Hex,
			"Contrast":      contrast,
			"Sample Image":  prettyImageValue(definition.Sample),
			"Matched Terms": strings.Join(definition.Matches, ", "),
			"Color Preview": prettyImageValue(fmt.Sprintf("%s %s", definition.Name, definition.Hex)),
		})
	}
	return renderListOutput(stdout, &renderOpts, []string{"OS", "Hex", "Contrast", "Sample Image", "Color Preview", "Matched Terms"}, rows)
}
