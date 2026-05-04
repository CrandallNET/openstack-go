package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

func executeForTest(args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Execute(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

func TestVersion(t *testing.T) {
	stdout, stderr, err := executeForTest("--version")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got, want := strings.TrimSpace(stdout), "openstack "+CLIVersion; got != want {
		t.Fatalf("version output mismatch: got %q want %q", got, want)
	}
}

func TestRootHelp(t *testing.T) {
	stdout, stderr, err := executeForTest("--help")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"OpenStack command line client", "--os-cloud", "--pretty", "command"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help output missing %q:\n%s", want, stdout)
		}
	}
}

func TestStubCommand(t *testing.T) {
	stdout, stderr, err := executeForTest("configuration", "show")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "auth_type") {
		t.Fatalf("configuration output missing auth_type:\n%s", stdout)
	}
}

func TestGeneratedStubCommandIgnoresCommandFlags(t *testing.T) {
	stdout, stderr, err := executeForTest("server", "start", "--wait")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got, want := strings.TrimSpace(stdout), notImplementedExitCodeText; got != want {
		t.Fatalf("stub output mismatch: got %q want %q", got, want)
	}
}

func TestGeneratedHelpUsesOracleSnapshot(t *testing.T) {
	stdout, stderr, err := executeForTest("server", "list", "--help")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "openstack server list") {
		t.Fatalf("expected server list oracle help, got:\n%s", stdout)
	}
}

func TestCommandListJSONMarksUnimplementedCommands(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.cli")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"command list"`) {
		t.Fatalf("expected implemented command without suffix, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"module list"`) {
		t.Fatalf("expected module list to be marked implemented, got:\n%s", stdout)
	}
}

func TestCommandListJSONMarksServiceStubs(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.compute.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"server start (Not Implemented Yet)"`) {
		t.Fatalf("expected server start to be marked unimplemented, got:\n%s", stdout)
	}
}

func TestCommandListJSONMarksCoreReadsImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.compute.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{`"server list"`, `"server show"`, `"flavor list"`, `"flavor show"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %s to be marked implemented, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"server list (Not Implemented Yet)"`) {
		t.Fatalf("expected server list without not-implemented suffix, got:\n%s", stdout)
	}
}

func TestCommandListJSONMarksIdentityReadsImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.identity.v3")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"project list"`) {
		t.Fatalf("expected project list to be marked implemented, got:\n%s", stdout)
	}
	if strings.Contains(stdout, `"project list (Not Implemented Yet)"`) {
		t.Fatalf("expected project list without not-implemented suffix, got:\n%s", stdout)
	}
}

func TestModuleListUsesPluginRegistry(t *testing.T) {
	stdout, stderr, err := executeForTest("module", "list", "-f", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"openstack.commands.core.local"`) {
		t.Fatalf("expected local Caddy command module, got:\n%s", stdout)
	}
}

func TestMaxWidthWrapsTableOutput(t *testing.T) {
	stdout, stderr, err := executeForTest("module", "list", "--max-width", "52")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := maxOutputLineLength(stdout); got > 52 {
		t.Fatalf("expected all table lines to fit within 52 columns, longest was %d:\n%s", got, stdout)
	}
}

func TestCliffFitWidthEnvUsesDetectedTerminalWidth(t *testing.T) {
	t.Setenv("CLIFF_FIT_WIDTH", "1")
	previousWidth := tableTerminalWidth
	previousTTY := tableWriterIsTerminal
	tableTerminalWidth = func(stdout io.Writer) (int, bool) {
		return 52, true
	}
	tableWriterIsTerminal = func(stdout io.Writer) bool {
		return false
	}
	defer func() {
		tableTerminalWidth = previousWidth
		tableWriterIsTerminal = previousTTY
	}()

	stdout, stderr, err := executeForTest("module", "list")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := maxOutputLineLength(stdout); got > 52 {
		t.Fatalf("expected CLIFF_FIT_WIDTH table lines to fit within 52 columns, longest was %d:\n%s", got, stdout)
	}
}

func TestCompleteUsesOracleSnapshot(t *testing.T) {
	stdout, stderr, err := executeForTest("complete")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "_openstack()") {
		t.Fatalf("expected bash completion function, got:\n%s", stdout)
	}
}

func TestPrettyFlagParses(t *testing.T) {
	stdout, stderr, err := executeForTest("--pretty", "configuration", "show")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "auth_type") {
		t.Fatalf("pretty configuration output missing auth_type:\n%s", stdout)
	}
}

func TestOSPrettyEnvDefaultsToPretty(t *testing.T) {
	t.Setenv("OS_PRETTY", "1")
	stdout, stderr, err := executeForTest("configuration", "show")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "+---") {
		t.Fatalf("expected OS_PRETTY=1 to use pretty output by default, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "auth_type") {
		t.Fatalf("pretty configuration output missing auth_type:\n%s", stdout)
	}
}

func TestOSPrettyEnvDoesNotOverrideExplicitFormat(t *testing.T) {
	t.Setenv("OS_PRETTY", "1")
	stdout, stderr, err := executeForTest("-f", "json", "configuration", "show")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"auth_type"`) {
		t.Fatalf("expected explicit JSON format to win over OS_PRETTY, got:\n%s", stdout)
	}
}

func TestPrettyFlagOverridesExplicitFormat(t *testing.T) {
	stdout, stderr, err := executeForTest("-f", "json", "--pretty", "configuration", "show")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, `"auth_type"`) || strings.Contains(stdout, "+---") {
		t.Fatalf("expected explicit --pretty to use pretty output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "auth_type") {
		t.Fatalf("pretty configuration output missing auth_type:\n%s", stdout)
	}
}

func TestPrettyListUsesTabularOutputWithoutANSIForNonTTY(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"ID", "Name", "Status"}, []outputRow{
		{"ID": "server-1", "Name": "alpha", "Status": "ACTIVE"},
		{"ID": "server-2", "Name": "beta", "Status": "SHUTOFF"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"ID", "Name", "Status", "server-1", "alpha", "SHUTOFF"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "ID: server-1") {
		t.Fatalf("expected pretty list output to be tabular, got old key/value output:\n%s", output)
	}
	if containsANSI(output) {
		t.Fatalf("expected non-TTY pretty output without ANSI escapes, got:\n%q", output)
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[1]) != "" {
		t.Fatalf("expected blank line between pretty headings and content, got:\n%s", output)
	}
	if strings.Contains(output, "\n  server-1") {
		t.Fatalf("expected first pretty row not to be shifted by selected-row padding, got:\n%s", output)
	}
}

