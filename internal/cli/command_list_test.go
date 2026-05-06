package cli

import (
	"testing"

	"github.com/crandallnet/golang-osc/compat/osc"
)

func TestCommandListPrettyRowsGroupByRootCommand(t *testing.T) {
	rows := commandListPrettyRows([]osc.CommandGroup{
		{
			CommandGroup: "openstack.compute.v2",
			Commands: []string{
				"server add network",
				"server create",
				"server volume set",
				"flavor list",
				"flavor show",
				"host list" + notImplementedSuffix,
			},
		},
	})

	if len(rows) != 3 {
		t.Fatalf("expected 3 grouped rows, got %d: %#v", len(rows), rows)
	}
	if rows[0].CommandGroup != "openstack.compute.v2" || rows[0].Command != "server" || rows[0].Subcommands != "add network\ncreate\nvolume set" {
		t.Fatalf("server grouping mismatch: %#v", rows[0])
	}
	if rows[1].CommandGroup != "" || rows[1].Command != "flavor" || rows[1].Subcommands != "list\nshow" {
		t.Fatalf("flavor grouping mismatch: %#v", rows[1])
	}
	if rows[2].CommandGroup != "" || rows[2].Command != "host" || rows[2].Subcommands != "list"+notImplementedSuffix {
		t.Fatalf("not implemented grouping mismatch: %#v", rows[2])
	}
}
