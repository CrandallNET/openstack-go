package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/crandallnet/golang-osc/compat/osc"
	"github.com/spf13/cobra"
)

type commandHandler func(*cobra.Command, []string) error

type commandRegistry struct {
	groups      []osc.CommandGroup
	implemented map[string]commandHandler
	stdout      io.Writer
}

func newCommandRegistry(groups []osc.CommandGroup, stdout io.Writer, opts *Options) *commandRegistry {
	registry := &commandRegistry{
		groups:      groups,
		implemented: map[string]commandHandler{},
		stdout:      stdout,
	}
	registry.implemented["command list"] = runCommandList(groups, stdout, opts, registry.implemented)
	registry.implemented["module list"] = runModuleList(stdout, opts)
	registry.implemented["token issue"] = runTokenIssue(stdout, opts)
	for _, path := range []string{
		"catalog list", "catalog show",
		"domain list", "domain show",
		"endpoint list", "endpoint show",
		"group list", "group show",
		"project list", "project show",
		"region list", "region show",
		"role list", "role show",
		"service list", "service show",
		"user list", "user show",
	} {
		registry.implemented[path] = runIdentityRead(path, stdout, opts)
	}
	for _, path := range []string{
		"allocation candidate list",
		"availability zone list",
		"container list", "container show",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip list", "floating ip show",
		"image list", "image show",
		"keypair list", "keypair show",
		"limits show",
		"network list", "network show",
		"object list", "object show",
		"object store account show",
		"port list", "port show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"server list", "server show",
		"server group list", "server group show",
		"subnet list", "subnet show",
		"trait list", "trait show",
		"volume list", "volume show",
		"volume snapshot list", "volume snapshot show",
		"volume type list", "volume type show",
	} {
		registry.implemented[path] = runCoreRead(path, stdout, opts)
	}
	return registry
}

func (r *commandRegistry) addCatalogCommands(root *cobra.Command) {
	for _, group := range r.groups {
		for _, command := range group.Commands {
			r.addCommand(root, command)
		}
	}
}

func (r *commandRegistry) addCommand(root *cobra.Command, path string) {
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return
	}

	parent := root
	for i, part := range parts {
		fullPath := strings.Join(parts[:i+1], " ")
		child := findChild(parent, part)
		if child == nil {
			child = &cobra.Command{
				Use:           part,
				Short:         fullPath,
				SilenceUsage:  true,
				SilenceErrors: true,
			}
			child.Flags().SortFlags = false
			child.PersistentFlags().SortFlags = false
			parent.AddCommand(child)
		}

		if i == len(parts)-1 {
			r.configureLeaf(child, fullPath)
		}
		parent = child
	}
}

func (r *commandRegistry) configureLeaf(cmd *cobra.Command, path string) {
	cmd.Short = path
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		if help, ok, err := osc.Help(path); err == nil && ok {
			fmt.Fprint(r.stdout, help)
			return
		}
		_ = command.Parent().Help()
	})

	if handler, ok := r.implemented[path]; ok {
		cmd.RunE = handler
		addImplementedCommandFlags(cmd, path)
		if path == "command list" {
			cmd.Flags().String("group", "", "filter by command group")
		}
		return
	}

	cmd.DisableFlagParsing = true
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(command *cobra.Command, args []string) error {
		if containsHelpFlag(args) {
			return command.Help()
		}
		_, err := fmt.Fprintln(r.stdout, notImplementedExitCodeText)
		return err
	}
}