func TestPrettyShowUsesTabularOutputWithoutANSIForNonTTY(t *testing.T) {
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: "pretty"}, []outputField{
		{Name: "id", Value: "server-1"},
		{Name: "status", Value: "ACTIVE"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"Field", "Value", "id", "server-1", "status", "ACTIVE"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty show output missing %q:\n%s", want, output)
		}
	}
	if containsANSI(output) {
		t.Fatalf("expected non-TTY pretty show output without ANSI escapes, got:\n%q", output)
	}
	if strings.Contains(output, "\n  id") {
		t.Fatalf("expected first pretty show row not to be shifted by selected-row padding, got:\n%s", output)
	}
}

func TestPrettyShowPreservesMultilineValues(t *testing.T) {
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: "pretty"}, []outputField{
		{Name: "cpu_info", Value: "arch: x86_64\nfeatures: sse avx"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	if strings.Contains(output, "arch: x86_64 features: sse avx") {
		t.Fatalf("expected pretty output to preserve multiline values, got:\n%s", output)
	}
	for _, want := range []string{"cpu_info", "arch: x86_64", "features: sse avx"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty multiline output missing %q:\n%s", want, output)
		}
	}
}

func TestPrettyShowFormatsJSONStringsAsStructuredBlocks(t *testing.T) {
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: "pretty"}, []outputField{
		{Name: "cpu_info", Value: `{"vendor":"Intel","features":["sse","avx"],"topology":{"cores":4}}`},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"vendor: Intel", "features:", "- sse", "- avx", "topology:", "cores: 4"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty structured output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, `"vendor"`) || strings.Contains(output, `{"vendor"`) {
		t.Fatalf("expected pretty output to avoid dense JSON, got:\n%s", output)
	}
}

func TestPrettyShowFormatsStructsAsStructuredBlocks(t *testing.T) {
	type topology struct {
		Cores int `json:"cores"`
	}
	type cpuInfo struct {
		Vendor   string   `json:"vendor"`
		Features []string `json:"features"`
		Topology topology `json:"topology"`
	}

	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: "pretty"}, []outputField{
		{Name: "cpu_info", Value: cpuInfo{Vendor: "Intel", Features: []string{"sse", "avx"}, Topology: topology{Cores: 4}}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"vendor: Intel", "features:", "- sse", "- avx", "topology:", "cores: 4"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty structured struct output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, `"vendor"`) || strings.Contains(output, `{"vendor"`) {
		t.Fatalf("expected pretty output to avoid dense JSON, got:\n%s", output)
	}
}

func TestPrettyStructuredObjectListsAvoidStandaloneMarkers(t *testing.T) {
	text, ok := prettyStructuredString([]map[string]any{{
		"attachment_id": "1f7222ef-c386-4523-9788-eb62f293e883",
		"device":        "/dev/vda",
	}})
	if !ok {
		t.Fatalf("expected structured list to format")
	}
	for _, want := range []string{"attachment_id: 1f7222ef-c386-4523-9788-eb62f293e883", "device: /dev/vda"} {
		if !strings.Contains(text, want) {
			t.Fatalf("pretty structured object list missing %q:\n%s", want, text)
		}
	}
	if strings.HasPrefix(text, "-") || strings.Contains(text, "\n-") {
		t.Fatalf("expected structured object list to avoid standalone item markers, got:\n%s", text)
	}
}

