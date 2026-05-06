package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
	groupedRows := commandListPrettyRows(rows)
	tableRows := make([]table.Row, 0, len(groupedRows))
	for _, row := range groupedRows {
		tableRows = append(tableRows, table.Row{row.CommandGroup, row.Command, row.Subcommands})
	}
	columns := []string{"Command Group", "Command", "Subcommands"}
	return renderPrettyTable(stdout, opts, columns, tableRows, prettyListCellColorizer(columns), prettyListCellContext(columns))
}

type commandListPrettyRow struct {
	CommandGroup string
	Command      string
	Subcommands  string
}

func commandListPrettyRows(rows []osc.CommandGroup) []commandListPrettyRow {
	tableRows := []commandListPrettyRow{}
	for _, row := range rows {
		grouped := groupPrettyCommands(row.Commands)
		for i, command := range grouped {
			group := ""
			if i == 0 {
				group = row.CommandGroup
			}
			tableRows = append(tableRows, commandListPrettyRow{
				CommandGroup: group,
				Command:      command.Command,
				Subcommands:  strings.Join(command.Subcommands, "\n"),
			})
		}
	}
	return tableRows
}

type prettyCommandGroup struct {
	Command     string
	Subcommands []string
}

func groupPrettyCommands(commands []string) []prettyCommandGroup {
	ordered := []prettyCommandGroup{}
	indexes := map[string]int{}
	for _, command := range commands {
		root, subcommand := splitPrettyCommand(command)
		index, ok := indexes[root]
		if !ok {
			index = len(ordered)
			indexes[root] = index
			ordered = append(ordered, prettyCommandGroup{Command: root})
		}
		ordered[index].Subcommands = append(ordered[index].Subcommands, subcommand)
	}
	return ordered
}

func splitPrettyCommand(command string) (string, string) {
	implementedSuffix := ""
	if strings.HasSuffix(command, notImplementedSuffix) {
		command = strings.TrimSuffix(command, notImplementedSuffix)
		implementedSuffix = notImplementedSuffix
	}
	root, subcommand, ok := strings.Cut(command, " ")
	if !ok {
		return command, strings.TrimSpace(implementedSuffix)
	}
	return root, subcommand + implementedSuffix
}
