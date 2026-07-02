package novaextras

import (
	"testing"

	"github.com/crandallnet/openstack-go/internal/cliplugin"
)

func TestNovaExtrasModuleRegistered(t *testing.T) {
	ids := cliplugin.ModuleIDs(cliplugin.NamespaceExtras)
	for _, id := range ids {
		if id == cliplugin.NamespaceExtras+".nova-extras" {
			return
		}
	}
	t.Fatalf("expected nova extras module in %v", ids)
}

func TestNovaExtrasCommands(t *testing.T) {
	commands := (Module{}).PluginCommands()
	if len(commands) != 1 {
		t.Fatalf("expected one nova extras command, got %d", len(commands))
	}
	command := commands[0]
	if command.Path != "server ssh" {
		t.Fatalf("path mismatch: got %q", command.Path)
	}
	if command.Group != "openstack.compute.v2" {
		t.Fatalf("group mismatch: got %q", command.Group)
	}
	if !command.Implemented {
		t.Fatal("expected server ssh to be implemented")
	}
}
