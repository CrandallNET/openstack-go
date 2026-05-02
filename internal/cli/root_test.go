package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
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
	if got, want := strings.TrimSpace(stdout), notImplementedExitCodeText; got != want {
		t.Fatalf("stub output mismatch: got %q want %q", got, want)
	}
}

func TestGeneratedStubCommandIgnoresCommandFlags(t *testing.T) {
	stdout, stderr, err := executeForTest("server", "list", "--long")
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
	if !strings.Contains(stdout, `"server list (Not Implemented Yet)"`) {
		t.Fatalf("expected server list to be marked unimplemented, got:\n%s", stdout)
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
	if got, want := strings.TrimSpace(stdout), notImplementedExitCodeText; got != want {
		t.Fatalf("stub output mismatch: got %q want %q", got, want)
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