func addImplementedCommandFlags(cmd *cobra.Command, path string) {
	if isIdentityReadCommand(path) && strings.HasSuffix(path, " list") {
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("domain", "", "filter by domain")
		cmd.Flags().Bool("enabled", false, "list enabled resources")
		cmd.Flags().Bool("disabled", false, "list disabled resources")
		cmd.Flags().String("service", "", "filter by service")
		cmd.Flags().String("interface", "", "filter by interface")
		cmd.Flags().String("region", "", "filter by region")
		cmd.Flags().String("parent-region", "", "filter by parent region")
		cmd.Flags().String("tags", "", "filter by tags")
		cmd.Flags().String("tags-any", "", "filter by any tags")
		cmd.Flags().String("not-tags", "", "exclude tags")
		cmd.Flags().String("not-tags-any", "", "exclude any tags")
	}
	if usesSharedCoreListFlags(path) {
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("name", "", "filter by name")
	}
	switch path {
	case "project show", "user show", "group show", "role show", "domain show":
		cmd.Flags().String("domain", "", "domain name or ID")
	case "project list":
		cmd.Flags().String("parent", "", "filter by parent")
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().Bool("my-projects", false, "list projects for the authenticated user")
		cmd.Flags().StringArray("sort", nil, "sort by key")
	case "user list":
		cmd.Flags().String("group", "", "filter by group")
		cmd.Flags().String("project", "", "filter by project")
	case "endpoint list":
		cmd.Flags().String("endpoint", "", "endpoint group")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "allocation candidate list":
		cmd.Flags().String("resource", "", "resource class amount")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("required", "", "required trait")
		cmd.Flags().String("forbidden", "", "forbidden trait")
		cmd.Flags().String("member-of", "", "aggregate membership")
		cmd.Flags().String("group", "", "granular request group")
		cmd.Flags().String("group-policy", "", "granular request group policy")
	case "availability zone list":
		cmd.Flags().Bool("compute", false, "list compute availability zones")
		cmd.Flags().Bool("network", false, "list network availability zones")
		cmd.Flags().Bool("volume", false, "list volume availability zones")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "container list":
		cmd.Flags().String("prefix", "", "filter by prefix")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().String("end-marker", "", "pagination end marker")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("all", false, "list all containers")
	case "object list":
		cmd.Flags().String("prefix", "", "filter by prefix")
		cmd.Flags().String("delimiter", "", "roll up objects by delimiter")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().String("end-marker", "", "pagination end marker")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("all", false, "list all objects")
	case "extension list":
		cmd.Flags().Bool("compute", false, "list compute extensions")
		cmd.Flags().Bool("identity", false, "list identity extensions")
		cmd.Flags().Bool("network", false, "list network extensions")
		cmd.Flags().Bool("volume", false, "list volume extensions")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "subnet list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Int("ip-version", 0, "filter by IP version")
		cmd.Flags().Bool("dhcp", false, "filter to DHCP-enabled subnets")
		cmd.Flags().Bool("no-dhcp", false, "filter to DHCP-disabled subnets")
		cmd.Flags().String("service-type", "", "filter by service type")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("network", "", "filter by network")
		cmd.Flags().String("gateway", "", "filter by gateway IP")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("subnet-range", "", "filter by subnet range")
		cmd.Flags().String("subnet-pool", "", "filter by subnet pool")
		addTagFilterFlags(cmd)
	case "port list":
		cmd.Flags().String("device-owner", "", "filter by device owner")
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("network", "", "filter by network")
		cmd.Flags().String("router", "", "filter by router")
		cmd.Flags().String("server", "", "filter by server")
		cmd.Flags().String("device-id", "", "filter by device ID")
		cmd.Flags().String("mac-address", "", "filter by MAC address")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("security-group", "", "filter by security group")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("fixed-ip", "", "filter by fixed IP attributes")
		addTagFilterFlags(cmd)
	case "floating ip list":
		cmd.Flags().String("network", "", "filter by network")
		cmd.Flags().String("port", "", "filter by port")
		cmd.Flags().String("fixed-ip-address", "", "filter by fixed IP address")
		cmd.Flags().String("floating-ip-address", "", "filter by floating IP address")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("router", "", "filter by router")
		addTagFilterFlags(cmd)
		cmd.Flags().Bool("long", false, "list additional fields")
	case "keypair list":
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().String("user-domain", "", "user domain")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	case "keypair show":
		cmd.Flags().Bool("public-key", false, "show only the public key")
		cmd.Flags().String("user", "", "keypair owner")
		cmd.Flags().String("user-domain", "", "user domain")
	case "limits show":
		cmd.Flags().Bool("absolute", false, "show absolute limits")
		cmd.Flags().Bool("rate", false, "show rate limits")
		cmd.Flags().Bool("reserved", false, "include reserved limits")
		cmd.Flags().String("project", "", "show limits for project")
		cmd.Flags().String("domain", "", "project domain")
	case "resource provider list":
		cmd.Flags().String("uuid", "", "filter by UUID")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("resource", "", "resource class amount")
		cmd.Flags().String("in-tree", "", "provider tree UUID")
		cmd.Flags().String("required", "", "required trait")
		cmd.Flags().String("forbidden", "", "forbidden trait")
		cmd.Flags().String("member-of", "", "aggregate membership")
	case "resource provider show":
		cmd.Flags().Bool("allocations", false, "include resource allocations")
	case "trait list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Bool("associated", false, "filter to associated traits")
	case "server group list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().Int("offset", 0, "collection offset")
	case "volume type list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("default", false, "list the default volume type")
		cmd.Flags().Bool("public", false, "list public volume types")
		cmd.Flags().Bool("private", false, "list private volume types")
		cmd.Flags().Bool("encryption-type", false, "show encryption information")
		cmd.Flags().String("property", "", "filter by property")
		cmd.Flags().Bool("multiattach", false, "filter multi-attach capable types")
		cmd.Flags().Bool("cacheable", false, "filter cacheable types")
		cmd.Flags().Bool("replicated", false, "filter replicated types")
		cmd.Flags().String("availability-zone", "", "filter by availability zone")
	case "volume type show":
		cmd.Flags().Bool("encryption-type", false, "show encryption information")
	case "volume snapshot list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("volume", "", "filter by volume")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	}
}

func addTagFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("tags", "", "filter by all tags")
	cmd.Flags().String("any-tags", "", "filter by any tags")
	cmd.Flags().String("not-tags", "", "exclude all tags")
	cmd.Flags().String("not-any-tags", "", "exclude any tags")
}

