package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

type generationSummary struct {
	CommandCount   int
	StatusCounts   map[string]int
	MatrixPath     string
	TestMatrixPath string
	TestCloudsPath string
}

var matrixStatusValues = []string{"unknown", "sdk-covered", "shim-needed", "implemented", "golden-matched", "cloud-verified", "blocked"}
var reportModes = map[string]bool{"summary": true, "command-status": true}
var reportFormats = map[string]bool{"terminal": true, "readme": true}

func main() {
	var commandsPath string
	var matrixPath string
	var testMatrixPath string
	var testCloudsPath string
	var reportMode string
	var reportFormat string
	var reportOutput string
	var summaryFormat string

	flag.StringVar(&commandsPath, "commands", "compat/osc/9.0.0/commands.json", "OSC command catalog JSON path")
	flag.StringVar(&matrixPath, "matrix", "compat/matrix.yaml", "command compatibility matrix output path")
	flag.StringVar(&testMatrixPath, "test-matrix", "compat/test-matrix.yaml", "test matrix output path")
	flag.StringVar(&testCloudsPath, "test-clouds", "compat/test-clouds.yaml", "test cloud capability config output path")
	flag.StringVar(&reportMode, "report", "summary", "stdout report to emit after generation: summary or command-status")
	flag.StringVar(&reportFormat, "report-format", "terminal", "stdout report format: terminal or readme")
	flag.StringVar(&reportOutput, "report-output", "", "write the selected stdout report to this path instead of stdout")
	flag.StringVar(&summaryFormat, "summary-format", "", "deprecated alias for --report-format")
	flag.Parse()
	reportMode = strings.ToLower(strings.TrimSpace(reportMode))
	reportFormat = strings.ToLower(strings.TrimSpace(reportFormat))
	summaryFormat = strings.ToLower(strings.TrimSpace(summaryFormat))
	if summaryFormat != "" {
		reportFormat = summaryFormat
	}
	if !reportModes[reportMode] {
		fmt.Fprintf(os.Stderr, "matrix: unsupported report %q; use summary or command-status\n", reportMode)
		os.Exit(1)
	}
	if !reportFormats[reportFormat] {
		fmt.Fprintf(os.Stderr, "matrix: unsupported report format %q; use terminal or readme\n", reportFormat)
		os.Exit(1)
	}

	groups, err := readGroups(commandsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "matrix: %v\n", err)
		os.Exit(1)
	}

	entries := commandEntries(groups)
	if err := writeFile(matrixPath, renderCommandMatrixEntries(entries)); err != nil {
		fmt.Fprintf(os.Stderr, "matrix: %v\n", err)
		os.Exit(1)
	}
	if err := writeFile(testMatrixPath, renderTestMatrix()); err != nil {
		fmt.Fprintf(os.Stderr, "matrix: %v\n", err)
		os.Exit(1)
	}
	if err := writeFile(testCloudsPath, renderTestClouds()); err != nil {
		fmt.Fprintf(os.Stderr, "matrix: %v\n", err)
		os.Exit(1)
	}
	summary := generationSummary{
		CommandCount:   len(entries),
		StatusCounts:   commandStatusCounts(entries),
		MatrixPath:     matrixPath,
		TestMatrixPath: testMatrixPath,
		TestCloudsPath: testCloudsPath,
	}
	report := renderReport(entries, summary, reportMode, reportFormat)
	if reportOutput != "" {
		if err := writeFile(reportOutput, report); err != nil {
			fmt.Fprintf(os.Stderr, "matrix: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Print(report)
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
	return renderCommandMatrixEntries(commandEntries(groups))
}

func commandEntries(groups []commandGroup) []commandEntry {
	var entries []commandEntry
	for _, group := range groups {
		for _, command := range group.Commands {
			entries = append(entries, newCommandEntry(group.CommandGroup, command))
		}
	}
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Command < entries[j].Command
	})
	return entries
}

func renderCommandMatrixEntries(entries []commandEntry) string {
	var b strings.Builder
	b.WriteString("# Generated from compat/osc/9.0.0/commands.json by tools/matrix.\n")
	b.WriteString("compatibility_target: \"9.0.0\"\n")
	b.WriteString("status_values:\n")
	for _, status := range matrixStatusValues {
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
		if len(entry.Tests) == 0 {
			b.WriteString("    tests: []\n")
		} else {
			b.WriteString("    tests:\n")
			for _, test := range entry.Tests {
				fmt.Fprintf(&b, "      - %s\n", yamlString(test))
			}
		}
		fmt.Fprintf(&b, "    notes: %s\n", yamlString(entry.Notes))
	}
	return b.String()
}

func commandStatusCounts(entries []commandEntry) map[string]int {
	counts := make(map[string]int, len(matrixStatusValues))
	for _, status := range matrixStatusValues {
		counts[status] = 0
	}
	for _, entry := range entries {
		counts[entry.Status]++
	}
	return counts
}

func printGenerationSummary(w io.Writer, summary generationSummary, format string) {
	switch format {
	case "readme":
		printReadmeGenerationSummary(w, summary)
	default:
		printTerminalGenerationSummary(w, summary)
	}
}

func printTerminalGenerationSummary(w io.Writer, summary generationSummary) {
	fmt.Fprintln(w, "matrix results:")
	fmt.Fprintf(w, "  commands: %d\n", summary.CommandCount)
	fmt.Fprintln(w, "  status counts:")
	for _, status := range matrixStatusValues {
		fmt.Fprintf(w, "    %s: %d\n", status, summary.StatusCounts[status])
	}
	fmt.Fprintln(w, "  wrote:")
	fmt.Fprintf(w, "    %s\n", summary.MatrixPath)
	fmt.Fprintf(w, "    %s\n", summary.TestMatrixPath)
	fmt.Fprintf(w, "    %s\n", summary.TestCloudsPath)
}

func printReadmeGenerationSummary(w io.Writer, summary generationSummary) {
	fmt.Fprintln(w, "### Matrix Results")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Generated command rows: `%d`\n", summary.CommandCount)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Status | Count |")
	fmt.Fprintln(w, "| --- | ---: |")
	for _, status := range matrixStatusValues {
		fmt.Fprintf(w, "| `%s` | %d |\n", status, summary.StatusCounts[status])
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Generated files:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "* `%s`\n", summary.MatrixPath)
	fmt.Fprintf(w, "* `%s`\n", summary.TestMatrixPath)
	fmt.Fprintf(w, "* `%s`\n", summary.TestCloudsPath)
}

