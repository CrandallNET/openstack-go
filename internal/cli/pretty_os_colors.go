package cli

import (
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
	Name    string
	Hex     string
	Sample  string
	Matches []string
}

const (
	prettyOSImageColorContrastBackground = "#282C34"
	prettyOSImageColorMinimumContrast    = 2.5
)

const (
	prettyColorOSAlmaLinux    = "#0069DA"
	prettyColorOSAlpine       = "#4BB4D8"
	prettyColorOSArch         = "#4DBBEB"
	prettyColorOSCentOS       = "#A14F8C"
	prettyColorOSCentOSCore   = prettyColorOSCentOS
	prettyColorOSCentOSStream = prettyColorOSCentOS
	prettyColorOSCirrOS       = "#FF6A7D"
	prettyColorOSCoreOS       = "#7EB7E6"
	prettyColorOSDebian       = "#CE0056"
	prettyColorOSDeepin       = "#5FB0FF"
	prettyColorOSElementary   = "#4CA7E4"
	prettyColorOSEndeavourOS  = "#A0A0FF"
	prettyColorOSFedora       = "#51A2DA"
	prettyColorOSFlatcar      = "#52B8D8"
	prettyColorOSFreeBSD      = "#FF5A5F"
	prettyColorOSGentoo       = "#DDDAEC"
	prettyColorOSKali         = "#84C8E8"
	prettyColorOSLinuxMint    = "#86BE43"
	prettyColorOSManjaro      = "#35BFA4"
	prettyColorOSNetBSD       = "#FF7A1A"
	prettyColorOSNixOS        = "#7EB7E6"
	prettyColorOSOpenBSD      = "#F2CA30"
	prettyColorOSOpenSUSE     = "#73BA25"
	prettyColorOSOracleLinux  = "#C74634"
	prettyColorOSPopOS        = "#48B9C7"
	prettyColorOSQubes        = "#6EA8FF"
	prettyColorOSRedHat       = "#EE0000"
	prettyColorOSRocky        = "#10B981"
	prettyColorOSSolus        = "#6AA8F7"
	prettyColorOSSUSE         = "#30BA78"
	prettyColorOSTails        = "#C7A4F4"
	prettyColorOSTalos        = "#E8312C"
	prettyColorOSUbuntu       = "#FF7A45"
	prettyColorOSVoid         = "#6FBF8F"
	prettyColorOSVyOS         = "#FFBF12"
	prettyColorOSWindows      = "#54B8FF"
	prettyColorOSZorin        = "#15A6F0"
)

