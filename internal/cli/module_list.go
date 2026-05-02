package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/crandallnet/golang-osc/internal/cliplugin"
	"github.com/spf13/cobra"
)

func runModuleList(stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		modules := map[string]string{
			"openstack":       CLIVersion,
			"openstackclient": OSCCompatibilityTarget + " compatibility target",
		}
		for _, namespace := range []string{cliplugin.NamespaceCore, cliplugin.NamespacePlugins, cliplugin.NamespaceExtras} {
			for _, id := range cliplugin.ModuleIDs(namespace) {
				modules[id] = "loaded"
			}
		}

		switch opts.Format {
		case "json":
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(modules)
		case "value":
			for field, value := range modules {
				if _, err := fmt.Fprintf(stdout, "%s %s\n", field, value); err != nil {
					return err
				}
			}
			return nil
		default:
			return renderFieldValueTable(stdout, modules)
		}
	}
}

func renderFieldValueTable(stdout io.Writer, fields map[string]string) error {
	if _, err := fmt.Fprintln(stdout, "+-------------------------------+-----------------------------+"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "| Field                         | Value                       |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "+-------------------------------+-----------------------------+"); err != nil {
		return err
	}
	for _, field := range sortedKeys(fields) {
		if _, err := fmt.Fprintf(stdout, "| %-29s | %-27s |\n", truncate(field, 29), truncate(fields[field], 27)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(stdout, "+-------------------------------+-----------------------------+")
	return err
}
