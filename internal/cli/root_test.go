package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
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