func renderReport(entries []commandEntry, summary generationSummary, report string, format string) string {
	var b strings.Builder
	switch report {
	case "command-status":
		printCommandStatusReport(&b, entries, format)
	default:
		printGenerationSummary(&b, summary, format)
	}
	return b.String()
}

func printCommandStatusReport(w io.Writer, entries []commandEntry, format string) {
	if format == "readme" {
		fmt.Fprintln(w, "### Command Compatibility Status")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "The Python column is sourced from the pinned Python OpenStackClient 9.0.0 command catalog. The Go status is conservative: `compatible` requires golden oracle parity, `partially compatible` means live cloud verification exists without full oracle parity, `implemented` means Go behavior exists but parity is still open, and `partially implemented` includes command paths that currently rely on generated stubs or incomplete SDK/shim work.")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "| Command | Python OSC 9.0.0 | golang-osc status | Source | Notes |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- |")
	for _, entry := range entries {
		fmt.Fprintf(w, "| `%s` | present | %s | %s | %s |\n",
			markdownTableCell(entry.Command),
			markdownTableCell(commandReportStatus(entry)),
			markdownTableCell(commandReportSource(entry)),
			markdownTableCell(commandReportNote(entry)),
		)
	}
}

func commandReportStatus(entry commandEntry) string {
	switch entry.Status {
	case "golden-matched":
		return "compatible"
	case "cloud-verified":
		return "partially compatible"
	case "implemented":
		return "implemented"
	default:
		return "partially implemented"
	}
}

func commandReportSource(entry commandEntry) string {
	if entry.PluginScope {
		return "plugin"
	}
	return "built-in"
}

func commandReportNote(entry commandEntry) string {
	var notes []string
	switch entry.Status {
	case "golden-matched":
		notes = append(notes, "Golden Python oracle parity recorded.")
	case "cloud-verified":
		notes = append(notes, "Live cloud verification recorded; full oracle parity may still be open.")
	case "implemented":
		notes = append(notes, "Implemented in Go; compatibility verification still open.")
	default:
		notes = append(notes, "Command path exists in the Go CLI, but behavior is not complete.")
	}
	if entry.PluginScope {
		notes = append(notes, "Plugin-scoped command.")
	}
	if entry.Shim {
		notes = append(notes, "Uses a raw REST shim.")
	}
	return strings.Join(notes, " ")
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "|", `\|`)
	value = strings.ReplaceAll(value, "\r\n", "<br>")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
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
	switch command {
	case "command list":
		entry.Status = "golden-matched"
		entry.ImplementedIn = "internal/cli"
		entry.Tests = []string{"compat-static: command-list-cli-table", "compat-static: command-list-cli-json"}
		entry.Notes = "Static Python oracle parity recorded for openstack.cli table and JSON output. Service command groups intentionally mark unfinished commands."
	case "module list":
		entry.Status = "implemented"
		entry.ImplementedIn = "internal/cli"
		entry.Notes = "Initial implementation lists golang-osc module and command-provider state through the Caddy-backed plugin registry."
	case "token issue":
		entry.Status = "cloud-verified"
		entry.SDKPackage = "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
		entry.ImplementedIn = "internal/cli"
		entry.Tests = []string{"unit: injected token issuer", "live: cloud6 token issue"}
		entry.Notes = "Implemented through Gophercloud auth/config and Identity v3 token extraction. JSON smoke passed on cloud6; broader formatter and auth precedence parity still need oracle tests."
	case "server ssh":
		entry.Status = "blocked"
		entry.ImplementedIn = ""
		entry.Tests = []string{"unit: generated command stub"}
		entry.Notes = "Python OSC delegates to local SSH behavior. The Go CLI keeps this as a stub until a self-contained, cross-platform SSH implementation can preserve the project rule against shelling out to the operating system."
	}
	if packagePath, ok := identityReadPackages()[command]; ok {
		entry.Status = "implemented"
		entry.SDKPackage = packagePath
		if identityReadShims()[command] {
			entry.Shim = true
		}
		entry.ImplementedIn = "internal/cli"
		entry.Tests = []string{"unit: command registry", "live: cloud6 identity read smoke"}
		entry.Notes = "Initial Gophercloud-backed read implementation. Basic JSON and table output work; full flag, lookup, sorting, and oracle parity coverage still need completion."
		if identityCloudVerified()[command] {
			entry.Status = "cloud-verified"
		}
	}
	if packagePath, ok := coreReadPackages()[command]; ok {
		entry.Status = "implemented"
		entry.SDKPackage = packagePath
		if coreReadShims()[command] {
			entry.Shim = true
		}
		entry.ImplementedIn = "internal/cli"
		entry.Tests = []string{"unit: command registry"}
		entry.Notes = "Initial Gophercloud-backed read implementation. Basic JSON and table output work; full flag, lookup, microversion, and oracle parity coverage still need completion."
		if coreWriteCommands()[command] {
			entry.Notes = "Initial Gophercloud-backed write implementation. Basic lifecycle behavior works for test-created resources; full flag, lookup, cleanup, and oracle parity coverage still need completion."
		}
		if command == "quota delete" || command == "quota set" {
			entry.Notes = "Initial shim-backed quota mutation implementation. Live mutation is intentionally deferred until a dedicated test project can capture defaults, mutate only test-owned quota values, reset, and retain diagnostics."
		}
		if coreCloudVerified()[command] {
			entry.Status = "cloud-verified"
			if coreWriteCommands()[command] {
				entry.Tests = append(entry.Tests, "live: configured cloud lifecycle smoke")
			} else {
				entry.Tests = append(entry.Tests, "live: configured cloud read smoke")
			}
		} else {
			entry.Notes += " Live fixture still needed before this row can be marked cloud-verified."
		}
		if coreGoldenMatched()[command] {
			entry.Status = "golden-matched"
			entry.Tests = append(entry.Tests, "compat-live: cloud6 default output")
			entry.Notes = "Python-vs-Go default output parity recorded against cloud6 with the live compatibility harness. Broader flag and cloud coverage still need completion."
		}
		if strings.HasPrefix(command, "floating ip port forwarding ") {
			entry.Tests = []string{"unit: command registry", "unit: floating ip port forwarding port range validation"}
			entry.Notes = "Implemented with Gophercloud port forwarding calls and raw Neutron list extraction for range fields. cloud6 returns Neutron 404 for floating IP port-forwarding endpoints in both Python OSC and the Go CLI, so a cloud exposing this extension is still needed for live lifecycle verification."
		}
		if command == "router add gateway" || command == "router remove gateway" {
			entry.Tests = []string{"unit: command registry", "live: cloud6 extension-gated error parity"}
			entry.Notes = "Implemented through Neutron external gateway multihoming action endpoints. cloud6 does not expose external-gateway-multihoming, and the Go CLI matches Python OSC's extension-gated error there; a cloud exposing the extension is still needed for successful gateway action verification."
		}
		if command == "router add route" || command == "router remove route" {
			entry.Tests = []string{"unit: command registry", "live: cloud6 router extra-route lifecycle smoke"}
			entry.Notes = "Implemented through Neutron add_extraroutes and remove_extraroutes action endpoints. Disposable route add, remove, repeated missing-route remove, and cleanup passed on cloud6."
		}
		if strings.HasPrefix(command, "network qos policy ") {
			entry.Tests = []string{"unit: command registry"}
			entry.Notes = "Implemented with Gophercloud QoS policy calls and custom request builders where explicit false booleans or extension fields are needed. cloud6 returns Neutron 404 for the QoS policy collection in both Python OSC and the Go CLI, so a cloud exposing this extension is still needed for live verification."
		}
		if strings.HasPrefix(command, "network qos rule ") && !strings.HasPrefix(command, "network qos rule type ") {
			entry.Tests = []string{"unit: command registry", "unit: QoS rule parameter validation"}
			entry.Notes = "Implemented with Gophercloud QoS rule calls for bandwidth-limit, dscp-marking, and minimum-bandwidth. minimum-packet-rate uses a narrow authenticated Neutron request because Gophercloud v2.12.0 does not expose that subtype. cloud6 returns Neutron 404 for the QoS policy collection in both Python OSC and the Go CLI, so a cloud exposing this extension is still needed for live lifecycle verification."
		}
		if strings.HasPrefix(command, "network segment ") {
			entry.Tests = []string{"unit: command registry"}
			entry.Notes = "Implemented with Gophercloud segment calls and custom request builders where extension fields are needed. cloud6 returns Neutron 404 for the segment collection in both Python OSC and the Go CLI, so a cloud exposing this extension is still needed for live lifecycle verification."
		}
		if strings.HasPrefix(command, "network trunk ") || command == "network subport list" {
			entry.Tests = []string{"unit: command registry", "unit: trunk subport parsing"}
			entry.Notes = "Implemented with Gophercloud trunk calls and custom request builders where partial subport dictionaries must match Python OSC. cloud6 returns Neutron 404 for the trunk collection in both Python OSC and the Go CLI, so a cloud exposing this extension is still needed for live lifecycle verification."
		}
	}
	return entry
}

