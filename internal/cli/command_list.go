package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"charm.land/bubbles/v2/table"
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
		case "pretty":
			return renderCommandListPretty(stdout, opts, rows)
		default:
			return renderCommandListTable(stdout, opts, rows)
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

func renderCommandListTable(stdout io.Writer, opts *Options, rows []osc.CommandGroup) error {
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		for i, command := range row.Commands {
			group := ""
			if i == 0 {
				group = row.CommandGroup
			}
			tableRows = append(tableRows, []string{group, command})
		}
	}
	return renderTable(stdout, opts, []string{"Command Group", "Commands"}, tableRows, 8, opts.PrintEmpty)
}

func renderCommandListPretty(stdout io.Writer, opts *Options, rows []osc.CommandGroup) error {
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		for i, command := range row.Commands {
			group := ""
			if i == 0 {
				group = row.CommandGroup
			}
			tableRows = append(tableRows, table.Row{group, command})
		}
	}
	return renderPrettyTable(stdout, opts, []string{"Command Group", "Commands"}, tableRows, prettyListCellColorizer([]string{"Command Group", "Commands"}))
}
