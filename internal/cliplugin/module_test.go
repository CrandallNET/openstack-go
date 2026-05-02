package cliplugin

import "testing"

func TestModuleIDsEmptyNamespace(t *testing.T) {
	if got := ModuleIDs("openstack.commands.unknown"); len(got) != 0 {
		t.Fatalf("expected no modules for unknown namespace, got %v", got)
	}
}
