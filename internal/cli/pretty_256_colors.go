package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// RenderPretty256ColorTest displays all 256 ANSI terminal colors using
// the same pretty renderer path used by the Go CLI.
func RenderPretty256ColorTest(stdout io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	renderOpts := *opts
	renderOpts.Format = "pretty"

	if _, ok := stdout.(*os.File); ok && detectTableWriterIsTerminal(stdout) {
		renderOpts.FitWidth = true
	}

	rows := make([]outputRow, 0, 256)
	for i := 0; i < 256; i++ {
		bgColor := lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i)))
		rows = append(rows, outputRow{
			"Color": prettyImageValue(bgColor.Render(fmt.Sprintf("%3d ", i))),
		})
	}
	return renderListOutput(stdout, &renderOpts, []string{"Color"}, rows)
}

// RenderPretty256ColorTestGrid displays all 256 ANSI terminal colors in an
// 8-colour-per-row grid (32 rows × 8 columns) using the same pretty renderer
// path used by the Go CLI.
func RenderPretty256ColorTestGrid(stdout io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	renderOpts := *opts
	renderOpts.Format = "pretty"

	if _, ok := stdout.(*os.File); ok && detectTableWriterIsTerminal(stdout) {
		renderOpts.FitWidth = true
	}

	rows := make([]outputRow, 0, 32)
	for row := 0; row < 32; row++ {
		parts := make([]string, 0, 8)
		for col := 0; col < 8; col++ {
			i := row*8 + col
			bgColor := lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i)))
			parts = append(parts, bgColor.Render(fmt.Sprintf("%3d ", i)))
		}
		rows = append(rows, outputRow{
			"Color": prettyImageValue(strings.Join(parts, "")),
		})
	}
	return renderListOutput(stdout, &renderOpts, []string{"Color"}, rows)
}
