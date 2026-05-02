package main

import "testing"

func TestFlattenCommandsSortsAcrossGroups(t *testing.T) {
	groups := []commandGroup{
		{CommandGroup: "b", Commands: []string{"server show", "server list"}},
		{CommandGroup: "a", Commands: []string{"command list"}},
	}

	got := flattenCommands(groups)
	want := []string{"command list", "server list", "server show"}
	if len(got) != len(want) {
		t.Fatalf("command count mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("command %d mismatch: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestCountByGroup(t *testing.T) {
	counts := countByGroup([]commandGroup{
		{CommandGroup: "openstack.cli", Commands: []string{"command list", "module list"}},
	})
	if got, want := counts["openstack.cli"], 2; got != want {
		t.Fatalf("group count mismatch: got %d want %d", got, want)
	}
}
