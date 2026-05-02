package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/crandallnet/golang-osc/compat/osc"
	"github.com/spf13/cobra"
)

const notImplementedSuffix = " (Not Implemented Yet)"

func runCommandList(groups []osc.CommandGroup, stdout io.Writer, opts *Options, implemented map[string]commandHandler) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		groupFilter, err := cmd.Flags().GetString("group")
		if err != nil {
			return err
		}

		rows := commandListRows(groups, implementedCommandNames(implemented), groupFilter)

		switch opts.Format {
		case "json":
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(rows)
		case "value":
			for _, group := range rows {
				for _, command := range group.Commands {
					if _, err := fmt.Fprintln(stdout, command); err != nil {
						return err
					}
				}
			}
			return nil
		default:
			return renderCommandListTable(stdout, rows)
		}
	}
}

func commandListRows(groups []osc.CommandGroup, implemented map[string]bool, groupFilter string) []osc.CommandGroup {
	var rows []osc.CommandGroup
	for _, group := range sortedGroups(groups) {
		if groupFilter != "" && group.CommandGroup != groupFilter {
			continue
		}
		row := osc.CommandGroup{CommandGroup: group.CommandGroup}
		for _, command := range group.Commands {
			if implemented[command] {
				row.Commands = append(row.Commands, command)
				continue
			}
			row.Commands = append(row.Commands, command+notImplementedSuffix)
		}
		rows = append(rows, row)
	}
	return rows
}

func renderCommandListTable(stdout io.Writer, rows []osc.CommandGroup) error {
	if _, err := fmt.Fprintln(stdout, "+---------------------------+--------------------------------+"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "| Command Group             | Commands                       |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "+---------------------------+--------------------------------+"); err != nil {
		return err
	}
	for _, row := range rows {
		commands := strings.Join(row.Commands, "\n")
		lines := strings.Split(commands, "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		for i, line := range lines {
			group := ""
			if i == 0 {
				group = row.CommandGroup
			}
			if _, err := fmt.Fprintf(stdout, "| %-25s | %-30s |\n", truncate(group, 25), truncate(line, 30)); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(stdout, "+---------------------------+--------------------------------+")
	return err
}

func truncate(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
