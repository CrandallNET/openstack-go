package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		if env := strings.TrimSpace(os.Getenv("PROGRESS_COLORS")); env != "" {
			args = strings.Fields(env)
		}
	}
	if err := run(os.Stdout, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stdout *os.File, args []string) error {
	model := progress.New(progress.WithWidth(72), progress.WithFillCharacters('█', ' '))

	switch len(args) {
	case 0:
		model = progress.New(progress.WithWidth(72), progress.WithFillCharacters('█', ' '))
		model.FullColor = lipgloss.Color("7")
		model.EmptyColor = lipgloss.NoColor{}
	case 1:
		full, err := parsePaletteColor(args[0])
		if err != nil {
			return err
		}
		model = progress.New(progress.WithWidth(72), progress.WithFillCharacters('█', ' '))
		model.FullColor = full
		model.EmptyColor = lipgloss.NoColor{}
	case 2:
		empty, err := parsePaletteColor(args[0])
		if err != nil {
			return err
		}
		full, err := parsePaletteColor(args[1])
		if err != nil {
			return err
		}
		model = progress.New(progress.WithWidth(72), progress.WithFillCharacters('█', '█'))
		model.FullColor = full
		model.EmptyColor = empty
	default:
		return fmt.Errorf("usage: go run ./tools/progress [FULL] or [EMPTY FULL] (values 0-255)")
	}

	for i := 0; i <= 100; i += 5 {
		pct := math.Min(1, float64(i)/100)
		if _, err := fmt.Fprintf(stdout, "\r%s", model.ViewAs(pct)); err != nil {
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func parsePaletteColor(value string) (color.Color, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "#") {
		if err := validateHexColor(value); err != nil {
			return nil, err
		}
		return lipgloss.Color(strings.ToLower(value)), nil
	}

	index, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid color %q: must be an integer from 0 to 255 or a #rrggbb value", value)
	}
	if index < 0 || index > 255 {
		return nil, fmt.Errorf("invalid color %q: must be between 0 and 255", value)
	}
	return lipgloss.Color(strconv.Itoa(index)), nil
}

func validateHexColor(value string) error {
	if len(value) != 7 {
		return fmt.Errorf("invalid color %q: expected #rrggbb", value)
	}
	for i := 1; i < 7; i += 2 {
		if _, err := strconv.ParseUint(value[i:i+2], 16, 8); err != nil {
			return fmt.Errorf("invalid color %q: expected #rrggbb", value)
		}
	}
	return nil
}
