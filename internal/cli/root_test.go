package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crandallnet/golang-osc/compat/osc"
)

func executeForTest(args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Execute(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
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
func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
func quotaResourceNames(rows []outputRow) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, valueString(row["Resource"]))
	}
	return names
}
func TestAggregateQuotaShowRowsMatchesOSCServiceMergeOrder(t *testing.T) {
	computeRows := []outputRow{
		{"Resource": "cores", "Limit": 20},
		{"Resource": "instances", "Limit": 10},
		{"Resource": "ram", "Limit": 51200},
		{"Resource": "fixed_ips", "Limit": nil},
		{"Resource": "floating_ips", "Limit": nil},
		{"Resource": "networks", "Limit": nil},
		{"Resource": "security_group_rules", "Limit": nil},
		{"Resource": "security_groups", "Limit": nil},
		{"Resource": "injected-file-size", "Limit": 10240},
		{"Resource": "injected-path-size", "Limit": 255},
		{"Resource": "injected-files", "Limit": 5},
		{"Resource": "key-pairs", "Limit": 100},
		{"Resource": "properties", "Limit": 128},
		{"Resource": "server-group-members", "Limit": 10},
		{"Resource": "server-groups", "Limit": 10},
	}
	volumeRows := []outputRow{
		{"Resource": "volumes", "Limit": 10},
		{"Resource": "snapshots", "Limit": 10},
		{"Resource": "gigabytes", "Limit": 1000},
		{"Resource": "backups", "Limit": 10},
		{"Resource": "volumes_LVM", "Limit": -1},
		{"Resource": "gigabytes_LVM", "Limit": -1},
		{"Resource": "snapshots_LVM", "Limit": -1},
		{"Resource": "groups", "Limit": 10},
		{"Resource": "backup-gigabytes", "Limit": 1000},
		{"Resource": "per-volume-gigabytes", "Limit": -1},
	}
	networkRows := []outputRow{
		{"Resource": "check_limit", "Limit": nil},
		{"Resource": "networks", "Limit": 100},
		{"Resource": "ports", "Limit": 500},
		{"Resource": "floating-ips", "Limit": 50},
		{"Resource": "secgroup-rules", "Limit": 100},
		{"Resource": "secgroups", "Limit": 10},
	}

	rows := aggregateQuotaShowRows(computeRows, volumeRows, networkRows)
	got := quotaResourceNames(rows)
	want := []string{
		"cores",
		"instances",
		"ram",
		"fixed_ips",
		"networks",
		"volumes",
		"snapshots",
		"gigabytes",
		"backups",
		"volumes_LVM",
		"gigabytes_LVM",
		"snapshots_LVM",
		"groups",
		"check_limit",
		"ports",
		"injected-file-size",
		"injected-path-size",
		"injected-files",
		"key-pairs",
		"properties",
		"server-group-members",
		"server-groups",
		"floating-ips",
		"secgroup-rules",
		"secgroups",
		"backup-gigabytes",
		"per-volume-gigabytes",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("quota aggregate order mismatch: got %#v want %#v", got, want)
	}
	if rows[4]["Limit"] != 100 {
		t.Fatalf("expected Neutron networks limit to replace Nova placeholder, got %#v", rows[4]["Limit"])
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
func TestCommandListJSONMarksCoreReadsImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.compute.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{`"server list"`, `"server show"`, `"flavor create"`, `"flavor delete"`, `"flavor list"`, `"flavor set"`, `"flavor show"`, `"flavor unset"`} {
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
func TestCommandListJSONMarksServiceStubs(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.compute.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, `"server ssh (Not Implemented Yet)"`) {
		t.Fatalf("expected server ssh to be marked implemented by nova-extras, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"server ssh"`) {
		t.Fatalf("expected server ssh to appear in command list, got:\n%s", stdout)
	}
	if strings.Contains(stdout, `"server start (Not Implemented Yet)"`) {
		t.Fatalf("expected server start to be marked implemented, got:\n%s", stdout)
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
func TestCommandListPrettyUsesPrettyRenderer(t *testing.T) {
	stdout, stderr, err := executeForTest("--pretty", "command", "list", "--group", "openstack.cli")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"Command Group", "Command", "Subcommands", "openstack.cli", "command", "module", "list"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("pretty command list output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "+---") {
		t.Fatalf("expected command list --pretty to avoid default table borders, got:\n%s", stdout)
	}
}

func TestSelectedPrettyTheme(t *testing.T) {
	theme, ok := selectedPrettyTheme(false, "pacman", "default")
	if ok || theme != "" {
		t.Fatalf("expected theme selection to be ignored when pretty is disabled")
	}

	theme, ok = selectedPrettyTheme(true, "", "")
	if ok || theme != "" {
		t.Fatalf("expected no theme selection when no flag/env is set")
	}

	theme, ok = selectedPrettyTheme(true, "", "pacman")
	if !ok || theme != "pacman" {
		t.Fatalf("expected OS_THEME to select pacman, got %q (ok=%v)", theme, ok)
	}

	theme, ok = selectedPrettyTheme(true, "default", "pacman")
	if !ok || theme != "default" {
		t.Fatalf("expected --theme to override OS_THEME, got %q (ok=%v)", theme, ok)
	}
}

func TestResolveCloudsYAMLPath(t *testing.T) {
	t.Run("os client config file takes precedence", func(t *testing.T) {
		expected := "/tmp/custom/clouds.yaml"
		t.Setenv("OS_CLIENT_CONFIG_FILE", expected)
		got, err := resolveCloudsYAMLPath()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("cwd clouds yaml is selected first", func(t *testing.T) {
		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "clouds.yaml"), []byte("clouds: {}\n"), 0o600); err != nil {
			t.Fatalf("write cwd clouds.yaml: %v", err)
		}
		t.Setenv("OS_CLIENT_CONFIG_FILE", "")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(originalWD)
		})

		got, err := resolveCloudsYAMLPath()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		want := filepath.Join(tmp, "clouds.yaml")
		if filepath.Clean(got) != filepath.Clean(want) {
			gotResolved, _ := filepath.EvalSymlinks(got)
			wantResolved, _ := filepath.EvalSymlinks(want)
			if filepath.Clean(gotResolved) != filepath.Clean(wantResolved) {
				t.Fatalf("expected %q, got %q", want, got)
			}
		}
	})

	t.Run("xdg fallback is selected when cwd has no clouds", func(t *testing.T) {
		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		tmpWD := t.TempDir()
		tmpXDG := t.TempDir()
		xdgClouds := filepath.Join(tmpXDG, "openstack", "clouds.yaml")
		if err := os.MkdirAll(filepath.Dir(xdgClouds), 0o755); err != nil {
			t.Fatalf("mkdir xdg: %v", err)
		}
		if err := os.WriteFile(xdgClouds, []byte("clouds: {}\n"), 0o600); err != nil {
			t.Fatalf("write xdg clouds.yaml: %v", err)
		}
		t.Setenv("OS_CLIENT_CONFIG_FILE", "")
		t.Setenv("XDG_CONFIG_HOME", tmpXDG)
		if err := os.Chdir(tmpWD); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(originalWD)
		})

		got, err := resolveCloudsYAMLPath()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != xdgClouds {
			t.Fatalf("expected %q, got %q", xdgClouds, got)
		}
	})
}

func TestUserThemeLoadsFromCloudsDirectory(t *testing.T) {
	previousTheme := currentTheme
	t.Cleanup(func() {
		currentTheme = previousTheme
	})
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	themeJSON := `{
  "schema_version": 1,
  "themes": {
    "user": {
      "label": "#00ff00",
      "uuid": "7",
      "name": "7",
      "ip_address": "7",
      "timestamp": "7",
      "number": "7",
      "boolean_true": "7",
      "boolean_false": "7",
      "warning": "7",
      "error": "7",
      "device": "7",
      "flavor": "7",
      "image": "7",
      "volume": "7",
      "na": "7",
      "cell_text": "7",
      "border": "7",
      "header": "7",
      "empty_state": "7",
      "progress_bar": ["7"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(tmp, "theme.json"), []byte(themeJSON), 0o600); err != nil {
		t.Fatalf("write theme.json: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")

	cmd := NewRootCommand(io.Discard, io.Discard)
	cmd.SetArgs([]string{"--pretty", "--theme", "user", "module", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if currentTheme == nil || currentTheme.LabelColor != "#00ff00" {
		t.Fatalf("expected user theme label color #00ff00, got %#v", currentTheme)
	}
}

func TestUserThemeSearchSkipsInvalidAndUsesNextLocation(t *testing.T) {
	previousTheme := currentTheme
	t.Cleanup(func() {
		currentTheme = previousTheme
	})
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	tmpXDG := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", tmpXDG)

	// First location (cwd) has invalid JSON and must be skipped.
	if err := os.WriteFile(filepath.Join(tmpWD, "theme.json"), []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write invalid theme.json: %v", err)
	}
	// Second location (XDG) has valid user theme and should be selected.
	xdgThemeDir := filepath.Join(tmpXDG, "openstack")
	if err := os.MkdirAll(xdgThemeDir, 0o755); err != nil {
		t.Fatalf("mkdir xdg theme dir: %v", err)
	}
	valid := `{"schema_version":1,"themes":{"user":{"label":"#123456","uuid":"7","name":"7","ip_address":"7","timestamp":"7","number":"7","boolean_true":"7","boolean_false":"7","warning":"7","error":"7","device":"7","flavor":"7","image":"7","volume":"7","na":"7","cell_text":"7","border":"7","header":"7","empty_state":"7","progress_bar":["7"]}}}`
	if err := os.WriteFile(filepath.Join(xdgThemeDir, "theme.json"), []byte(valid), 0o600); err != nil {
		t.Fatalf("write valid xdg theme.json: %v", err)
	}

	cmd := NewRootCommand(io.Discard, io.Discard)
	cmd.SetArgs([]string{"--pretty", "--theme", "user", "module", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if currentTheme == nil || currentTheme.LabelColor != "#123456" {
		t.Fatalf("expected xdg user theme label color #123456, got %#v", currentTheme)
	}
}

func TestUserThemeMissingReturnsExpectedError(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	tmpXDG := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", tmpXDG)

	cmd := NewRootCommand(io.Discard, io.Discard)
	cmd.SetArgs([]string{"--pretty", "--theme", "user", "module", "list"})
	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected error when no user-defined theme exists")
	}
	if err.Error() != "No user-defined theme found" {
		t.Fatalf("expected exact error %q, got %q", "No user-defined theme found", err.Error())
	}
}

func TestUserThemeIgnoresNonUserThemes(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	nonUser := `{"schema_version":1,"themes":{"default":{"label":"2","uuid":"2","name":"2","ip_address":"2","timestamp":"2","number":"2","boolean_true":"2","boolean_false":"2","warning":"2","error":"2","device":"2","flavor":"2","image":"2","volume":"2","na":"2","cell_text":"2","border":"2","header":"2","empty_state":"2","progress_bar":["2"]}}}`
	if err := os.WriteFile(filepath.Join(tmpWD, "theme.json"), []byte(nonUser), 0o600); err != nil {
		t.Fatalf("write non-user theme.json: %v", err)
	}

	cmd := NewRootCommand(io.Discard, io.Discard)
	cmd.SetArgs([]string{"--pretty", "--theme", "user", "module", "list"})
	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected error when only non-user themes are defined")
	}
	if err.Error() != "No user-defined theme found" {
		t.Fatalf("expected exact error %q, got %q", "No user-defined theme found", err.Error())
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
func TestInvalidFlagUsesOracleUsageAndExitCode(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "--bogus")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("expected exit code 2, got %d", got)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{
		"usage: openstack command list",
		"openstack command list: error: unrecognized arguments: --bogus",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("invalid flag output missing %q:\n%s", want, stderr)
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
func TestLifecycleCleanupRunsLIFOAndRecordsFailures(t *testing.T) {
	run, err := newLifecycleRun("golang-osc-test")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	run.addCleanup("first", func(context.Context) error {
		order = append(order, "first")
		return nil
	})
	run.addCleanup("second", func(context.Context) error {
		order = append(order, "second")
		return errors.New("cleanup failed")
	})
	err = run.cleanup(context.Background())
	if err == nil || !strings.Contains(err.Error(), "second") {
		t.Fatalf("expected cleanup failure to include label, got %v", err)
	}
	if strings.Join(order, ",") != "second,first" {
		t.Fatalf("expected cleanup to run LIFO, got %v", order)
	}
	if len(run.CleanupResults) != 2 || run.CleanupResults[0].Label != "second" || run.CleanupResults[0].Error == "" {
		t.Fatalf("cleanup diagnostics mismatch: %+v", run.CleanupResults)
	}
}
func TestLifecycleDiagnosticsWriteFixtureAndCleanupState(t *testing.T) {
	run, err := newLifecycleRun("golang-osc-test")
	if err != nil {
		t.Fatal(err)
	}
	run.recordFixture("image", "test-image")
	run.CleanupResults = append(run.CleanupResults, lifecycleCleanupResult{Label: "image delete"})
	path := filepath.Join(t.TempDir(), "diagnostics.json")
	if err := run.writeDiagnostics(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"image": "test-image"`, `"label": "image delete"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("diagnostics missing %q:\n%s", want, string(data))
		}
	}
}
func TestLifecycleRunNamesResourcesWithUniquePrefix(t *testing.T) {
	run, err := newLifecycleRun("golang-osc-test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(run.ID, "golang-osc-test-") {
		t.Fatalf("expected lifecycle id to use test prefix, got %q", run.ID)
	}
	name := run.resourceName("network")
	if !strings.HasPrefix(name, run.ID+"-network") {
		t.Fatalf("expected resource name to include lifecycle id, got %q", name)
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
func TestNoGeneratedStubCommandsRemain(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, notImplementedSuffix) {
		t.Fatalf("command list still contains generated stubs:\n%s", stdout)
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
func TestPrettyOSColorTestReportsContrast(t *testing.T) {
	var stdout bytes.Buffer
	if err := RenderPrettyOSColorTest(&stdout, &Options{Format: "pretty"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stripANSI(stdout.String())
	if !strings.Contains(output, "Contrast") {
		t.Fatalf("expected os-test output to include contrast column, got:\n%s", output)
	}
	if !strings.Contains(output, "4.5") && !strings.Contains(output, ":1") {
		t.Fatalf("expected os-test output to include measured contrast ratios, got:\n%s", output)
	}
}
func TestQuotaVolumeTypeFieldsFollowVolumeTypeOrder(t *testing.T) {
	fields := quotaVolumeTypeFields(map[string]any{
		"volumes_LVM":          -1,
		"gigabytes_LVM":        -1,
		"snapshots_LVM":        -1,
		"volumes_debug":        -1,
		"gigabytes_debug":      -1,
		"snapshots_debug":      -1,
		"volumes_alpha":        -1,
		"gigabytes_alpha":      -1,
		"snapshots_alpha":      -1,
		"volumes_incomplete":   -1,
		"gigabytes_incomplete": -1,
	}, []string{"debug", "LVM"})
	got := make([]string, 0, len(fields))
	for _, field := range fields {
		got = append(got, field.Name)
	}
	want := []string{
		"volumes_debug",
		"gigabytes_debug",
		"snapshots_debug",
		"volumes_LVM",
		"gigabytes_LVM",
		"snapshots_LVM",
		"volumes_alpha",
		"gigabytes_alpha",
		"snapshots_alpha",
		"volumes_incomplete",
		"gigabytes_incomplete",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("volume type quota field order mismatch: got %#v want %#v", got, want)
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
	want, ok, err := osc.Help("")
	if err != nil || !ok {
		t.Fatalf("load oracle root help: ok=%t err=%v", ok, err)
	}
	if stdout != want {
		t.Fatalf("root help mismatch")
	}
}
func TestServerImageShowValueRendersOSCDisplayString(t *testing.T) {
	imageID := "da8beb8e-7301-49a3-b952-ebde206f9a0b"
	display := "cirros (" + imageID + ")"
	value := serverImageShowValue(map[string]any{
		"id": imageID,
		"properties": map[string]any{
			"owner_specified.openstack.object": "images/cirros",
		},
	})
	if got := valueString(value); got != display {
		t.Fatalf("server image table display mismatch: got %q want %q", got, display)
	}

	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: "json"}, []outputField{{Name: "image", Value: value}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("expected image output to decode as JSON object, got %v:\n%s", err, stdout.String())
	}
	if got := decoded["image"]; got != display {
		t.Fatalf("server image JSON display mismatch: got %q want %q", got, display)
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
func TestServerRawTaskStateMustClearForWaitCompletion(t *testing.T) {
	if serverRawTaskStateCleared(map[string]any{"OS-EXT-STS:task_state": "shelving_offloading"}) {
		t.Fatalf("expected non-empty task_state to keep server wait open")
	}
	if !serverRawTaskStateCleared(map[string]any{"OS-EXT-STS:task_state": nil}) {
		t.Fatalf("expected nil task_state to complete server wait")
	}
	if !serverRawTaskStateCleared(map[string]any{}) {
		t.Fatalf("expected missing task_state to complete server wait")
	}
}
func TestServerShowTableValuesMatchOSCRepresentations(t *testing.T) {
	groupID := "f3adc01c-13d6-4087-bf56-3f2b9c22fd10"
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "properties",
			value: serverPropertyTableValue(map[string]any{"golang_osc_test": "golang-osc-test"}),
			want:  "golang_osc_test='golang-osc-test'",
		},
		{
			name:  "scheduler_hints",
			value: serverSchedulerHintsTableValue(map[string]any{"group": []any{groupID}}),
			want:  "group=" + groupID,
		},
		{
			name:  "server_groups",
			value: serverPythonListTableValue([]any{groupID}),
			want:  "['" + groupID + "']",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := valueString(tc.value); got != tc.want {
				t.Fatalf("%s table value mismatch: got %q want %q", tc.name, got, tc.want)
			}
		})
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
func TestUnixSecondsISOFormatsUTC(t *testing.T) {
	if got, want := unixSecondsISO(float64(1700000000)), "2023-11-14T22:13:20"; got != want {
		t.Fatalf("timestamp mismatch: got %v want %v", got, want)
	}
}
func TestUnknownCommandUsesFuzzySuggestionsAndExitCode(t *testing.T) {
	stdout, stderr, err := executeForTest("nosuch")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("expected exit code 2, got %d", got)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"openstack: 'nosuch' is not an openstack command. See 'openstack --help'.",
		"Did you mean one of these?",
		"  consumer create",
		"  resource usage show",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("unknown command output missing %q:\n%s", want, stdout)
		}
	}
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
