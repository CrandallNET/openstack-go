package cli

import (
	"fmt"
	"io"
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
	Name    string
	Hex     string
	Sample  string
	Matches []string
}

const (
	prettyColorOSAlpine   = "#0D597F"
	prettyColorOSArch     = "#1793D1"
	prettyColorOSCentOS   = "#262577"
	prettyColorOSDebian   = "#CE0056"
	prettyColorOSFedora   = "#3C6EB4"
	prettyColorOSOpenSUSE = "#73BA25"
	prettyColorOSRedHat   = "#EE0000"
	prettyColorOSRocky    = "#10B981"
	prettyColorOSSUSE     = "#30BA78"
	prettyColorOSUbuntu   = "#E95420"
	prettyColorOSWindows  = "#0078D7"
)

var prettyOSImageColorDefinitions = []prettyOSImageColorDefinition{
	{
		Name:    "Ubuntu",
		Hex:     prettyColorOSUbuntu,
		Sample:  "Ubuntu 24.04",
		Matches: []string{"ubuntu"},
	},
	{
		Name:    "Debian",
		Hex:     prettyColorOSDebian,
		Sample:  "Debian 12",
		Matches: []string{"debian"},
	},
	{
		Name:    "Rocky Linux",
		Hex:     prettyColorOSRocky,
		Sample:  "Rocky Linux 9",
		Matches: []string{"rocky", "rockylinux"},
	},
	{
		Name:    "Red Hat Enterprise Linux",
		Hex:     prettyColorOSRedHat,
		Sample:  "Red Hat Enterprise Linux 9",
		Matches: []string{"red hat", "redhat", "rhel"},
	},
	{
		Name:    "Fedora",
		Hex:     prettyColorOSFedora,
		Sample:  "Fedora 41",
		Matches: []string{"fedora"},
	},
	{
		Name:    "CentOS",
		Hex:     prettyColorOSCentOS,
		Sample:  "CentOS Stream 9",
		Matches: []string{"centos"},
	},
	{
		Name:    "openSUSE",
		Hex:     prettyColorOSOpenSUSE,
		Sample:  "openSUSE Leap 15",
		Matches: []string{"opensuse", "open suse", "tumbleweed", "leap"},
	},
	{
		Name:    "SUSE",
		Hex:     prettyColorOSSUSE,
		Sample:  "SUSE Linux Enterprise Server 15",
		Matches: []string{"suse linux enterprise", "sles", "suse"},
	},
	{
		Name:    "Alpine Linux",
		Hex:     prettyColorOSAlpine,
		Sample:  "Alpine Linux 3.20",
		Matches: []string{"alpine"},
	},
	{
		Name:    "Arch Linux",
		Hex:     prettyColorOSArch,
		Sample:  "Arch Linux",
		Matches: []string{"arch linux", "archlinux", "arch"},
	},
	{
		Name:    "Windows",
		Hex:     prettyColorOSWindows,
		Sample:  "Windows Server 2022",
		Matches: []string{"windows"},
	},
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
		rows = append(rows, outputRow{
			"OS":            prettyImageValue(definition.Name),
			"Hex":           definition.Hex,
			"Sample Image":  prettyImageValue(definition.Sample),
			"Matched Terms": strings.Join(definition.Matches, ", "),
			"Color Preview": prettyImageValue(fmt.Sprintf("%s %s", definition.Name, definition.Hex)),
		})
	}
	return renderListOutput(stdout, &renderOpts, []string{"OS", "Hex", "Sample Image", "Color Preview", "Matched Terms"}, rows)
}
