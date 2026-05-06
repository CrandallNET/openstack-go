package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
)

type quotaCleanup struct {
	name string
	env  map[string]string
	args []string
	fn   func() error
}

type quotaLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []quotaCleanup
}

func runQuotaLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "quota",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "quota",
		},
	}
	run := &quotaLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(quotaLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	projectID, err := run.createProjectFixture(id)
	if err != nil {
		return diagnostics, err
	}
	run.must("show quota project", "project", "show", projectID, "-f", "json")
	run.must("show initial quotas", "quota", "show", projectID, "-f", "json")
	run.mustOracle("oracle parity initial quota show json", nil, "quota", "show", projectID, "-f", "json")

	run.mustOracle("oracle parity quota set output", nil, "quota", "set",
		"--instances", "9",
		"--cores", "19",
		"--ram", "51199",
		"--volumes", "9",
		"--snapshots", "9",
		"--gigabytes", "999",
		"--floating-ips", "49",
		"--secgroup-rules", "99",
		"--secgroups", "9",
		"--networks", "99",
		"--subnets", "99",
		"--ports", "499",
		"--routers", "9",
		"--rbac-policies", "9",
		projectID,
	)
	run.must("show quotas after set", "quota", "show", projectID, "-f", "json")
	run.mustOracle("oracle parity quota show after set json", nil, "quota", "show", projectID, "-f", "json")

	run.mustOracle("oracle parity quota delete compute output", nil, "quota", "delete", "--compute", projectID)
	run.must("show compute quota after delete", "quota", "show", "--compute", projectID, "-f", "json")
	run.mustOracle("oracle parity quota reset compute output", nil, "quota", "set", "--instances", "9", "--cores", "19", "--ram", "51199", projectID)

	run.mustOracle("oracle parity quota delete volume output", nil, "quota", "delete", "--volume", projectID)
	run.must("show volume quota after delete", "quota", "show", "--volume", projectID, "-f", "json")
	run.mustOracle("oracle parity quota reset volume output", nil, "quota", "set", "--volumes", "9", "--snapshots", "9", "--gigabytes", "999", projectID)

	run.mustOracle("oracle parity quota delete network output", nil, "quota", "delete", "--network", projectID)
	run.must("show network quota after delete", "quota", "show", "--network", projectID, "-f", "json")
	run.mustOracle("oracle parity quota reset network output", nil, "quota", "set", "--floating-ips", "49", "--secgroup-rules", "99", "--secgroups", "9", "--networks", "99", "--subnets", "99", "--ports", "499", "--routers", "9", "--rbac-policies", "9", projectID)

	run.mustOracle("oracle parity quota delete all output", nil, "quota", "delete", projectID)
	run.must("show quotas after delete all", "quota", "show", projectID, "-f", "json")
	run.mustOracle("oracle parity quota show after delete all json", nil, "quota", "show", projectID, "-f", "json")
	run.recordRiskSkipped("quota set --class and --default", "changes default quota class state rather than only the dedicated test project")

	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func (run *quotaLifecycle) createProjectFixture(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := identityClientForCloud(ctx, run.cloud)
	if err != nil {
		return "", err
	}
	enabled := true
	name := id + "-project"
	project, err := projects.Create(ctx, client, projects.CreateOpts{
		Name:        name,
		Enabled:     &enabled,
		Description: "golang-osc quota lifecycle project",
	}).Extract()
	if err != nil {
		return "", err
	}
	run.diagnostics.Fixtures["project_id"] = project.ID
	run.diagnostics.Fixtures["project_name"] = project.Name
	run.addCleanupFunc("cleanup quota project", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return projects.Delete(ctx, client, project.ID).ExtractErr()
	})
	return project.ID, nil
}

func identityClientForCloud(ctx context.Context, cloud string) (*gophercloud.ServiceClient, error) {
	authOptions, endpointOptions, tlsConfig, err := clouds.Parse(clouds.WithCloudName(cloud))
	if err != nil {
		return nil, err
	}
	provider, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		return nil, err
	}
	return openstack.NewIdentityV3(provider, endpointOptions)
}

func (run *quotaLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *quotaLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *quotaLifecycle) must(name string, args ...string) stepResult {
	return run.mustWithEnv(name, nil, args...)
}

func (run *quotaLifecycle) mustWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := run.runWithEnv(name, env, args...)
	if result.ExitCode != 0 {
		_ = run.cleanupAll()
		panic(quotaLifecycleFailure{name: name})
	}
	return result
}

func (run *quotaLifecycle) mustOracle(name string, env map[string]string, args ...string) stepResult {
	result := compareWithOracle(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(quotaLifecycleFailure{name: name})
	}
	return result
}

func (run *quotaLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *quotaLifecycle) recordRiskSkipped(command string, reason string) {
	run.skip("skip "+command, reason)
}

func (run *quotaLifecycle) addCleanup(name string, args ...string) {
	run.addCleanupWithEnv(name, nil, args...)
}

func (run *quotaLifecycle) addCleanupWithEnv(name string, env map[string]string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, quotaCleanup{name: name, env: env, args: args})
}

func (run *quotaLifecycle) addCleanupFunc(name string, fn func() error) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, quotaCleanup{name: name, fn: fn})
}

func (run *quotaLifecycle) cleanupAll() error {
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

type quotaLifecycleFailure struct {
	name string
}

func (err quotaLifecycleFailure) Error() string {
	return "quota lifecycle failed at " + err.name
}

func quotaValue(stdout string, resource string) string {
	for _, row := range jsonRows(stdout) {
		if strings.EqualFold(jsonRowString(row, "Resource", "resource"), resource) {
			return jsonRowString(row, "Limit", "limit")
		}
	}
	return ""
}
