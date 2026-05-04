package main

import (
	"strings"
	"testing"
)

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

func TestNewCommandEntryMarksImplementedModuleList(t *testing.T) {
	entry := newCommandEntry("openstack.cli", "module list")
	if entry.Status != "implemented" {
		t.Fatalf("expected module list to be implemented, got %q", entry.Status)
	}
	if entry.ImplementedIn != "internal/cli" {
		t.Fatalf("unexpected implementation owner: %q", entry.ImplementedIn)
	}
}

func TestNewCommandEntryMarksCinderResourceFilterShim(t *testing.T) {
	entry := newCommandEntry("openstack.volume.v3", "block storage resource filter list")
	if entry.Status != "cloud-verified" {
		t.Fatalf("expected resource filter list to be cloud-verified, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected resource filter list to be marked as a shim")
	}
	if entry.ImplementedIn != "internal/cli" {
		t.Fatalf("unexpected implementation owner: %q", entry.ImplementedIn)
	}
}

func TestNewCommandEntryMarksCinderMessageShim(t *testing.T) {
	entry := newCommandEntry("openstack.volume.v3", "volume message show")
	if entry.Status != "cloud-verified" {
		t.Fatalf("expected volume message show to be cloud-verified, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected volume message show to be marked as a shim")
	}
}

func TestNewCommandEntryMarksCinderGroupTypeShowImplemented(t *testing.T) {
	entry := newCommandEntry("openstack.volume.v3", "volume group type show")
	if entry.Status != "implemented" {
		t.Fatalf("expected volume group type show to be implemented, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected volume group type show to be marked as a shim")
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

func TestCommandStatusCounts(t *testing.T) {
	entries := []commandEntry{
		{Command: "command list", Status: "implemented"},
		{Command: "server list", Status: "cloud-verified"},
		{Command: "server create", Status: "unknown"},
	}
	counts := commandStatusCounts(entries)
	if counts["implemented"] != 1 {
		t.Fatalf("expected one implemented command, got %d", counts["implemented"])
	}
	if counts["cloud-verified"] != 1 {
		t.Fatalf("expected one cloud-verified command, got %d", counts["cloud-verified"])
	}
	if counts["unknown"] != 1 {
		t.Fatalf("expected one unknown command, got %d", counts["unknown"])
	}
	if counts["blocked"] != 0 {
		t.Fatalf("expected zero blocked commands, got %d", counts["blocked"])
	}
}

func TestPrintGenerationSummary(t *testing.T) {
	var output strings.Builder
	printGenerationSummary(&output, generationSummary{
		CommandCount: 3,
		StatusCounts: map[string]int{
			"unknown":        1,
			"implemented":    1,
			"cloud-verified": 1,
		},
		MatrixPath:     "compat/matrix.yaml",
		TestMatrixPath: "compat/test-matrix.yaml",
		TestCloudsPath: "compat/test-clouds.yaml",
	}, "terminal")
	text := output.String()
	for _, want := range []string{
		"matrix results:",
		"  commands: 3",
		"    unknown: 1",
		"    implemented: 1",
		"    cloud-verified: 1",
		"    compat/matrix.yaml",
		"    compat/test-matrix.yaml",
		"    compat/test-clouds.yaml",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestPrintGenerationSummaryReadmeFormat(t *testing.T) {
	var output strings.Builder
	printGenerationSummary(&output, generationSummary{
		CommandCount: 3,
		StatusCounts: map[string]int{
			"unknown":        1,
			"implemented":    1,
			"cloud-verified": 1,
		},
		MatrixPath:     "compat/matrix.yaml",
		TestMatrixPath: "compat/test-matrix.yaml",
		TestCloudsPath: "compat/test-clouds.yaml",
	}, "readme")
	text := output.String()
	for _, want := range []string{
		"### Matrix Results",
		"Generated command rows: `3`",
		"| Status | Count |",
		"| `unknown` | 1 |",
		"| `implemented` | 1 |",
		"| `cloud-verified` | 1 |",
		"* `compat/matrix.yaml`",
		"* `compat/test-matrix.yaml`",
		"* `compat/test-clouds.yaml`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README summary missing %q:\n%s", want, text)
		}
	}
}

func TestYAMLStringEscapesQuotes(t *testing.T) {
	if got, want := yamlString(`a "quoted" value`), `"a \"quoted\" value"`; got != want {
		t.Fatalf("quoted string mismatch: got %q want %q", got, want)
	}
}
