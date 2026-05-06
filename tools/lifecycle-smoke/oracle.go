package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func runOracleCLI(cloud string, extraEnv map[string]string, args ...string) stepResult {
	oracle := os.Getenv("OSC_ORACLE")
	if oracle == "" {
		oracle = "/Users/ken/.local/bin/openstack"
	}
	fullArgs := append([]string{"--os-cloud", cloud}, args...)
	command := exec.Command(oracle, fullArgs...)
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
	return result
}

func compareWithOracle(cloud string, extraEnv map[string]string, args ...string) stepResult {
	goResult := runCLIWithEnv(cloud, extraEnv, args...)
	oracleResult := runOracleCLI(cloud, extraEnv, args...)
	goResult.OracleArgs = oracleResult.Args
	goResult.OracleExitCode = oracleResult.ExitCode
	goResult.OracleStdout = oracleResult.Stdout
	goResult.OracleStderr = oracleResult.Stderr
	goResult.OracleError = oracleResult.Error
	if goResult.ExitCode != oracleResult.ExitCode {
		goResult.Error = fmt.Sprintf("oracle exit code differs: go=%d oracle=%d", goResult.ExitCode, oracleResult.ExitCode)
		return goResult
	}
	if goResult.Stdout != oracleResult.Stdout {
		goResult.Error = "oracle stdout differs"
		return goResult
	}
	if goResult.Stderr != oracleResult.Stderr {
		goResult.Error = "oracle stderr differs"
		return goResult
	}
	return goResult
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
