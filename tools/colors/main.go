package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

func main() {
	if err := run(os.Stdout, os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stdout io.Writer, args []string, getenv func(string) string) error {
	fs := flag.NewFlagSet("colors", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	ansiFlag := fs.Bool("ansi", false, "show the 16 ANSI named colors")
	asciiFlag := fs.Bool("ascii", false, "show the 256-color palette table")
	shortFlag := fs.Bool("short", false, "alias for --ascii")
	hexFlag := fs.Bool("hex", false, "show the 256-color palette with hex values")
	if err := fs.Parse(args); err != nil {
		return err
	}

	showANSI := *ansiFlag || envBool(getenv("COLORS_ANSI"))
	showASCII := *asciiFlag || *shortFlag || envBool(getenv("COLORS_ASCII")) || envBool(getenv("COLORS_SHORT"))
	showHex := *hexFlag || envBool(getenv("COLORS_HEX"))

	// Default mode keeps the prior behavior: show ANSI names and the 256 table.
	if !showANSI && !showASCII && !showHex {
		showANSI = true
		showASCII = true
	}

	first := true
	if showANSI {
		renderANSI16(stdout)
		first = false
	}
	if showASCII {
		if !first {
			fmt.Fprintln(stdout)
		}
		render256Table(stdout)
		first = false
	}
	if showHex {
		if !first {
			fmt.Fprintln(stdout)
		}
		render256Hex(stdout)
	}
	return nil
}

func envBool(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func renderANSI16(stdout io.Writer) {
	names := []string{
		"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
		"bright-black", "bright-red", "bright-green", "bright-yellow",
		"bright-blue", "bright-magenta", "bright-cyan", "bright-white",
	}
	fmt.Fprintln(stdout, "ANSI 16")
	for index, name := range names {
		swatch := lipgloss.NewStyle().Background(lipgloss.Color(strconv.Itoa(index))).Render("  ")
		fmt.Fprintf(stdout, "%2d %s %s\n", index, swatch, name)
	}
}

func render256Table(stdout io.Writer) {
	fmt.Fprintln(stdout, "ANSI 256 (16 columns)")
	for i := 0; i < 256; i++ {
		swatch := lipgloss.NewStyle().Background(lipgloss.Color(strconv.Itoa(i))).Render("  ")
		fmt.Fprintf(stdout, "%3d %s", i, swatch)
		if (i+1)%16 == 0 {
			fmt.Fprintln(stdout)
		} else {
			fmt.Fprint(stdout, "  ")
		}
	}
}

func render256Hex(stdout io.Writer) {
	fmt.Fprintln(stdout, "ANSI 256 + HEX")
	for i := 0; i < 256; i++ {
		hex := xterm256Hex(i)
		swatch := lipgloss.NewStyle().Background(lipgloss.Color(strconv.Itoa(i))).Render("  ")
		fmt.Fprintf(stdout, "%3d %s %s\n", i, swatch, hex)
	}
}

func xterm256Hex(index int) string {
	if index < 0 {
		index = 0
	}
	if index > 255 {
		index = 255
	}

	base := [16][3]int{
		{0x00, 0x00, 0x00}, {0x80, 0x00, 0x00}, {0x00, 0x80, 0x00}, {0x80, 0x80, 0x00},
		{0x00, 0x00, 0x80}, {0x80, 0x00, 0x80}, {0x00, 0x80, 0x80}, {0xc0, 0xc0, 0xc0},
		{0x80, 0x80, 0x80}, {0xff, 0x00, 0x00}, {0x00, 0xff, 0x00}, {0xff, 0xff, 0x00},
		{0x00, 0x00, 0xff}, {0xff, 0x00, 0xff}, {0x00, 0xff, 0xff}, {0xff, 0xff, 0xff},
	}
	if index < 16 {
		c := base[index]
		return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
	}
	if index >= 232 {
		v := 8 + (index-232)*10
		return fmt.Sprintf("#%02x%02x%02x", v, v, v)
	}

	cube := index - 16
	r := cube / 36
	g := (cube / 6) % 6
	b := cube % 6
	levels := [6]int{0, 95, 135, 175, 215, 255}
	return fmt.Sprintf("#%02x%02x%02x", levels[r], levels[g], levels[b])
}
