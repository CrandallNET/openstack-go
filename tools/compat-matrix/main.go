package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type commandGroup struct {
	CommandGroup string   `json:"Command Group"`
	Commands     []string `json:"Commands"`
}

type commandEntry struct {
	Command       string
	Group         string
	Service       string
	API           string
	Status        string
	SDKPackage    string
	Shim          bool
	PluginScope   bool
	Tests         []string
	Notes         string
	ImplementedIn string
}

type testSuite struct {
	Name          string
	Oracle        string
	Service       string
	Risk          string
	Role          string
	AllowedClouds []string
	Setup         string
	Cleanup       string
	Skip          []string
	Notes         string
}

func main() {
	var commandsPath string
	var matrixPath string
	var testMatrixPath string
	var testCloudsPath string

	flag.StringVar(&commandsPath, "commands", "compat/osc/9.0.0/commands.json", "OSC command catalog JSON path")
	flag.StringVar(&matrixPath, "matrix", "compat/matrix.yaml", "command compatibility matrix output path")
	flag.StringVar(&testMatrixPath, "test-matrix", "compat/test-matrix.yaml", "test matrix output path")
	flag.StringVar(&testCloudsPath, "test-clouds", "compat/test-clouds.yaml", "test cloud capability config output path")
	flag.Parse()

	groups, err := readGroups(commandsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-matrix: %v\n", err)
		os.Exit(1)
	}

	if err := writeFile(matrixPath, renderCommandMatrix(groups)); err != nil {
		fmt.Fprintf(os.Stderr, "compat-matrix: %v\n", err)
		os.Exit(1)
	}
	if err := writeFile(testMatrixPath, renderTestMatrix()); err != nil {
		fmt.Fprintf(os.Stderr, "compat-matrix: %v\n", err)
		os.Exit(1)
	}
	if err := writeFile(testCloudsPath, renderTestClouds()); err != nil {
		fmt.Fprintf(os.Stderr, "compat-matrix: %v\n", err)
		os.Exit(1)
	}
}

func readGroups(path string) ([]commandGroup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var groups []commandGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}
	sort.Slice(groups, func(i int, j int) bool {
		return groups[i].CommandGroup < groups[j].CommandGroup
	})
	for i := range groups {
		sort.Strings(groups[i].Commands)
	}
	return groups, nil
}

func renderCommandMatrix(groups []commandGroup) string {
	var entries []commandEntry
	for _, group := range groups {
		for _, command := range group.Commands {
			entries = append(entries, newCommandEntry(group.CommandGroup, command))
		}
	}
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Command < entries[j].Command
	})

	var b strings.Builder
	b.WriteString("# Generated from compat/osc/9.0.0/commands.json by tools/compat-matrix.\n")
	b.WriteString("compatibility_target: \"9.0.0\"\n")
	b.WriteString("status_values:\n")
	for _, status := range []string{"unknown", "sdk-covered", "shim-needed", "implemented", "golden-matched", "cloud-verified", "blocked"} {
		fmt.Fprintf(&b, "  - %s\n", yamlString(status))
	}
	b.WriteString("commands:\n")
	for _, entry := range entries {
		fmt.Fprintf(&b, "  - command: %s\n", yamlString(entry.Command))
		fmt.Fprintf(&b, "    group: %s\n", yamlString(entry.Group))
		fmt.Fprintf(&b, "    service: %s\n", yamlString(entry.Service))
		fmt.Fprintf(&b, "    api: %s\n", yamlString(entry.API))
		fmt.Fprintf(&b, "    status: %s\n", yamlString(entry.Status))
		fmt.Fprintf(&b, "    sdk_package: %s\n", yamlString(entry.SDKPackage))
		fmt.Fprintf(&b, "    shim: %t\n", entry.Shim)
		fmt.Fprintf(&b, "    plugin_scope: %t\n", entry.PluginScope)
		fmt.Fprintf(&b, "    implemented_in: %s\n", yamlString(entry.ImplementedIn))
		b.WriteString("    tests: []\n")
		fmt.Fprintf(&b, "    notes: %s\n", yamlString(entry.Notes))
	}
	return b.String()
}

