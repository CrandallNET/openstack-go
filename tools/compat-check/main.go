package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type metadata struct {
	OraclePath string `json:"oracle_path"`
}

type commandGroup struct {
	CommandGroup string   `json:"Command Group"`
	Commands     []string `json:"Commands"`
}

type checkCase struct {
	Name     string
	Args     []string
	Env      []string
	KnownGap bool
	Reason   string
	Skip     bool
}

type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type checkResult struct {
	Case       checkCase
	Oracle     commandResult
	Go         commandResult
	Matched    bool
	Skipped    bool
	Difference string
	Error      error
}

var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z0-9_-]+)\}`)

var liveFixtureCommands = map[string][]string{
	"flavor":         {"flavor", "list", "-f", "json"},
	"floating_ip":    {"floating", "ip", "list", "-f", "json"},
	"group":          {"group", "list", "-f", "json"},
	"image":          {"image", "list", "-f", "json"},
	"keypair":        {"keypair", "list", "-f", "json"},
	"network":        {"network", "list", "-f", "json"},
	"port":           {"port", "list", "-f", "json"},
	"project":        {"project", "list", "-f", "json"},
	"router":         {"router", "list", "-f", "json"},
	"security_group": {"security", "group", "list", "-f", "json"},
	"server":         {"server", "list", "-f", "json"},
	"subnet":         {"subnet", "list", "-f", "json"},
	"user":           {"user", "list", "-f", "json"},
	"volume":         {"volume", "list", "-f", "json"},
}

func main() {
	var oraclePath string
	var goBinary string
	var metadataPath string
	var commandsPath string
	var allHelp bool
	var includeKnownGaps bool
	var liveCloud string
	var liveCommands string
	var timeout time.Duration

	flag.StringVar(&oraclePath, "oracle", "", "path to the Python OpenStackClient oracle")
	flag.StringVar(&goBinary, "go-binary", "./bin/openstack", "path to the golang-osc binary")
	flag.StringVar(&metadataPath, "metadata", "compat/osc/9.0.0/metadata.json", "OSC oracle metadata path")
	flag.StringVar(&commandsPath, "commands", "compat/osc/9.0.0/commands.json", "OSC command catalog path")
	flag.BoolVar(&allHelp, "all-help", false, "compare --help output for every cataloged command")
	flag.BoolVar(&includeKnownGaps, "known-gaps", true, "run known-gap cases and report them without failing the check")
	flag.StringVar(&liveCloud, "live-cloud", "", "OS_CLOUD value for live Python-vs-Go command comparisons")
	flag.StringVar(&liveCommands, "live-command", "", "comma-separated live commands to compare against --live-cloud")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "timeout per command")
	flag.Parse()

	if oraclePath == "" {
		path, err := oraclePathFromMetadata(metadataPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compat-check: %v\n", err)
			os.Exit(1)
		}
		oraclePath = path
	}

	cases := defaultCases(includeKnownGaps)
	if allHelp {
		commands, err := readCommands(commandsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compat-check: %v\n", err)
			os.Exit(1)
		}
		for _, command := range commands {
			cases = append(cases, checkCase{
				Name: "help: " + command,
				Args: append(strings.Fields(command), "--help"),
			})
		}
	}
	if strings.TrimSpace(liveCommands) != "" {
		if strings.TrimSpace(liveCloud) == "" {
			fmt.Fprintln(os.Stderr, "compat-check: --live-command requires --live-cloud")
			os.Exit(1)
		}
		resolver := newFixtureResolver(oraclePath, liveCloud, timeout)
		for _, command := range splitComma(liveCommands) {
			args := strings.Fields(command)
			resolvedArgs, skipReason := resolver.resolveArgs(args)
			if skipReason != "" {
				cases = append(cases, checkCase{
					Name:   "live:" + liveCloud + ":" + command,
					Args:   args,
					Env:    []string{"OS_CLOUD=" + liveCloud},
					Skip:   true,
					Reason: skipReason,
				})
				continue
			}
			cases = append(cases, checkCase{
				Name: "live:" + liveCloud + ":" + command,
				Args: resolvedArgs,
				Env:  []string{"OS_CLOUD=" + liveCloud},
			})
		}
	}

	results := runCases(oraclePath, goBinary, cases, timeout)
	if err := printResults(os.Stdout, results); err != nil {
		fmt.Fprintf(os.Stderr, "compat-check: %v\n", err)
		os.Exit(1)
	}
	if requiredFailures(results) > 0 {
		os.Exit(1)
	}
}

func defaultCases(includeKnownGaps bool) []checkCase {
	cases := []checkCase{
		{Name: "complete", Args: []string{"complete"}},
		{Name: "command-list-cli-table", Args: []string{"command", "list", "--group", "openstack.cli"}},
		{Name: "command-list-cli-json", Args: []string{"command", "list", "-f", "json", "--group", "openstack.cli"}},
		{Name: "help-command-list", Args: []string{"command", "list", "--help"}},
		{Name: "help-server-list-flag", Args: []string{"server", "list", "--help"}},
		{Name: "help-server-list-command", Args: []string{"help", "server", "list"}},
		{Name: "help-volume-list", Args: []string{"volume", "list", "--help"}},
		{Name: "help-image-list", Args: []string{"image", "list", "--help"}},
		{Name: "invalid-command", Args: []string{"nosuch"}},
		{Name: "invalid-flag", Args: []string{"command", "list", "--bogus"}},
	}
	if includeKnownGaps {
		cases = append(cases,
			checkCase{Name: "root-help", Args: []string{"--help"}, KnownGap: true, Reason: "root help uses the embedded OSC snapshot, but Python OSC root help is nondeterministic because auth plugin option groups can be ordered differently between invocations"},
			checkCase{Name: "module-list-json", Args: []string{"module", "list", "-f", "json"}, KnownGap: true, Reason: "Go module list intentionally reports Go plugin/module state; Python reports installed Python modules"},
		)
	}
	return cases
}

func splitComma(value string) []string {
	var values []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func oraclePathFromMetadata(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}
	if strings.TrimSpace(meta.OraclePath) == "" {
		return "", fmt.Errorf("%s does not contain oracle_path", path)
	}
	return meta.OraclePath, nil
}

func readCommands(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var groups []commandGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}
	var commands []string
	for _, group := range groups {
		commands = append(commands, group.Commands...)
	}
	sort.Strings(commands)
	return commands, nil
}

type fixtureResolver struct {
	oraclePath string
	cloud      string
	timeout    time.Duration
	cache      map[string]fixtureLookup
}

type fixtureLookup struct {
	Value string
	Error string
}

func newFixtureResolver(oraclePath string, cloud string, timeout time.Duration) *fixtureResolver {
	return &fixtureResolver{
		oraclePath: oraclePath,
		cloud:      cloud,
		timeout:    timeout,
		cache:      map[string]fixtureLookup{},
	}
}

func (r *fixtureResolver) resolveArgs(args []string) ([]string, string) {
	resolved := append([]string(nil), args...)
	for index, arg := range resolved {
		matches := placeholderPattern.FindAllStringSubmatch(arg, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			name := normalizeFixtureName(match[1])
			lookup := r.lookup(name)
			if lookup.Error != "" {
				return args, lookup.Error
			}
			resolved[index] = strings.ReplaceAll(resolved[index], match[0], lookup.Value)
		}
	}
	return resolved, ""
}

func normalizeFixtureName(name string) string {
	name = strings.TrimSuffix(name, "_id")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

func (r *fixtureResolver) lookup(name string) fixtureLookup {
	if lookup, ok := r.cache[name]; ok {
		return lookup
	}
	args, ok := liveFixtureCommands[name]
	if !ok {
		lookup := fixtureLookup{Error: fmt.Sprintf("unsupported live fixture placeholder {%s}", name)}
		r.cache[name] = lookup
		return lookup
	}
	result := runCommand(r.timeout, []string{"OS_CLOUD=" + r.cloud}, r.oraclePath, args...)
	if result.ExitCode != 0 {
		lookup := fixtureLookup{Error: fmt.Sprintf("fixture {%s} query failed: %s", name, strings.TrimSpace(result.Stderr+result.Stdout))}
		r.cache[name] = lookup
		return lookup
	}
	value, err := firstFixtureID(result.Stdout)
	if err != nil {
		lookup := fixtureLookup{Error: fmt.Sprintf("fixture {%s} unavailable: %s", name, err)}
		r.cache[name] = lookup
		return lookup
	}
	lookup := fixtureLookup{Value: value}
	r.cache[name] = lookup
	return lookup
}

func firstFixtureID(jsonText string) (string, error) {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
		return "", err
	}
	for _, row := range rows {
		for _, key := range []string{"ID", "Id", "id", "Name", "name"} {
			if value, ok := row[key]; ok {
				text := strings.TrimSpace(fmt.Sprint(value))
				if text != "" && text != "<nil>" {
					return text, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no rows with an ID or name")
}

func runCases(oraclePath string, goBinary string, cases []checkCase, timeout time.Duration) []checkResult {
	results := make([]checkResult, 0, len(cases))
	for _, testCase := range cases {
		if testCase.Skip {
			results = append(results, checkResult{
				Case:    testCase,
				Skipped: true,
			})
			continue
		}
		oracle := runCommand(timeout, testCase.Env, oraclePath, testCase.Args...)
		goResult := runCommand(timeout, testCase.Env, goBinary, testCase.Args...)
		matched, diff := compareResults(oracle, goResult)
		results = append(results, checkResult{
			Case:       testCase,
			Oracle:     oracle,
			Go:         goResult,
			Matched:    matched,
			Difference: diff,
		})
	}
	return results
}

func runCommand(timeout time.Duration, extraEnv []string, name string, args ...string) commandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "CLIFF_FIT_WIDTH=0", "OS_PRETTY=0")
	cmd.Env = append(cmd.Env, extraEnv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{
		Stdout:   normalizeText(stdout.String()),
		Stderr:   normalizeText(stderr.String()),
		ExitCode: 0,
	}
	if err == nil {
		return result
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = -1
		result.Stderr += fmt.Sprintf("command timed out after %s\n", timeout)
		return result
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result
	}
	result.ExitCode = -1
	result.Stderr += err.Error() + "\n"
	return result
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value
}

func compareResults(oracle commandResult, goResult commandResult) (bool, string) {
	if oracle.ExitCode != goResult.ExitCode {
		return false, fmt.Sprintf("exit code: oracle=%d go=%d", oracle.ExitCode, goResult.ExitCode)
	}
	if oracle.Stdout != goResult.Stdout {
		return false, firstDiff("stdout", oracle.Stdout, goResult.Stdout)
	}
	if oracle.Stderr != goResult.Stderr {
		return false, firstDiff("stderr", oracle.Stderr, goResult.Stderr)
	}
	return true, ""
}

func firstDiff(stream string, oracle string, goValue string) string {
	oracleLines := strings.Split(oracle, "\n")
	goLines := strings.Split(goValue, "\n")
	limit := len(oracleLines)
	if len(goLines) > limit {
		limit = len(goLines)
	}
	for i := 0; i < limit; i++ {
		oracleLine := ""
		goLine := ""
		if i < len(oracleLines) {
			oracleLine = oracleLines[i]
		}
		if i < len(goLines) {
			goLine = goLines[i]
		}
		if oracleLine != goLine {
			return fmt.Sprintf("%s line %d differs\n  oracle: %q\n  go:     %q", stream, i+1, oracleLine, goLine)
		}
	}
	return stream + " differs"
}

func printResults(stdout *os.File, results []checkResult) error {
	passed := 0
	skipped := 0
	knownGapFailures := 0
	requiredFailed := 0
	for _, result := range results {
		status := "PASS"
		if result.Skipped {
			status = "SKIP"
			skipped++
		} else if !result.Matched && result.Case.KnownGap {
			status = "KNOWN-GAP"
			knownGapFailures++
		} else if !result.Matched {
			status = "FAIL"
			requiredFailed++
		} else {
			passed++
		}
		if _, err := fmt.Fprintf(stdout, "%-9s %s\n", status, result.Case.Name); err != nil {
			return err
		}
		if result.Skipped {
			if result.Case.Reason != "" {
				if _, err := fmt.Fprintf(stdout, "          reason: %s\n", result.Case.Reason); err != nil {
					return err
				}
			}
		} else if !result.Matched {
			if result.Case.Reason != "" {
				if _, err := fmt.Fprintf(stdout, "          reason: %s\n", result.Case.Reason); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(stdout, "          %s\n", strings.ReplaceAll(result.Difference, "\n", "\n          ")); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(stdout, "summary: pass=%d fail=%d known-gap=%d skip=%d total=%d\n", passed, requiredFailed, knownGapFailures, skipped, len(results))
	return err
}

func requiredFailures(results []checkResult) int {
	failures := 0
	for _, result := range results {
		if result.Skipped {
			continue
		}
		if !result.Matched && !result.Case.KnownGap {
			failures++
		}
	}
	return failures
}
