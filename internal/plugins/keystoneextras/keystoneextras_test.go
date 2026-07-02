package keystoneextras

import (
	"testing"

	"github.com/crandallnet/openstack-go/internal/cliplugin"
)

func TestKeystoneExtrasModuleIsRegistered(t *testing.T) {
	ids := cliplugin.ModuleIDs(cliplugin.NamespaceExtras)
	found := false
	for _, id := range ids {
		if id == cliplugin.NamespaceExtras+".keystone-extras" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("keystone-extras module was not registered: %v", ids)
	}
}

func TestPluginCommands(t *testing.T) {
	commands := (Module{}).PluginCommands()
	if len(commands) == 0 {
		t.Fatalf("expected plugin commands")
	}
	seen := map[string]bool{}
	for _, command := range commands {
		if command.Group != "openstack.identity.v3" {
			t.Fatalf("%s group = %s, want openstack.identity.v3", command.Path, command.Group)
		}
		if !command.Implemented {
			t.Fatalf("%s is not marked implemented", command.Path)
		}
		seen[command.Path] = true
	}
	for _, path := range []string{
		"endpoint group list",
		"federation protocol show",
		"identity provider set",
		"service provider create",
	} {
		if !seen[path] {
			t.Fatalf("missing %q", path)
		}
	}
}