func newCommandEntry(group string, command string) commandEntry {
	service, api := serviceForGroup(group)
	entry := commandEntry{
		Command:    command,
		Group:      group,
		Service:    service,
		API:        api,
		Status:     "unknown",
		SDKPackage: "",
		Shim:       false,
		Tests:      nil,
		Notes:      "",
	}
	if group == "openstack.placement.v1" || strings.HasPrefix(command, "tap ") {
		entry.PluginScope = true
	}
	if command == "command list" {
		entry.Status = "implemented"
		entry.ImplementedIn = "internal/cli"
		entry.Notes = "Initial implementation uses the embedded OSC catalog and marks unfinished commands."
	}
	return entry
}

func serviceForGroup(group string) (string, string) {
	switch group {
	case "openstack.cli":
		return "cli", "local"
	case "openstack.common":
		return "common", "multi-service"
	case "openstack.compute.v2":
		return "compute", "v2"
	case "openstack.identity.v3":
		return "identity", "v3"
	case "openstack.image.v2":
		return "image", "v2"
	case "openstack.network.v2":
		return "network", "v2"
	case "openstack.object_store.v1":
		return "object-store", "v1"
	case "openstack.placement.v1":
		return "placement", "v1"
	case "openstack.volume.v3":
		return "volume", "v3"
	default:
		return "unknown", ""
	}
}

