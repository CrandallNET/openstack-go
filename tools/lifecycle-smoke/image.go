package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
)

type imageCleanup struct {
	name string
	env  map[string]string
	args []string
	fn   func() error
}

type imageLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []imageCleanup
}

func runImageLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "image",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "image",
		},
	}
	run := &imageLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(imageLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	run.mustOracle("oracle parity image import info json", nil, "image", "import", "info", "-f", "json")
	imageFile, cleanupImageFile, err := writeLifecycleFile("golang-osc-image-*", "golang-osc image lifecycle content\n")
	if err != nil {
		return diagnostics, err
	}
	defer cleanupImageFile()
	diagnostics.Fixtures["image_file"] = imageFile

	imageID, oracleImageID, replacements := run.directImageLifecycle(id, imageFile)
	run.queuedImportLifecycle(id, imageFile)
	run.imageMetadefLifecycle(id)
	if imageID != "" && oracleImageID != "" {
		run.imageMemberLifecycle(id, imageID, oracleImageID, replacements)
	}

	if imageID != "" {
		run.mustOraclePair("oracle parity image delete output", nil,
			[]string{"image", "delete", imageID},
			[]string{"image", "delete", oracleImageID},
			replacements,
		)
		run.dropCleanup("cleanup image")
		run.dropCleanup("cleanup oracle image")
	}
	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func (run *imageLifecycle) directImageLifecycle(id string, imageFile string) (string, string, []parityReplacement) {
	goName := id + "-image"
	oracleName := id + "-oracle-image"
	replacements := []parityReplacement{
		pairedValue("<image-name>", goName, oracleName),
	}
	goArgs := []string{"image", "create", "--disk-format", "raw", "--container-format", "bare", "--shared", "--unprotected", "--file", imageFile, "--property", "golang_osc_test=" + id, "--tag", "golang-osc-lifecycle", goName, "-f", "json"}
	oracleArgs := []string{"image", "create", "--disk-format", "raw", "--container-format", "bare", "--shared", "--unprotected", "--file", imageFile, "--property", "golang_osc_test=" + id, "--tag", "golang-osc-lifecycle", oracleName, "-f", "json"}
	create := runCLIWithEnv(run.cloud, nil, goArgs...)
	create.Name = "create image"
	run.diagnostics.Steps = append(run.diagnostics.Steps, create)
	imageID := jsonStringField(create.Stdout, "id", "ID")
	if create.ExitCode != 0 || imageID == "" {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: "create image"})
	}
	run.diagnostics.Fixtures["image_id"] = imageID
	run.addCleanup("cleanup image", "image", "delete", imageID)

	oracle := runOracleCLI(run.cloud, nil, oracleArgs...)
	oracleImageID := jsonStringField(oracle.Stdout, "id", "ID")
	if oracle.ExitCode == 0 && oracleImageID != "" {
		run.diagnostics.Fixtures["oracle_image_id"] = oracleImageID
		run.addCleanup("cleanup oracle image", "image", "delete", oracleImageID)
	}
	replacements = appendPairedValues(replacements, pairedValue("<image-id>", imageID, oracleImageID))
	parity := compareStepResults(create, oracle, replacements)
	parity.Name = "oracle parity image create output"
	run.diagnostics.Steps = append(run.diagnostics.Steps, parity)
	if parity.Error != "" {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: "oracle parity image create output"})
	}
	run.mustWaitStatus("wait image active", []string{"image", "show", imageID, "-f", "json"}, "status", []string{"active"}, 3*time.Minute)
	run.mustWaitStatus("wait oracle image active", []string{"image", "show", oracleImageID, "-f", "json"}, "status", []string{"active"}, 3*time.Minute)
	run.mustOracle("oracle parity image show json", nil, "image", "show", imageID, "-f", "json")
	run.mustOracle("oracle parity image list json", nil, "image", "list", "-f", "json")
	run.mustOracle("oracle parity image save stdout", nil, "image", "save", imageID)
	run.mustOraclePair("oracle parity image set output", nil,
		[]string{"image", "set", "--property", "phase=set", "--tag", "phase-set", imageID},
		[]string{"image", "set", "--property", "phase=set", "--tag", "phase-set", oracleImageID},
		replacements,
	)
	run.mustOraclePair("oracle parity image unset output", nil,
		[]string{"image", "unset", "--property", "phase", "--tag", "phase-set", imageID},
		[]string{"image", "unset", "--property", "phase", "--tag", "phase-set", oracleImageID},
		replacements,
	)
	return imageID, oracleImageID, replacements
}

