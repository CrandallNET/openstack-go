package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type commandGroup struct {
	CommandGroup string   `json:"Command Group"`
	Commands     []string `json:"Commands"`
}

type metadata struct {
	GeneratedAt          string         `json:"generated_at"`
	OraclePath           string         `json:"oracle_path"`
	OracleVersionOutput  string         `json:"oracle_version_output"`
	CompatibilityTarget  string         `json:"compatibility_target"`
	CommandGroupCount    int            `json:"command_group_count"`
	CommandCount         int            `json:"command_count"`
	HelpSnapshotCount    int            `json:"help_snapshot_count"`
	Generator            string         `json:"generator"`
	GenerationCommands   []string       `json:"generation_commands"`
	CommandCountByGroup  map[string]int `json:"command_count_by_group"`
	SecretsRedactionNote string         `json:"secrets_redaction_note"`
}

func main() {
	var openstackPath string
	var outputDir string
	var targetVersion string
	var skipHelp bool
	var timeout time.Duration

	flag.StringVar(&openstackPath, "openstack", "/Users/ken/.local/bin/openstack", "path to the Python OpenStackClient oracle")
	flag.StringVar(&outputDir, "output", "compat/osc/9.0.0", "output directory for generated compatibility artifacts")
	flag.StringVar(&targetVersion, "target-version", "9.0.0", "pinned compatibility target version")
	flag.BoolVar(&skipHelp, "skip-help", false, "skip per-command help snapshots")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "timeout per oracle command")
	flag.Parse()

	if err := run(openstackPath, outputDir, targetVersion, skipHelp, timeout); err != nil {
		fmt.Fprintf(os.Stderr, "osc-catalog: %v\n", err)
		os.Exit(1)
	}
}

func run(openstackPath string, outputDir string, targetVersion string, skipHelp bool, timeout time.Duration) error {
	versionOutput, err := capture(timeout, openstackPath, "--version")
	if err != nil {
		return err
	}

	commandJSON, err := capture(timeout, openstackPath, "command", "list", "-f", "json")
	if err != nil {
		return err
	}

	var groups []commandGroup
	if err := json.Unmarshal(commandJSON, &groups); err != nil {
		return fmt.Errorf("decode command list: %w", err)
	}
	sortCommandGroups(groups)

	globalHelp, err := capture(timeout, openstackPath, "--help")
	if err != nil {
		return err
	}

	completion, err := capture(timeout, openstackPath, "complete")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(outputDir, "help"), 0o755); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(outputDir, "commands.json"), groups); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "global-help.txt"), globalHelp, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "completion.bash"), completion, 0o644); err != nil {
		return err
	}

	helpCount := 0
	if !skipHelp {
		for _, command := range flattenCommands(groups) {
			help, err := capture(timeout, openstackPath, append([]string{"help"}, strings.Fields(command)...)...)
			if err != nil {
				return fmt.Errorf("capture help for %q: %w", command, err)
			}
			helpPath := filepath.Join(append([]string{outputDir, "help"}, strings.Fields(command)...)...) + ".txt"
			if err := os.MkdirAll(filepath.Dir(helpPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(helpPath, help, 0o644); err != nil {
				return err
			}
			helpCount++
		}
	}

	meta := metadata{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		OraclePath:          openstackPath,
		OracleVersionOutput: strings.TrimSpace(string(versionOutput)),
		CompatibilityTarget: targetVersion,
		CommandGroupCount:   len(groups),
		CommandCount:        len(flattenCommands(groups)),
		HelpSnapshotCount:   helpCount,
		Generator:           "tools/osc-catalog",
		GenerationCommands: []string{
			openstackPath + " --version",
			openstackPath + " command list -f json",
			openstackPath + " --help",
			openstackPath + " complete",
			openstackPath + " help <command>",
		},
		CommandCountByGroup:  countByGroup(groups),
		SecretsRedactionNote: "Generated artifacts must not include clouds.yaml contents, tokens, passwords, application credentials, or unsanitized debug logs.",
	}
	if skipHelp {
		meta.GenerationCommands = meta.GenerationCommands[:4]
	}
	if err := writeJSON(filepath.Join(outputDir, "metadata.json"), meta); err != nil {
		return err
	}

	return nil
}

func capture(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s %s timed out after %s", name, strings.Join(args, " "), timeout)
		}
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func sortCommandGroups(groups []commandGroup) {
	sort.Slice(groups, func(i int, j int) bool {
		return groups[i].CommandGroup < groups[j].CommandGroup
	})
	for i := range groups {
		sort.Strings(groups[i].Commands)
	}
}

func flattenCommands(groups []commandGroup) []string {
	var commands []string
	for _, group := range groups {
		commands = append(commands, group.Commands...)
	}
	sort.Strings(commands)
	return commands
}

func countByGroup(groups []commandGroup) map[string]int {
	counts := make(map[string]int, len(groups))
	for _, group := range groups {
		counts[group.CommandGroup] = len(group.Commands)
	}
	return counts
}