func identityReadPackages() map[string]string {
	base := "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/"
	return map[string]string{
		"access rule list":            base + "applicationcredentials",
		"access rule show":            base + "applicationcredentials",
		"application credential list": base + "applicationcredentials",
		"application credential show": base + "applicationcredentials",
		"catalog list":                base + "catalog",
		"catalog show":                base + "catalog",
		"credential list":             base + "credentials",
		"credential show":             base + "credentials",
		"domain list":                 base + "domains",
		"domain show":                 base + "domains",
		"ec2 credentials list":        base + "ec2credentials",
		"ec2 credentials show":        base + "ec2credentials",
		"endpoint list":               base + "endpoints",
		"endpoint show":               base + "endpoints",
		"federation protocol list":    "Keystone OS-FEDERATION protocol reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"federation protocol show":    "Keystone OS-FEDERATION protocol reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"group list":                  base + "groups",
		"group show":                  base + "groups",
		"identity provider list":      "Keystone OS-FEDERATION identity provider reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"identity provider show":      "Keystone OS-FEDERATION identity provider reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"implied role list":           base + "roles",
		"limit list":                  base + "limits",
		"limit show":                  base + "limits",
		"mapping list":                base + "federation",
		"mapping show":                base + "federation",
		"policy list":                 base + "policies",
		"policy show":                 base + "policies",
		"project list":                base + "projects",
		"project show":                base + "projects",
		"region list":                 base + "regions",
		"region show":                 base + "regions",
		"registered limit list":       base + "registeredlimits",
		"registered limit show":       base + "registeredlimits",
		"role assignment list":        base + "roles",
		"role list":                   base + "roles",
		"role show":                   base + "roles",
		"service list":                base + "services",
		"service provider list":       "Keystone OS-FEDERATION service provider reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"service provider show":       "Keystone OS-FEDERATION service provider reads via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"service show":                base + "services",
		"trust list":                  base + "trusts",
		"trust show":                  base + "trusts",
		"user list":                   base + "users",
		"user show":                   base + "users",
	}
}

func identityReadShims() map[string]bool {
	return map[string]bool{
		"federation protocol list": true,
		"federation protocol show": true,
		"identity provider list":   true,
		"identity provider show":   true,
		"service provider list":    true,
		"service provider show":    true,
	}
}

func identityCloudVerified() map[string]bool {
	return map[string]bool{
		"access rule list":            true,
		"application credential list": true,
		"catalog list":                true,
		"catalog show":                true,
		"credential list":             true,
		"domain list":                 true,
		"domain show":                 true,
		"ec2 credentials list":        true,
		"endpoint list":               true,
		"endpoint show":               true,
		"identity provider list":      true,
		"implied role list":           true,
		"mapping list":                true,
		"group list":                  true,
		"limit list":                  true,
		"policy list":                 true,
		"project list":                true,
		"project show":                true,
		"region list":                 true,
		"region show":                 true,
		"registered limit list":       true,
		"role assignment list":        true,
		"role list":                   true,
		"role show":                   true,
		"service list":                true,
		"service provider list":       true,
		"service show":                true,
		"trust list":                  true,
		"user list":                   true,
		"user show":                   true,
	}
}

