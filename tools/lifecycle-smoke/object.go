package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type objectCleanup struct {
	name string
	env  map[string]string
	args []string
}

type objectLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []objectCleanup
}

func runObjectLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "object",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "object",
		},
	}
	run := &objectLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(objectLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	preflight := run.optional("preflight object store account show", "object", "store", "account", "show", "-f", "json")
	if preflight.ExitCode != 0 {
		run.skip("object lifecycle", "object-store endpoint is unavailable or rejected account metadata discovery on this cloud")
		return diagnostics, nil
	}

	objectFile, cleanupObjectFile, err := writeLifecycleFile("golang-osc-object-*", "golang-osc object lifecycle content\n")
	if err != nil {
		return diagnostics, err
	}
	defer cleanupObjectFile()
	diagnostics.Fixtures["object_file"] = objectFile

	goContainer := id + "-container"
	oracleContainer := id + "-oracle-container"
	containerReplacements := []parityReplacement{
		pairedValue("<container-name>", goContainer, oracleContainer),
	}
	run.addCleanup("cleanup container", "container", "delete", "--recursive", goContainer)
	run.addCleanup("cleanup oracle container", "container", "delete", "--recursive", oracleContainer)
	run.mustOraclePair("oracle parity container create output", nil,
		[]string{"container", "create", goContainer, "-f", "json"},
		[]string{"container", "create", oracleContainer, "-f", "json"},
		containerReplacements,
	)
	run.mustOracle("oracle parity container show json", nil, "container", "show", goContainer, "-f", "json")
	run.mustOracle("oracle parity container list json", nil, "container", "list", "-f", "json")
	run.mustOracle("oracle parity container set output", nil, "container", "set", "--property", "phase=set", goContainer)
	run.mustOracle("oracle parity container unset output", nil, "container", "unset", "--property", "phase", goContainer)

	objectName := id + "-object.txt"
	oracleObjectName := id + "-oracle-object.txt"
	objectReplacements := appendPairedValues(containerReplacements,
		pairedValue("<object-name>", objectName, oracleObjectName),
	)
	run.addCleanup("cleanup object", "object", "delete", goContainer, objectName)
	run.addCleanup("cleanup oracle object", "object", "delete", oracleContainer, oracleObjectName)
	run.mustOraclePair("oracle parity object create output", nil,
		[]string{"object", "create", goContainer, objectFile, "--name", objectName, "-f", "json"},
		[]string{"object", "create", oracleContainer, objectFile, "--name", oracleObjectName, "-f", "json"},
		objectReplacements,
	)
	run.mustOracle("oracle parity object show json", nil, "object", "show", goContainer, objectName, "-f", "json")
	run.mustOracle("oracle parity object list json", nil, "object", "list", goContainer, "-f", "json")
	run.mustOracle("oracle parity object save stdout", nil, "object", "save", goContainer, objectName, "--file", "-")
	run.mustOracle("oracle parity object set output", nil, "object", "set", "--property", "phase=set", goContainer, objectName)
	run.mustOracle("oracle parity object unset output", nil, "object", "unset", "--property", "phase", goContainer, objectName)
	run.mustOraclePair("oracle parity object delete output", nil,
		[]string{"object", "delete", goContainer, objectName},
		[]string{"object", "delete", oracleContainer, oracleObjectName},
		objectReplacements,
	)
	run.dropCleanup("cleanup object")
	run.dropCleanup("cleanup oracle object")

	run.mustOraclePair("oracle parity container delete output", nil,
		[]string{"container", "delete", goContainer},
		[]string{"container", "delete", oracleContainer},
		containerReplacements,
	)
	run.dropCleanup("cleanup container")
	run.dropCleanup("cleanup oracle container")
	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func writeLifecycleFile(pattern string, content string) (string, func(), error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	if _, err := file.WriteString(content); err != nil {
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

func (run *objectLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *objectLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *objectLifecycle) mustOracle(name string, env map[string]string, args ...string) stepResult {
	result := compareWithOracle(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(objectLifecycleFailure{name: name})
	}
	return result
}

func (run *objectLifecycle) mustOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	result := compareWithOracleArgs(run.cloud, env, goArgs, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(objectLifecycleFailure{name: name})
	}
	return result
}

func (run *objectLifecycle) optional(name string, args ...string) stepResult {
	result := run.run(name, args...)
	if result.ExitCode != 0 {
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(result.Error, result.Stderr, result.Stdout)))
	}
	return result
}

func (run *objectLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *objectLifecycle) addCleanup(name string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, objectCleanup{name: name, args: args})
}

func (run *objectLifecycle) dropCleanup(name string) {
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		if run.cleanups[i].name == name {
			run.cleanups = append(run.cleanups[:i], run.cleanups[i+1:]...)
			break
		}
	}
	if len(run.cleanups) == 0 {
		run.diagnostics.CleanupRequired = false
	}
}

func (run *objectLifecycle) cleanupAll() error {
	var failures []error
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		cleanup := run.cleanups[i]
		result := run.runWithEnv(cleanup.name, cleanup.env, cleanup.args...)
		if result.ExitCode != 0 && !looksDeleted(result) {
			failures = append(failures, fmt.Errorf("%s failed", cleanup.name))
		}
	}
	run.cleanups = nil
	run.diagnostics.CleanupRequired = len(failures) > 0
	return errors.Join(failures...)
}

type objectLifecycleFailure struct {
	name string
}

func (err objectLifecycleFailure) Error() string {
	return "object lifecycle failed at " + err.name
}