func isIdentityReadCommand(path string) bool {
	switch path {
	case "catalog list", "catalog show",
		"domain list", "domain show",
		"endpoint list", "endpoint show",
		"group list", "group show",
		"project list", "project show",
		"region list", "region show",
		"role list", "role show",
		"service list", "service show",
		"user list", "user show":
		return true
	default:
		return false
	}
}

func isCoreReadCommand(path string) bool {
	switch path {
	case "allocation candidate list",
		"availability zone list",
		"container list", "container show",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip list", "floating ip show",
		"image list", "image show",
		"keypair list", "keypair show",
		"limits show",
		"network list", "network show",
		"object list", "object show",
		"object store account show",
		"port list", "port show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"server list", "server show",
		"server group list", "server group show",
		"subnet list", "subnet show",
		"trait list", "trait show",
		"volume list", "volume show",
		"volume snapshot list", "volume snapshot show",
		"volume type list", "volume type show":
		return true
	default:
		return false
	}
}

func usesSharedCoreListFlags(path string) bool {
	switch path {
	case "flavor list", "image list", "network list", "server list", "volume list":
		return true
	default:
		return false
	}
}

func findChild(parent *cobra.Command, name string) *cobra.Command {
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func implementedCommandNames(implemented map[string]commandHandler) map[string]bool {
	names := make(map[string]bool, len(implemented))
	for name := range implemented {
		names[name] = true
	}
	return names
}

func sortedGroups(groups []osc.CommandGroup) []osc.CommandGroup {
	copied := make([]osc.CommandGroup, len(groups))
	for i, group := range groups {
		commands := append([]string(nil), group.Commands...)
		sort.Strings(commands)
		copied[i] = osc.CommandGroup{
			CommandGroup: group.CommandGroup,
			Commands:     commands,
		}
	}
	sort.Slice(copied, func(i int, j int) bool {
		return copied[i].CommandGroup < copied[j].CommandGroup
	})
	return copied
}
