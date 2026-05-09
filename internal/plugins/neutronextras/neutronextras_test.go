package neutronextras

import (
	"testing"

	"github.com/crandallnet/golang-osc/internal/cliplugin"
)

func TestModuleRegisters(t *testing.T) {
	ids := cliplugin.ModuleIDs(cliplugin.NamespaceExtras)
	found := false
	for _, id := range ids {
		if id == cliplugin.NamespaceExtras+".neutron-extras" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("neutron-extras module was not registered: %v", ids)
	}
}

func TestPluginCommands(t *testing.T) {
	commands := (Module{}).PluginCommands()
	if len(commands) == 0 {
		t.Fatal("expected neutron-extras commands")
	}
	for _, command := range commands {
		if command.Group != "openstack.network.v2" {
			t.Fatalf("command %q group = %q, want openstack.network.v2", command.Path, command.Group)
		}
		if !command.Implemented {
			t.Fatalf("command %q is not marked implemented", command.Path)
		}
	}
}
