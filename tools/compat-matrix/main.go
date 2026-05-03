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
		entry.Status = "implemented"
		entry.ImplementedIn = "internal/cli"
		entry.Notes = "Initial implementation uses the embedded OSC catalog and marks unfinished commands."
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
		"address group list":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address group show":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups",
		"address scope list":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"address scope show":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes",
		"aggregate list":                     "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates",
		"aggregate show":                     "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates",
		"allocation candidate list":          "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/allocationcandidates",
		"availability zone list":             "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones; github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones; Neutron availability_zones via gophercloud.ServiceClient",
		"block storage cluster list":         "Cinder clusters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage cluster show":         "Cinder clusters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage log level list":       "Cinder os-services get-log via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage resource filter list": "Cinder resource_filters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"block storage resource filter show": "Cinder resource_filters via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"cached image list":                  "Glance image cache via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"compute agent list":                 "Nova os-agents via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"compute service list":               "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services",
		"console connection show":            "Nova os-console-auth-tokens via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"console log show":                   "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"console url show":                   "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles",
		"container list":                     "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"container show":                     "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers",
		"extension list":                     "github.com/gophercloud/gophercloud/v2/openstack/common/extensions",
		"extension show":                     "github.com/gophercloud/gophercloud/v2/openstack/common/extensions",
		"flavor list":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors",
		"flavor show":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors",
		"floating ip list":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"floating ip pool list":              "Neutron floating IP pool compatibility error; Nova-network pool API is not implemented",
		"floating ip show":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips",
		"host list":                          "Nova os-hosts via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"host show":                          "Nova os-hosts via gophercloud.ServiceClient; OpenStackSDK intentionally does not support the deprecated API",
		"hypervisor list":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"hypervisor show":                    "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"hypervisor stats show":              "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors",
		"image list":                         "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images",
		"image member get":                   "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image member list":                  "github.com/gophercloud/gophercloud/v2/openstack/image/v2/members",
		"image metadef namespace list":       "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image metadef namespace show":       "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image metadef resource type list":   "Glance metadef resource types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"image show":                         "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images",
		"image stores list":                  "Image store discovery via gophercloud.ServiceClient",
		"image task list":                    "github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks",
		"image task show":                    "github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks",
		"ip availability list":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities",
		"ip availability show":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities",
		"keypair list":                       "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"keypair show":                       "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs",
		"limits show":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits; github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits",
		"network agent list":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents",
		"network agent show":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents",
		"network list":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks",
		"network qos policy list":            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos policy show":            "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies",
		"network qos rule type list":         "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes",
		"network qos rule type show":         "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes",
		"network rbac list":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies",
		"network rbac show":                  "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies",
		"network service provider list":      "Neutron service-providers via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"network segment list":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments",
		"network segment show":               "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments",
		"network show":                       "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks",
		"network trunk list":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"network trunk show":                 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks",
		"object list":                        "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object show":                        "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects",
		"object store account show":          "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/accounts",
		"port list":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"port show":                          "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports",
		"quota list":                         "Compute/Volume/Network quota reads via gophercloud.ServiceClient; typed SDK packages are incomplete for OSC-shaped aggregate output",
		"quota show":                         "Compute/Volume/Network quota reads via gophercloud.ServiceClient; typed SDK packages are incomplete for OSC-shaped aggregate output",
		"resource class list":                "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses",
		"resource class show":                "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses",
		"resource provider aggregate list":   "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider allocation show":  "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider inventory list":   "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider inventory show":   "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider list":             "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider show":             "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider trait list":       "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"resource provider usage show":       "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders",
		"router list":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"router show":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers",
		"security group list":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups",
		"security group show":                "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups",
		"security group rule list":           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"security group rule show":           "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules",
		"server event list":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions",
		"server event show":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions",
		"server list":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server show":                        "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers",
		"server group list":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server group show":                  "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups",
		"server migration list":              "Nova os-migrations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"server volume list":                 "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach",
		"subnet list":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets",
		"subnet pool list":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools",
		"subnet pool show":                   "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools",
		"subnet show":                        "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets",
		"trait list":                         "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits",
		"trait show":                         "github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits",
		"usage list":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage",
		"usage show":                         "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage",
		"versions show":                      "Service catalog version discovery via gophercloud.ProviderClient",
		"volume attachment list":             "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume attachment show":             "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments",
		"volume backend pool list":           "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats",
		"volume backup list":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume backup show":                 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups",
		"volume group list":                  "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group show":                  "Cinder groups via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot list":         "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group snapshot show":         "Cinder group_snapshots via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type list":             "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume group type show":             "Cinder group_types via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume list":                        "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume message list":                "Cinder messages via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume message show":                "Cinder messages via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume qos list":                    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume qos show":                    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos",
		"volume service list":                "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services",
		"volume show":                        "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes",
		"volume snapshot list":               "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume snapshot show":               "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots",
		"volume summary":                     "Cinder volume summary via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0",
		"volume transfer request list":       "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume transfer request show":       "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers",
		"volume type list":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
		"volume type show":                   "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes",
	}
	packages["image metadef namespace create"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef namespace delete"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef namespace set"] = "Glance metadef namespaces via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object list"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object property show"] = "Glance metadef object properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef object show"] = "Glance metadef objects via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property list"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef property show"] = "Glance metadef namespace properties via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association create"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association delete"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	packages["image metadef resource type association list"] = "Glance metadef namespace resource type associations via gophercloud.ServiceClient; no typed Gophercloud helper in v2.12.0"
	return packages
}