func coreReadPackages() map[string]string {
	packages := map[string]string{
		"address group create":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group delete":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group list":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group set":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group show":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group unset":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address scope create":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"address scope delete":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"address scope list":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"address scope set":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"address scope show":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"aggregate list":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates",
		"aggregate show":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates",
		"allocation candidate list":              "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/allocationcandidates",
		"availability zone list":                 "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones; github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones; Neutron availability_zones via gophercloud.ServiceClient",
		"block storage cleanup":                  "Cinder workers cleanup via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage cluster list":             "Cinder clusters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage cluster set":              "Cinder clusters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage cluster show":             "Cinder clusters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage log level list":           "Cinder os-services get-log via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage log level set":            "Cinder os-services set-log via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage resource filter list":     "Cinder resource_filters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage resource filter show":     "Cinder resource_filters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage snapshot manageable list": "Cinder manageable_snapshots via gophercloud.ServiceClient; no typed Gophercloud list helper in v2.12.0",
		"block storage volume manageable list":   "Cinder manageable_volumes via gophercloud.ServiceClient; Gophercloud v2.12.0 has manage-create support but not OSC-shaped manageable list output",
		"cached image clear":                     "Glance image cache via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"cached image delete":                    "Glance image cache via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"cached image list":                      "Glance image cache via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"cached image queue":                     "Glance image cache via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"configuration show":                     "github.com/gophercloud/gophercloud/v2/openstack/config/clouds",
		"compute agent list":                     "Nova os-agents via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"compute service list":                   "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services",
		"consistency group add volume":           "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group create":               "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group delete":               "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group list":                 "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group remove volume":        "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group set":                  "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group show":                 "Cinder consistencygroups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group snapshot create":      "Cinder cgsnapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group snapshot delete":      "Cinder cgsnapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group snapshot list":        "Cinder cgsnapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"consistency group snapshot show":        "Cinder cgsnapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"console connection show":                "Nova os-console-auth-tokens via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"console log show":                       "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"console url show":                       "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles",
		"container create":                       "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"container delete":                       "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers; recursive object deletion via github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"container list":                         "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"container set":                          "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"container show":                         "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"container unset":                        "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"extension list":                         "github.com/gophercloud/gophercloud/v2/openstack/common/extensions",
		"extension show":                         "github.com/gophercloud/gophercloud/v2/openstack/common/extensions",
		"flavor list":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors",
		"flavor show":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors",
		"floating ip create":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips; standard tags and extension fields via gophercloud.ServiceClient",
		"floating ip delete":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"floating ip list":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"floating ip pool list":                  "Neutron floating IP pool compatibility error; Nova-network pool API is not implemented",
		"floating ip port forwarding create":     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding; port range and extension fields via gophercloud.ServiceClient",
		"floating ip port forwarding delete":     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding",
		"floating ip port forwarding list":       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding; raw Neutron list extraction preserves port range fields",
		"floating ip port forwarding set":        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding; port range and extension fields via gophercloud.ServiceClient",
		"floating ip port forwarding show":       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding",
		"floating ip set":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips; standard tags and extension fields via gophercloud.ServiceClient",
		"floating ip show":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"floating ip unset":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips; standard tags and extension fields via gophercloud.ServiceClient",
		"host list":                              "Nova os-hosts via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"host show":                              "Nova os-hosts via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"hypervisor list":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"hypervisor show":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"hypervisor stats show":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"image add project":                      "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image create":                           "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images; github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata; github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport",
		"image delete":                           "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images; Glance multi-store delete via gophercloud.ServiceClient",
		"image import":                           "Glance image import via gophercloud.ServiceClient; typed Gophercloud imageimport helper does not cover stores or remote import options in v2.12.0",
		"image import info":                      "github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport",
		"image list":                             "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images",
		"image member get":                       "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image member list":                      "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image metadef namespace list":           "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image metadef namespace show":           "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image metadef resource type list":       "Glance metadef resource types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image show":                             "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images",
		"image remove project":                   "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image save":                             "github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata",
		"image set":                              "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images; github.com/gophercloud/gophercloud/v2/openstack/image/v2/members; image activation via gophercloud.ServiceClient",
		"image stage":                            "github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata",
		"image unset":                            "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images; image tag delete via gophercloud.ServiceClient",
		"image stores list":                      "Image store discovery via gophercloud.ServiceClient",
		"image task list":                        "github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks",
		"image task show":                        "github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks",
		"ip availability list":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities",
		"ip availability show":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities",
		"keypair create":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"keypair delete":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"keypair list":                           "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"keypair show":                           "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"limits show":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits; github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits",
		"network agent list":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents",
		"network agent show":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents",
		"network create":                         "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks; standard tags via gophercloud.ServiceClient",
		"network delete":                         "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks",
		"network list":                           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks",
		"network qos policy create":              "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies; false booleans and extension fields via custom request builders",
		"network qos policy delete":              "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos policy list":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos policy set":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies; false booleans and extension fields via custom request builders",
		"network qos policy show":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos rule create":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules; minimum-packet-rate via authenticated Neutron request because Gophercloud v2.12.0 lacks that subtype",
		"network qos rule delete":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules; minimum-packet-rate via authenticated Neutron request because Gophercloud v2.12.0 lacks that subtype",
		"network qos rule list":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos rule set":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules; minimum-packet-rate via authenticated Neutron request because Gophercloud v2.12.0 lacks that subtype",
		"network qos rule show":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules; minimum-packet-rate via authenticated Neutron request because Gophercloud v2.12.0 lacks that subtype",
		"network qos rule type list":             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes",
		"network qos rule type show":             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes",
		"network rbac create":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies; owner project and extension fields via custom request builders",
		"network rbac delete":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies",
		"network rbac list":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies",
		"network rbac set":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies; extension fields via custom request builders",
		"network rbac show":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies",
		"network segment create":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments; extension fields via custom request builders",
		"network segment delete":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments",
		"network set":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks; standard tags via gophercloud.ServiceClient",
		"network service provider list":          "Neutron service-providers via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"network segment list":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments",
		"network segment set":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments; extension fields via custom request builders",
		"network segment show":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments",
		"network subport list":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"network trunk create":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks; subport request shape via custom request builders",
		"network trunk delete":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"network show":                           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks",
		"network trunk set":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks; subport request shape via custom request builders",
		"network trunk unset":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks; subport request shape via custom request builders",
		"network trunk list":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"network trunk show":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"network unset":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks; standard tags via gophercloud.ServiceClient",
		"object create":                          "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object delete":                          "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object list":                            "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object save":                            "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object set":                             "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object show":                            "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object unset":                           "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object store account show":              "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/accounts",
		"port create":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports; standard tags and extension fields via gophercloud.ServiceClient",
		"port delete":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"port list":                              "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"port set":                               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports; standard tags and extension fields via gophercloud.ServiceClient",
		"port show":                              "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"port unset":                             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports; standard tags and extension fields via gophercloud.ServiceClient",
		"quota delete":                           "Compute, Volume, and Network quota reset via gophercloud.ServiceClient",
		"quota list":                             "Compute/Volume/Network quota reads via gophercloud.ServiceClient; typed SDK packages are incomplete for OSC-shaped aggregate output",
		"quota set":                              "Compute, Volume, and Network quota update via gophercloud.ServiceClient",
		"quota show":                             "Compute/Volume/Network quota reads via gophercloud.ServiceClient; typed SDK packages are incomplete for OSC-shaped aggregate output",
		"resource class list":                    "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses",
		"resource class show":                    "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses",
		"resource provider aggregate list":       "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider allocation show":      "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider inventory list":       "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider inventory show":       "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider list":                 "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider show":                 "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider trait list":           "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider usage show":           "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"router add gateway":                     "Neutron router add_external_gateways action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"router add port":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router add route":                       "Neutron router add_extraroutes action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"router add subnet":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router create":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers; standard tags via gophercloud.ServiceClient",
		"router delete":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router list":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router remove gateway":                  "Neutron router remove_external_gateways action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"router remove port":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router remove route":                    "Neutron router remove_extraroutes action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"router remove subnet":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router set":                             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers; standard tags via gophercloud.ServiceClient",
		"router show":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router unset":                           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers; standard tags via gophercloud.ServiceClient",
		"security group create":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups; standard tags via gophercloud.ServiceClient",
		"security group delete":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups",
		"security group list":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups",
		"security group set":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups; standard tags via gophercloud.ServiceClient",
		"security group show":                    "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups",
		"security group unset":                   "Neutron standard tags via gophercloud.ServiceClient",
		"security group rule create":             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"security group rule delete":             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"security group rule list":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"security group rule show":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"server add fixed ip":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces",
		"server add floating ip":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips; github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"server add network":                     "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces",
		"server add port":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces",
		"server add security group":              "Nova server addSecurityGroup action via gophercloud.ServiceClient",
		"server add volume":                      "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach",
		"server backup create":                   "Nova createBackup action via gophercloud.ServiceClient",
		"server create":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server delete":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server dump create":                     "Nova trigger_crash_dump action via gophercloud.ServiceClient",
		"server evacuate":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server event list":                      "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions",
		"server event show":                      "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions",
		"server group create":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server group delete":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server group list":                      "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server group show":                      "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server image create":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server list":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server lock":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers; lock reason via gophercloud.ServiceClient",
		"server migrate":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers; cold migrate host via gophercloud.ServiceClient",
		"server migrate confirm":                 "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server migrate revert":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server migration abort":                 "Nova server migration action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"server migration confirm":               "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server migration force complete":        "Nova server migration action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"server migration list":                  "Nova os-migrations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"server migration revert":                "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server migration show":                  "Nova server migrations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"server pause":                           "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server reboot":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server rebuild":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server remove fixed ip":                 "Nova removeFixedIp action via gophercloud.ServiceClient",
		"server remove floating ip":              "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"server remove network":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces; github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"server remove port":                     "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces",
		"server remove security group":           "Nova server removeSecurityGroup action via gophercloud.ServiceClient",
		"server remove volume":                   "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach",
		"server rescue":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server resize":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server resize confirm":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server resize revert":                   "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server restore":                         "Nova restore action via gophercloud.ServiceClient",
		"server resume":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server set":                             "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers; github.com/gophercloud/gophercloud/v2/openstack/compute/v2/tags",
		"server shelve":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server show":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server start":                           "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server stop":                            "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server suspend":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server unlock":                          "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server unpause":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server unrescue":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server unset":                           "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers; github.com/gophercloud/gophercloud/v2/openstack/compute/v2/tags",
		"server unshelve":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers; host/no-availability-zone via gophercloud.ServiceClient",
		"server volume list":                     "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach",
		"server volume set":                      "Nova server volume attachment update via gophercloud.ServiceClient",
		"server volume update":                   "Nova server volume attachment update via gophercloud.ServiceClient",
		"subnet create":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets; standard tags via gophercloud.ServiceClient",
		"subnet delete":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets",
		"subnet list":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets",
		"subnet pool create":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools; standard tags via gophercloud.ServiceClient",
		"subnet pool delete":                     "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools",
		"subnet pool list":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools",
		"subnet pool set":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools; standard tags via gophercloud.ServiceClient",
		"subnet pool show":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools",
		"subnet pool unset":                      "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools; standard tags via gophercloud.ServiceClient",
		"subnet set":                             "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets; standard tags via gophercloud.ServiceClient",
		"subnet show":                            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets",
		"subnet unset":                           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets; standard tags via gophercloud.ServiceClient",
		"trait list":                             "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits",
		"trait show":                             "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits",
		"usage list":                             "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage",
		"usage show":                             "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage",
		"versions show":                          "Service catalog version discovery via gophercloud.ProviderClient",
		"volume attachment complete":             "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment create":               "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment delete":               "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment list":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment set":                  "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment show":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume backend capability show":         "Cinder capabilities via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume backend pool list":               "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats",
		"volume backup create":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup delete":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup list":                     "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup record export":            "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup record import":            "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup restore":                  "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup set":                      "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup show":                     "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup unset":                    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume create":                          "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes; manageable remote-source via gophercloud.ServiceClient",
		"volume delete":                          "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume group create":                    "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group delete":                    "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group failover":                  "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group list":                      "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group set":                       "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group show":                      "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot create":           "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot delete":           "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot list":             "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot show":             "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type create":               "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type delete":               "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type list":                 "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type set":                  "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type show":                 "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume host set":                        "Cinder os-services freeze/thaw via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume list":                            "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume message delete":                  "Cinder messages via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume message list":                    "Cinder messages via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume message show":                    "Cinder messages via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume migrate":                         "Cinder volume migrate action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume qos associate":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos create":                      "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos delete":                      "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos disassociate":                "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos list":                        "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos set":                         "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos show":                        "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos unset":                       "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume revert":                          "Cinder revert volume action via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume service list":                    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services",
		"volume service set":                     "Cinder os-services enable/disable via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume set":                             "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume show":                            "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume snapshot create":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots; manageable remote-source via gophercloud.ServiceClient",
		"volume snapshot delete":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume snapshot list":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume snapshot set":                    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume snapshot show":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume snapshot unset":                  "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume summary":                         "Cinder volume summary via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume transfer request accept":         "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume transfer request create":         "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume transfer request delete":         "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume transfer request list":           "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume transfer request show":           "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume type create":                     "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type delete":                     "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type list":                       "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type set":                        "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type show":                       "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type unset":                      "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume unset":                           "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
	}
	packages["image metadef namespace create"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef namespace delete"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef namespace set"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object create"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object delete"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object list"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object property show"] = "Glance metadef object properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object show"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object update"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property create"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property delete"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property list"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property set"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property show"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association create"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association delete"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association list"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	return packages
}