func (run *imageLifecycle) queuedImportLifecycle(id string, imageFile string) {
	goName := id + "-queued-image"
	oracleName := id + "-oracle-queued-image"
	create := run.optionalOraclePair("oracle parity queued image create output", nil,
		[]string{"image", "create", "--disk-format", "raw", "--container-format", "bare", "--private", "--unprotected", "--property", "golang_osc_test=" + id, goName, "-f", "json"},
		[]string{"image", "create", "--disk-format", "raw", "--container-format", "bare", "--private", "--unprotected", "--property", "golang_osc_test=" + id, oracleName, "-f", "json"},
		[]parityReplacement{pairedValue("<image-name>", goName, oracleName)},
	)
	if create.ExitCode != 0 || create.Error != "" {
		return
	}
	imageID := jsonStringField(create.Stdout, "id", "ID")
	oracleImageID := jsonStringField(create.OracleStdout, "id", "ID")
	if imageID == "" || oracleImageID == "" {
		run.skip("image stage/import follow-up", "queued image create did not return paired image IDs")
		return
	}
	run.addCleanup("cleanup queued image", "image", "delete", imageID)
	run.addCleanup("cleanup oracle queued image", "image", "delete", oracleImageID)
	replacements := []parityReplacement{
		pairedValue("<image-name>", goName, oracleName),
		pairedValue("<image-id>", imageID, oracleImageID),
	}
	if result := run.optionalOraclePair("oracle parity image stage output", nil,
		[]string{"image", "stage", "--file", imageFile, imageID},
		[]string{"image", "stage", "--file", imageFile, oracleImageID},
		replacements,
	); result.ExitCode != 0 || result.Error != "" {
		return
	}
	if result := run.optionalOraclePair("oracle parity image import output", nil,
		[]string{"image", "import", imageID, "-f", "json"},
		[]string{"image", "import", oracleImageID, "-f", "json"},
		replacements,
	); result.ExitCode == 0 && result.Error == "" {
		run.optionalWaitStatus("wait imported image active", []string{"image", "show", imageID, "-f", "json"}, "status", []string{"active"}, 3*time.Minute)
		run.optionalWaitStatus("wait oracle imported image active", []string{"image", "show", oracleImageID, "-f", "json"}, "status", []string{"active"}, 3*time.Minute)
	}
	run.mustDeleteOrGone("delete queued image", "image", "delete", imageID)
	run.dropCleanup("cleanup queued image")
	run.mustDeleteOrGone("delete oracle queued image", "image", "delete", oracleImageID)
	run.dropCleanup("cleanup oracle queued image")
}

func (run *imageLifecycle) imageMemberLifecycle(id string, imageID string, oracleImageID string, replacements []parityReplacement) {
	projectID, cleanup, err := run.createProjectFixture(id)
	if err != nil {
		run.skip("image add/remove project", strings.TrimSpace(err.Error()))
		return
	}
	run.addCleanupFunc("cleanup image member project", cleanup)
	replacements = appendPairedValues(replacements, pairedValue("<project-id>", projectID, projectID))
	run.mustOraclePair("oracle parity image add project output", nil,
		[]string{"image", "add", "project", imageID, projectID, "-f", "json"},
		[]string{"image", "add", "project", oracleImageID, projectID, "-f", "json"},
		replacements,
	)
	run.mustOraclePair("oracle parity image remove project output", nil,
		[]string{"image", "remove", "project", imageID, projectID},
		[]string{"image", "remove", "project", oracleImageID, projectID},
		replacements,
	)
}

