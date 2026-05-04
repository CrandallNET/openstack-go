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
	prettyColorOSAlmaLinux    = "#0069DA"
	prettyColorOSAlpine       = "#0D597F"
	prettyColorOSArch         = "#1793D1"
	prettyColorOSCentOS       = "#262577"
	prettyColorOSCentOSStream = "#A14F8C"
	prettyColorOSCirrOS       = "#ED1844"
	prettyColorOSDebian       = "#CE0056"
	prettyColorOSDeepin       = "#007CFF"
	prettyColorOSElementary   = "#64BAFF"
	prettyColorOSEndeavourOS  = "#7F7FFF"
	prettyColorOSFedora       = "#3C6EB4"
	prettyColorOSFreeBSD      = "#E31E26"
	prettyColorOSGentoo       = "#54487A"
	prettyColorOSKali         = "#557C94"
	prettyColorOSLinuxMint    = "#86BE43"
	prettyColorOSManjaro      = "#35BFA4"
	prettyColorOSNetBSD       = "#F26711"
	prettyColorOSNixOS        = "#5277C3"
	prettyColorOSOpenBSD      = "#F2CA30"
	prettyColorOSOpenSUSE     = "#73BA25"
	prettyColorOSOracleLinux  = "#E32124"
	prettyColorOSPopOS        = "#48B9C7"
	prettyColorOSQubes        = "#3874D8"
	prettyColorOSRedHat       = "#EE0000"
	prettyColorOSRocky        = "#10B981"
	prettyColorOSSolus        = "#5294E2"
	prettyColorOSSUSE         = "#30BA78"
	prettyColorOSTails        = "#56347C"
	prettyColorOSUbuntu       = "#E95420"
	prettyColorOSVoid         = "#478061"
	prettyColorOSVyOS         = "#FFBF12"
	prettyColorOSWindows      = "#0078D7"
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
