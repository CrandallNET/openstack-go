package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type parityReplacement struct {
	Go     string
	Oracle string
	Token  string
}

var parityUUIDPattern = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
var parityHexIDPattern = regexp.MustCompile(`\b[0-9a-fA-F]{32,64}\b`)
var parityTimestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)
var parityIPv4Pattern = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
var parityMACPattern = regexp.MustCompile(`\b[0-9a-fA-F]{2}(?::[0-9a-fA-F]{2}){5}\b`)
var paritySwiftTransactionPattern = regexp.MustCompile(`\btx[A-Za-z0-9-]+\b`)
var parityInstanceNamePattern = regexp.MustCompile(`\binstance-[0-9a-fA-F-]+\b`)
var parityReservationPattern = regexp.MustCompile(`\br-[A-Za-z0-9]+\b`)
var parityAdminPassPattern = regexp.MustCompile(`("adminPass":\s*)"[^"]*"`)
var parityCinderAuthPattern = regexp.MustCompile(`("(?:auth_username|auth_password)":\s*)"[^"]*"`)
var parityNeutronRevisionPattern = regexp.MustCompile(`("revision_number":\s*)\d+`)
var parityNeutronSegmentationPattern = regexp.MustCompile(`("provider:segmentation_id":\s*)\d+`)
var parityNeutronStandardAttrPattern = regexp.MustCompile(`("standard_attr_id":\s*)\d+`)

func runOracleCLI(cloud string, extraEnv map[string]string, args ...string) stepResult {
	return runOracleCLICommand(cloud, extraEnv, 0, args...)
}

func runOracleCLIWithTimeout(cloud string, extraEnv map[string]string, timeout time.Duration, args ...string) stepResult {
	return runOracleCLICommand(cloud, extraEnv, timeout, args...)
}

func runOracleCLICommand(cloud string, extraEnv map[string]string, timeout time.Duration, args ...string) stepResult {
	oracle := os.Getenv("OSC_ORACLE")
	if oracle == "" {
		oracle = "/Users/ken/.local/bin/openstack"
	}
	fullArgs := append([]string{"--os-cloud", cloud}, args...)
	var command *exec.Cmd
	var cancel context.CancelFunc
	var ctx context.Context
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		command = exec.CommandContext(ctx, oracle, fullArgs...)
	} else {
		command = exec.Command(oracle, fullArgs...)
	}
	command.Env = append(os.Environ(), envMapValues(extraEnv)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := stepResult{
		Args:     fullArgs,
		Env:      envKeys(extraEnv),
		ExitCode: exitCode(err),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
	if err != nil {
		result.Error = err.Error()
	}
	if timeout > 0 && ctx != nil && ctx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Sprintf("timeout after %s", timeout)
	}
	return result
}

func compareWithOracle(cloud string, extraEnv map[string]string, args ...string) stepResult {
	goResult := runCLIWithEnv(cloud, extraEnv, args...)
	oracleResult := runOracleCLI(cloud, extraEnv, args...)
	return compareStepResults(goResult, oracleResult, nil)
}

func compareWithOracleArgs(cloud string, extraEnv map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	goResult := runCLIWithEnv(cloud, extraEnv, goArgs...)
	if goResult.ExitCode != 0 {
		return goResult
	}
	oracleResult := runOracleCLI(cloud, extraEnv, oracleArgs...)
	return compareStepResults(goResult, oracleResult, replacements)
}

func compareExistingWithOracle(cloud string, extraEnv map[string]string, goResult stepResult, oracleArgs []string, replacements []parityReplacement) stepResult {
	oracleResult := runOracleCLI(cloud, extraEnv, oracleArgs...)
	return compareStepResults(goResult, oracleResult, replacements)
}

func compareStepResults(goResult stepResult, oracleResult stepResult, replacements []parityReplacement) stepResult {
	goResult.OracleArgs = oracleResult.Args
	goResult.OracleExitCode = oracleResult.ExitCode
	goResult.OracleStdout = oracleResult.Stdout
	goResult.OracleStderr = oracleResult.Stderr
	goResult.OracleError = oracleResult.Error
	if goResult.ExitCode != oracleResult.ExitCode {
		goResult.Error = fmt.Sprintf("oracle exit code differs: go=%d oracle=%d", goResult.ExitCode, oracleResult.ExitCode)
		return goResult
	}
	goStdout := normalizeParityOutput(goResult.Stdout, replacements, true)
	oracleStdout := normalizeParityOutput(oracleResult.Stdout, replacements, false)
	goStderr := normalizeParityOutput(goResult.Stderr, replacements, true)
	oracleStderr := normalizeParityOutput(oracleResult.Stderr, replacements, false)
	if goStdout != oracleStdout {
		goResult.Error = "oracle stdout differs"
		return goResult
	}
	if goStderr != oracleStderr {
		goResult.Error = "oracle stderr differs"
		return goResult
	}
	return goResult
}

func normalizeParityOutput(value string, replacements []parityReplacement, goSide bool) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	if replacements == nil {
		return value
	}
	for _, replacement := range replacements {
		token := replacement.Token
		if token == "" {
			token = "<value>"
		}
		sideValue := replacement.Oracle
		if goSide {
			sideValue = replacement.Go
		}
		if strings.TrimSpace(sideValue) == "" {
			continue
		}
		value = strings.ReplaceAll(value, sideValue, token)
	}
	value = parityTimestampPattern.ReplaceAllString(value, "<timestamp>")
	value = parityUUIDPattern.ReplaceAllString(value, "<uuid>")
	value = parityHexIDPattern.ReplaceAllString(value, "<hex-id>")
	value = parityIPv4Pattern.ReplaceAllString(value, "<ip>")
	value = parityMACPattern.ReplaceAllString(value, "<mac>")
	value = paritySwiftTransactionPattern.ReplaceAllString(value, "<swift-tx>")
	value = parityInstanceNamePattern.ReplaceAllString(value, "<instance>")
	value = parityReservationPattern.ReplaceAllString(value, "<reservation>")
	value = parityAdminPassPattern.ReplaceAllString(value, `${1}"<admin-pass>"`)
	value = parityCinderAuthPattern.ReplaceAllString(value, `${1}"<cinder-auth>"`)
	value = parityNeutronRevisionPattern.ReplaceAllString(value, `${1}"<revision>"`)
	value = parityNeutronSegmentationPattern.ReplaceAllString(value, `${1}"<segment>"`)
	value = parityNeutronStandardAttrPattern.ReplaceAllString(value, `${1}"<standard-attr>"`)
	if canonical, ok := canonicalParityJSON(value); ok {
		value = canonical
	}
	return value
}

func canonicalParityJSON(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		return value, false
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return value, false
	}
	canonical := canonicalParityValue(decoded)
	encoded, err := json.MarshalIndent(canonical, "", "  ")
	if err != nil {
		return value, false
	}
	return string(encoded) + "\n", true
}

func canonicalParityValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = canonicalParityValue(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, canonicalParityValue(item))
		}
		sort.SliceStable(result, func(i int, j int) bool {
			left, _ := json.Marshal(result[i])
			right, _ := json.Marshal(result[j])
			return string(left) < string(right)
		})
		return result
	default:
		return value
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}
