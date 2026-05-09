package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	if entry.Status != "golden-matched" {
		t.Fatalf("expected command list to be golden-matched, got %q", entry.Status)
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
	if entry.Status != "golden-matched" {
		t.Fatalf("expected resource filter list to be golden-matched, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected resource filter list to be marked as a shim")
	}
	if entry.ImplementedIn != "internal/plugins/cinderextras" {
		t.Fatalf("unexpected implementation owner: %q", entry.ImplementedIn)
	}
	if got := commandReportSource(entry); got != "plugin" {
		t.Fatalf("expected plugin report source, got %q", got)
	}
	if !strings.Contains(strings.Join(entry.Tests, " "), "mocked extras REST endpoint") {
		t.Fatalf("expected resource filter list to record mocked endpoint evidence, got %v", entry.Tests)
	}
}

func TestNewCommandEntryMarksNeutronExtrasShim(t *testing.T) {
	entry := newCommandEntry("openstack.network.v2", "network flavor list")
	if entry.Status != "golden-matched" {
		t.Fatalf("expected network flavor list to be golden-matched, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected network flavor list to be marked as a shim")
	}
	if entry.ImplementedIn != "internal/plugins/neutronextras" {
		t.Fatalf("unexpected implementation owner: %q", entry.ImplementedIn)
	}
	if got := commandReportSource(entry); got != "plugin" {
		t.Fatalf("expected plugin report source, got %q", got)
	}
}

func TestNewCommandEntryMarksGoldenMatchedCoreReads(t *testing.T) {
	for _, command := range []string{"flavor list", "image list", "network list"} {
		entry := newCommandEntry("openstack.compute.v2", command)
		if strings.HasPrefix(command, "image ") {
			entry = newCommandEntry("openstack.image.v2", command)
		}
		if strings.HasPrefix(command, "network ") {
			entry = newCommandEntry("openstack.network.v2", command)
		}
		if entry.Status != "golden-matched" {
			t.Fatalf("expected %s to be golden-matched, got %q", command, entry.Status)
		}
		if !strings.Contains(strings.Join(entry.Tests, " "), "compat-live") {
			t.Fatalf("expected %s to record compat-live evidence, got %v", command, entry.Tests)
		}
	}
}

func TestNewCommandEntryMarksCinderMessageShim(t *testing.T) {
	entry := newCommandEntry("openstack.volume.v3", "volume message show")
	if entry.Status != "golden-matched" {
		t.Fatalf("expected volume message show to be golden-matched, got %q", entry.Status)
	}
	if !entry.Shim {
		t.Fatal("expected volume message show to be marked as a shim")
	}
}