func coreReadShims() map[string]bool {
	shims := map[string]bool{
		"availability zone list":                 true,
		"block storage cleanup":                  true,
		"block storage cluster list":             true,
		"block storage cluster set":              true,
		"block storage cluster show":             true,
		"block storage log level list":           true,
		"block storage log level set":            true,
		"block storage resource filter list":     true,
		"block storage resource filter show":     true,
		"block storage snapshot manageable list": true,
		"block storage volume manageable list":   true,
		"cached image clear":                     true,
		"cached image delete":                    true,
		"cached image list":                      true,
		"cached image queue":                     true,
		"configuration show":                     true,
		"compute agent list":                     true,
		"consistency group add volume":           true,
		"consistency group create":               true,
		"consistency group delete":               true,
		"consistency group list":                 true,
		"consistency group remove volume":        true,
		"consistency group set":                  true,
		"consistency group show":                 true,
		"consistency group snapshot create":      true,
		"consistency group snapshot delete":      true,
		"consistency group snapshot list":        true,
		"consistency group snapshot show":        true,
		"console connection show":                true,
		"floating ip pool list":                  true,
		"host list":                              true,
		"host show":                              true,
		"image delete":                           true,
		"image import":                           true,
		"image set":                              true,
		"image stores list":                      true,
		"image unset":                            true,
		"image metadef namespace list":           true,
		"image metadef namespace show":           true,
		"image metadef resource type list":       true,
		"network service provider list":          true,
		"quota delete":                           true,
		"quota list":                             true,
		"quota set":                              true,
		"quota show":                             true,
		"router add gateway":                     true,
		"router add route":                       true,
		"router remove gateway":                  true,
		"router remove route":                    true,
		"server add security group":              true,
		"server backup create":                   true,
		"server dump create":                     true,
		"server lock":                            true,
		"server migrate":                         true,
		"server migration abort":                 true,
		"server migration force complete":        true,
		"server migration list":                  true,
		"server migration show":                  true,
		"server remove fixed ip":                 true,
		"server remove security group":           true,
		"server restore":                         true,
		"server unshelve":                        true,
		"server volume set":                      true,
		"server volume update":                   true,
		"volume backend capability show":         true,
		"volume create":                          true,
		"volume group create":                    true,
		"volume group delete":                    true,
		"volume group failover":                  true,
		"volume group list":                      true,
		"volume group set":                       true,
		"volume group show":                      true,
		"volume group snapshot create":           true,
		"volume group snapshot delete":           true,
		"volume group snapshot list":             true,
		"volume group snapshot show":             true,
		"volume group type create":               true,
		"volume group type delete":               true,
		"volume group type list":                 true,
		"volume group type set":                  true,
		"volume group type show":                 true,
		"volume host set":                        true,
		"volume message delete":                  true,
		"volume message list":                    true,
		"volume message show":                    true,
		"volume migrate":                         true,
		"volume revert":                          true,
		"volume service set":                     true,
		"volume snapshot create":                 true,
		"volume summary":                         true,
		"versions show":                          true,
	}
	shims["image metadef namespace create"] = true
	shims["image metadef namespace delete"] = true
	shims["image metadef namespace set"] = true
	shims["image metadef object create"] = true
	shims["image metadef object delete"] = true
	shims["image metadef object list"] = true
	shims["image metadef object property show"] = true
	shims["image metadef object show"] = true
	shims["image metadef object update"] = true
	shims["image metadef property create"] = true
	shims["image metadef property delete"] = true
	shims["image metadef property list"] = true
	shims["image metadef property set"] = true
	shims["image metadef property show"] = true
	shims["image metadef resource type association create"] = true
	shims["image metadef resource type association delete"] = true
	shims["image metadef resource type association list"] = true
	return shims
}