func coreReadShims() map[string]bool {
	shims := map[string]bool{
		"availability zone list":             true,
		"block storage cluster list":         true,
		"block storage cluster show":         true,
		"block storage log level list":       true,
		"block storage resource filter list": true,
		"block storage resource filter show": true,
		"cached image list":                  true,
		"compute agent list":                 true,
		"console connection show":            true,
		"floating ip pool list":              true,
		"host list":                          true,
		"host show":                          true,
		"image stores list":                  true,
		"image metadef namespace list":       true,
		"image metadef namespace show":       true,
		"image metadef resource type list":   true,
		"network service provider list":      true,
		"quota list":                         true,
		"quota show":                         true,
		"server migration list":              true,
		"volume group list":                  true,
		"volume group show":                  true,
		"volume group snapshot list":         true,
		"volume group snapshot show":         true,
		"volume group type list":             true,
		"volume group type show":             true,
		"volume message list":                true,
		"volume message show":                true,
		"volume summary":                     true,
		"versions show":                      true,
	}
	shims["image metadef namespace create"] = true
	shims["image metadef namespace delete"] = true
	shims["image metadef namespace set"] = true
	shims["image metadef object list"] = true
	shims["image metadef object property show"] = true
	shims["image metadef object show"] = true
	shims["image metadef property list"] = true
	shims["image metadef property show"] = true
	shims["image metadef resource type association create"] = true
	shims["image metadef resource type association delete"] = true
	shims["image metadef resource type association list"] = true
	return shims
}

func coreCloudVerified() map[string]bool {
	verified := map[string]bool{
		"address group list":                 true,
		"address scope list":                 true,
		"aggregate list":                     true,
		"allocation candidate list":          true,
		"availability zone list":             true,
		"block storage cluster list":         true,
		"block storage log level list":       true,
		"block storage resource filter list": true,
		"block storage resource filter show": true,
		"cached image list":                  true,
		"compute agent list":                 true,
		"compute service list":               true,
		"console connection show":            true,
		"console log show":                   true,
		"console url show":                   true,
		"container list":                     true,
		"container show":                     true,
		"extension list":                     true,
		"extension show":                     true,
		"flavor list":                        true,
		"flavor show":                        true,
		"floating ip list":                   true,
		"floating ip pool list":              true,
		"floating ip show":                   true,
		"host list":                          true,
		"host show":                          true,
		"hypervisor list":                    true,
		"hypervisor show":                    true,
		"hypervisor stats show":              true,
		"image list":                         true,
		"image member list":                  true,
		"image metadef namespace list":       true,
		"image metadef namespace show":       true,
		"image metadef resource type list":   true,
		"image show":                         true,
		"image stores list":                  true,
		"image task list":                    true,
		"ip availability list":               true,
		"ip availability show":               true,
		"keypair list":                       true,
		"keypair show":                       true,
		"limits show":                        true,
		"network agent list":                 true,
		"network agent show":                 true,
		"network list":                       true,
		"network rbac list":                  true,
		"network rbac show":                  true,
		"network service provider list":      true,
		"network show":                       true,
		"object list":                        true,
		"object show":                        true,
		"object store account show":          true,
		"port list":                          true,
		"port show":                          true,
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
		"router list":                        true,
		"router show":                        true,
		"security group list":                true,
		"security group show":                true,
		"security group rule list":           true,
		"security group rule show":           true,
		"server event list":                  true,
		"server event show":                  true,
		"server list":                        true,
		"server show":                        true,
		"server group list":                  true,
		"server volume list":                 true,
		"server migration list":              true,
		"subnet list":                        true,
		"subnet pool list":                   true,
		"subnet show":                        true,
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
	verified["image metadef object list"] = true
	verified["image metadef object property show"] = true
	verified["image metadef object show"] = true
	verified["image metadef property list"] = true
	verified["image metadef property show"] = true
	verified["image metadef resource type association create"] = true
	verified["image metadef resource type association delete"] = true
	verified["image metadef resource type association list"] = true
	return verified
}

func coreWriteCommands() map[string]bool {
	return map[string]bool{
		"image metadef namespace create":                 true,
		"image metadef namespace delete":                 true,
		"image metadef namespace set":                    true,
		"image metadef resource type association create": true,
		"image metadef resource type association delete": true,
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
