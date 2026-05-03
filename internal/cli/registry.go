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
		"access rule list", "access rule show",
		"application credential list", "application credential show",
		"catalog list", "catalog show",
		"credential list", "credential show",
		"domain list", "domain show",
		"ec2 credentials list", "ec2 credentials show",
		"endpoint list", "endpoint show",
		"federation protocol list", "federation protocol show",
		"group list", "group show",
		"identity provider list", "identity provider show",
		"implied role list",
		"limit list", "limit show",
		"mapping list", "mapping show",
		"policy list", "policy show",
		"project list", "project show",
		"region list", "region show",
		"registered limit list", "registered limit show",
		"role assignment list",
		"role list", "role show",
		"service list", "service show",
		"service provider list", "service provider show",
		"trust list", "trust show",
		"user list", "user show",
	} {
		registry.implemented[path] = runIdentityRead(path, stdout, opts)
	}
	for _, path := range []string{
		"address group list", "address group show",
		"address scope list", "address scope show",
		"aggregate list", "aggregate show",
		"allocation candidate list",
		"availability zone list",
		"block storage cluster list", "block storage cluster show",
		"block storage log level list",
		"block storage resource filter list", "block storage resource filter show",
		"cached image list",
		"compute agent list",
		"compute service list",
		"console connection show",
		"console log show", "console url show",
		"container list", "container show",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip list", "floating ip pool list", "floating ip show",
		"host list", "host show",
		"hypervisor list", "hypervisor show", "hypervisor stats show",
		"image list", "image member get", "image member list", "image show",
		"image metadef namespace create", "image metadef namespace delete",
		"image metadef namespace list", "image metadef namespace set",
		"image metadef namespace show",
		"image metadef object list", "image metadef object property show",
		"image metadef object show", "image metadef property list",
		"image metadef property show",
		"image metadef resource type association create",
		"image metadef resource type association delete",
		"image metadef resource type association list",
		"image metadef resource type list",
		"image stores list", "image task list", "image task show",
		"ip availability list", "ip availability show",
		"keypair list", "keypair show",
		"limits show",
		"network agent list", "network agent show",
		"network list", "network show",
		"network service provider list",
		"network qos policy list", "network qos policy show",
		"network qos rule type list", "network qos rule type show",
		"network rbac list", "network rbac show",
		"network segment list", "network segment show",
		"network trunk list", "network trunk show",
		"object list", "object show",
		"object store account show",
		"port list", "port show",
		"quota list", "quota show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"router list", "router show",
		"security group list", "security group show",
		"security group rule list", "security group rule show",
		"server event list", "server event show",
		"server list", "server show",
		"server group list", "server group show",
		"server migration list",
		"server volume list",
		"subnet list", "subnet show",
		"subnet pool list", "subnet pool show",
		"trait list", "trait show",
		"usage list", "usage show",
		"versions show",
		"volume attachment list", "volume attachment show",
		"volume backend pool list",
		"volume backup list", "volume backup show",
		"volume group list", "volume group show",
		"volume group snapshot list", "volume group snapshot show",
		"volume group type list", "volume group type show",
		"volume list", "volume show",
		"volume message list", "volume message show",
		"volume qos list", "volume qos show",
		"volume service list",
		"volume snapshot list", "volume snapshot show",
		"volume summary",
		"volume transfer request list", "volume transfer request show",
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
	case "address group list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "address scope list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Int("ip-version", 0, "filter by IP version")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "list shared resources")
		cmd.Flags().Bool("no-share", false, "list non-shared resources")
	case "access rule list", "application credential list", "ec2 credentials list":
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().String("user-domain", "", "user domain")
	case "aggregate list":
		cmd.Flags().Bool("long", false, "list additional fields")
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
	case "credential list":
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().String("user-domain", "", "user domain")
		cmd.Flags().String("type", "", "filter by credential type")
	case "endpoint list":
		cmd.Flags().String("endpoint", "", "endpoint group")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "federation protocol list", "federation protocol show":
		cmd.Flags().String("identity-provider", "", "identity provider")
	case "ec2 credentials show":
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().String("user-domain", "", "user domain")
	case "identity provider list":
		cmd.Flags().String("id", "", "filter by ID")
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
	case "compute service list":
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("service", "", "filter by service binary")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "console log show":
		cmd.Flags().Int("lines", 0, "number of log lines")
	case "console url show":
		cmd.Flags().Bool("novnc", false, "show noVNC console URL")
		cmd.Flags().Bool("xvpvnc", false, "show xvpvnc console URL")
		cmd.Flags().Bool("spice", false, "show SPICE console URL")
		cmd.Flags().Bool("spice-direct", false, "show SPICE direct console URL")
		cmd.Flags().Bool("rdp", false, "show RDP console URL")
		cmd.Flags().Bool("serial", false, "show serial console URL")
		cmd.Flags().Bool("mks", false, "show WebMKS console URL")
	case "network agent list":
		cmd.Flags().String("agent-type", "", "filter by agent type")
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("network", "", "filter by hosted network")
		cmd.Flags().String("router", "", "filter by hosted router")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "network qos policy list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "list shared policies")
		cmd.Flags().Bool("no-share", false, "list non-shared policies")
	case "network qos rule type list":
		cmd.Flags().Bool("all-supported", false, "list all supported rule types")
		cmd.Flags().Bool("all-rules", false, "list all implemented rule types")
	case "network rbac list":
		cmd.Flags().String("type", "", "filter by object type")
		cmd.Flags().String("action", "", "filter by action")
		cmd.Flags().String("target-project", "", "filter by target project")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "network segment list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("network", "", "filter by network")
	case "network trunk list":
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
	case "subnet pool list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("share", false, "list shared subnet pools")
		cmd.Flags().Bool("no-share", false, "list non-shared subnet pools")
		cmd.Flags().Bool("default", false, "list default subnet pools")
		cmd.Flags().Bool("no-default", false, "list non-default subnet pools")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("address-scope", "", "filter by address scope")
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
	case "ip availability list":
		cmd.Flags().Int("ip-version", 0, "filter by IP version")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
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
	case "hypervisor list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("matching", "", "filter by hypervisor hostname")
		cmd.Flags().Bool("with-servers", false, "include servers")
	case "image member get", "image member list":
		cmd.Flags().String("project-domain", "", "project domain")
	case "image metadef namespace create", "image metadef namespace set":
		cmd.Flags().String("display-name", "", "display name")
		cmd.Flags().String("description", "", "description")
		cmd.Flags().Bool("public", false, "public visibility")
		cmd.Flags().Bool("private", false, "private visibility")
		cmd.Flags().Bool("protected", false, "protected namespace")
		cmd.Flags().Bool("unprotected", false, "unprotected namespace")
	case "image metadef namespace list":
		cmd.Flags().String("resource-types", "", "filter resource types")
		cmd.Flags().String("visibility", "", "filter on visibility")
	case "image metadef resource type association create":
		cmd.Flags().String("properties-target", "", "properties target")
	case "image metadef resource type association delete":
		cmd.Flags().Bool("force", false, "force delete protected association")
	case "image stores list":
		cmd.Flags().Bool("detail", false, "show store details")
	case "image task list":
		cmd.Flags().String("sort-key", "", "sort key")
		cmd.Flags().String("sort-dir", "", "sort direction")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().String("type", "", "filter by task type")
		cmd.Flags().String("status", "", "filter by task status")
	case "limits show":
		cmd.Flags().Bool("absolute", false, "show absolute limits")
		cmd.Flags().Bool("rate", false, "show rate limits")
		cmd.Flags().Bool("reserved", false, "include reserved limits")
		cmd.Flags().String("project", "", "show limits for project")
		cmd.Flags().String("domain", "", "project domain")
	case "limit list":
		cmd.Flags().String("resource-name", "", "filter by resource name")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "quota list":
		cmd.Flags().Bool("compute", false, "list compute quotas")
		cmd.Flags().Bool("volume", false, "list volume quotas")
		cmd.Flags().Bool("network", false, "list network quotas")
	case "quota show":
		cmd.Flags().Bool("default", false, "show default quotas")
		cmd.Flags().Bool("usage", false, "show quota usage")
		cmd.Flags().Bool("all", false, "show quotas for all services")
		cmd.Flags().Bool("compute", false, "show compute quota")
		cmd.Flags().Bool("volume", false, "show volume quota")
		cmd.Flags().Bool("network", false, "show network quota")
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
	case "registered limit list":
		cmd.Flags().String("resource-name", "", "filter by resource name")
	case "role assignment list":
		cmd.Flags().Bool("effective", false, "return only effective role assignments")
		cmd.Flags().String("role", "", "role to filter")
		cmd.Flags().String("role-domain", "", "role domain")
		cmd.Flags().Bool("names", false, "display names instead of IDs")
		cmd.Flags().String("user", "", "user to filter")
		cmd.Flags().String("user-domain", "", "user domain")
		cmd.Flags().String("group", "", "group to filter")
		cmd.Flags().String("group-domain", "", "group domain")
		cmd.Flags().String("project", "", "project to filter")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("system", "", "system to filter")
		cmd.Flags().Bool("inherited", false, "filter inherited assignments")
		cmd.Flags().Bool("auth-user", false, "filter to authenticated user")
		cmd.Flags().Bool("auth-project", false, "filter to authenticated project")
	case "router list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Bool("enable", false, "list enabled routers")
		cmd.Flags().Bool("disable", false, "list disabled routers")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("agent", "", "filter by agent")
		addTagFilterFlags(cmd)
	case "security group list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "list shared security groups")
		cmd.Flags().Bool("no-share", false, "list non-shared security groups")
		addTagFilterFlags(cmd)
	case "security group rule list":
		cmd.Flags().String("protocol", "", "filter by IP protocol")
		cmd.Flags().String("ethertype", "", "filter by ethertype")
		cmd.Flags().Bool("ingress", false, "list ingress rules")
		cmd.Flags().Bool("egress", false, "list egress rules")
		cmd.Flags().Bool("long", false, "deprecated")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "trait list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Bool("associated", false, "filter to associated traits")
	case "trust list":
		cmd.Flags().String("trustor", "", "filter by trustor user")
		cmd.Flags().String("trustee", "", "filter by trustee user")
		cmd.Flags().String("trustor-domain", "", "trustor domain")
		cmd.Flags().String("trustee-domain", "", "trustee domain")
		cmd.Flags().Bool("auth-user", false, "filter to authenticated user")
	case "server group list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().Int("offset", 0, "collection offset")
	case "server event list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("changes-since", "", "show events changed since this timestamp")
		cmd.Flags().String("changes-before", "", "show events changed before this timestamp")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	case "server migration list":
		cmd.Flags().String("server", "", "filter migrations by server")
		cmd.Flags().String("host", "", "filter migrations by source or destination host")
		cmd.Flags().String("status", "", "filter migrations by status")
		cmd.Flags().String("type", "", "filter migrations by type")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().String("changes-since", "", "show migrations changed since this timestamp")
		cmd.Flags().String("changes-before", "", "show migrations changed before this timestamp")
		cmd.Flags().String("project", "", "filter migrations by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("user", "", "filter migrations by user")
		cmd.Flags().String("user-domain", "", "user domain")
	case "usage list", "usage show":
		cmd.Flags().String("start", "", "usage range start date")
		cmd.Flags().String("end", "", "usage range end date")
		if path == "usage show" {
			cmd.Flags().String("project", "", "project name or ID")
		}
	case "volume backend pool list":
		cmd.Flags().Bool("long", false, "show detailed information about pools")
	case "block storage cluster list":
		cmd.Flags().String("cluster", "", "filter by cluster name")
		cmd.Flags().String("binary", "", "filter by cluster binary")
		cmd.Flags().Bool("up", false, "filter by up status")
		cmd.Flags().Bool("down", false, "filter by down status")
		cmd.Flags().Bool("disabled", false, "filter by disabled status")
		cmd.Flags().Bool("enabled", false, "filter by enabled status")
		cmd.Flags().Int("num-hosts", 0, "filter by number of hosts")
		cmd.Flags().Int("num-down-hosts", 0, "filter by number of down hosts")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "block storage cluster show":
		cmd.Flags().String("binary", "", "service binary")
	case "block storage log level list":
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("service", "", "filter by service binary")
		cmd.Flags().String("log-prefix", "", "filter by log prefix")
	case "compute agent list":
		cmd.Flags().String("hypervisor", "", "type of hypervisor")
	case "host list":
		cmd.Flags().String("zone", "", "only return hosts in the availability zone")
	case "volume attachment list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().String("volume-id", "", "filter by volume ID")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	case "volume backup list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("volume", "", "filter by volume")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume service list":
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("service", "", "filter by service binary")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "volume group list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume group show":
		cmd.Flags().Bool("volumes", false, "show volumes in the group")
		cmd.Flags().Bool("no-volumes", false, "do not show volumes in the group")
		cmd.Flags().Bool("replication-targets", false, "show replication targets")
		cmd.Flags().Bool("no-replication-targets", false, "do not show replication targets")
	case "volume group snapshot list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume group type list":
		cmd.Flags().Bool("default", false, "list the default volume group type")
	case "volume message list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
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
	case "volume summary":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "versions show":
		cmd.Flags().Bool("all-interfaces", false, "show all interfaces")
		cmd.Flags().String("interface", "", "show a specific interface")
		cmd.Flags().String("region-name", "", "show a specific region")
		cmd.Flags().String("service", "", "show a specific service")
		cmd.Flags().String("status", "", "show a specific version status")
	case "volume transfer request list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
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
	case "access rule list", "access rule show",
		"application credential list", "application credential show",
		"catalog list", "catalog show",
		"credential list", "credential show",
		"domain list", "domain show",
		"ec2 credentials list", "ec2 credentials show",
		"endpoint list", "endpoint show",
		"federation protocol list", "federation protocol show",
		"group list", "group show",
		"identity provider list", "identity provider show",
		"implied role list",
		"limit list", "limit show",
		"mapping list", "mapping show",
		"policy list", "policy show",
		"project list", "project show",
		"region list", "region show",
		"registered limit list", "registered limit show",
		"role assignment list",
		"role list", "role show",
		"service list", "service show",
		"service provider list", "service provider show",
		"trust list", "trust show",
		"user list", "user show":
		return true
	default:
		return false
	}
}