func coreCloudVerified() map[string]bool {
	verified := map[string]bool{
		"address group create":               true,
		"address group delete":               true,
		"address group list":                 true,
		"address group set":                  true,
		"address group unset":                true,
		"address scope create":               true,
		"address scope delete":               true,
		"address scope list":                 true,
		"address scope set":                  true,
		"address scope show":                 true,
		"aggregate list":                     true,
		"allocation candidate list":          true,
		"availability zone list":             true,
		"block storage cluster list":         true,
		"block storage log level list":       true,
		"block storage resource filter list": true,
		"block storage resource filter show": true,
		"cached image clear":                 true,
		"cached image delete":                true,
		"cached image list":                  true,
		"cached image queue":                 true,
		"compute agent list":                 true,
		"compute service list":               true,
		"console connection show":            true,
		"console log show":                   true,
		"console url show":                   true,
		"container create":                   true,
		"container delete":                   true,
		"container list":                     true,
		"container set":                      true,
		"container show":                     true,
		"container unset":                    true,
		"extension list":                     true,
		"extension show":                     true,
		"flavor list":                        true,
		"flavor show":                        true,
		"floating ip create":                 true,
		"floating ip delete":                 true,
		"floating ip list":                   true,
		"floating ip pool list":              true,
		"floating ip set":                    true,
		"floating ip show":                   true,
		"floating ip unset":                  true,
		"host list":                          true,
		"host show":                          true,
		"hypervisor list":                    true,
		"hypervisor show":                    true,
		"hypervisor stats show":              true,
		"image create":                       true,
		"image delete":                       true,
		"image import":                       true,
		"image import info":                  true,
		"image list":                         true,
		"image member list":                  true,
		"image metadef namespace list":       true,
		"image metadef namespace show":       true,
		"image metadef resource type list":   true,
		"image show":                         true,
		"image save":                         true,
		"image set":                          true,
		"image stage":                        true,
		"image unset":                        true,
		"image stores list":                  true,
		"image task list":                    true,
		"ip availability list":               true,
		"ip availability show":               true,
		"keypair create":                     true,
		"keypair delete":                     true,
		"keypair list":                       true,
		"keypair show":                       true,
		"limits show":                        true,
		"network agent list":                 true,
		"network agent show":                 true,
		"network create":                     true,
		"network delete":                     true,
		"network list":                       true,
		"network rbac list":                  true,
		"network rbac show":                  true,
		"network rbac create":                true,
		"network rbac delete":                true,
		"network rbac set":                   true,
		"network set":                        true,
		"network service provider list":      true,
		"network show":                       true,
		"network unset":                      true,
		"object list":                        true,
		"object create":                      true,
		"object delete":                      true,
		"object save":                        true,
		"object set":                         true,
		"object show":                        true,
		"object unset":                       true,
		"object store account show":          true,
		"port create":                        true,
		"port delete":                        true,
		"port list":                          true,
		"port set":                           true,
		"port show":                          true,
		"port unset":                         true,
		"quota list":                         true,
		"quota show":                         true,
		"resource class list":                true,
		"resource class show":                true,
		"resource provider aggregate list":   true,
		"resource provider allocation show":  true,
		"resource provider inventory list":   true,
		"resource provider inventory show":   true,
		"resource provider list":             true,
		"resource provider show":             true,
		"resource provider trait list":       true,
		"resource provider usage show":       true,
		"router add port":                    true,
		"router add route":                   true,
		"router add subnet":                  true,
		"router create":                      true,
		"router delete":                      true,
		"router list":                        true,
		"router remove port":                 true,
		"router remove route":                true,
		"router remove subnet":               true,
		"router set":                         true,
		"router show":                        true,
		"router unset":                       true,
		"security group list":                true,
		"security group show":                true,
		"security group create":              true,
		"security group delete":              true,
		"security group set":                 true,
		"security group unset":               true,
		"security group rule create":         true,
		"security group rule delete":         true,
		"security group rule list":           true,
		"security group rule show":           true,
		"server event list":                  true,
		"server event show":                  true,
		"server group create":                true,
		"server group delete":                true,
		"server list":                        true,
		"server show":                        true,
		"server group list":                  true,
		"server group show":                  true,
		"server volume list":                 true,
		"server migration list":              true,
		"subnet create":                      true,
		"subnet delete":                      true,
		"subnet list":                        true,
		"subnet pool create":                 true,
		"subnet pool delete":                 true,
		"subnet pool list":                   true,
		"subnet pool set":                    true,
		"subnet pool show":                   true,
		"subnet pool unset":                  true,
		"subnet set":                         true,
		"subnet show":                        true,
		"subnet unset":                       true,
		"trait list":                         true,
		"trait show":                         true,
		"usage list":                         true,
		"usage show":                         true,
		"versions show":                      true,
		"volume attachment list":             true,
		"volume attachment show":             true,
		"volume backend pool list":           true,
		"volume backup list":                 true,
		"volume group list":                  true,
		"volume group snapshot list":         true,
		"volume group type list":             true,
		"volume list":                        true,
		"volume message list":                true,
		"volume message show":                true,
		"volume qos list":                    true,
		"volume qos show":                    true,
		"volume summary":                     true,
		"volume service list":                true,
		"volume show":                        true,
		"volume snapshot list":               true,
		"volume transfer request list":       true,
		"volume type list":                   true,
		"volume type show":                   true,
	}
	verified["image metadef namespace create"] = true
	verified["image metadef namespace delete"] = true
	verified["image metadef namespace set"] = true
	verified["image metadef object create"] = true
	verified["image metadef object delete"] = true
	verified["image metadef object list"] = true
	verified["image metadef object property show"] = true
	verified["image metadef object show"] = true
	verified["image metadef object update"] = true
	verified["image metadef property create"] = true
	verified["image metadef property delete"] = true
	verified["image metadef property list"] = true
	verified["image metadef property set"] = true
	verified["image metadef property show"] = true
	verified["image metadef resource type association create"] = true
	verified["image metadef resource type association delete"] = true
	verified["image metadef resource type association list"] = true
	return verified
}

