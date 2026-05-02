package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/crandallnet/golang-osc/compat/osc"
	"github.com/spf13/cobra"
)

type commandHandler func(*cobra.Command, []string) error

type commandRegistry struct {
	groups      []osc.CommandGroup
	implemented map[string]commandHandler
	stdout      io.Writer
}

func newCommandRegistry(groups []osc.CommandGroup, stdout io.Writer, opts *Options) *commandRegistry {
	registry := &commandRegistry{
		groups: groups,
		implemented: map[string]commandHandler{
			"command list": runCommandList(groups, stdout, opts),
		},
		stdout: stdout,
	}
	return registry
}

func (r *commandRegistry) addCatalogCommands(root *cobra.Command) {
	for _, group := range r.groups {
		for _, command := range group.Commands {
			r.addCommand(root, command)
		}
	}
}

func (r *commandRegistry) addCommand(root *cobra.Command, path string) {
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return
	}

	parent := root
	for i, part := range parts {
		fullPath := strings.Join(parts[:i+1], " ")
		child := findChild(parent, part)
		if child == nil {
			child = &cobra.Command{
				Use:           part,
				Short:         fullPath,
				SilenceUsage:  true,
				SilenceErrors: true,
			}
			child.Flags().SortFlags = false
			child.PersistentFlags().SortFlags = false
			parent.AddCommand(child)
		}

		if i == len(parts)-1 {
			r.configureLeaf(child, fullPath)
		}
		parent = child
	}
}

func (r *commandRegistry) configureLeaf(cmd *cobra.Command, path string) {
	cmd.Short = path
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		if help, ok, err := osc.Help(path); err == nil && ok {
			fmt.Fprint(r.stdout, help)
			return
		}
		_ = command.Parent().Help()
	})

	if handler, ok := r.implemented[path]; ok {
		cmd.RunE = handler
		if path == "command list" {
			cmd.Flags().String("group", "", "filter by command group")
		}
		return
	}

	cmd.DisableFlagParsing = true
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(command *cobra.Command, args []string) error {
		if containsHelpFlag(args) {
			return command.Help()
		}
		_, err := fmt.Fprintln(r.stdout, notImplementedExitCodeText)
		return err
	}
}

func findChild(parent *cobra.Command, name string) *cobra.Command {
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func implementedCommandNames(implemented map[string]commandHandler) map[string]bool {
	names := make(map[string]bool, len(implemented))
	for name := range implemented {
		names[name] = true
	}
	return names
}

func sortedGroups(groups []osc.CommandGroup) []osc.CommandGroup {
	copied := make([]osc.CommandGroup, len(groups))
	for i, group := range groups {
		commands := append([]string(nil), group.Commands...)
		sort.Strings(commands)
		copied[i] = osc.CommandGroup{
			CommandGroup: group.CommandGroup,
			Commands:     commands,
		}
	}
	sort.Slice(copied, func(i int, j int) bool {
		return copied[i].CommandGroup < copied[j].CommandGroup
	})
	return copied
}
