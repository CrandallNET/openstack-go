package cinderextras

import (
	"testing"

	"github.com/crandallnet/golang-osc/internal/cliplugin"
)

func TestCinderExtrasModuleIsRegistered(t *testing.T) {
	ids := cliplugin.ModuleIDs(cliplugin.NamespaceExtras)
	if len(ids) == 0 {
		t.Fatal("expected at least one extras command module")
	}
	if got, want := ids[0], cliplugin.NamespaceExtras+".cinder-extras"; got != want {
		t.Fatalf("module ID mismatch: got %q want %q", got, want)
	}
}

func TestCinderExtrasModuleProvidesResourceFilterCommands(t *testing.T) {
	providers, err := cliplugin.Providers(cliplugin.NamespaceExtras)
	if err != nil {
		t.Fatalf("expected providers, got error %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}

	commands := providers[0].PluginCommands()
	if len(commands) != 2 {
		t.Fatalf("command count mismatch: got %d want 2", len(commands))
	}
	for _, command := range commands {
		if command.Group != "openstack.volume.v3" {
			t.Fatalf("group mismatch for %q: got %q", command.Path, command.Group)
		}
		if !command.Implemented {
			t.Fatalf("expected %q to be implemented", command.Path)
		}
	}
}