func coreGoldenMatched() map[string]bool {
	return map[string]bool{
		"flavor list":  true,
		"image list":   true,
		"network list": true,
	}
}

func coreWriteCommands() map[string]bool {
	return map[string]bool{
		"address group create":               true,
		"address group delete":               true,
		"address group set":                  true,
		"address group unset":                true,
		"address scope create":               true,
		"address scope delete":               true,
		"address scope set":                  true,
		"image metadef namespace create":     true,
		"cached image clear":                 true,
		"cached image delete":                true,
		"cached image queue":                 true,
		"image add project":                  true,
		"container create":                   true,
		"container delete":                   true,
		"container set":                      true,
		"container unset":                    true,
		"object create":                      true,
		"object delete":                      true,
		"object save":                        true,
		"object set":                         true,
		"object unset":                       true,
		"image create":                       true,
		"image delete":                       true,
		"image import":                       true,
		"image metadef namespace delete":     true,
		"image metadef namespace set":        true,
		"image metadef object create":        true,
		"image metadef object delete":        true,
		"image metadef object update":        true,
		"image metadef property create":      true,
		"image metadef property delete":      true,
		"image metadef property set":         true,
		"image remove project":               true,
		"floating ip create":                 true,
		"floating ip delete":                 true,
		"floating ip port forwarding create": true,
		"floating ip port forwarding delete": true,
		"floating ip port forwarding set":    true,
		"floating ip set":                    true,
		"floating ip unset":                  true,
		"keypair create":                     true,
		"keypair delete":                     true,
		"network create":                     true,
		"network delete":                     true,
		"network qos policy create":          true,
		"network qos policy delete":          true,
		"network qos policy set":             true,
		"network qos rule create":            true,
		"network qos rule delete":            true,
		"network qos rule set":               true,
		"network rbac create":                true,
		"network rbac delete":                true,
		"network rbac set":                   true,
		"network segment create":             true,
		"network segment delete":             true,
		"network segment set":                true,
		"network set":                        true,
		"network trunk create":               true,
		"network trunk delete":               true,
		"network trunk set":                  true,
		"network trunk unset":                true,
		"network unset":                      true,
		"port create":                        true,
		"port delete":                        true,
		"port set":                           true,
		"port unset":                         true,
		"quota delete":                       true,
		"quota set":                          true,
		"router add gateway":                 true,
		"router add port":                    true,
		"router add route":                   true,
		"router add subnet":                  true,
		"router create":                      true,
		"router delete":                      true,
		"router remove gateway":              true,
		"router remove port":                 true,
		"router remove route":                true,
		"router remove subnet":               true,
		"router set":                         true,
		"router unset":                       true,
		"security group create":              true,
		"security group delete":              true,
		"security group set":                 true,
		"security group unset":               true,
		"security group rule create":         true,
		"security group rule delete":         true,
		"server add fixed ip":                true,
		"server add floating ip":             true,
		"server add network":                 true,
		"server add port":                    true,
		"server add security group":          true,
		"server add volume":                  true,
		"server backup create":               true,
		"server create":                      true,
		"server delete":                      true,
		"server dump create":                 true,
		"server evacuate":                    true,
		"server group create":                true,
		"server group delete":                true,
		"server image create":                true,
		"server lock":                        true,
		"server migrate":                     true,
		"server migrate confirm":             true,
		"server migrate revert":              true,
		"server migration abort":             true,
		"server migration confirm":           true,
		"server migration force complete":    true,
		"server migration revert":            true,
		"server pause":                       true,
		"server reboot":                      true,
		"server rebuild":                     true,
		"server remove fixed ip":             true,
		"server remove floating ip":          true,
		"server remove network":              true,
		"server remove port":                 true,
		"server remove security group":       true,
		"server remove volume":               true,
		"server rescue":                      true,
		"server resize":                      true,
		"server resize confirm":              true,
		"server resize revert":               true,
		"server restore":                     true,
		"server resume":                      true,
		"server set":                         true,
		"server shelve":                      true,
		"server start":                       true,
		"server stop":                        true,
		"server suspend":                     true,
		"server unlock":                      true,
		"server unpause":                     true,
		"server unrescue":                    true,
		"server unset":                       true,
		"server unshelve":                    true,
		"server volume set":                  true,
		"server volume update":               true,
		"subnet create":                      true,
		"subnet delete":                      true,
		"subnet pool create":                 true,
		"subnet pool delete":                 true,
		"subnet pool set":                    true,
		"subnet pool unset":                  true,
		"subnet set":                         true,
		"subnet unset":                       true,
		"image set":                          true,
		"image stage":                        true,
		"image unset":                        true,
		"image metadef resource type association create": true,
		"image metadef resource type association delete": true,
		"block storage cleanup":                          true,
		"block storage cluster set":                      true,
		"block storage log level set":                    true,
		"consistency group add volume":                   true,
		"consistency group create":                       true,
		"consistency group delete":                       true,
		"consistency group remove volume":                true,
		"consistency group set":                          true,
		"consistency group snapshot create":              true,
		"consistency group snapshot delete":              true,
		"volume attachment complete":                     true,
		"volume attachment create":                       true,
		"volume attachment delete":                       true,
		"volume attachment set":                          true,
		"volume backup create":                           true,
		"volume backup delete":                           true,
		"volume backup record export":                    true,
		"volume backup record import":                    true,
		"volume backup restore":                          true,
		"volume backup set":                              true,
		"volume backup unset":                            true,
		"volume create":                                  true,
		"volume delete":                                  true,
		"volume group create":                            true,
		"volume group delete":                            true,
		"volume group failover":                          true,
		"volume group set":                               true,
		"volume group snapshot create":                   true,
		"volume group snapshot delete":                   true,
		"volume group type create":                       true,
		"volume group type delete":                       true,
		"volume group type set":                          true,
		"volume host set":                                true,
		"volume message delete":                          true,
		"volume migrate":                                 true,
		"volume qos associate":                           true,
		"volume qos create":                              true,
		"volume qos delete":                              true,
		"volume qos disassociate":                        true,
		"volume qos set":                                 true,
		"volume qos unset":                               true,
		"volume revert":                                  true,
		"volume service set":                             true,
		"volume set":                                     true,
		"volume snapshot create":                         true,
		"volume snapshot delete":                         true,
		"volume snapshot set":                            true,
		"volume snapshot unset":                          true,
		"volume transfer request accept":                 true,
		"volume transfer request create":                 true,
		"volume transfer request delete":                 true,
		"volume type create":                             true,
		"volume type delete":                             true,
		"volume type set":                                true,
		"volume type unset":                              true,
		"volume unset":                                   true,
	}
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
