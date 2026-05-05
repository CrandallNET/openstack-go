package main

import "testing"

func TestSplitClouds(t *testing.T) {
	got := splitClouds(" cloud6, flex-sjc ,,flex-dfw ")
	want := []string{"cloud6", "flex-sjc", "flex-dfw"}
	if len(got) != len(want) {
		t.Fatalf("cloud count mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cloud %d mismatch: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestSelectedFixtureSetSortsLimitsAndMarksFirstCandidate(t *testing.T) {
	items := []fixtureItem{
		{ID: "3", Name: "zeta"},
		{ID: "2", Name: "alpha"},
		{ID: "1", Name: "alpha"},
	}
	set := selectedFixtureSet(items, 2)
	if set.Status != "ok" {
		t.Fatalf("expected ok fixture status, got %q", set.Status)
	}
	if len(set.Items) != 2 {
		t.Fatalf("expected limit to keep two items, got %d", len(set.Items))
	}
	if set.Items[0].ID != "1" || set.Items[0].Name != "alpha" || !set.Items[0].Selected {
		t.Fatalf("first selected fixture mismatch: %+v", set.Items[0])
	}
	if set.Items[1].ID != "2" || set.Items[1].Selected {
		t.Fatalf("second fixture mismatch: %+v", set.Items[1])
	}
}