func TestNewCommandEntryMarksCinderGroupTypeShowImplemented(t *testing.T) {
	entry := newCommandEntry("openstack.volume.v3", "volume group type show")
	if entry.Status != "cloud-verified" {
		t.Fatalf("expected volume group type show to be cloud-verified, got %q", entry.Status)
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

func TestNewCommandEntryMarksServerSSHImplementedByPlugin(t *testing.T) {
	entry := newCommandEntry("openstack.compute.v2", "server ssh")
	if entry.Status != "implemented" {
		t.Fatalf("expected server ssh to be implemented, got %q", entry.Status)
	}
	if !entry.PluginScope {
		t.Fatal("expected server ssh to be plugin scoped")
	}
	if entry.ImplementedIn != "internal/plugins/novaextras" {
		t.Fatalf("implemented_in mismatch: got %q", entry.ImplementedIn)
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

func TestCommandStatusPercentage(t *testing.T) {
	if got := commandStatusPercentage(1, 4); got != 25 {
		t.Fatalf("expected 25%%, got %.2f", got)
	}
	if got := commandStatusPercentage(1, 0); got != 0 {
		t.Fatalf("expected zero percent for empty total, got %.2f", got)
	}
}

func TestRenderCommandMatrixIncludesStatusSummary(t *testing.T) {
	output := renderCommandMatrixEntries([]commandEntry{
		{Command: "command list", Group: "openstack.cli", Service: "cli", API: "internal", Status: "golden-matched", ImplementedIn: "internal/cli"},
		{Command: "server list", Group: "openstack.compute.v2", Service: "compute", API: "v2", Status: "implemented", ImplementedIn: "internal/cli"},
	})
	for _, want := range []string{
		"status_summary:",
		"  total: 2",
		"    - status: \"unknown\"",
		"      count: 0",
		"      percentage: 0.00",
		"    - status: \"implemented\"",
		"      count: 1",
		"      percentage: 50.00",
		"    - status: \"golden-matched\"",
		"      count: 1",
		"commands:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("matrix summary missing %q:\n%s", want, output)
		}
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
		"    unknown: 1 (33.33%)",
		"    implemented: 1 (33.33%)",
		"    cloud-verified: 1 (33.33%)",
		"    blocked: 0 (0.00%)",
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
		"| Status | Count | Percent |",
		"| `unknown` | 1 | 33.33% |",
		"| `implemented` | 1 | 33.33% |",
		"| `cloud-verified` | 1 | 33.33% |",
		"| `blocked` | 0 | 0.00% |",
		"* `compat/matrix.yaml`",
		"* `compat/test-matrix.yaml`",
		"* `compat/test-clouds.yaml`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README summary missing %q:\n%s", want, text)
		}
	}
}

func TestReadmeCommandMatrixStatusIsCurrent(t *testing.T) {
	groups, err := readGroups(filepath.Join("..", "..", "compat", "osc", "9.0.0", "commands.json"))
	if err != nil {
		t.Fatalf("read command groups: %v", err)
	}
	entries := commandEntries(groups)
	counts := commandStatusCounts(entries)
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(readme)
	if !strings.Contains(text, "## Command Matrix Status") {
		t.Fatalf("README.md missing bottom command matrix status section")
	}
	for _, status := range matrixStatusValues {
		want := fmt.Sprintf("| `%s` | %d | %.2f%% |", status, counts[status], commandStatusPercentage(counts[status], len(entries)))
		if !strings.Contains(text, want) {
			t.Fatalf("README.md command status table is stale; missing %q", want)
		}
	}
}

func TestCommandReportStatusMapping(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{status: "golden-matched", want: "compatible"},
		{status: "cloud-verified", want: "partially compatible"},
		{status: "implemented", want: "implemented"},
		{status: "unknown", want: "partially implemented"},
		{status: "sdk-covered", want: "partially implemented"},
		{status: "shim-needed", want: "partially implemented"},
		{status: "blocked", want: "partially implemented"},
		{status: "local-client-needed", want: "partially implemented"},
	}
	for _, tt := range tests {
		got := commandReportStatus(commandEntry{Status: tt.status})
		if got != tt.want {
			t.Fatalf("status %q mapped to %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestPrintCommandStatusReportReadmeFormat(t *testing.T) {
	entries := []commandEntry{
		{Command: "server list", Status: "golden-matched", ImplementedIn: "internal/cli"},
		{Command: "volume message show", Status: "cloud-verified", Shim: true, ImplementedIn: "internal/cli"},
		{Command: "resource provider list", Status: "implemented", PluginScope: true, ImplementedIn: "internal/cli"},
		{Command: "tap flow list", Status: "unknown", PluginScope: true},
	}
	var output strings.Builder
	printCommandStatusReport(&output, entries, "readme")
	text := output.String()
	for _, want := range []string{
		"### Command Compatibility Status",
		"| Command | Python OSC 9.0.0 | golang-osc status | Source | Notes |",
		"| `server list` | present | compatible | built-in | Golden Python oracle parity recorded. |",
		"| `volume message show` | present | partially compatible | built-in | Live cloud verification recorded; full oracle parity may still be open. Uses a raw REST shim. |",
		"| `resource provider list` | present | implemented | plugin | Implemented in Go; compatibility verification still open. Plugin-scoped command. |",
		"| `tap flow list` | present | partially implemented | plugin | Command path exists in the Go CLI, but behavior is not complete. Plugin-scoped command. |",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("command report missing %q:\n%s", want, text)
		}
	}
}

func TestMarkdownTableCellEscapesPipesAndNewlines(t *testing.T) {
	got := markdownTableCell("left|right\nnext")
	want := `left\|right<br>next`
	if got != want {
		t.Fatalf("escaped cell mismatch: got %q want %q", got, want)
	}
}

func TestYAMLStringEscapesQuotes(t *testing.T) {
	if got, want := yamlString(`a "quoted" value`), `"a \"quoted\" value"`; got != want {
		t.Fatalf("quoted string mismatch: got %q want %q", got, want)
	}
}