func (run *imageLifecycle) imageMetadefLifecycle(id string) {
	goNamespace := strings.ReplaceAll(id, "-", "_") + "_ns"
	oracleNamespace := strings.ReplaceAll(id, "-", "_") + "_oracle_ns"
	replacements := []parityReplacement{pairedValue("<namespace>", goNamespace, oracleNamespace)}
	result := run.optionalOraclePair("oracle parity image metadef namespace create output", nil,
		[]string{"image", "metadef", "namespace", "create", "--display-name", "golang-osc lifecycle namespace", "--description", "golang-osc lifecycle namespace", "--private", "--unprotected", goNamespace, "-f", "json"},
		[]string{"image", "metadef", "namespace", "create", "--display-name", "golang-osc lifecycle namespace", "--description", "golang-osc lifecycle namespace", "--private", "--unprotected", oracleNamespace, "-f", "json"},
		replacements,
	)
	if result.ExitCode != 0 || result.Error != "" {
		return
	}
	run.addCleanup("cleanup metadef namespace", "image", "metadef", "namespace", "delete", goNamespace)
	run.addCleanup("cleanup oracle metadef namespace", "image", "metadef", "namespace", "delete", oracleNamespace)
	run.mustOraclePair("oracle parity image metadef namespace set output", nil,
		[]string{"image", "metadef", "namespace", "set", "--display-name", "golang-osc lifecycle namespace updated", goNamespace},
		[]string{"image", "metadef", "namespace", "set", "--display-name", "golang-osc lifecycle namespace updated", oracleNamespace},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef property create output", nil,
		[]string{"image", "metadef", "property", "create", "--name", "phase", "--title", "Phase", "--type", "string", "--schema", `{"type":"string"}`, goNamespace, "-f", "json"},
		[]string{"image", "metadef", "property", "create", "--name", "phase", "--title", "Phase", "--type", "string", "--schema", `{"type":"string"}`, oracleNamespace, "-f", "json"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef property set output", nil,
		[]string{"image", "metadef", "property", "set", "--title", "Phase Updated", "--type", "string", "--schema", `{"type":"string"}`, goNamespace, "phase"},
		[]string{"image", "metadef", "property", "set", "--title", "Phase Updated", "--type", "string", "--schema", `{"type":"string"}`, oracleNamespace, "phase"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef object create output", nil,
		[]string{"image", "metadef", "object", "create", "--namespace", goNamespace, "lifecycle_object", "-f", "json"},
		[]string{"image", "metadef", "object", "create", "--namespace", oracleNamespace, "lifecycle_object", "-f", "json"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef object update output", nil,
		[]string{"image", "metadef", "object", "update", goNamespace, "lifecycle_object"},
		[]string{"image", "metadef", "object", "update", oracleNamespace, "lifecycle_object"},
		replacements,
	)
	run.optionalOraclePair("oracle parity image metadef resource association create output", nil,
		[]string{"image", "metadef", "resource", "type", "association", "create", goNamespace, "OS::Nova::Flavor", "-f", "json"},
		[]string{"image", "metadef", "resource", "type", "association", "create", oracleNamespace, "OS::Nova::Flavor", "-f", "json"},
		replacements,
	)
	run.optionalOraclePair("oracle parity image metadef resource association delete output", nil,
		[]string{"image", "metadef", "resource", "type", "association", "delete", goNamespace, "OS::Nova::Flavor"},
		[]string{"image", "metadef", "resource", "type", "association", "delete", oracleNamespace, "OS::Nova::Flavor"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef object delete output", nil,
		[]string{"image", "metadef", "object", "delete", goNamespace, "lifecycle_object"},
		[]string{"image", "metadef", "object", "delete", oracleNamespace, "lifecycle_object"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef property delete output", nil,
		[]string{"image", "metadef", "property", "delete", goNamespace, "phase"},
		[]string{"image", "metadef", "property", "delete", oracleNamespace, "phase"},
		replacements,
	)
	run.mustOraclePair("oracle parity image metadef namespace delete output", nil,
		[]string{"image", "metadef", "namespace", "delete", goNamespace},
		[]string{"image", "metadef", "namespace", "delete", oracleNamespace},
		replacements,
	)
	run.dropCleanup("cleanup metadef namespace")
	run.dropCleanup("cleanup oracle metadef namespace")
}

func (run *imageLifecycle) createProjectFixture(id string) (string, func() error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := identityClientForCloud(ctx, run.cloud)
	if err != nil {
		return "", nil, err
	}
	enabled := true
	project, err := projects.Create(ctx, client, projects.CreateOpts{
		Name:        id + "-image-member-project",
		Enabled:     &enabled,
		Description: "golang-osc image lifecycle project",
	}).Extract()
	if err != nil {
		return "", nil, err
	}
	run.diagnostics.Fixtures["image_member_project_id"] = project.ID
	return project.ID, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return projects.Delete(ctx, client, project.ID).ExtractErr()
	}, nil
}

func (run *imageLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *imageLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *imageLifecycle) mustOracle(name string, env map[string]string, args ...string) stepResult {
	result := compareWithOracle(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: name})
	}
	return result
}

func (run *imageLifecycle) mustOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	result := compareWithOracleArgs(run.cloud, env, goArgs, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: name})
	}
	return result
}