func renderTestMatrix() string {
	suites := []testSuite{
		{Name: "command catalog generation", Oracle: "python-openstackclient", Service: "cli", Risk: "static", Role: "none", AllowedClouds: nil, Setup: "none", Cleanup: "none", Skip: []string{}, Notes: "Uses local Python OSC only."},
		{Name: "parser and help parity", Oracle: "python-openstackclient", Service: "cli", Risk: "static", Role: "none", AllowedClouds: nil, Setup: "none", Cleanup: "none", Skip: []string{}, Notes: "Covers command paths, help snapshots, invalid commands, invalid flags, and completion."},
		{Name: "renderer parity", Oracle: "python-openstackclient", Service: "cli", Risk: "static", Role: "none", AllowedClouds: nil, Setup: "fixed rows", Cleanup: "none", Skip: []string{}, Notes: "Covers OSC formats and Go-only pretty output."},
		{Name: "auth config precedence", Oracle: "python-openstackclient", Service: "identity", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "local non-secret cloud config", Cleanup: "none", Skip: []string{"endpoint-missing", "fixture-missing"}, Notes: "May use cloud names but must not persist secrets."},
		{Name: "token issue and service catalog", Oracle: "python-openstackclient", Service: "identity", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "configured cloud credential", Cleanup: "none", Skip: []string{"endpoint-missing", "policy-denied"}, Notes: "First live gate for every configured cloud."},
		{Name: "service discovery", Oracle: "python-openstackclient", Service: "common", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "configured cloud credential", Cleanup: "none", Skip: []string{"endpoint-missing"}, Notes: "Records catalog services, regions, interfaces, versions, and extension lists."},
		{Name: "identity admin read write", Oracle: "python-openstackclient", Service: "identity", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "golang-osc-testing project", Cleanup: "test-created resources only", Skip: []string{"role-missing", "policy-denied"}, Notes: "Admin tests are cloud6-only unless future credentials grant admin elsewhere."},
		{Name: "compute read", Oracle: "python-openstackclient", Service: "compute", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "policy-denied"}, Notes: "Run where Nova exists."},
		{Name: "compute project lifecycle", Oracle: "python-openstackclient", Service: "compute", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "dynamic image flavor network quota discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "quota-missing", "fixture-missing", "unsafe-remote-write"}, Notes: "Never delete existing servers, keys, networks, or images."},
		{Name: "compute admin operations", Oracle: "python-openstackclient", Service: "compute", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "role-missing", "policy-denied"}, Notes: "Admin failures on flex clouds are expected and not product gaps."},
		{Name: "network read", Oracle: "python-openstackclient", Service: "network", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "extension-missing"}, Notes: "Run where Neutron exists."},
		{Name: "network project lifecycle", Oracle: "python-openstackclient", Service: "network", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "dynamic external network quota extension discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "quota-missing", "fixture-missing", "unsafe-remote-write"}, Notes: "Never mutate existing network resources."},
		{Name: "network admin and extension writes", Oracle: "python-openstackclient", Service: "network", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service and extension discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "extension-missing", "role-missing", "policy-denied"}, Notes: "Includes provider and Tap-as-a-Service admin paths when available."},
		{Name: "volume read", Oracle: "python-openstackclient", Service: "volume", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "policy-denied"}, Notes: "Run where Cinder v3 exists."},
		{Name: "volume project lifecycle", Oracle: "python-openstackclient", Service: "volume", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "dynamic volume type quota backend discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "quota-missing", "fixture-missing", "unsafe-remote-write"}, Notes: "Never delete existing volumes, snapshots, or backups."},
		{Name: "volume admin operations", Oracle: "python-openstackclient", Service: "volume", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "role-missing", "policy-denied"}, Notes: "Covers services, clusters, hosts, cleanup, manageable resources, and groups."},
		{Name: "image read", Oracle: "python-openstackclient", Service: "image", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "policy-denied"}, Notes: "Run where Glance v2 exists."},
		{Name: "image project lifecycle", Oracle: "python-openstackclient", Service: "image", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "dynamic import store quota discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "quota-missing", "fixture-missing", "unsafe-remote-write"}, Notes: "Never delete existing images."},
		{Name: "image admin cache metadef", Oracle: "python-openstackclient", Service: "image", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service and feature discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "extension-missing", "role-missing", "policy-denied"}, Notes: "Covers cache, metadef, protected images, and stores where exposed."},
		{Name: "object store read lifecycle", Oracle: "python-openstackclient", Service: "object-store", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "dynamic quota discovery", Cleanup: "test-created containers and objects only", Skip: []string{"service-missing", "quota-missing", "unsafe-remote-write"}, Notes: "Swift lifecycle is safe on remote clouds only when cleanup is proven."},
		{Name: "placement read", Oracle: "python-openstackclient", Service: "placement", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "policy-denied"}, Notes: "Plugin scope, but visible in the local command catalog."},
		{Name: "placement admin writes", Oracle: "python-openstackclient", Service: "placement", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service discovery", Cleanup: "test-created resources only", Skip: []string{"service-missing", "role-missing", "policy-denied"}, Notes: "Normally requires admin."},
		{Name: "common read commands", Oracle: "python-openstackclient", Service: "common", Risk: "read", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "policy-denied"}, Notes: "Covers availability zones, extensions, limits, quota show, and versions."},
		{Name: "common admin destructive commands", Oracle: "python-openstackclient", Service: "common", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "golang-osc-testing project", Cleanup: "test-created resources only", Skip: []string{"service-missing", "role-missing", "policy-denied"}, Notes: "Covers quota mutations and project cleanup."},
		{Name: "name or id lookup ambiguity", Oracle: "python-openstackclient", Service: "common", Risk: "mocked-http", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "prefer mocked duplicate names", Cleanup: "test-created resources only", Skip: []string{"fixture-missing", "unsafe-remote-write"}, Notes: "Live ambiguity tests may create duplicate test-owned names only."},
		{Name: "microversion negotiation", Oracle: "python-openstackclient", Service: "common", Risk: "mocked-http", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service version discovery", Cleanup: "none", Skip: []string{"service-missing", "endpoint-missing", "fixture-missing"}, Notes: "Static and mocked tests are mandatory before live variants."},
		{Name: "wait async timeout behavior", Oracle: "python-openstackclient", Service: "common", Risk: "write-cleanup", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "test-created async resources", Cleanup: "test-created resources only", Skip: []string{"quota-missing", "fixture-missing", "unsafe-remote-write"}, Notes: "Can consume resources; remote variants need proven cleanup."},
		{Name: "raw rest shims", Oracle: "python-openstackclient", Service: "common", Risk: "mocked-http", Role: "project", AllowedClouds: []string{"cloud6", "flex-sjc", "flex-dfw", "flex-iad"}, Setup: "mocked HTTP first", Cleanup: "depends on service", Skip: []string{"service-missing", "extension-missing", "role-missing"}, Notes: "Live eligibility follows the command using the shim."},
		{Name: "remote service breadth smoke", Oracle: "python-openstackclient", Service: "common", Risk: "read", Role: "project", AllowedClouds: []string{"flex-sjc", "flex-dfw", "flex-iad"}, Setup: "service discovery", Cleanup: "none", Skip: []string{"endpoint-missing"}, Notes: "Finds services missing on cloud6."},
		{Name: "admin coverage smoke", Oracle: "python-openstackclient", Service: "common", Risk: "admin-write", Role: "admin", AllowedClouds: []string{"cloud6"}, Setup: "service discovery", Cleanup: "test-created resources only", Skip: []string{"role-missing", "service-missing"}, Notes: "Documents cloud6 admin coverage and missing services separately."},
		{Name: "osc version drift", Oracle: "python-openstackclient", Service: "cli", Risk: "static", Role: "none", AllowedClouds: nil, Setup: "regenerated catalogs", Cleanup: "none", Skip: []string{}, Notes: "Does not require OpenStack credentials."},
	}

	var b strings.Builder
	b.WriteString("# Initial live/static test matrix for golang-osc.\n")
	b.WriteString("suites:\n")
	for _, suite := range suites {
		fmt.Fprintf(&b, "  - name: %s\n", yamlString(suite.Name))
		fmt.Fprintf(&b, "    oracle: %s\n", yamlString(suite.Oracle))
		fmt.Fprintf(&b, "    service: %s\n", yamlString(suite.Service))
		fmt.Fprintf(&b, "    risk: %s\n", yamlString(suite.Risk))
		fmt.Fprintf(&b, "    role: %s\n", yamlString(suite.Role))
		writeStringList(&b, "    allowed_clouds", suite.AllowedClouds)
		fmt.Fprintf(&b, "    setup: %s\n", yamlString(suite.Setup))
		fmt.Fprintf(&b, "    cleanup: %s\n", yamlString(suite.Cleanup))
		writeStringList(&b, "    skip_reasons", suite.Skip)
		fmt.Fprintf(&b, "    notes: %s\n", yamlString(suite.Notes))
	}
	return b.String()
}