func isCoreReadCommand(path string) bool {
	switch path {
	case "address group list", "address group show",
		"address scope list", "address scope show",
		"aggregate list", "aggregate show",
		"allocation candidate list",
		"availability zone list",
		"block storage cluster list", "block storage cluster show",
		"block storage log level list",
		"block storage resource filter list", "block storage resource filter show",
		"cached image list",
		"compute agent list",
		"compute service list",
		"console connection show",
		"console log show", "console url show",
		"container list", "container show",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip list", "floating ip pool list", "floating ip show",
		"host list", "host show",
		"hypervisor list", "hypervisor show", "hypervisor stats show",
		"image list", "image member get", "image member list", "image show",
		"image metadef namespace create", "image metadef namespace delete",
		"image metadef namespace list", "image metadef namespace set",
		"image metadef namespace show",
		"image metadef object list", "image metadef object property show",
		"image metadef object show", "image metadef property list",
		"image metadef property show",
		"image metadef resource type association create",
		"image metadef resource type association delete",
		"image metadef resource type association list",
		"image metadef resource type list",
		"image stores list", "image task list", "image task show",
		"ip availability list", "ip availability show",
		"keypair list", "keypair show",
		"limits show",
		"network agent list", "network agent show",
		"network list", "network show",
		"network service provider list",
		"network qos policy list", "network qos policy show",
		"network qos rule type list", "network qos rule type show",
		"network rbac list", "network rbac show",
		"network segment list", "network segment show",
		"network trunk list", "network trunk show",
		"object list", "object show",
		"object store account show",
		"port list", "port show",
		"quota list", "quota show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"router list", "router show",
		"security group list", "security group show",
		"security group rule list", "security group rule show",
		"server event list", "server event show",
		"server list", "server show",
		"server group list", "server group show",
		"server migration list",
		"server volume list",
		"subnet list", "subnet show",
		"subnet pool list", "subnet pool show",
		"trait list", "trait show",
		"usage list", "usage show",
		"versions show",
		"volume attachment list", "volume attachment show",
		"volume backend pool list",
		"volume backup list", "volume backup show",
		"volume group list", "volume group show",
		"volume group snapshot list", "volume group snapshot show",
		"volume group type list", "volume group type show",
		"volume list", "volume show",
		"volume message list", "volume message show",
		"volume qos list", "volume qos show",
		"volume service list",
		"volume snapshot list", "volume snapshot show",
		"volume summary",
		"volume transfer request list", "volume transfer request show",
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