func TestPrettyShowWrapsLongValuesInsteadOfTruncating(t *testing.T) {
	var stdout bytes.Buffer
	longValue := strings.Repeat("abcdef ", 12)
	err := renderShowOutput(&stdout, &Options{Format: "pretty", MaxWidth: 40}, []outputField{
		{Name: "cpu_info", Value: longValue},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	if strings.Contains(output, "\u2026") {
		t.Fatalf("expected pretty output to wrap long values instead of truncating, got:\n%s", output)
	}
	if strings.Count(output, "\n") < 3 {
		t.Fatalf("expected wrapped pretty output to span multiple rows, got:\n%s", output)
	}
}

func TestPrettyWrapRowsAddsSpacerBetweenEntries(t *testing.T) {
	rows := prettyWrapRows(
		[]table.Row{{"server-1", "alpha"}, {"server-2", "beta"}},
		[]table.Column{{Title: "ID", Width: 8}, {Title: "Name", Width: 8}},
		nil,
		nil,
	)
	if len(rows) != 3 {
		t.Fatalf("expected spacer row between pretty entries, got %#v", rows)
	}
	for column, value := range rows[1] {
		if value != "" {
			t.Fatalf("expected spacer column %d to be empty, got %q in %#v", column, value, rows)
		}
	}
}

func TestPrettyWrapRowsCanMarkSeparatorsBetweenEntries(t *testing.T) {
	rows, rowKinds := prettyWrapRowsWithKinds(
		[]table.Row{{"server-1", "alpha"}, {"server-2", "beta"}},
		[]table.Column{{Title: "ID", Width: 8}, {Title: "Name", Width: 8}},
		nil,
		nil,
		true,
	)
	if len(rows) != 3 {
		t.Fatalf("expected separator row between pretty entries, got %#v", rows)
	}
	wantKinds := []prettyTableRowKind{prettyTableRowContent, prettyTableRowSeparator, prettyTableRowContent}
	for index, want := range wantKinds {
		if rowKinds[index] != want {
			t.Fatalf("expected row kind %d to be %v, got %v in %#v", index, want, rowKinds[index], rowKinds)
		}
	}
}

func TestPrettyBubbleTableAddsSymmetricCellPadding(t *testing.T) {
	if got, want := prettyBubbleTableCellValue("ID"), " ID "; got != want {
		t.Fatalf("expected bubble-table cell padding %q, got %q", want, got)
	}
	if got := prettyBubbleTableCellValue(""); got != "" {
		t.Fatalf("expected empty bubble-table spacer cells to stay empty, got %q", got)
	}
}

func TestPrettyBubbleTableSeparatorLineMatchesColumnWidths(t *testing.T) {
	line := prettyBubbleTableSeparatorLine([]table.Column{{Title: "ID", Width: 2}, {Title: "Name", Width: 4}})
	if want := "├────┼──────┤"; !strings.Contains(line, want) {
		t.Fatalf("expected bubble-table separator to contain %q, got %q", want, line)
	}
}

func TestPrettyTableColumnsFitBubbleTableToTerminalWidth(t *testing.T) {
	headers := []string{"ID", "Name", "Status", "Networks", "Image", "Flavor", "Project", "Host"}
	rows := []table.Row{{
		"7dbf33e2-6d96-43b1-961b-ae58925a382c",
		"rocky",
		"ACTIVE",
		"os6-lan: 172.16.86.56",
		"N/A (booted from volume)",
		"m1.small",
		"17613f25c8d742f8ab589c0fa7b6a66b",
		"dell6.crandall.haus",
	}}
	columns := prettyTableColumns(headers, rows, 80, true)
	if got := prettyBubbleTableViewWidth(columns); got > 80 {
		t.Fatalf("expected Fancy bubble-table columns to fit 80 columns, got width %d from %#v", got, columns)
	}
	if columns[0].Width < 12 {
		t.Fatalf("expected ID column to stay wide enough for UUID hyphen wrapping, got %#v", columns)
	}
}

func TestPrettyApplyBubbleTableSeparatorsReplacesMarkedRows(t *testing.T) {
	view := strings.Join([]string{
		"╭────╮",
		"│ ID │",
		"├────┤",
		"│ a  │",
		"│    │",
		"│ b  │",
		"╰────╯",
	}, "\n")
	got := prettyApplyBubbleTableSeparators(view, []table.Column{{Title: "ID", Width: 2}}, []prettyTableRowKind{
		prettyTableRowContent,
		prettyTableRowSeparator,
		prettyTableRowContent,
	})
	lines := strings.Split(got, "\n")
	if lines[3] != "│ a  │" {
		t.Fatalf("expected first body row to stay directly under heading, got %q in:\n%s", lines[3], got)
	}
	if want := "├────┤"; !strings.Contains(lines[4], want) {
		t.Fatalf("expected marked spacer to become separator %q, got %q in:\n%s", want, lines[4], got)
	}
}

func TestPrettyTTYOutputFitsDetectedTerminalWidth(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	previousWidth := tableTerminalWidth
	previousTTY := tableWriterIsTerminal
	tableTerminalWidth = func(stdout io.Writer) (int, bool) {
		return 80, true
	}
	tableWriterIsTerminal = func(stdout io.Writer) bool {
		return true
	}
	defer func() {
		tableTerminalWidth = previousWidth
		tableWriterIsTerminal = previousTTY
	}()

	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"ID", "Name", "Status", "Networks", "Image", "Flavor", "Project", "Host"}, []outputRow{
		{
			"ID":       "7dbf33e2-6d96-43b1-961b-ae58925a382c",
			"Name":     "rocky",
			"Status":   "ACTIVE",
			"Networks": "os6-lan: 172.16.86.56",
			"Image":    "N/A (booted from volume)",
			"Flavor":   "m1.small",
			"Project":  "17613f25c8d742f8ab589c0fa7b6a66b",
			"Host":     "dell6.crandall.haus",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := maxVisibleOutputLineLength(stdout.String()); got > 80 {
		t.Fatalf("expected Fancy output to fit detected 80-column terminal, longest visible line was %d:\n%s", got, stdout.String())
	}
}

func TestPrettyTTYOutputDoesNotInsertTopBlankRow(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	previousWidth := tableTerminalWidth
	previousTTY := tableWriterIsTerminal
	tableTerminalWidth = func(stdout io.Writer) (int, bool) {
		return 80, true
	}
	tableWriterIsTerminal = func(stdout io.Writer) bool {
		return true
	}
	defer func() {
		tableTerminalWidth = previousWidth
		tableWriterIsTerminal = previousTTY
	}()

	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"ID", "Name"}, []outputRow{
		{"ID": "7dbf33e2-6d96-43b1-961b-ae58925a382c", "Name": "rocky"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	lines := strings.Split(strings.TrimRight(stripANSI(stdout.String()), "\n"), "\n")
	if len(lines) < 4 || !strings.Contains(lines[3], "7dbf33e2") {
		t.Fatalf("expected first body row directly under header separator, got:\n%s", stdout.String())
	}
}

func TestPrettyWrapsIDsAtHyphenBoundaries(t *testing.T) {
	uuid := "1c77f920-e72d-45d0-8198-5a4a11722214"
	wrapped := prettyWrapTableCell(uuid, 12, "id")
	if got := strings.Join(wrapped, ""); got != uuid {
		t.Fatalf("expected wrapped UUID fragments to preserve value, got %q want %q from %#v", got, uuid, wrapped)
	}
	for index, line := range wrapped[:len(wrapped)-1] {
		if !strings.HasSuffix(line, "-") {
			t.Fatalf("expected wrapped UUID line %d to end at a hyphen, got %#v", index, wrapped)
		}
	}
}

func TestPrettyListFormatsServerNetworksVertically(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"Name", "Networks"}, []outputRow{
		{
			"Name":     "server-1",
			"Networks": prettyNetworkAddresses{"testNet": {"172.16.86.110", "172.17.36.42"}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"testNet:", "172.16.86.110", "172.17.36.42"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty server network output missing %q:\n%s", want, output)
		}
	}
	if got, want := strings.Count(output, "testNet:"), 2; got != want {
		t.Fatalf("expected pretty server networks to repeat the network name for each IP, got %d occurrences:\n%s", got, output)
	}
	if strings.Contains(output, "testNet=172.16.86.110, 172.17.36.42") {
		t.Fatalf("expected pretty server networks to avoid comma-delimited summary, got:\n%s", output)
	}
}

func TestPrettyListFormatsPortFixedIPsVertically(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"Fixed IP Addresses"}, []outputRow{
		{
			"Fixed IP Addresses": prettyPortFixedIPAddresses{
				ports.IP{IPAddress: "172.16.86.117", SubnetID: "d8da6273-ec5f-47da-b269-c276b3e734b0"},
				ports.IP{IPAddress: "172.17.36.42", SubnetID: "e1c1eb6b-d245-411f-af36-9480e8ac83e2"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"172.16.86.117",
		"172.17.36.42",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty fixed IP output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, ", subnet_id=") || strings.Contains(output, "subnet:") {
		t.Fatalf("expected pretty list fixed IPs to stay focused on IP addresses, got:\n%s", output)
	}
}

func TestPrettyShowFormatsNetworkIPFieldsVertically(t *testing.T) {
	opts := &Options{Format: "pretty"}
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, opts, prettyNetworkIPOutputFields(opts, []outputField{
		{
			Name: "fixed_ips",
			Value: []map[string]any{
				{"ip_address": "172.16.86.117", "subnet_id": "d8da6273-ec5f-47da-b269-c276b3e734b0"},
				{"ip_address": "172.17.36.42", "subnet_id": "e1c1eb6b-d245-411f-af36-9480e8ac83e2"},
			},
		},
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"fixed_ips",
		"172.16.86.117",
		"subnet: d8da6273-ec5f-47da-b269-c276b3e734b0",
		"172.17.36.42",
		"subnet: e1c1eb6b-d245-411f-af36-9480e8ac83e2",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty show fixed IP output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "ip_address:") || strings.Contains(output, "subnet_id:") {
		t.Fatalf("expected pretty show fixed IPs to use human-readable labels, got:\n%s", output)
	}
}

func TestPrettyShowFormatsRouterGatewayFixedIPsVertically(t *testing.T) {
	opts := &Options{Format: "pretty"}
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, opts, prettyNetworkIPOutputFields(opts, []outputField{
		{
			Name: "external_gateway_info",
			Value: map[string]any{
				"enable_snat": true,
				"external_fixed_ips": []map[string]any{
					{"ip_address": "172.16.86.32", "subnet_id": "d8da6273-ec5f-47da-b269-c276b3e734b0"},
				},
				"network_id": "cb696f01-32be-4dc3-b562-14ab121bda16",
			},
		},
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"external_gateway_info",
		"enable snat: True",
		"external fixed IPs:",
		"172.16.86.32",
		"subnet: d8da6273-ec5f-47da-b269-c276b3e734b0",
		"network: cb696f01-32be-4dc3-b562-14ab121bda16",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty router gateway output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "ip_address:") || strings.Contains(output, "subnet_id:") || strings.Contains(output, "network_id:") {
		t.Fatalf("expected pretty router gateway output to use human-readable labels, got:\n%s", output)
	}
}

func TestCommandListPrettyUsesPrettyRenderer(t *testing.T) {
	stdout, stderr, err := executeForTest("--pretty", "command", "list", "--group", "openstack.cli")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"Command Group", "openstack.cli", "command list", "module list"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("pretty command list output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "+---") {
		t.Fatalf("expected command list --pretty to avoid default table borders, got:\n%s", stdout)
	}
}

func TestPrettyProgressUsesBubblesProgressWithoutANSIForNonTTY(t *testing.T) {
	var stdout bytes.Buffer
	if err := renderPrettyProgress(&stdout, &Options{}, "waiting", 0.5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"waiting", "50%"} {
		if !strings.Contains(output, want) {
			t.Fatalf("pretty progress output missing %q:\n%s", want, output)
		}
	}
	if containsANSI(output) {
		t.Fatalf("expected non-TTY pretty progress output without ANSI escapes, got:\n%q", output)
	}
}

func TestPrettyColorHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	previousTTY := tableWriterIsTerminal
	tableWriterIsTerminal = func(stdout io.Writer) bool {
		return true
	}
	defer func() {
		tableWriterIsTerminal = previousTTY
	}()

	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"ID", "Name", "Networks"}, []outputRow{
		{
			"ID":       "7dbf33e2-6d96-43b1-961b-ae58925a382c",
			"Name":     "alpha",
			"Networks": "testNet:\n  172.16.86.56",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if containsANSI(stdout.String()) {
		t.Fatalf("expected NO_COLOR pretty output without ANSI escapes, got:\n%q", stdout.String())
	}
}

func TestPrettySemanticColorUsesANSIForTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	previousTTY := tableWriterIsTerminal
	tableWriterIsTerminal = func(stdout io.Writer) bool {
		return true
	}
	defer func() {
		tableWriterIsTerminal = previousTTY
	}()

	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "pretty"}, []string{"ID", "Name", "Networks", "Flavor", "RAM", "Status"}, []outputRow{
		{
			"ID":       "7dbf33e2-6d96-43b1-961b-ae58925a382c",
			"Name":     "rocky",
			"Networks": "os6-lan:\n  172.16.86.56",
			"Flavor":   "m1.small",
			"RAM":      2048,
			"Status":   "ACTIVE",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	if !containsANSI(output) {
		t.Fatalf("expected TTY pretty output to contain ANSI color, got:\n%s", output)
	}
	for _, want := range []string{"7dbf33e2", "ae58925a382c", "rocky", "172.16.86.56", "m1.small", "2048", "ACTIVE"} {
		if !strings.Contains(output, want) {
			t.Fatalf("colored pretty output missing raw text %q:\n%s", want, output)
		}
	}
}

func TestPrettySemanticColorizersStyleKnownValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "ID", value: "7dbf33e2-6d96-43b1-961b-ae58925a382c"},
		{name: "Name", value: "rocky"},
		{name: "Networks", value: "172.16.86.56"},
		{name: "Flavor", value: "m1.small"},
		{name: "Image", value: "N/A (booted from volume)"},
		{name: "RAM", value: "2048"},
		{name: "Status", value: "ACTIVE"},
	}
	for _, tc := range cases {
		colored := prettyColorizeByName(tc.name, tc.value)
		if !containsANSI(colored) {
			t.Fatalf("expected %s value %q to be colorized, got %q", tc.name, tc.value, colored)
		}
		if tc.name == "ID" {
			for _, segment := range strings.Split(tc.value, "-") {
				if !strings.Contains(colored, segment) {
					t.Fatalf("expected colorized UUID to preserve segment %q, got %q", segment, colored)
				}
			}
			continue
		}
		if tc.name == "Image" {
			for _, want := range []string{"N/A", "(booted from volume)"} {
				if !strings.Contains(colored, want) {
					t.Fatalf("expected colorized image value to preserve %q, got %q", want, colored)
				}
			}
			continue
		}
		if !strings.Contains(colored, tc.value) {
			t.Fatalf("expected colorized value to preserve %q, got %q", tc.value, colored)
		}
	}
}

func TestPrettyUUIDColorLeavesHyphensPlain(t *testing.T) {
	uuid := "7dbf33e2-6d96-43b1-961b-ae58925a382c"
	colored := prettyColorizeByName("ID", uuid)
	parts := strings.Split(colored, "-")
	if len(parts) != 5 {
		t.Fatalf("expected colored UUID to keep four plain hyphens, got %q", colored)
	}
	for index, part := range parts {
		if !containsANSI(part) {
			t.Fatalf("expected UUID segment %d to be colored, got %q from %q", index, part, colored)
		}
	}
}

func TestPrettyImageNAIsColored(t *testing.T) {
	colored := prettyColorizeByName("Image", "N/A (booted from volume)")
	want := prettyNAStyle.Render("N/A") + " (booted from volume)"
	if colored != want {
		t.Fatalf("expected only image N/A token to be colored, got %q want %q", colored, want)
	}
}

func TestPrettyWrappedImageNAContinuationStaysNeutral(t *testing.T) {
	rows := prettyWrapRows(
		[]table.Row{{"N/A (booted from volume)"}},
		[]table.Column{{Title: "Image", Width: 8}},
		prettyListCellColorizer([]string{"Image"}),
		prettyListCellContext([]string{"Image"}),
	)
	if len(rows) != 4 {
		t.Fatalf("expected wrapped N/A image value rows, got %#v", rows)
	}
	if !strings.Contains(rows[0][0], prettyNAStyle.Render("N/A")) {
		t.Fatalf("expected first wrapped image row to color N/A, got %#v", rows)
	}
	for index, row := range rows[1:] {
		if containsANSI(row[0]) {
			t.Fatalf("expected wrapped N/A explanation continuation row %d to stay neutral, got %#v", index+1, rows)
		}
	}
}

func TestPrettyPaletteColorsDomainValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "Volume ID", value: "52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b", want: prettyColorizeUUID("52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b")},
		{name: "Server ID", value: "1c77f920-e72d-45d0-8198-5a4a11722214", want: prettyColorizeUUID("1c77f920-e72d-45d0-8198-5a4a11722214")},
		{name: "Subnet ID", value: "f44a40a0-f159-4923-8d79-55d101533f67", want: prettyColorizeUUID("f44a40a0-f159-4923-8d79-55d101533f67")},
		{name: "Network ID", value: "6f911ff3-5259-4dbe-a29d-684a11bfd828", want: prettyColorizeUUID("6f911ff3-5259-4dbe-a29d-684a11bfd828")},
		{name: "Image ID", value: "da8beb8e-7301-49a3-b952-ebde206f9a0b", want: prettyColorizeUUID("da8beb8e-7301-49a3-b952-ebde206f9a0b")},
		{name: "Flavor ID", value: "56e015e0-79f4-4962-82f0-8f8a1b2c771f", want: prettyColorizeUUID("56e015e0-79f4-4962-82f0-8f8a1b2c771f")},
		{name: "Image", value: "rocky9", want: prettyImageStyle.Render("rocky9")},
		{name: "Flavor", value: "m1.small", want: prettyFlavorStyle.Render("m1.small")},
		{name: "device", value: "/dev/vda", want: prettyDeviceStyle.Render("/dev/vda")},
		{name: "created_at", value: "2026-05-04T19:10:59.000000", want: prettyTimestampStyle.Render("2026-05-04T19:10:59.000000")},
		{name: "Status", value: "in-use", want: prettyBooleanTrueStyle.Render("in-use")},
	}
	for _, tc := range cases {
		if got := prettyColorizeByName(tc.name, tc.value); got != tc.want {
			t.Fatalf("expected %s value %q to use palette color, got %q want %q", tc.name, tc.value, got, tc.want)
		}
	}
}

func TestPrettyValueStylesAreNotBoldExceptLabels(t *testing.T) {
	values := []string{
		prettyBooleanFalseStyle.Render("False"),
		prettyBooleanTrueStyle.Render("True"),
		prettyDeviceStyle.Render("/dev/vda"),
		prettyErrorStyle.Render("ERROR"),
		prettyFlavorStyle.Render("m1.small"),
		prettyImageStyle.Render("rocky9"),
		prettyIPStyle.Render("172.16.86.56"),
		prettyNAStyle.Render("N/A"),
		prettyNameStyle.Render("rocky"),
		prettyNumberStyle.Render("2048"),
		prettyTimestampStyle.Render("2026-05-04T19:10:59.000000"),
		prettyUUIDStyle.Render("52c4cf8d"),
		prettyVolumeStyle.Render("Rocky10-Boot"),
		prettyWarningStyle.Render("BUILD"),
	}
	for _, value := range values {
		if containsANSIBold(value) {
			t.Fatalf("expected pretty value style to avoid bold/bright SGR, got %q", value)
		}
	}
	label := prettyLabelStyle.Render("id:")
	if !containsANSIBold(label) {
		t.Fatalf("expected pretty label style to keep bold/bright SGR, got %q", label)
	}
}

func TestPrettyShowFieldColumnUsesLabelStyle(t *testing.T) {
	colorizer := prettyShowCellColorizer([]outputField{{Name: "id", Value: "server-1"}})
	want := prettyLabelStyle.Render("id")
	if got := colorizer(0, 0, "id"); got != want {
		t.Fatalf("expected pretty show field column to use label style, got %q want %q", got, want)
	}
}

func TestPrettyVolumeSemanticRoleStillColorsIDsAsGenericUUID(t *testing.T) {
	colorizer := prettyListCellColorizer([]string{"ID"}, [][]string{{"volume"}})
	uuid := "52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b"
	want := prettyColorizeUUID(uuid)
	if got := colorizer(0, 0, uuid); got != want {
		t.Fatalf("expected volume role to color ID as generic UUID, got %q want %q", got, want)
	}
}

func TestPrettyVolumeListIDUsesGenericUUIDRole(t *testing.T) {
	uuid := "52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b"
	row := volumeListRow(volumes.Volume{ID: uuid, Name: "Rocky10-Boot"})
	if role := prettySemanticRole(row["ID"]); role != "" {
		t.Fatalf("expected volume list ID to use generic UUID styling, got semantic role %q", role)
	}
	if role := prettySemanticRole(row["Name"]); role != "" {
		t.Fatalf("expected volume list name to use generic name styling, got semantic role %q", role)
	}
	colorizer := prettyListCellColorizer([]string{"ID"})
	want := prettyColorizeUUID(uuid)
	if got := colorizer(0, 0, uuid); got != want {
		t.Fatalf("expected volume list ID colorizer to use generic UUID style, got %q want %q", got, want)
	}
	wrapped := prettyWrapRows(
		[]table.Row{{"52c4cf8d-4ef3-4479-"}},
		[]table.Column{{Title: "ID", Width: 8}},
		colorizer,
		prettyListCellContext([]string{"ID"}),
	)
	if len(wrapped) != 3 {
		t.Fatalf("expected wrapped UUID fragment rows, got %#v", wrapped)
	}
	for rowIndex, row := range wrapped {
		if !containsANSI(row[0]) {
			t.Fatalf("expected wrapped volume list ID row %d to use generic UUID fragment color, got %#v", rowIndex, wrapped)
		}
	}
}

func TestPrettyVolumeListNameUsesGenericNameColor(t *testing.T) {
	colorizer := prettyListCellColorizer([]string{"Name"})
	want := prettyNameStyle.Render("Rocky10-Boot")
	if got := colorizer(0, 0, "Rocky10-Boot"); got != want {
		t.Fatalf("expected volume list name to use the generic name color, got %q want %q", got, want)
	}
}

func TestPrettyVolumeUUIDFragmentsKeepHyphensPlain(t *testing.T) {
	colored := prettyColorizeVolume("52c4cf8d-4ef3-4479-")
	parts := strings.Split(colored, "-")
	if len(parts) != 4 {
		t.Fatalf("expected volume UUID fragment to keep plain hyphen separators, got %q", colored)
	}
	for index, part := range parts[:len(parts)-1] {
		if !containsANSI(part) {
			t.Fatalf("expected volume UUID fragment segment %d to be colored, got %q from %q", index, part, colored)
		}
	}
}

func TestPrettyVolumeUUIDLeavesHyphensPlain(t *testing.T) {
	uuid := "52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b"
	colored := prettyColorizeByName("Volume ID", uuid)
	want := prettyColorizeUUID(uuid)
	if colored != want {
		t.Fatalf("expected volume ID to use generic UUID style, got %q want %q", colored, want)
	}
	parts := strings.Split(colored, "-")
	if len(parts) != 5 {
		t.Fatalf("expected colored volume UUID to keep four plain hyphens, got %q", colored)
	}
	for index, part := range parts {
		if !containsANSI(part) {
			t.Fatalf("expected volume UUID segment %d to be colored, got %q from %q", index, part, colored)
		}
	}
}

func TestPrettyAttachedToIDUsesGenericUUIDStyle(t *testing.T) {
	uuid := "52c4cf8d-4ef3-4479-b7b9-ffb814f9dd5b"
	colored := prettyColorizeByName("Attached to", "id: "+uuid)
	want := prettyLabelStyle.Render("id:") + prettyColorizeID(" "+uuid)
	if colored != want {
		t.Fatalf("expected attached-to id label to use generic UUID style, got %q want %q", colored, want)
	}
}

func TestPrettyWrappedAttachedToIDUsesGenericUUIDStyle(t *testing.T) {
	rows := prettyWrapRows(
		[]table.Row{{"  id: 52c4cf8d"}},
		[]table.Column{{Title: "Attached to", Width: 8}},
		prettyListCellColorizer([]string{"Attached to"}),
		prettyListCellContext([]string{"Attached to"}),
	)
	if len(rows) != 2 {
		t.Fatalf("expected wrapped label and value rows, got %#v", rows)
	}
	wantLabel := "  " + prettyLabelStyle.Render("id:")
	if rows[0][0] != wantLabel {
		t.Fatalf("expected wrapped id label row to be bright, got %q want %q", rows[0][0], wantLabel)
	}
	wantValue := prettyUUIDStyle.Render("52c4cf8d")
	if rows[1][0] != wantValue {
		t.Fatalf("expected wrapped attached-to id value to use generic UUID color, got %q want %q", rows[1][0], wantValue)
	}
}

func TestPrettyDeviceLabelUsesDeviceColor(t *testing.T) {
	colored := prettyColorizeByName("Attached to", "device: /dev/vda")
	want := prettyLabelStyle.Render("device:") + prettyDeviceStyle.Render(" /dev/vda")
	if colored != want {
		t.Fatalf("expected device label value to use device color, got %q want %q", colored, want)
	}
}

func TestPrettyHostnamesUseIPColor(t *testing.T) {
	colored := prettyColorizeByName("host_name", "dell6.crandall.haus")
	want := prettyIPStyle.Render("dell6.crandall.haus")
	if colored != want {
		t.Fatalf("expected hostname to use IP color, got %q want %q", colored, want)
	}
}

func TestPrettyLabelPrefixIsBrightWhite(t *testing.T) {
	colored := prettyColorizeByName("Networks", "  testNet: 172.16.86.56")
	wantPrefix := "  " + prettyLabelStyle.Render("testNet:")
	if !strings.HasPrefix(colored, wantPrefix) {
		t.Fatalf("expected pretty label prefix to be bright white, got %q want prefix %q", colored, wantPrefix)
	}
	wantValue := " " + prettyIPStyle.Render("172.16.86.56")
	if !strings.Contains(colored, wantValue) {
		t.Fatalf("expected pretty label value to keep semantic color, got %q want value %q", colored, wantValue)
	}
}

func TestPrettyWrappedLabelContinuationValueKeepsSemanticColor(t *testing.T) {
	rows := prettyWrapRows(
		[]table.Row{{"  attachment_id: 1f7222ef"}},
		[]table.Column{{Title: "Attached to", Width: 16}},
		prettyListCellColorizer([]string{"Attached to"}),
		prettyListCellContext([]string{"Attached to"}),
	)
	if len(rows) != 2 {
		t.Fatalf("expected wrapped label and value rows, got %#v", rows)
	}
	wantLabel := "  " + prettyLabelStyle.Render("attachment_id:")
	if rows[0][0] != wantLabel {
		t.Fatalf("expected wrapped label row to be bright, got %q want %q", rows[0][0], wantLabel)
	}
	wantValue := prettyUUIDStyle.Render("1f7222ef")
	if rows[1][0] != wantValue {
		t.Fatalf("expected wrapped label value row to keep semantic color, got %q want %q", rows[1][0], wantValue)
	}
}

func TestPrettyWrappedInlineLabelContinuationValueKeepsSemanticColor(t *testing.T) {
	rows := prettyWrapRows(
		[]table.Row{{"  attachment_id: 1f7222ef-c386-4523-9788"}},
		[]table.Column{{Title: "Attached to", Width: 28}},
		prettyListCellColorizer([]string{"Attached to"}),
		prettyListCellContext([]string{"Attached to"}),
	)
	if len(rows) != 2 {
		t.Fatalf("expected wrapped inline label value rows, got %#v", rows)
	}
	if !strings.Contains(rows[0][0], prettyUUIDStyle.Render("1f7222ef")) {
		t.Fatalf("expected first wrapped inline value row to color UUID fragment, got %#v", rows)
	}
	if !strings.Contains(rows[1][0], prettyUUIDStyle.Render("c386")) {
		t.Fatalf("expected continued inline value row to keep UUID color, got %#v", rows)
	}
}

func TestDefaultOutputStillUsesOSCCompatibleTableRenderer(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: defaultOutputFormat}, []string{"ID", "Name"}, []outputRow{
		{"ID": "server-1", "Name": "alpha"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	if !strings.HasPrefix(output, "+") || !strings.Contains(output, "| ID       | Name  |") {
		t.Fatalf("expected default output to keep OSC-compatible table rendering, got:\n%s", output)
	}
	if containsANSI(output) {
		t.Fatalf("expected default table output without ANSI escapes, got:\n%q", output)
	}
}

func TestDefaultListRightAlignsNumericColumns(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: defaultOutputFormat}, []string{"ID", "Name", "RAM", "Disk"}, []outputRow{
		{"ID": "0", "Name": "m1.tiny", "RAM": 512, "Disk": 10},
		{"ID": "1", "Name": "m1.small", "RAM": 1024, "Disk": 10},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"| ID | Name     |  RAM | Disk |",
		"| 0  | m1.tiny  |  512 |   10 |",
		"| 1  | m1.small | 1024 |   10 |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("default list output missing numeric alignment %q:\n%s", want, output)
		}
	}
}

func TestDefaultShowFormatsOSCEmptyAndNoneValues(t *testing.T) {
	type projectOption string
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: defaultOutputFormat}, []outputField{
		{Name: "nil_value", Value: nil},
		{Name: "empty_slice", Value: []string{}},
		{Name: "empty_typed_map", Value: map[projectOption]any{}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"| nil_value       | None  |",
		"| empty_slice     | []    |",
		"| empty_typed_map | {}    |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("default show output missing %q:\n%s", want, output)
		}
	}
}

