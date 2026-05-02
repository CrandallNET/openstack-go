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
			return renderFieldValueTable(stdout, opts, modules)
		}
	}
}
