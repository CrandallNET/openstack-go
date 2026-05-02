package cli

import (
	"bytes"
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
	stdout, stderr, err := executeForTest("module", "list")
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