func renderTestClouds() string {
	return `# Non-secret cloud capability config. Credentials live in clouds.yaml, not here.
clouds:
  - name: "cloud6"
    access_profile: "local admin cloud"
    admin: true
    default_risks:
      - "read"
      - "write-cleanup"
      - "destructive"
      - "admin-write"
    destructive_project: "golang-osc-testing"
    resource_prefix: "golang-osc-test-"
    notes: "Full admin access, but not all services are present. Missing services are environment gaps, not product gaps."
  - name: "flex-sjc"
    access_profile: "remote project cloud"
    admin: false
    default_risks:
      - "read"
      - "write-cleanup"
    destructive_project: ""
    resource_prefix: "golang-osc-test-"
    notes: "Broader service coverage, no admin-level access. Project-level writes may run only on test-created resources."
  - name: "flex-dfw"
    access_profile: "remote project cloud"
    admin: false
    default_risks:
      - "read"
      - "write-cleanup"
    destructive_project: ""
    resource_prefix: "golang-osc-test-"
    notes: "Broader service coverage, no admin-level access. Project-level writes may run only on test-created resources."
  - name: "flex-iad"
    access_profile: "remote project cloud"
    admin: false
    default_risks:
      - "read"
      - "write-cleanup"
    destructive_project: ""
    resource_prefix: "golang-osc-test-"
    notes: "Broader service coverage, no admin-level access. Project-level writes may run only on test-created resources."
fixture_discovery:
  required_before_lifecycle_tests: true
  record_selected_values: true
  examples:
    - "images"
    - "flavors"
    - "networks"
    - "volume types"
    - "external networks"
    - "quotas"
    - "extensions"
cleanup_policy:
  delete_only_test_created_resources: true
  retain_diagnostics_on_failure: true
  never_mutate_existing_resources: true
`
}

func writeStringList(b *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(b, "%s: []\n", name)
		return
	}
	fmt.Fprintf(b, "%s:\n", name)
	for _, value := range values {
		fmt.Fprintf(b, "      - %s\n", yamlString(value))
	}
}

func writeFile(path string, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}

func yamlString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