func TestJSONOutputDoesNotEscapeHTML(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: "json"}, []string{"Properties"}, []outputRow{
		{"Properties": map[string]any{"consistent_group_snapshot_enabled": "<is> True"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.Contains(stdout.String(), `\u003c`) || strings.Contains(stdout.String(), `\u003e`) {
		t.Fatalf("expected JSON output to preserve angle brackets, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"<is> True"`) {
		t.Fatalf("expected JSON output to contain unescaped value, got:\n%s", stdout.String())
	}
}

func TestOrderedJSONTopObjectPreservesNestedOrder(t *testing.T) {
	item, err := orderedJSONTopObject([]byte(`{"properties":{"z":{"type":"string","title":"Z"},"a":{"default":"1","minimum":1}},"required":[]}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	properties, ok := orderedMapValueAsObject(item, "properties")
	if !ok {
		t.Fatalf("expected properties object")
	}
	if got, want := valueString(properties), `{"z":{"type":"string","title":"Z"},"a":{"default":"1","minimum":1}}`; got != want {
		t.Fatalf("ordered JSON mismatch: got %q want %q", got, want)
	}
	encodedRequired, err := json.Marshal(orderedMapValueOrNil(item, "required"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got, want := string(encodedRequired), `[]`; got != want {
		t.Fatalf("empty array mismatch: got %q want %q", got, want)
	}
}

func TestOSCHTTPExceptionFormatsOpenStackFault(t *testing.T) {
	err := oscHTTPException(gophercloud.ErrUnexpectedResponseCode{
		URL:    "http://example.test/v2.1/os-agents",
		Method: "GET",
		Actual: 410,
		Body:   []byte(`{"computeFault":{"code":410,"message":"This resource is no longer available. No forwarding address is given."}}`),
	})
	if got, want := err.Error(), "HttpException: 410: Client Error for url: http://example.test/v2.1/os-agents, This resource is no longer available. No forwarding address is given."; got != want {
		t.Fatalf("HTTP exception mismatch: got %q want %q", got, want)
	}
}

func TestOSCResourceNotFoundFormatsSDKLookupError(t *testing.T) {
	err := oscResourceNotFoundError(gophercloud.ErrUnexpectedResponseCode{
		URL:    "http://example.test/v2.1/os-console-auth-tokens/bad-token",
		Method: "GET",
		Actual: 404,
		Body:   []byte(`{"itemNotFound":{"code":404,"message":"Token not found"}}`),
	}, "ConsoleAuthToken", "bad-token")
	if got, want := err.Error(), "No ConsoleAuthToken found for bad-token: Client Error for url: http://example.test/v2.1/os-console-auth-tokens/bad-token, Token not found"; got != want {
		t.Fatalf("resource not found mismatch: got %q want %q", got, want)
	}
}

func TestOpenStackFaultMessageFormatsFlatGlanceError(t *testing.T) {
	body := []byte(`{"message":"Caching via API is not supported at this site.<br /><br />\n\n\n","code":"404 Not Found","title":"Not Found"}`)
	if got, want := openStackFaultMessage(body), "404 Not Found: Caching via API is not supported at this site."; got != want {
		t.Fatalf("fault message mismatch: got %q want %q", got, want)
	}
}

func TestServerMigrationListColumnsMatchMicroversion(t *testing.T) {
	columns, keys := serverMigrationListColumns("2.80", &Options{CommandFlags: map[string]string{
		"project": "admin",
		"user":    "admin",
	}})
	wantColumns := []string{"Id", "UUID", "Source Node", "Dest Node", "Source Compute", "Dest Compute", "Dest Host", "Status", "Server UUID", "Old Flavor", "New Flavor", "Type", "Project", "User", "Created At", "Updated At"}
	wantKeys := []string{"id", "uuid", "source_node", "dest_node", "source_compute", "dest_compute", "dest_host", "status", "instance_uuid", "old_instance_type_id", "new_instance_type_id", "migration_type", "project_id", "user_id", "created_at", "updated_at"}
	if strings.Join(columns, "|") != strings.Join(wantColumns, "|") {
		t.Fatalf("columns mismatch: got %#v want %#v", columns, wantColumns)
	}
	if strings.Join(keys, "|") != strings.Join(wantKeys, "|") {
		t.Fatalf("keys mismatch: got %#v want %#v", keys, wantKeys)
	}
}

func TestUnixSecondsISOFormatsUTC(t *testing.T) {
	if got, want := unixSecondsISO(float64(1700000000)), "2023-11-14T22:13:20"; got != want {
		t.Fatalf("timestamp mismatch: got %v want %v", got, want)
	}
}

func TestFloatingIPPortForwardingPortRangeValidation(t *testing.T) {
	cases := []struct {
		name     string
		internal string
		external string
		want     map[string]any
		wantErr  string
	}{
		{
			name:     "single ports",
			internal: "22",
			external: "2222",
			want:     map[string]any{"internal_port": 22, "external_port": 2222},
		},
		{
			name:     "one to many",
			internal: "80",
			external: "8080:8082",
			want:     map[string]any{"internal_port": 80, "external_port_range": "8080:8082"},
		},
		{
			name:     "many to many",
			internal: "8000:8002",
			external: "9000:9002",
			want:     map[string]any{"internal_port_range": "8000:8002", "external_port_range": "9000:9002"},
		},
		{
			name:     "mismatched ranges",
			internal: "8000:8003",
			external: "9000:9002",
			wantErr:  "The relation between internal and external ports does not match the pattern 1:N and N:N",
		},
		{
			name:     "descending range",
			internal: "8002:8000",
			external: "9000:9002",
			wantErr:  "The last number in port range must be greater or equal to the first",
		},
		{
			name:     "out of range",
			internal: "0",
			external: "9000",
			wantErr:  "The port number range is <1-65535>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]any{}
			err := floatingIPPortForwardingApplyPortRanges(values, tc.internal, tc.external)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error mismatch: got %v want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if encoded, _ := json.Marshal(values); string(encoded) != mustJSON(tc.want) {
				t.Fatalf("values mismatch: got %s want %s", encoded, mustJSON(tc.want))
			}
		})
	}
}

func TestNetworkQoSRuleParameterValidation(t *testing.T) {
	cases := []struct {
		name    string
		flags   map[string]string
		want    map[string]any
		wantErr string
	}{
		{
			name:  "bandwidth limit",
			flags: map[string]string{"type": "bandwidth-limit", "max-kbps": "1000", "max-burst-kbits": "80", "egress": "true"},
			want:  map[string]any{"max_kbps": 1000, "max_burst_kbps": 80, "direction": "egress"},
		},
		{
			name:    "minimum bandwidth requires direction",
			flags:   map[string]string{"type": "minimum-bandwidth", "min-kbps": "1000"},
			wantErr: "\"Create\" rule command for type \"minimum-bandwidth\" requires arguments: direction, min_kbps",
		},
		{
			name:  "minimum packet allows any direction",
			flags: map[string]string{"type": "minimum-packet-rate", "min-kpps": "25", "any": "true"},
			want:  map[string]any{"min_kpps": 25, "direction": "any"},
		},
		{
			name:    "any direction is minimum packet only",
			flags:   map[string]string{"type": "bandwidth-limit", "max-kbps": "1000", "any": "true"},
			wantErr: "Direction \"any\" can only be used with minimum-packet-rate rule type",
		},
		{
			name:    "reject other rule mandatory parameter",
			flags:   map[string]string{"type": "dscp-marking", "dscp-mark": "8", "max-kbps": "1000"},
			wantErr: "Rule type \"dscp-marking\" only requires arguments: dscp_mark",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &Options{CommandFlags: tc.flags}
			ruleType, values, err := networkQoSRuleCreateValues(opts)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error mismatch: got %v want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if ruleType != tc.flags["type"] {
				t.Fatalf("rule type mismatch: got %q want %q", ruleType, tc.flags["type"])
			}
			if encoded, _ := json.Marshal(values); string(encoded) != mustJSON(tc.want) {
				t.Fatalf("values mismatch: got %s want %s", encoded, mustJSON(tc.want))
			}
		})
	}
}

func TestNetworkQoSRuleRawFieldsMatchOSCOrder(t *testing.T) {
	fields := networkQoSRuleRawFields(map[string]any{
		"id":             "rule-id",
		"max_kbps":       float64(1000),
		"custom":         "value",
		"qos_policy_id":  "policy-id",
		"max_burst_kbps": nil,
	}, networkQoSRuleBandwidthLimit)
	var names []string
	for _, field := range fields {
		names = append(names, field.Name)
	}
	want := []string{"custom", "direction", "id", "max_burst_kbps", "max_kbps"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Fatalf("field order mismatch: got %#v want %#v", names, want)
	}
}

func TestNetworkSegmentSetValues(t *testing.T) {
	opts := &Options{
		CommandFlags: map[string]string{
			"description": "updated",
			"name":        "segment-name",
		},
		CommandFlagList: map[string][]string{
			"extra-property": {"type=int,name=custom,value=7"},
		},
	}
	values, err := networkSegmentSetValues(opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]any{"description": "updated", "name": "segment-name", "custom": 7}
	if encoded, _ := json.Marshal(values); string(encoded) != mustJSON(want) {
		t.Fatalf("values mismatch: got %s want %s", encoded, mustJSON(want))
	}
}

func TestNetworkSegmentTypeValidation(t *testing.T) {
	for _, networkType := range []string{"flat", "geneve", "gre", "local", "vlan", "vxlan"} {
		if !networkSegmentTypeValid(networkType) {
			t.Fatalf("expected %q to be valid", networkType)
		}
	}
	if networkSegmentTypeValid("invalid") {
		t.Fatal("expected invalid network segment type to be rejected")
	}
}

func TestNetworkTrunkSubportMap(t *testing.T) {
	subport, err := networkTrunkSubportMap(map[string]string{
		"segmentation-id":   "42",
		"segmentation-type": "vlan",
	}, "port-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]any{"port_id": "port-id", "segmentation_id": 42, "segmentation_type": "vlan"}
	if encoded, _ := json.Marshal(subport); string(encoded) != mustJSON(want) {
		t.Fatalf("subport mismatch: got %s want %s", encoded, mustJSON(want))
	}
	if _, err := networkTrunkSubportMap(map[string]string{"segmentation-id": "not-int"}, "port-id"); err == nil || err.Error() != "Segmentation-id \"not-int\" is not an integer" {
		t.Fatalf("error mismatch: got %v", err)
	}
}

func TestNetworkTrunkSetValues(t *testing.T) {
	values, err := networkTrunkSetValues(&Options{CommandFlags: map[string]string{
		"name":        "trunk-name",
		"description": "updated",
		"disable":     "true",
	}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]any{"name": "trunk-name", "description": "updated", "admin_state_up": false}
	if encoded, _ := json.Marshal(values); string(encoded) != mustJSON(want) {
		t.Fatalf("values mismatch: got %s want %s", encoded, mustJSON(want))
	}
	_, err = networkTrunkSetValues(&Options{CommandFlags: map[string]string{"enable": "true", "disable": "true"}})
	if err == nil || err.Error() != "argument --disable: not allowed with argument --enable" {
		t.Fatalf("error mismatch: got %v", err)
	}
}

func TestTokenIssueUsesInjectedIssuer(t *testing.T) {
	previous := issueToken
	issueToken = func(ctx context.Context, opts *Options) (tokenIssueRow, error) {
		return tokenIssueRow{
			Expires:   "2026-05-03T00:00:00+0000",
			ID:        "token-id",
			ProjectID: "project-id",
			UserID:    "user-id",
		}, nil
	}
	defer func() {
		issueToken = previous
	}()

	stdout, stderr, err := executeForTest("token", "issue", "-f", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{`"expires": "2026-05-03T00:00:00+0000"`, `"id": "token-id"`, `"project_id": "project-id"`, `"user_id": "user-id"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("token issue output missing %q:\n%s", want, stdout)
		}
	}
}

func maxOutputLineLength(output string) int {
	longest := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		longest = max(longest, len([]rune(line)))
	}
	return longest
}

func maxVisibleOutputLineLength(output string) int {
	longest := 0
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		longest = max(longest, displayWidth(stripANSI(line)))
	}
	return longest
}

func stripANSI(output string) string {
	var builder strings.Builder
	for i := 0; i < len(output); {
		if output[i] != '\x1b' {
			builder.WriteByte(output[i])
			i++
			continue
		}
		i++
		if i >= len(output) {
			break
		}
		switch output[i] {
		case '[':
			i++
			for i < len(output) {
				b := output[i]
				i++
				if b >= '@' && b <= '~' {
					break
				}
			}
		case ']':
			i++
			for i < len(output) {
				if output[i] == '\a' {
					i++
					break
				}
				if output[i] == '\x1b' && i+1 < len(output) && output[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return builder.String()
}

func containsANSI(output string) bool {
	return strings.Contains(output, "\x1b[") || strings.Contains(output, "\x1b]")
}

func containsANSIBold(output string) bool {
	parts := strings.Split(output, "\x1b[")
	for _, part := range parts[1:] {
		end := strings.IndexByte(part, 'm')
		if end < 0 {
			continue
		}
		for _, code := range strings.Split(part[:end], ";") {
			if code == "1" {
				return true
			}
		}
	}
	return false
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
