package local

import (
	"testing"

	"github.com/crandallnet/golang-osc/internal/cliplugin"
)

func TestLocalModuleIsRegistered(t *testing.T) {
	ids := cliplugin.ModuleIDs(cliplugin.NamespaceCore)
	if len(ids) == 0 {
		t.Fatal("expected at least one core command module")
	}
	if got, want := ids[0], cliplugin.NamespaceCore+".local"; got != want {
		t.Fatalf("module ID mismatch: got %q want %q", got, want)
	}
}

func TestLocalModuleProvidesCommands(t *testing.T) {
	providers, err := cliplugin.Providers(cliplugin.NamespaceCore)
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
}
