package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crandallnet/golang-osc/internal/cli"
	"golang.org/x/crypto/ssh"
)

type lifecycleDiagnostics struct {
	ID              string            `json:"id"`
	Cloud           string            `json:"cloud"`
	Resource        string            `json:"resource"`
	Status          string            `json:"status"`
	StartedAt       string            `json:"started_at"`
	FinishedAt      string            `json:"finished_at"`
	Fixtures        map[string]string `json:"fixtures,omitempty"`
	Steps           []stepResult      `json:"steps"`
	CleanupRequired bool              `json:"cleanup_required"`
	DiagnosticsPath string            `json:"diagnostics_path,omitempty"`
}

type stepResult struct {
	Name           string   `json:"name"`
	Args           []string `json:"args,omitempty"`
	Env            []string `json:"env,omitempty"`
	ExitCode       int      `json:"exit_code,omitempty"`
	Stdout         string   `json:"stdout,omitempty"`
	Stderr         string   `json:"stderr,omitempty"`
	Error          string   `json:"error,omitempty"`
	OracleArgs     []string `json:"oracle_args,omitempty"`
	OracleExitCode int      `json:"oracle_exit_code,omitempty"`
	OracleStdout   string   `json:"oracle_stdout,omitempty"`
	OracleStderr   string   `json:"oracle_stderr,omitempty"`
	OracleError    string   `json:"oracle_error,omitempty"`
	Skipped        bool     `json:"skipped,omitempty"`
	SkipReason     string   `json:"skip_reason,omitempty"`
}

func main() {
	var suite string
	var cloud string
	var prefix string
	var diagnosticsDir string
	var keepSuccess bool

	flag.StringVar(&suite, "suite", "keypair", "lifecycle suite to run: keypair, server, volume, quota, image, network, or object")
	flag.StringVar(&cloud, "cloud", os.Getenv("OS_CLOUD"), "cloud name to test")
	flag.StringVar(&prefix, "prefix", "golang-osc-test", "unique resource prefix")
	flag.StringVar(&diagnosticsDir, "diagnostics-dir", "compat/lifecycle-diagnostics", "directory for retained failure diagnostics")
	flag.BoolVar(&keepSuccess, "keep-success", false, "write diagnostics even when the lifecycle passes")
	flag.Parse()

	if strings.TrimSpace(cloud) == "" {
		fmt.Fprintln(os.Stderr, "lifecycle-smoke: provide --cloud or OS_CLOUD")
		os.Exit(1)
	}

	var diagnostics lifecycleDiagnostics
	var err error
	switch suite {
	case "keypair":
		diagnostics, err = runKeypairLifecycle(cloud, prefix)
	case "server":
		diagnostics, err = runServerLifecycle(cloud, prefix)
	case "volume":
		diagnostics, err = runVolumeLifecycle(cloud, prefix)
	case "quota":
		diagnostics, err = runQuotaLifecycle(cloud, prefix)
	case "image":
		diagnostics, err = runImageLifecycle(cloud, prefix)
	case "network":
		diagnostics, err = runNetworkLifecycle(cloud, prefix)
	case "object":
		diagnostics, err = runObjectLifecycle(cloud, prefix)
	default:
		fmt.Fprintf(os.Stderr, "lifecycle-smoke: unknown suite %q\n", suite)
		os.Exit(1)
	}
	if err != nil {
		diagnostics.Status = "failed"
	} else {
		diagnostics.Status = "passed"
	}
	diagnostics.FinishedAt = time.Now().UTC().Format(time.RFC3339)

	if err != nil || keepSuccess {
		path, writeErr := writeDiagnostics(diagnosticsDir, diagnostics)
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "lifecycle-smoke: write diagnostics: %v\n", writeErr)
			os.Exit(1)
		}
		diagnostics.DiagnosticsPath = path
	}

	fmt.Printf("%s %s lifecycle: %s\n", cloud, suite, diagnostics.Status)
	if diagnostics.DiagnosticsPath != "" {
		fmt.Printf("diagnostics: %s\n", diagnostics.DiagnosticsPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "lifecycle-smoke: %v\n", err)
		os.Exit(1)
	}
}

func runKeypairLifecycle(cloud string, prefix string) (lifecycleDiagnostics, error) {
	id := uniqueID(prefix)
	name := id + "-keypair"
	diagnostics := lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  name,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_name": name,
		},
	}

	publicKey, cleanupPublicKey, err := writePublicKey()
	if err != nil {
		return diagnostics, err
	}
	defer cleanupPublicKey()
	diagnostics.Fixtures["public_key_file"] = publicKey

	run := func(step string, args ...string) stepResult {
		result := runCLI(cloud, args...)
		result.Name = step
		diagnostics.Steps = append(diagnostics.Steps, result)
		return result
	}

	preflight := run("preflight keypair list", "keypair", "list", "-f", "json")
	if preflight.ExitCode != 0 {
		return diagnostics, fmt.Errorf("preflight keypair list failed")
	}
	create := run("create keypair", "keypair", "create", "--public-key", publicKey, name, "-f", "json")
	if create.ExitCode != 0 {
		diagnostics.CleanupRequired = true
		cleanup := run("cleanup keypair after create failure", "keypair", "delete", name)
		if cleanup.ExitCode == 0 {
			diagnostics.CleanupRequired = false
		}
		return diagnostics, fmt.Errorf("create keypair failed")
	}
	diagnostics.CleanupRequired = true

	show := run("show keypair", "keypair", "show", name, "-f", "json")
	if show.ExitCode != 0 {
		cleanup := run("cleanup keypair after show failure", "keypair", "delete", name)
		if cleanup.ExitCode == 0 {
			diagnostics.CleanupRequired = false
		}
		return diagnostics, fmt.Errorf("show keypair failed")
	}

	deleteResult := run("delete keypair", "keypair", "delete", name)
	if deleteResult.ExitCode != 0 {
		return diagnostics, fmt.Errorf("delete keypair failed")
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func runCLI(cloud string, args ...string) stepResult {
	return runCLIWithEnv(cloud, nil, args...)
}

func runCLIWithEnv(cloud string, extraEnv map[string]string, args ...string) stepResult {
	env := envKeys(extraEnv)
	if len(extraEnv) > 0 {
		saved := map[string]*string{}
		for key, value := range extraEnv {
			if current, ok := os.LookupEnv(key); ok {
				currentCopy := current
				saved[key] = &currentCopy
			} else {
				saved[key] = nil
			}
			_ = os.Setenv(key, value)
		}
		defer func() {
			for key, value := range saved {
				if value == nil {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, *value)
				}
			}
		}()
	}
	fullArgs := append([]string{"--os-cloud", cloud}, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Execute(fullArgs, &stdout, &stderr)
	if err != nil && cli.ShouldPrintError(err) {
		fmt.Fprintln(&stderr, err)
	}
	result := stepResult{
		Args:     fullArgs,
		Env:      env,
		ExitCode: cli.ExitCode(err),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func writePublicKey() (string, func(), error) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", func() {}, err
	}
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", func() {}, err
	}
	file, err := os.CreateTemp("", "golang-osc-keypair-*.pub")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	if _, err := file.Write(ssh.MarshalAuthorizedKey(sshPublicKey)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func uniqueID(prefix string) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%x", prefix, random)
}

func writeDiagnostics(dir string, diagnostics lifecycleDiagnostics) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, diagnostics.ID+".json")
	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