func (run *imageLifecycle) optionalOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	goResult := runCLIWithEnv(run.cloud, env, goArgs...)
	if goResult.ExitCode != 0 {
		goResult.Name = name
		run.diagnostics.Steps = append(run.diagnostics.Steps, goResult)
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(goResult.Error, goResult.Stderr, goResult.Stdout)))
		return goResult
	}
	result := compareExistingWithOracle(run.cloud, env, goResult, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(result.Error, result.OracleStderr, result.Stderr, result.OracleStdout, result.Stdout)))
	}
	return result
}

func (run *imageLifecycle) optionalWaitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	result, ok := run.waitStatus(name, args, field, accepted, timeout)
	if !ok {
		run.skip(name+" follow-up", "timed out waiting for an accepted status")
	}
	return result
}

func (run *imageLifecycle) mustWaitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	result, ok := run.waitStatus(name, args, field, accepted, timeout)
	if !ok {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: name})
	}
	return result
}

func (run *imageLifecycle) waitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) (stepResult, bool) {
	deadline := time.Now().Add(timeout)
	var last stepResult
	for {
		last = run.run(name, args...)
		if last.ExitCode == 0 {
			status := strings.ToLower(jsonStringField(last.Stdout, field, strings.ToLower(field), strings.ToUpper(field)))
			for _, value := range accepted {
				if status == strings.ToLower(value) {
					return last, true
				}
			}
		}
		if time.Now().After(deadline) {
			return last, false
		}
		time.Sleep(time.Second)
	}
}

func (run *imageLifecycle) mustDeleteOrGone(name string, args ...string) stepResult {
	result := run.run(name, args...)
	if result.ExitCode != 0 && !looksDeleted(result) {
		_ = run.cleanupAll()
		panic(imageLifecycleFailure{name: name})
	}
	return result
}

func (run *imageLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *imageLifecycle) addCleanup(name string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, imageCleanup{name: name, args: args})
}

func (run *imageLifecycle) addCleanupFunc(name string, fn func() error) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, imageCleanup{name: name, fn: fn})
}

func (run *imageLifecycle) dropCleanup(name string) {
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

func (run *imageLifecycle) cleanupAll() error {
	var failures []error
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		cleanup := run.cleanups[i]
		if cleanup.fn != nil {
			if err := cleanup.fn(); err != nil {
				failures = append(failures, fmt.Errorf("%s failed: %w", cleanup.name, err))
			}
			continue
		}
		result := run.runWithEnv(cleanup.name, cleanup.env, cleanup.args...)
		if result.ExitCode != 0 && !looksDeleted(result) {
			failures = append(failures, fmt.Errorf("%s failed", cleanup.name))
		}
	}
	run.cleanups = nil
	run.diagnostics.CleanupRequired = len(failures) > 0
	return errors.Join(failures...)
}

type imageLifecycleFailure struct {
	name string
}

func (err imageLifecycleFailure) Error() string {
	return "image lifecycle failed at " + err.name
}
