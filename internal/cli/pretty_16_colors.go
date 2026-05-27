package cli

import (
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
)

// RenderPretty16ColorTest displays the 16 standard terminal colors using
// the same pretty renderer path used by the Go CLI.
func RenderPretty16ColorTest(stdout io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	renderOpts := *opts
	renderOpts.Format = "pretty"

	if _, ok := stdout.(*os.File); ok && detectTableWriterIsTerminal(stdout) {
		renderOpts.FitWidth = true
	}

	rows := make([]outputRow, 0, 16)
	for i := 0; i < 16; i++ {
		bgColor := lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i)))
		rows = append(rows, outputRow{
			"Number": fmt.Sprintf("%d", i),
			"Color":  prettyImageValue(bgColor.Render("   ")),
		})
	}
	return renderListOutput(stdout, &renderOpts, []string{"Number", "Color"}, rows)
}