var prettyOSImageColorDefinitions = []prettyOSImageColorDefinition{
	{
		Name:    "AlmaLinux",
		Hex:     prettyColorOSAlmaLinux,
		Sample:  "AlmaLinux 9",
		Matches: []string{"almalinux", "alma linux", "alma"},
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
		Name:    "CentOS Stream",
		Hex:     prettyColorOSCentOSStream,
		Sample:  "CentOS Stream 10",
		Matches: []string{"centos stream", "centosstream"},
	},
	{
		Name:    "CentOS Core",
		Hex:     prettyColorOSCentOSCore,
		Sample:  "CentOS Core",
		Matches: []string{"centos core", "centoscore"},
	},
	{
		Name:    "CentOS",
		Hex:     prettyColorOSCentOS,
		Sample:  "CentOS 7",
		Matches: []string{"centos"},
	},
	{
		Name:    "CirrOS",
		Hex:     prettyColorOSCirrOS,
		Sample:  "CirrOS 0.6.2",
		Matches: []string{"cirros"},
	},
	{
		Name:    "CoreOS",
		Hex:     prettyColorOSCoreOS,
		Sample:  "Fedora CoreOS",
		Matches: []string{"fedora coreos", "coreos", "core os"},
	},
	{
		Name:    "Debian",
		Hex:     prettyColorOSDebian,
		Sample:  "Debian 12",
		Matches: []string{"debian"},
	},
	{
		Name:    "deepin",
		Hex:     prettyColorOSDeepin,
		Sample:  "deepin 23",
		Matches: []string{"deepin"},
	},
	{
		Name:    "elementary OS",
		Hex:     prettyColorOSElementary,
		Sample:  "elementary OS 8",
		Matches: []string{"elementary os", "elementaryos", "elementary"},
	},
	{
		Name:    "EndeavourOS",
		Hex:     prettyColorOSEndeavourOS,
		Sample:  "EndeavourOS",
		Matches: []string{"endeavouros", "endeavour os"},
	},
	{
		Name:    "Fedora",
		Hex:     prettyColorOSFedora,
		Sample:  "Fedora 41",
		Matches: []string{"fedora"},
	},
	{
		Name:    "Flatcar Container Linux",
		Hex:     prettyColorOSFlatcar,
		Sample:  "Flatcar Container Linux",
		Matches: []string{"flatcar container linux", "flatcar linux", "flatcar"},
	},
	{
		Name:    "FreeBSD",
		Hex:     prettyColorOSFreeBSD,
		Sample:  "FreeBSD 14",
		Matches: []string{"freebsd", "free bsd"},
	},
	{
		Name:    "Gentoo",
		Hex:     prettyColorOSGentoo,
		Sample:  "Gentoo Linux",
		Matches: []string{"gentoo"},
	},
	{
		Name:    "Kali Linux",
		Hex:     prettyColorOSKali,
		Sample:  "Kali Linux",
		Matches: []string{"kali linux", "kalilinux", "kali"},
	},
	{
		Name:    "Linux Mint",
		Hex:     prettyColorOSLinuxMint,
		Sample:  "Linux Mint 22",
		Matches: []string{"linux mint", "linuxmint", "mint"},
	},
	{
		Name:    "Manjaro",
		Hex:     prettyColorOSManjaro,
		Sample:  "Manjaro Linux",
		Matches: []string{"manjaro"},
	},
	{
		Name:    "NetBSD",
		Hex:     prettyColorOSNetBSD,
		Sample:  "NetBSD 10",
		Matches: []string{"netbsd", "net bsd"},
	},
	{
		Name:    "NixOS",
		Hex:     prettyColorOSNixOS,
		Sample:  "NixOS 25.05",
		Matches: []string{"nixos", "nix os"},
	},
	{
		Name:    "OpenBSD",
		Hex:     prettyColorOSOpenBSD,
		Sample:  "OpenBSD 7.7",
		Matches: []string{"openbsd", "open bsd"},
	},
	{
		Name:    "openSUSE",
		Hex:     prettyColorOSOpenSUSE,
		Sample:  "openSUSE Leap 15",
		Matches: []string{"opensuse", "open suse", "tumbleweed", "leap"},
	},
	{
		Name:    "Oracle Linux",
		Hex:     prettyColorOSOracleLinux,
		Sample:  "Oracle Linux 9",
		Matches: []string{"oracle linux", "oraclelinux", "ol"},
	},
	{
		Name:    "Pop!_OS",
		Hex:     prettyColorOSPopOS,
		Sample:  "Pop!_OS 22.04",
		Matches: []string{"pop os", "popos"},
	},
	{
		Name:    "Qubes OS",
		Hex:     prettyColorOSQubes,
		Sample:  "Qubes OS 4.2",
		Matches: []string{"qubes os", "qubesos", "qubes"},
	},
	{
		Name:    "Red Hat Enterprise Linux",
		Hex:     prettyColorOSRedHat,
		Sample:  "Red Hat Enterprise Linux 9",
		Matches: []string{"red hat", "redhat", "rhel"},
	},
	{
		Name:    "Rocky Linux",
		Hex:     prettyColorOSRocky,
		Sample:  "Rocky Linux 9",
		Matches: []string{"rocky", "rockylinux"},
	},
	{
		Name:    "Solus",
		Hex:     prettyColorOSSolus,
		Sample:  "Solus 4.5",
		Matches: []string{"solus"},
	},
	{
		Name:    "SUSE",
		Hex:     prettyColorOSSUSE,
		Sample:  "SUSE Linux Enterprise Server 15",
		Matches: []string{"suse linux enterprise", "sles", "suse"},
	},
	{
		Name:    "Tails",
		Hex:     prettyColorOSTails,
		Sample:  "Tails 6.0",
		Matches: []string{"tails"},
	},
	{
		Name:    "Talos Linux",
		Hex:     prettyColorOSTalos,
		Sample:  "Talos Linux",
		Matches: []string{"talos linux", "talos"},
	},
	{
		Name:    "Ubuntu",
		Hex:     prettyColorOSUbuntu,
		Sample:  "Ubuntu 24.04",
		Matches: []string{"ubuntu"},
	},
	{
		Name:    "Void Linux",
		Hex:     prettyColorOSVoid,
		Sample:  "Void Linux",
		Matches: []string{"void linux", "voidlinux", "void"},
	},
	{
		Name:    "VyOS",
		Hex:     prettyColorOSVyOS,
		Sample:  "VyOS 1.5",
		Matches: []string{"vyos"},
	},
	{
		Name:    "Windows",
		Hex:     prettyColorOSWindows,
		Sample:  "Windows Server 2022",
		Matches: []string{"windows"},
	},
	{
		Name:    "Zorin OS",
		Hex:     prettyColorOSZorin,
		Sample:  "Zorin OS 17",
		Matches: []string{"zorin os", "zorinos", "zorin"},
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
