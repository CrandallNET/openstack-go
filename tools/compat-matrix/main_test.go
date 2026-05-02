package main

import "testing"

func TestServiceForGroup(t *testing.T) {
	service, api := serviceForGroup("openstack.compute.v2")
	if service != "compute" || api != "v2" {
		t.Fatalf("unexpected compute mapping: %q %q", service, api)
	}
}

func TestNewCommandEntryMarksImplementedCommandList(t *testing.T) {
	entry := newCommandEntry("openstack.cli", "command list")
	if entry.Status != "implemented" {
		t.Fatalf("expected command list to be implemented, got %q", entry.Status)
	}
	if entry.ImplementedIn != "internal/cli" {
		t.Fatalf("unexpected implementation owner: %q", entry.ImplementedIn)
	}
}

func TestNewCommandEntryMarksPluginScope(t *testing.T) {
	entry := newCommandEntry("openstack.placement.v1", "resource provider list")
	if !entry.PluginScope {
		t.Fatal("expected placement command to be plugin scoped")
	}

	entry = newCommandEntry("openstack.network.v2", "tap flow list")
	if !entry.PluginScope {
		t.Fatal("expected tap command to be plugin scoped")
	}
}

func TestYAMLStringEscapesQuotes(t *testing.T) {
	if got, want := yamlString(`a "quoted" value`), `"a \"quoted\" value"`; got != want {
		t.Fatalf("quoted string mismatch: got %q want %q", got, want)
	}
}
