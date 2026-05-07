package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/crandallnet/golang-osc/compat/osc"
	"github.com/crandallnet/golang-osc/internal/cliplugin"
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
	registry.implemented["configuration show"] = runConfigurationShow(stdout, opts)
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
		"address group create", "address group delete", "address group list", "address group set", "address group show", "address group unset",
		"address scope create", "address scope delete", "address scope list", "address scope set", "address scope show",
		"aggregate list", "aggregate show",
		"allocation candidate list",
		"availability zone list",
		"block storage cleanup",
		"block storage cluster list", "block storage cluster set", "block storage cluster show",
		"block storage log level list", "block storage log level set",
		"block storage snapshot manageable list", "block storage volume manageable list",
		"cached image clear", "cached image delete",
		"cached image list", "cached image queue",
		"compute agent list",
		"compute service list",
		"consistency group add volume", "consistency group create", "consistency group delete", "consistency group list",
		"consistency group remove volume", "consistency group set", "consistency group show",
		"consistency group snapshot create", "consistency group snapshot delete", "consistency group snapshot list", "consistency group snapshot show",
		"console connection show",
		"console log show", "console url show",
		"container create", "container delete", "container list", "container set", "container show", "container unset",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip create", "floating ip delete", "floating ip list", "floating ip pool list",
		"floating ip port forwarding create", "floating ip port forwarding delete", "floating ip port forwarding list", "floating ip port forwarding set", "floating ip port forwarding show",
		"floating ip set", "floating ip show", "floating ip unset",
		"host list", "host show",
		"hypervisor list", "hypervisor show", "hypervisor stats show",
		"image add project",
		"image create", "image delete",
		"image import", "image import info", "image list",
		"image member get", "image member list", "image show",
		"image remove project",
		"image save", "image set", "image stage", "image unset",
		"image metadef namespace create", "image metadef namespace delete",
		"image metadef namespace list", "image metadef namespace set",
		"image metadef namespace show",
		"image metadef object create", "image metadef object delete",
		"image metadef object list", "image metadef object property show",
		"image metadef object show", "image metadef object update",
		"image metadef property create", "image metadef property delete",
		"image metadef property list", "image metadef property set",
		"image metadef property show",
		"image metadef resource type association create",
		"image metadef resource type association delete",
		"image metadef resource type association list",
		"image metadef resource type list",
		"image stores list", "image task list", "image task show",
		"ip availability list", "ip availability show",
		"keypair create", "keypair delete", "keypair list", "keypair show",
		"limits show",
		"network agent list", "network agent show",
		"network create", "network delete", "network list", "network set", "network show", "network unset",
		"network service provider list",
		"network qos policy create", "network qos policy delete", "network qos policy list", "network qos policy set", "network qos policy show",
		"network qos rule create", "network qos rule delete", "network qos rule list", "network qos rule set", "network qos rule show",
		"network qos rule type list", "network qos rule type show",
		"network rbac create", "network rbac delete", "network rbac list", "network rbac set", "network rbac show",
		"network segment create", "network segment delete", "network segment list", "network segment set", "network segment show",
		"network subport list",
		"network trunk create", "network trunk delete", "network trunk list", "network trunk set", "network trunk show", "network trunk unset",
		"object create", "object delete", "object list", "object save", "object set", "object show", "object unset",
		"object store account show",
		"port create", "port delete", "port list", "port set", "port show", "port unset",
		"quota delete", "quota list", "quota set", "quota show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"router add gateway", "router add port", "router add route", "router add subnet",
		"router create", "router delete", "router list",
		"router remove gateway", "router remove port", "router remove route", "router remove subnet",
		"router set", "router show", "router unset",
		"security group create", "security group delete", "security group list", "security group set", "security group show", "security group unset",
		"security group rule create", "security group rule delete", "security group rule list", "security group rule show",
		"server add fixed ip", "server add floating ip", "server add network", "server add port", "server add security group", "server add volume",
		"server backup create",
		"server create", "server delete", "server dump create", "server evacuate",
		"server event list", "server event show",
		"server group create", "server group delete", "server group list", "server group show",
		"server image create",
		"server list", "server lock", "server migrate", "server migrate confirm", "server migrate revert",
		"server migration abort", "server migration confirm", "server migration force complete", "server migration list", "server migration revert", "server migration show",
		"server pause", "server reboot", "server rebuild",
		"server remove fixed ip", "server remove floating ip", "server remove network", "server remove port", "server remove security group", "server remove volume",
		"server rescue", "server resize", "server resize confirm", "server resize revert", "server restore", "server resume",
		"server set", "server shelve", "server show", "server start", "server stop", "server suspend",
		"server unlock", "server unpause", "server unrescue", "server unset", "server unshelve",
		"server volume list", "server volume set", "server volume update",
		"subnet create", "subnet delete", "subnet list", "subnet set", "subnet show", "subnet unset",
		"subnet pool create", "subnet pool delete", "subnet pool list", "subnet pool set", "subnet pool show", "subnet pool unset",
		"trait list", "trait show",
		"usage list", "usage show",
		"versions show",
		"volume attachment complete", "volume attachment create", "volume attachment delete",
		"volume attachment list", "volume attachment set", "volume attachment show",
		"volume backend capability show",
		"volume backend pool list",
		"volume backup create", "volume backup delete", "volume backup list",
		"volume backup record export", "volume backup record import", "volume backup restore",
		"volume backup set", "volume backup show", "volume backup unset",
		"volume group create", "volume group delete", "volume group failover", "volume group list", "volume group set", "volume group show",
		"volume group snapshot create", "volume group snapshot delete", "volume group snapshot list", "volume group snapshot show",
		"volume group type create", "volume group type delete", "volume group type list", "volume group type set", "volume group type show",
		"volume host set",
		"volume create", "volume delete", "volume list", "volume set", "volume show", "volume unset",
		"volume message delete", "volume message list", "volume message show",
		"volume migrate",
		"volume qos associate", "volume qos create", "volume qos delete", "volume qos disassociate",
		"volume qos list", "volume qos set", "volume qos show", "volume qos unset",
		"volume revert",
		"volume service list", "volume service set",
		"volume snapshot create", "volume snapshot delete", "volume snapshot list",
		"volume snapshot set", "volume snapshot show", "volume snapshot unset",
		"volume summary",
		"volume transfer request accept", "volume transfer request create", "volume transfer request delete",
		"volume transfer request list", "volume transfer request show",
		"volume type create", "volume type delete", "volume type list", "volume type set", "volume type show", "volume type unset",
	} {
		registry.implemented[path] = runCoreRead(path, stdout, opts)
	}
	registry.addProviderCommands(cliplugin.NamespaceExtras, stdout, opts)
	return registry
}

func (r *commandRegistry) addProviderCommands(namespace string, stdout io.Writer, opts *Options) {
	providers, err := cliplugin.Providers(namespace)
	if err != nil {
		panic(fmt.Sprintf("load command providers for %s: %v", namespace, err))
	}
	for _, provider := range providers {
		for _, command := range provider.PluginCommands() {
			if !command.Implemented {
				continue
			}
			handler, ok := extrasCommandHandler(command.Path, stdout, opts)
			if !ok {
				panic(fmt.Sprintf("no registered extras handler for %q", command.Path))
			}
			r.implemented[command.Path] = handler
		}
	}
}

func extrasCommandHandler(path string, stdout io.Writer, opts *Options) (commandHandler, bool) {
	if isCinderExtrasCommand(path) {
		return runCinderExtras(path, stdout, opts), true
	}
	if isNovaExtrasCommand(path) {
		return runNovaExtras(path, stdout, opts), true
	}
	return nil, false
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
	cmd.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return parserFlagError(path, err)
	})
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		if help, ok, err := osc.Help(path); err == nil && ok {
			fmt.Fprint(r.stdout, help)
			if path == "server ssh" {
				fmt.Fprint(r.stdout, serverSSHPassThroughHelp())
			}
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
	case "address group create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "address group description")
		cmd.Flags().StringArray("address", nil, "IP address or CIDR")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "address group set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new address group name")
		cmd.Flags().String("description", "", "new address group description")
		cmd.Flags().StringArray("address", nil, "IP address or CIDR")
	case "address group unset":
		cmd.Flags().StringArray("address", nil, "IP address or CIDR")
	case "address scope list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Int("ip-version", 0, "filter by IP version")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "list shared resources")
		cmd.Flags().Bool("no-share", false, "list non-shared resources")
	case "address scope create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().Int("ip-version", 4, "IP version")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "share resource")
		cmd.Flags().Bool("no-share", false, "do not share resource")
	case "address scope set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new address scope name")
		cmd.Flags().Bool("share", false, "share resource")
		cmd.Flags().Bool("no-share", false, "do not share resource")
	case "configuration show":
		cmd.Flags().Bool("mask", true, "mask passwords")
		cmd.Flags().Bool("unmask", false, "show passwords")
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
	case "container create":
		cmd.Flags().Bool("public", false, "make container public")
		cmd.Flags().String("storage-policy", "", "storage policy")
	case "container delete":
		cmd.Flags().BoolP("recursive", "r", false, "recursively delete objects and container")
	case "container set":
		cmd.Flags().StringArray("property", nil, "container property key=value")
	case "container unset":
		cmd.Flags().StringArray("property", nil, "container property key")
	case "object list":
		cmd.Flags().String("prefix", "", "filter by prefix")
		cmd.Flags().String("delimiter", "", "roll up objects by delimiter")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().String("end-marker", "", "pagination end marker")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("all", false, "list all objects")
	case "object create":
		cmd.Flags().String("name", "", "object name")
	case "object save":
		cmd.Flags().String("file", "", "destination filename")
	case "object set":
		cmd.Flags().StringArray("property", nil, "object property key=value")
	case "object unset":
		cmd.Flags().StringArray("property", nil, "object property key")
	case "port create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("network", "", "network")
		cmd.Flags().String("description", "", "port description")
		cmd.Flags().String("device", "", "device ID")
		cmd.Flags().String("mac-address", "", "MAC address")
		cmd.Flags().String("device-owner", "", "device owner")
		cmd.Flags().String("vnic-type", "", "VNIC type")
		cmd.Flags().String("host", "", "binding host ID")
		cmd.Flags().String("dns-domain", "", "DNS domain")
		cmd.Flags().String("dns-name", "", "DNS name")
		cmd.Flags().Bool("numa-policy-required", false, "NUMA affinity policy required")
		cmd.Flags().Bool("numa-policy-preferred", false, "NUMA affinity policy preferred")
		cmd.Flags().Bool("numa-policy-socket", false, "NUMA affinity policy socket")
		cmd.Flags().Bool("numa-policy-legacy", false, "NUMA affinity policy legacy")
		cmd.Flags().StringArray("hint", nil, "port hint alias=value or JSON")
		cmd.Flags().Bool("trusted", false, "trusted port")
		cmd.Flags().Bool("not-trusted", false, "not trusted port")
		cmd.Flags().StringArray("fixed-ip", nil, "fixed IP subnet=<subnet>,ip-address=<ip>")
		cmd.Flags().Bool("no-fixed-ip", false, "create without fixed IPs")
		cmd.Flags().StringArray("binding-profile", nil, "binding profile key=value or JSON")
		cmd.Flags().Bool("enable", true, "enable port")
		cmd.Flags().Bool("disable", false, "disable port")
		cmd.Flags().Bool("enable-uplink-status-propagation", false, "enable uplink status propagation")
		cmd.Flags().Bool("disable-uplink-status-propagation", false, "disable uplink status propagation")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().StringArray("extra-dhcp-option", nil, "extra DHCP option name=<name>,value=<value>,ip-version=<version>")
		cmd.Flags().StringArray("security-group", nil, "security group")
		cmd.Flags().Bool("no-security-group", false, "create without security groups")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().Bool("enable-port-security", false, "enable port security")
		cmd.Flags().Bool("disable-port-security", false, "disable port security")
		cmd.Flags().StringArray("allowed-address", nil, "allowed address ip-address=<ip>,mac-address=<mac>")
		cmd.Flags().String("device-profile", "", "device profile")
		cmd.Flags().String("hardware-offload-type", "", "hardware offload type")
		cmd.Flags().StringArray("tag", nil, "port tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "port set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "port description")
		cmd.Flags().String("device", "", "device ID")
		cmd.Flags().String("mac-address", "", "MAC address")
		cmd.Flags().String("device-owner", "", "device owner")
		cmd.Flags().String("vnic-type", "", "VNIC type")
		cmd.Flags().String("host", "", "binding host ID")
		cmd.Flags().String("dns-domain", "", "DNS domain")
		cmd.Flags().String("dns-name", "", "DNS name")
		cmd.Flags().Bool("numa-policy-required", false, "NUMA affinity policy required")
		cmd.Flags().Bool("numa-policy-preferred", false, "NUMA affinity policy preferred")
		cmd.Flags().Bool("numa-policy-socket", false, "NUMA affinity policy socket")
		cmd.Flags().Bool("numa-policy-legacy", false, "NUMA affinity policy legacy")
		cmd.Flags().StringArray("hint", nil, "port hint alias=value or JSON")
		cmd.Flags().Bool("trusted", false, "trusted port")
		cmd.Flags().Bool("not-trusted", false, "not trusted port")
		cmd.Flags().Bool("enable", false, "enable port")
		cmd.Flags().Bool("disable", false, "disable port")
		cmd.Flags().String("name", "", "new port name")
		cmd.Flags().StringArray("fixed-ip", nil, "fixed IP subnet=<subnet>,ip-address=<ip>")
		cmd.Flags().Bool("no-fixed-ip", false, "clear fixed IPs")
		cmd.Flags().StringArray("binding-profile", nil, "binding profile key=value or JSON")
		cmd.Flags().Bool("no-binding-profile", false, "clear binding profile")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().StringArray("security-group", nil, "security group")
		cmd.Flags().Bool("no-security-group", false, "clear security groups")
		cmd.Flags().Bool("enable-port-security", false, "enable port security")
		cmd.Flags().Bool("disable-port-security", false, "disable port security")
		cmd.Flags().StringArray("allowed-address", nil, "allowed address ip-address=<ip>,mac-address=<mac>")
		cmd.Flags().Bool("no-allowed-address", false, "clear allowed addresses")
		cmd.Flags().StringArray("extra-dhcp-option", nil, "extra DHCP option name=<name>,value=<value>,ip-version=<version>")
		cmd.Flags().String("data-plane-status", "", "data plane status")
		cmd.Flags().Bool("enable-uplink-status-propagation", false, "enable uplink status propagation")
		cmd.Flags().Bool("disable-uplink-status-propagation", false, "disable uplink status propagation")
		cmd.Flags().StringArray("tag", nil, "port tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
	case "port unset":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().StringArray("fixed-ip", nil, "fixed IP subnet=<subnet>,ip-address=<ip>")
		cmd.Flags().StringArray("binding-profile", nil, "binding profile key")
		cmd.Flags().StringArray("security-group", nil, "security group")
		cmd.Flags().StringArray("allowed-address", nil, "allowed address ip-address=<ip>,mac-address=<mac>")
		cmd.Flags().Bool("qos-policy", false, "remove QoS policy")
		cmd.Flags().Bool("data-plane-status", false, "clear data plane status")
		cmd.Flags().Bool("numa-policy", false, "clear NUMA affinity policy")
		cmd.Flags().Bool("host", false, "clear binding host")
		cmd.Flags().Bool("hints", false, "clear hints")
		cmd.Flags().Bool("device", false, "clear device ID")
		cmd.Flags().Bool("device-owner", false, "clear device owner")
		cmd.Flags().StringArray("tag", nil, "port tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
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
	case "network qos policy create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "QoS policy description")
		cmd.Flags().Bool("share", false, "share policy")
		cmd.Flags().Bool("no-share", false, "do not share policy")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("default", false, "set as default policy")
		cmd.Flags().Bool("no-default", false, "set as non-default policy")
	case "network qos policy set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new QoS policy name")
		cmd.Flags().String("description", "", "QoS policy description")
		cmd.Flags().Bool("share", false, "share policy")
		cmd.Flags().Bool("no-share", false, "do not share policy")
		cmd.Flags().Bool("default", false, "set as default policy")
		cmd.Flags().Bool("no-default", false, "set as non-default policy")
	case "network qos rule create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("type", "", "QoS rule type")
		cmd.Flags().Int("max-kbps", 0, "maximum bandwidth in kbps")
		cmd.Flags().Int("max-burst-kbits", 0, "maximum burst in kilobits")
		cmd.Flags().Int("dscp-mark", 0, "DSCP mark")
		cmd.Flags().Int("min-kbps", 0, "minimum bandwidth in kbps")
		cmd.Flags().Int("min-kpps", 0, "minimum packet rate in kpps")
		cmd.Flags().Bool("ingress", false, "ingress direction")
		cmd.Flags().Bool("egress", false, "egress direction")
		cmd.Flags().Bool("any", false, "any direction")
	case "network qos rule set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().Int("max-kbps", 0, "maximum bandwidth in kbps")
		cmd.Flags().Int("max-burst-kbits", 0, "maximum burst in kilobits")
		cmd.Flags().Int("dscp-mark", 0, "DSCP mark")
		cmd.Flags().Int("min-kbps", 0, "minimum bandwidth in kbps")
		cmd.Flags().Int("min-kpps", 0, "minimum packet rate in kpps")
		cmd.Flags().Bool("ingress", false, "ingress direction")
		cmd.Flags().Bool("egress", false, "egress direction")
		cmd.Flags().Bool("any", false, "any direction")
	case "network qos rule type list":
		cmd.Flags().Bool("all-supported", false, "list all supported rule types")
		cmd.Flags().Bool("all-rules", false, "list all implemented rule types")
	case "network rbac list":
		cmd.Flags().String("type", "", "filter by object type")
		cmd.Flags().String("action", "", "filter by action")
		cmd.Flags().String("target-project", "", "filter by target project")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "network rbac create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("type", "", "object type")
		cmd.Flags().String("action", "", "RBAC action")
		cmd.Flags().String("target-project", "", "target project")
		cmd.Flags().Bool("target-all-projects", false, "target all projects")
		cmd.Flags().String("target-project-domain", "", "target project domain")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "network rbac set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("target-project", "", "target project")
		cmd.Flags().String("target-project-domain", "", "target project domain")
	case "network segment create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "segment description")
		cmd.Flags().String("physical-network", "", "physical network")
		cmd.Flags().Int("segment", 0, "segment identifier")
		cmd.Flags().String("network", "", "parent network")
		cmd.Flags().String("network-type", "", "network type")
	case "network segment list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("network", "", "filter by network")
	case "network segment set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "segment description")
		cmd.Flags().String("name", "", "segment name")
	case "network subport list":
		cmd.Flags().String("trunk", "", "parent trunk")
	case "network trunk create":
		cmd.Flags().String("description", "", "trunk description")
		cmd.Flags().String("parent-port", "", "parent port")
		cmd.Flags().StringArray("subport", nil, "subport port=<port>,segmentation-type=<type>,segmentation-id=<id>")
		cmd.Flags().Bool("enable", false, "enable trunk")
		cmd.Flags().Bool("disable", false, "disable trunk")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "network trunk list":
		cmd.Flags().Bool("long", false, "list additional fields")
	case "network trunk set":
		cmd.Flags().String("name", "", "trunk name")
		cmd.Flags().String("description", "", "trunk description")
		cmd.Flags().StringArray("subport", nil, "subport port=<port>,segmentation-type=<type>,segmentation-id=<id>")
		cmd.Flags().Bool("enable", false, "enable trunk")
		cmd.Flags().Bool("disable", false, "disable trunk")
	case "network trunk unset":
		cmd.Flags().StringArray("subport", nil, "subport port name or ID")
	case "network create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().Bool("share", false, "share resource")
		cmd.Flags().Bool("no-share", false, "do not share resource")
		cmd.Flags().Bool("enable", true, "enable network")
		cmd.Flags().Bool("disable", false, "disable network")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("description", "", "network description")
		cmd.Flags().String("mtu", "", "network MTU")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().StringArray("availability-zone-hint", nil, "availability zone hint")
		cmd.Flags().Bool("enable-port-security", false, "enable port security")
		cmd.Flags().Bool("disable-port-security", false, "disable port security")
		cmd.Flags().Bool("external", false, "external network")
		cmd.Flags().Bool("internal", false, "internal network")
		cmd.Flags().Bool("default", false, "default external network")
		cmd.Flags().Bool("no-default", false, "not default external network")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().Bool("transparent-vlan", false, "enable VLAN transparency")
		cmd.Flags().Bool("no-transparent-vlan", false, "disable VLAN transparency")
		cmd.Flags().Bool("qinq-vlan", false, "enable QinQ VLAN")
		cmd.Flags().Bool("no-qinq-vlan", false, "disable QinQ VLAN")
		cmd.Flags().String("provider-network-type", "", "provider network type")
		cmd.Flags().String("provider-physical-network", "", "provider physical network")
		cmd.Flags().String("provider-segment", "", "provider segment")
		cmd.Flags().String("dns-domain", "", "DNS domain")
		cmd.Flags().StringArray("tag", nil, "network tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "network set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new network name")
		cmd.Flags().Bool("enable", false, "enable network")
		cmd.Flags().Bool("disable", false, "disable network")
		cmd.Flags().Bool("share", false, "share resource")
		cmd.Flags().Bool("no-share", false, "do not share resource")
		cmd.Flags().String("description", "", "network description")
		cmd.Flags().String("mtu", "", "network MTU")
		cmd.Flags().Bool("enable-port-security", false, "enable port security")
		cmd.Flags().Bool("disable-port-security", false, "disable port security")
		cmd.Flags().Bool("external", false, "external network")
		cmd.Flags().Bool("internal", false, "internal network")
		cmd.Flags().Bool("default", false, "default external network")
		cmd.Flags().Bool("no-default", false, "not default external network")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().Bool("no-qos-policy", false, "remove QoS policy")
		cmd.Flags().StringArray("tag", nil, "network tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
		cmd.Flags().String("provider-network-type", "", "provider network type")
		cmd.Flags().String("provider-physical-network", "", "provider physical network")
		cmd.Flags().String("provider-segment", "", "provider segment")
		cmd.Flags().String("dns-domain", "", "DNS domain")
	case "network unset":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().StringArray("tag", nil, "network tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
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
	case "subnet create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("subnet-pool", "", "subnet pool")
		cmd.Flags().Bool("use-prefix-delegation", false, "use prefix delegation")
		cmd.Flags().Bool("use-default-subnet-pool", false, "use default subnet pool")
		cmd.Flags().String("prefix-length", "", "prefix length")
		cmd.Flags().String("subnet-range", "", "subnet range")
		cmd.Flags().Bool("dhcp", false, "enable DHCP")
		cmd.Flags().Bool("no-dhcp", false, "disable DHCP")
		cmd.Flags().Bool("dns-publish-fixed-ip", false, "publish fixed IPs in DNS")
		cmd.Flags().Bool("no-dns-publish-fixed-ip", false, "do not publish fixed IPs in DNS")
		cmd.Flags().String("gateway", "", "gateway IP, auto, or none")
		cmd.Flags().Int("ip-version", 4, "IP version")
		cmd.Flags().String("ipv6-ra-mode", "", "IPv6 RA mode")
		cmd.Flags().String("ipv6-address-mode", "", "IPv6 address mode")
		cmd.Flags().String("network-segment", "", "network segment")
		cmd.Flags().String("network", "", "network")
		cmd.Flags().String("description", "", "subnet description")
		cmd.Flags().StringArray("allocation-pool", nil, "allocation pool start=<ip>,end=<ip>")
		cmd.Flags().StringArray("dns-nameserver", nil, "DNS nameserver")
		cmd.Flags().StringArray("host-route", nil, "host route destination=<cidr>,gateway=<ip>")
		cmd.Flags().StringArray("service-type", nil, "service type")
		cmd.Flags().StringArray("tag", nil, "subnet tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "subnet set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new subnet name")
		cmd.Flags().Bool("dhcp", false, "enable DHCP")
		cmd.Flags().Bool("no-dhcp", false, "disable DHCP")
		cmd.Flags().Bool("dns-publish-fixed-ip", false, "publish fixed IPs in DNS")
		cmd.Flags().Bool("no-dns-publish-fixed-ip", false, "do not publish fixed IPs in DNS")
		cmd.Flags().String("gateway", "", "gateway IP or none")
		cmd.Flags().String("network-segment", "", "network segment")
		cmd.Flags().String("description", "", "subnet description")
		cmd.Flags().StringArray("tag", nil, "subnet tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
		cmd.Flags().StringArray("allocation-pool", nil, "allocation pool start=<ip>,end=<ip>")
		cmd.Flags().Bool("no-allocation-pool", false, "clear allocation pools")
		cmd.Flags().StringArray("dns-nameserver", nil, "DNS nameserver")
		cmd.Flags().Bool("no-dns-nameservers", false, "clear DNS nameservers")
		cmd.Flags().StringArray("host-route", nil, "host route destination=<cidr>,gateway=<ip>")
		cmd.Flags().Bool("no-host-route", false, "clear host routes")
		cmd.Flags().StringArray("service-type", nil, "service type")
	case "subnet unset":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().StringArray("allocation-pool", nil, "allocation pool start=<ip>,end=<ip>")
		cmd.Flags().Bool("gateway", false, "remove gateway")
		cmd.Flags().StringArray("dns-nameserver", nil, "DNS nameserver")
		cmd.Flags().StringArray("host-route", nil, "host route destination=<cidr>,gateway=<ip>")
		cmd.Flags().StringArray("service-type", nil, "service type")
		cmd.Flags().StringArray("tag", nil, "subnet tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
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
	case "subnet pool create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().StringArray("pool-prefix", nil, "subnet pool prefix")
		cmd.Flags().Int("default-prefix-length", 0, "default prefix length")
		cmd.Flags().Int("min-prefix-length", 0, "minimum prefix length")
		cmd.Flags().Int("max-prefix-length", 0, "maximum prefix length")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("address-scope", "", "address scope")
		cmd.Flags().Bool("default", false, "set as default")
		cmd.Flags().Bool("no-default", false, "set as non-default")
		cmd.Flags().Bool("share", false, "share resource")
		cmd.Flags().Bool("no-share", false, "do not share resource")
		cmd.Flags().String("description", "", "subnet pool description")
		cmd.Flags().Int("default-quota", 0, "default quota")
		cmd.Flags().StringArray("tag", nil, "subnet pool tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "subnet pool set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new subnet pool name")
		cmd.Flags().StringArray("pool-prefix", nil, "subnet pool prefix")
		cmd.Flags().Int("default-prefix-length", 0, "default prefix length")
		cmd.Flags().Int("min-prefix-length", 0, "minimum prefix length")
		cmd.Flags().Int("max-prefix-length", 0, "maximum prefix length")
		cmd.Flags().String("address-scope", "", "address scope")
		cmd.Flags().Bool("no-address-scope", false, "remove address scope")
		cmd.Flags().Bool("default", false, "set as default")
		cmd.Flags().Bool("no-default", false, "set as non-default")
		cmd.Flags().String("description", "", "subnet pool description")
		cmd.Flags().Int("default-quota", 0, "default quota")
		cmd.Flags().StringArray("tag", nil, "subnet pool tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
	case "subnet pool unset":
		cmd.Flags().StringArray("tag", nil, "subnet pool tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
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
	case "floating ip create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("subnet", "", "subnet")
		cmd.Flags().String("port", "", "port")
		cmd.Flags().String("floating-ip-address", "", "floating IP address")
		cmd.Flags().String("fixed-ip-address", "", "fixed IP address")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().String("description", "", "floating IP description")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("dns-domain", "", "DNS domain")
		cmd.Flags().String("dns-name", "", "DNS name")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().StringArray("tag", nil, "floating IP tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "floating ip set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("port", "", "port")
		cmd.Flags().String("fixed-ip-address", "", "fixed IP address")
		cmd.Flags().String("description", "", "floating IP description")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().Bool("no-qos-policy", false, "remove QoS policy")
		cmd.Flags().StringArray("tag", nil, "floating IP tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
	case "floating ip unset":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().Bool("port", false, "disassociate port")
		cmd.Flags().Bool("qos-policy", false, "remove QoS policy")
		cmd.Flags().StringArray("tag", nil, "floating IP tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
	case "floating ip port forwarding create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("internal-ip-address", "", "internal IP address")
		cmd.Flags().String("port", "", "internal network port")
		cmd.Flags().String("internal-protocol-port", "", "internal protocol port or range")
		cmd.Flags().String("external-protocol-port", "", "external protocol port or range")
		cmd.Flags().String("protocol", "", "protocol")
		cmd.Flags().String("description", "", "port forwarding description")
	case "floating ip port forwarding list":
		cmd.Flags().String("port", "", "filter by internal network port")
		cmd.Flags().String("external-protocol-port", "", "filter by external protocol port or range")
		cmd.Flags().String("protocol", "", "filter by protocol")
	case "floating ip port forwarding set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("port", "", "internal network port")
		cmd.Flags().String("internal-ip-address", "", "internal IP address")
		cmd.Flags().String("internal-protocol-port", "", "internal protocol port or range")
		cmd.Flags().String("external-protocol-port", "", "external protocol port or range")
		cmd.Flags().String("protocol", "", "protocol")
		cmd.Flags().String("description", "", "port forwarding description")
	case "ip availability list":
		cmd.Flags().Int("ip-version", 0, "filter by IP version")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "cached image clear":
		cmd.Flags().Bool("cache", false, "clear cached images")
		cmd.Flags().Bool("queue", false, "clear queued images")
	case "keypair list":
		cmd.Flags().String("user", "", "filter by user")
		cmd.Flags().String("user-domain", "", "user domain")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	case "keypair create":
		cmd.Flags().String("public-key", "", "public key file")
		cmd.Flags().String("private-key", "", "private key output file")
		cmd.Flags().String("type", "", "keypair type")
		cmd.Flags().String("user", "", "keypair owner")
		cmd.Flags().String("user-domain", "", "user domain")
	case "keypair delete":
		cmd.Flags().String("user", "", "keypair owner")
		cmd.Flags().String("user-domain", "", "user domain")
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
	case "image add project", "image remove project":
		cmd.Flags().String("project-domain", "", "project domain")
	case "image create":
		cmd.Flags().String("id", "", "image ID to reserve")
		cmd.Flags().String("container-format", "bare", "image container format")
		cmd.Flags().String("disk-format", "raw", "image disk format")
		cmd.Flags().Int("min-disk", 0, "minimum disk size in GB")
		cmd.Flags().Int("min-ram", 0, "minimum RAM size in MB")
		cmd.Flags().String("file", "", "upload image from local file")
		cmd.Flags().String("volume", "", "create image from a volume")
		cmd.Flags().Bool("force", false, "force image creation from volume")
		cmd.Flags().Bool("progress", false, "show upload progress")
		cmd.Flags().String("sign-key-path", "", "private key path for image signing")
		cmd.Flags().String("sign-cert-id", "", "certificate ID for image signing")
		cmd.Flags().Bool("protected", false, "prevent image deletion")
		cmd.Flags().Bool("unprotected", false, "allow image deletion")
		cmd.Flags().Bool("public", false, "public image visibility")
		cmd.Flags().Bool("private", false, "private image visibility")
		cmd.Flags().Bool("community", false, "community image visibility")
		cmd.Flags().Bool("shared", false, "shared image visibility")
		cmd.Flags().StringArray("property", nil, "image property key=value")
		cmd.Flags().StringArray("tag", nil, "image tag")
		cmd.Flags().String("project", "", "alternate project owner")
		cmd.Flags().Bool("import", false, "use Glance image import")
		cmd.Flags().String("project-domain", "", "project domain")
	case "image delete":
		cmd.Flags().String("store", "", "store to delete image from")
	case "image import":
		cmd.Flags().String("method", "glance-direct", "image import method")
		cmd.Flags().String("uri", "", "web-download URI")
		cmd.Flags().String("remote-image", "", "remote Glance image ID")
		cmd.Flags().String("remote-region", "", "remote Glance region")
		cmd.Flags().String("remote-service-interface", "", "remote Glance service interface")
		cmd.Flags().StringArray("store", nil, "backend store")
		cmd.Flags().Bool("all-stores", false, "import to all stores")
		cmd.Flags().Bool("allow-failure", false, "allow partial multi-store failure")
		cmd.Flags().Bool("disallow-failure", false, "disallow partial multi-store failure")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "image save":
		cmd.Flags().Int("chunk-size", 1024, "download buffer size")
		cmd.Flags().String("file", "", "downloaded image filename")
	case "image set":
		cmd.Flags().String("name", "", "new image name")
		cmd.Flags().Int("min-disk", 0, "minimum disk size in GB")
		cmd.Flags().Int("min-ram", 0, "minimum RAM size in MB")
		cmd.Flags().String("container-format", "", "image container format")
		cmd.Flags().String("disk-format", "", "image disk format")
		cmd.Flags().Bool("protected", false, "prevent image deletion")
		cmd.Flags().Bool("unprotected", false, "allow image deletion")
		cmd.Flags().Bool("public", false, "public image visibility")
		cmd.Flags().Bool("private", false, "private image visibility")
		cmd.Flags().Bool("community", false, "community image visibility")
		cmd.Flags().Bool("shared", false, "shared image visibility")
		cmd.Flags().StringArray("property", nil, "image property key=value")
		cmd.Flags().StringArray("tag", nil, "image tag")
		cmd.Flags().String("architecture", "", "operating system architecture")
		cmd.Flags().String("instance-id", "", "server instance ID")
		cmd.Flags().String("instance-uuid", "", "server instance ID")
		cmd.Flags().String("kernel-id", "", "kernel image ID")
		cmd.Flags().String("os-distro", "", "operating system distribution")
		cmd.Flags().String("os-version", "", "operating system version")
		cmd.Flags().String("ramdisk-id", "", "ramdisk image ID")
		cmd.Flags().Bool("deactivate", false, "deactivate image")
		cmd.Flags().Bool("activate", false, "activate image")
		cmd.Flags().String("project", "", "alternate project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("accept", false, "accept image membership")
		cmd.Flags().Bool("reject", false, "reject image membership")
		cmd.Flags().Bool("pending", false, "reset image membership to pending")
		cmd.Flags().Bool("hidden", false, "hide image")
		cmd.Flags().Bool("unhidden", false, "unhide image")
	case "image stage":
		cmd.Flags().String("file", "", "local file to stage")
		cmd.Flags().Bool("progress", false, "show upload progress")
	case "image unset":
		cmd.Flags().StringArray("tag", nil, "image tag")
		cmd.Flags().StringArray("property", nil, "image property key")
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
	case "image metadef object create":
		cmd.Flags().String("namespace", "", "metadef namespace")
	case "image metadef object update":
		cmd.Flags().String("name", "", "new object name")
	case "image metadef property create":
		cmd.Flags().String("name", "", "property name")
		cmd.Flags().String("title", "", "property title")
		cmd.Flags().String("type", "", "property type")
		cmd.Flags().String("schema", "", "property JSON schema")
	case "image metadef property set":
		cmd.Flags().String("name", "", "property name")
		cmd.Flags().String("title", "", "property title")
		cmd.Flags().String("type", "", "property type")
		cmd.Flags().String("schema", "", "property JSON schema")
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
	case "quota delete":
		cmd.Flags().Bool("all", false, "delete all service quotas")
		cmd.Flags().Bool("compute", false, "delete compute quotas")
		cmd.Flags().Bool("volume", false, "delete volume quotas")
		cmd.Flags().Bool("network", false, "delete network quotas")
	case "quota set":
		cmd.Flags().Bool("class", false, "set quota class")
		cmd.Flags().Bool("default", false, "set default quotas")
		for _, name := range []string{"cores", "injected-file-size", "injected-path-size", "injected-files", "instances", "key-pairs", "properties", "ram", "server-group-members", "server-groups", "backups", "backup-gigabytes", "gigabytes", "per-volume-gigabytes", "snapshots", "volumes", "floating-ips", "secgroup-rules", "secgroups", "networks", "subnets", "ports", "routers", "rbac-policies", "subnetpools"} {
			cmd.Flags().Int(name, 0, "quota value")
		}
		cmd.Flags().String("volume-type", "", "volume type")
		cmd.Flags().Bool("force", false, "force quota update")
		cmd.Flags().Bool("no-force", false, "do not force quota update")
		cmd.Flags().Bool("check-limit", false, "do not force quota update")
		_ = cmd.Flags().MarkHidden("check-limit")
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
	case "router add gateway":
		cmd.Flags().StringArray("fixed-ip", nil, "gateway fixed IP")
	case "router add route":
		cmd.Flags().StringArray("route", nil, "route destination=<cidr>,gateway=<ip>")
	case "router list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().Bool("enable", false, "list enabled routers")
		cmd.Flags().Bool("disable", false, "list disabled routers")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("agent", "", "filter by agent")
		addTagFilterFlags(cmd)
	case "router create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().Bool("enable", true, "enable router")
		cmd.Flags().Bool("disable", false, "disable router")
		cmd.Flags().Bool("distributed", false, "distributed router")
		cmd.Flags().Bool("centralized", false, "centralized router")
		cmd.Flags().Bool("ha", false, "high availability router")
		cmd.Flags().Bool("no-ha", false, "legacy router")
		cmd.Flags().String("description", "", "router description")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().StringArray("availability-zone-hint", nil, "availability zone hint")
		cmd.Flags().StringArray("tag", nil, "router tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
		cmd.Flags().StringArray("external-gateway", nil, "external gateway network")
		cmd.Flags().StringArray("fixed-ip", nil, "gateway fixed IP")
		cmd.Flags().Bool("enable-snat", false, "enable SNAT")
		cmd.Flags().Bool("disable-snat", false, "disable SNAT")
		cmd.Flags().Bool("enable-ndp-proxy", false, "enable NDP proxy")
		cmd.Flags().Bool("disable-ndp-proxy", false, "disable NDP proxy")
		cmd.Flags().String("flavor", "", "router flavor")
		cmd.Flags().Bool("enable-default-route-bfd", false, "enable default route BFD")
		cmd.Flags().Bool("disable-default-route-bfd", false, "disable default route BFD")
		cmd.Flags().Bool("enable-default-route-ecmp", false, "enable default route ECMP")
		cmd.Flags().Bool("disable-default-route-ecmp", false, "disable default route ECMP")
		cmd.Flags().String("qos-policy", "", "QoS policy")
	case "router set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new router name")
		cmd.Flags().String("description", "", "router description")
		cmd.Flags().Bool("enable", false, "enable router")
		cmd.Flags().Bool("disable", false, "disable router")
		cmd.Flags().Bool("distributed", false, "distributed router")
		cmd.Flags().Bool("centralized", false, "centralized router")
		cmd.Flags().StringArray("route", nil, "route destination=<cidr>,gateway=<ip>")
		cmd.Flags().Bool("no-route", false, "clear routes")
		cmd.Flags().Bool("ha", false, "high availability router")
		cmd.Flags().Bool("no-ha", false, "legacy router")
		cmd.Flags().StringArray("external-gateway", nil, "external gateway network")
		cmd.Flags().StringArray("fixed-ip", nil, "gateway fixed IP")
		cmd.Flags().Bool("enable-snat", false, "enable SNAT")
		cmd.Flags().Bool("disable-snat", false, "disable SNAT")
		cmd.Flags().Bool("enable-ndp-proxy", false, "enable NDP proxy")
		cmd.Flags().Bool("disable-ndp-proxy", false, "disable NDP proxy")
		cmd.Flags().String("qos-policy", "", "QoS policy")
		cmd.Flags().Bool("no-qos-policy", false, "remove QoS policy")
		cmd.Flags().StringArray("tag", nil, "router tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
		cmd.Flags().Bool("enable-default-route-bfd", false, "enable default route BFD")
		cmd.Flags().Bool("disable-default-route-bfd", false, "disable default route BFD")
		cmd.Flags().Bool("enable-default-route-ecmp", false, "enable default route ECMP")
		cmd.Flags().Bool("disable-default-route-ecmp", false, "disable default route ECMP")
	case "router unset":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().StringArray("route", nil, "route destination=<cidr>,gateway=<ip>")
		cmd.Flags().Bool("external-gateway", false, "remove external gateway")
		cmd.Flags().Bool("qos-policy", false, "remove QoS policy")
		cmd.Flags().StringArray("tag", nil, "router tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
	case "router remove gateway":
		cmd.Flags().StringArray("fixed-ip", nil, "gateway fixed IP")
	case "router remove route":
		cmd.Flags().StringArray("route", nil, "route destination=<cidr>,gateway=<ip>")
	case "security group create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("description", "", "security group description")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().Bool("stateful", false, "create a stateful security group")
		cmd.Flags().Bool("stateless", false, "create a stateless security group")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().StringArray("tag", nil, "security group tag")
		cmd.Flags().Bool("no-tag", false, "create without tags")
	case "security group set":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("name", "", "new security group name")
		cmd.Flags().String("description", "", "new security group description")
		cmd.Flags().Bool("stateful", false, "set a stateful security group")
		cmd.Flags().Bool("stateless", false, "set a stateless security group")
		cmd.Flags().StringArray("tag", nil, "security group tag")
		cmd.Flags().Bool("no-tag", false, "clear tags before applying tags")
	case "security group list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "list shared security groups")
		cmd.Flags().Bool("no-share", false, "list non-shared security groups")
		addTagFilterFlags(cmd)
	case "security group unset":
		cmd.Flags().StringArray("tag", nil, "security group tag")
		cmd.Flags().Bool("all-tag", false, "clear all tags")
	case "security group rule create":
		cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
		cmd.Flags().String("remote-ip", "", "remote IP address block")
		cmd.Flags().String("remote-group", "", "remote security group")
		cmd.Flags().String("remote-address-group", "", "remote address group")
		cmd.Flags().String("dst-port", "", "destination port or port range")
		cmd.Flags().String("protocol", "", "IP protocol")
		cmd.Flags().String("proto", "", "IP protocol")
		_ = cmd.Flags().MarkHidden("proto")
		cmd.Flags().String("description", "", "security group rule description")
		cmd.Flags().Int("icmp-type", 0, "ICMP type")
		cmd.Flags().Int("icmp-code", 0, "ICMP code")
		cmd.Flags().Bool("ingress", false, "ingress rule")
		cmd.Flags().Bool("egress", false, "egress rule")
		cmd.Flags().String("ethertype", "", "ethertype")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
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
	case "server group create":
		cmd.Flags().String("policy", "", "server group policy")
		cmd.Flags().StringArray("rule", nil, "server group rule key=value")
	case "server create":
		cmd.Flags().String("flavor", "", "server flavor")
		cmd.Flags().String("image", "", "server image")
		cmd.Flags().String("image-property", "", "image property key=value")
		cmd.Flags().String("volume", "", "boot volume")
		cmd.Flags().String("snapshot", "", "boot snapshot")
		cmd.Flags().Int("boot-from-volume", 0, "boot volume size")
		cmd.Flags().StringArray("block-device-mapping", nil, "deprecated block device mapping")
		cmd.Flags().StringArray("block-device", nil, "block device mapping")
		cmd.Flags().Int("swap", 0, "swap size")
		cmd.Flags().StringArray("ephemeral", nil, "ephemeral disk")
		cmd.Flags().StringArray("network", nil, "network")
		cmd.Flags().StringArray("port", nil, "port")
		cmd.Flags().Bool("no-network", false, "do not attach a network")
		cmd.Flags().Bool("auto-network", false, "automatically allocate network")
		cmd.Flags().StringArray("nic", nil, "network interface")
		cmd.Flags().String("password", "", "admin password")
		cmd.Flags().Bool("no-security-group", false, "do not attach security groups")
		cmd.Flags().StringArray("security-group", nil, "security group")
		cmd.Flags().String("key-name", "", "keypair name")
		cmd.Flags().StringArray("property", nil, "server metadata key=value")
		cmd.Flags().StringArray("file", nil, "file injection destination=source")
		cmd.Flags().String("user-data", "", "user data file")
		cmd.Flags().String("description", "", "server description")
		cmd.Flags().String("availability-zone", "", "availability zone")
		cmd.Flags().String("host", "", "requested host")
		cmd.Flags().String("hypervisor-hostname", "", "requested hypervisor hostname")
		cmd.Flags().String("server-group", "", "server group")
		cmd.Flags().StringArray("hint", nil, "scheduler hint key=value")
		cmd.Flags().Bool("use-config-drive", false, "enable config drive")
		cmd.Flags().Bool("no-config-drive", false, "disable config drive")
		cmd.Flags().String("config-drive", "", "config drive")
		cmd.Flags().Int("min", 0, "minimum number of servers")
		cmd.Flags().Int("max", 0, "maximum number of servers")
		cmd.Flags().StringArray("tag", nil, "server tag")
		cmd.Flags().String("hostname", "", "server hostname")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
		cmd.Flags().StringArray("trusted-image-cert", nil, "trusted image certificate")
	case "server delete":
		cmd.Flags().Bool("force", false, "force delete")
		cmd.Flags().Bool("all-projects", false, "delete in all projects")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server lock":
		cmd.Flags().String("reason", "", "lock reason")
	case "server migrate":
		cmd.Flags().Bool("live-migration", false, "live migrate server")
		cmd.Flags().String("host", "", "destination host")
		cmd.Flags().Bool("shared-migration", false, "shared live migration")
		cmd.Flags().Bool("block-migration", false, "block live migration")
		cmd.Flags().Bool("disk-overcommit", false, "allow disk over-commit")
		cmd.Flags().Bool("no-disk-overcommit", false, "disable disk over-commit")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server reboot":
		cmd.Flags().Bool("hard", false, "hard reboot")
		cmd.Flags().Bool("soft", false, "soft reboot")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server rebuild":
		cmd.Flags().String("image", "", "server image")
		cmd.Flags().String("name", "", "new server name")
		cmd.Flags().String("password", "", "admin password")
		cmd.Flags().StringArray("property", nil, "server metadata key=value")
		cmd.Flags().String("description", "", "server description")
		cmd.Flags().Bool("preserve-ephemeral", false, "preserve ephemeral disk")
		cmd.Flags().Bool("no-preserve-ephemeral", false, "do not preserve ephemeral disk")
		cmd.Flags().String("key-name", "", "keypair name")
		cmd.Flags().Bool("no-key-name", false, "unset keypair name")
		cmd.Flags().String("user-data", "", "user data file")
		cmd.Flags().Bool("no-user-data", false, "remove user data")
		cmd.Flags().StringArray("trusted-image-cert", nil, "trusted image certificate")
		cmd.Flags().Bool("no-trusted-image-certs", false, "remove trusted image certificates")
		cmd.Flags().String("hostname", "", "server hostname")
		cmd.Flags().Bool("reimage-boot-volume", false, "reimage boot volume")
		cmd.Flags().Bool("no-reimage-boot-volume", false, "do not reimage boot volume")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server resize":
		cmd.Flags().String("flavor", "", "server flavor")
		cmd.Flags().Bool("confirm", false, "confirm resize")
		cmd.Flags().Bool("revert", false, "revert resize")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server rescue":
		cmd.Flags().String("image", "", "rescue image")
		cmd.Flags().String("password", "", "admin password")
	case "server evacuate":
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
		cmd.Flags().String("host", "", "destination host")
		cmd.Flags().String("password", "", "admin password")
		cmd.Flags().Bool("shared-storage", false, "shared storage")
	case "server shelve":
		cmd.Flags().Bool("offload", false, "offload after shelving")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server unshelve":
		cmd.Flags().String("availability-zone", "", "availability zone")
		cmd.Flags().Bool("no-availability-zone", false, "unset availability zone")
		cmd.Flags().String("host", "", "destination host")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server set":
		cmd.Flags().String("name", "", "new server name")
		cmd.Flags().String("password", "", "admin password")
		cmd.Flags().Bool("no-password", false, "clear stored admin password")
		cmd.Flags().StringArray("property", nil, "server metadata key=value")
		cmd.Flags().Bool("auto-approve", false, "auto approve state reset")
		cmd.Flags().String("state", "", "server state")
		cmd.Flags().String("description", "", "server description")
		cmd.Flags().StringArray("tag", nil, "server tag")
		cmd.Flags().String("hostname", "", "server hostname")
	case "server unset":
		cmd.Flags().StringArray("property", nil, "metadata property")
		cmd.Flags().Bool("all-properties", false, "remove all metadata")
		cmd.Flags().Bool("description", false, "unset description")
		cmd.Flags().StringArray("tag", nil, "server tag")
		cmd.Flags().Bool("all-tags", false, "remove all tags")
	case "server image create":
		cmd.Flags().String("name", "", "image name")
		cmd.Flags().StringArray("property", nil, "image metadata key=value")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server backup create":
		cmd.Flags().String("name", "", "image name")
		cmd.Flags().String("type", "", "backup type")
		cmd.Flags().Int("rotate", 0, "backup rotation count")
		cmd.Flags().Bool("wait", false, "wait for operation to complete")
	case "server ssh":
		cmd.Flags().StringP("login", "l", "", "SSH login name")
		cmd.Flags().IntP("port", "p", 0, "SSH port")
		cmd.Flags().StringP("identity", "i", "", "SSH private key file")
		cmd.Flags().StringP("option", "o", "", "SSH option")
		cmd.Flags().BoolP("ipv4", "4", false, "use only IPv4 addresses")
		cmd.Flags().BoolP("ipv6", "6", false, "use only IPv6 addresses")
		cmd.Flags().Bool("public", false, "use public IP address")
		cmd.Flags().Bool("private", false, "use private IP address")
		cmd.Flags().String("address-type", "", "use other IP address type")
		cmd.Flags().BoolP("verbose", "v", false, "verbose SSH logging")
	case "server add fixed ip":
		cmd.Flags().String("fixed-ip-address", "", "fixed IP address")
		cmd.Flags().String("tag", "", "interface tag")
	case "server add network", "server add port":
		cmd.Flags().String("tag", "", "interface tag")
	case "server add floating ip":
		cmd.Flags().String("fixed-ip-address", "", "fixed IP address")
	case "server add volume":
		cmd.Flags().String("device", "", "device name")
		cmd.Flags().String("tag", "", "volume tag")
		cmd.Flags().Bool("enable-delete-on-termination", false, "delete volume when server is destroyed")
		cmd.Flags().Bool("disable-delete-on-termination", false, "preserve volume when server is destroyed")
	case "server volume set", "server volume update":
		cmd.Flags().Bool("delete-on-termination", false, "delete volume when server is destroyed")
		cmd.Flags().Bool("preserve-on-termination", false, "preserve volume when server is destroyed")
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
	case "block storage cleanup":
		cmd.Flags().String("cluster", "", "cluster name")
		cmd.Flags().String("host", "", "host name")
		cmd.Flags().String("binary", "", "service binary")
		cmd.Flags().Bool("up", false, "filter by up status")
		cmd.Flags().Bool("down", false, "filter by down status")
		cmd.Flags().Bool("disabled", false, "filter by disabled status")
		cmd.Flags().Bool("enabled", false, "filter by enabled status")
		cmd.Flags().String("resource-id", "", "resource UUID")
		cmd.Flags().String("resource-type", "", "resource type")
		cmd.Flags().String("service-id", "", "service database ID")
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
	case "block storage cluster set":
		cmd.Flags().String("binary", "cinder-volume", "service binary")
		cmd.Flags().Bool("enable", false, "enable cluster")
		cmd.Flags().Bool("disable", false, "disable cluster")
		cmd.Flags().String("disable-reason", "", "disable reason")
	case "block storage cluster show":
		cmd.Flags().String("binary", "", "service binary")
	case "block storage log level list":
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("service", "", "filter by service binary")
		cmd.Flags().String("log-prefix", "", "filter by log prefix")
	case "block storage log level set":
		cmd.Flags().String("host", "", "host name")
		cmd.Flags().String("service", "", "service binary")
		cmd.Flags().String("log-prefix", "", "log prefix")
	case "block storage volume manageable list", "block storage snapshot manageable list":
		cmd.Flags().String("cluster", "", "cluster name")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().Int("offset", 0, "number of entries to skip")
		cmd.Flags().String("sort", "", "sort expression")
	case "consistency group list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "consistency group create":
		cmd.Flags().String("volume-type", "", "volume type")
		cmd.Flags().String("source", "", "source consistency group")
		cmd.Flags().String("consistency-group-source", "", "source consistency group")
		cmd.Flags().String("snapshot", "", "source consistency group snapshot")
		cmd.Flags().String("consistency-group-snapshot", "", "source consistency group snapshot")
		cmd.Flags().String("description", "", "consistency group description")
		cmd.Flags().String("availability-zone", "", "availability zone")
	case "consistency group delete":
		cmd.Flags().Bool("force", false, "force delete")
	case "consistency group set":
		cmd.Flags().String("name", "", "new consistency group name")
		cmd.Flags().String("description", "", "new consistency group description")
	case "consistency group snapshot create":
		cmd.Flags().String("consistency-group", "", "consistency group")
		cmd.Flags().String("description", "", "consistency group snapshot description")
	case "consistency group snapshot list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("consistency-group", "", "filter by consistency group")
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
	case "volume attachment create":
		cmd.Flags().Bool("connect", false, "make an active connection")
		cmd.Flags().Bool("no-connect", false, "do not make an active connection")
		cmd.Flags().String("initiator", "", "connector initiator")
		cmd.Flags().String("ip", "", "connector IP")
		cmd.Flags().String("host", "", "connector host")
		cmd.Flags().String("platform", "", "connector platform")
		cmd.Flags().String("os-type", "", "connector OS type")
		cmd.Flags().Bool("multipath", false, "use multipath")
		cmd.Flags().Bool("no-multipath", false, "do not use multipath")
		cmd.Flags().String("mountpoint", "", "mountpoint")
		cmd.Flags().String("mode", "", "attachment mode")
	case "volume attachment set":
		cmd.Flags().String("initiator", "", "connector initiator")
		cmd.Flags().String("ip", "", "connector IP")
		cmd.Flags().String("host", "", "connector host")
		cmd.Flags().String("platform", "", "connector platform")
		cmd.Flags().String("os-type", "", "connector OS type")
		cmd.Flags().Bool("multipath", false, "use multipath")
		cmd.Flags().Bool("no-multipath", false, "do not use multipath")
		cmd.Flags().String("mountpoint", "", "mountpoint")
	case "volume backup list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("status", "", "filter by status")
		cmd.Flags().String("volume", "", "filter by volume")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume backup create":
		cmd.Flags().Bool("force", false, "force backup")
		cmd.Flags().Bool("incremental", false, "incremental backup")
		cmd.Flags().String("name", "", "backup name")
		cmd.Flags().String("description", "", "backup description")
		cmd.Flags().String("container", "", "backup container")
		cmd.Flags().String("snapshot", "", "source snapshot")
		cmd.Flags().String("availability-zone", "", "availability zone")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
	case "volume backup delete":
		cmd.Flags().Bool("force", false, "force delete")
	case "volume backup restore":
		cmd.Flags().Bool("force", false, "restore to existing volume")
	case "volume backup set":
		cmd.Flags().String("name", "", "backup name")
		cmd.Flags().String("description", "", "backup description")
		cmd.Flags().String("state", "", "backup state")
		cmd.Flags().Bool("no-property", false, "clear metadata")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
	case "volume backup unset":
		cmd.Flags().StringArray("property", nil, "metadata key")
	case "volume service list":
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("service", "", "filter by service binary")
		cmd.Flags().Bool("long", false, "list additional fields")
	case "volume service set":
		cmd.Flags().Bool("enable", false, "enable service")
		cmd.Flags().Bool("disable", false, "disable service")
		cmd.Flags().String("disable-reason", "", "disable reason")
	case "volume host set":
		cmd.Flags().Bool("disable", false, "freeze and disable host")
		cmd.Flags().Bool("enable", false, "thaw and enable host")
	case "volume group list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume group create":
		cmd.Flags().String("volume-group-type", "", "volume group type")
		cmd.Flags().StringArray("volume-type", nil, "volume type")
		cmd.Flags().String("source-group", "", "source volume group")
		cmd.Flags().String("group-snapshot", "", "source group snapshot")
		cmd.Flags().String("name", "", "volume group name")
		cmd.Flags().String("description", "", "volume group description")
		cmd.Flags().String("availability-zone", "", "availability zone")
	case "volume group delete":
		cmd.Flags().Bool("force", false, "delete group volumes")
	case "volume group failover":
		cmd.Flags().Bool("allow-attached-volume", false, "allow attached volumes")
		cmd.Flags().Bool("disallow-attached-volume", false, "disallow attached volumes")
		cmd.Flags().String("secondary-backend-id", "", "secondary backend ID")
	case "volume group set":
		cmd.Flags().String("name", "", "volume group name")
		cmd.Flags().String("description", "", "volume group description")
		cmd.Flags().Bool("enable-replication", false, "enable replication")
		cmd.Flags().Bool("disable-replication", false, "disable replication")
	case "volume group show":
		cmd.Flags().Bool("volumes", false, "show volumes in the group")
		cmd.Flags().Bool("no-volumes", false, "do not show volumes in the group")
		cmd.Flags().Bool("replication-targets", false, "show replication targets")
		cmd.Flags().Bool("no-replication-targets", false, "do not show replication targets")
	case "volume group snapshot list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume group snapshot create":
		cmd.Flags().String("name", "", "volume group snapshot name")
		cmd.Flags().String("description", "", "volume group snapshot description")
	case "volume group type list":
		cmd.Flags().Bool("default", false, "list the default volume group type")
	case "volume group type create":
		cmd.Flags().String("description", "", "volume group type description")
		cmd.Flags().Bool("public", false, "public group type")
		cmd.Flags().Bool("private", false, "private group type")
	case "volume group type set":
		cmd.Flags().String("name", "", "volume group type name")
		cmd.Flags().String("description", "", "volume group type description")
		cmd.Flags().Bool("public", false, "public group type")
		cmd.Flags().Bool("private", false, "private group type")
		cmd.Flags().Bool("no-property", false, "clear properties")
		cmd.Flags().StringArray("property", nil, "property key=value")
	case "volume message list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Int("limit", 0, "maximum number of entries")
		cmd.Flags().String("marker", "", "pagination marker")
	case "volume migrate":
		cmd.Flags().String("host", "", "destination host")
		cmd.Flags().Bool("force-host-copy", false, "force host copy")
		cmd.Flags().Bool("lock-volume", false, "lock volume")
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
	case "volume snapshot create":
		cmd.Flags().String("volume", "", "source volume")
		cmd.Flags().Bool("force", false, "force snapshot")
		cmd.Flags().String("description", "", "snapshot description")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
		cmd.Flags().StringArray("remote-source", nil, "remote source key=value")
	case "volume snapshot delete":
		cmd.Flags().Bool("force", false, "force delete")
		cmd.Flags().Bool("remote", false, "unmanage remote snapshot")
	case "volume snapshot set":
		cmd.Flags().String("name", "", "snapshot name")
		cmd.Flags().String("description", "", "snapshot description")
		cmd.Flags().String("state", "", "snapshot state")
		cmd.Flags().String("progress", "", "snapshot progress")
		cmd.Flags().Bool("no-property", false, "clear metadata")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
	case "volume snapshot unset":
		cmd.Flags().StringArray("property", nil, "metadata key")
	case "volume summary":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume create":
		cmd.Flags().Int("size", 0, "volume size in GB")
		cmd.Flags().String("type", "", "volume type")
		cmd.Flags().String("image", "", "source image")
		cmd.Flags().String("snapshot", "", "source snapshot")
		cmd.Flags().String("source", "", "source volume")
		cmd.Flags().String("backup", "", "source backup")
		cmd.Flags().StringArray("remote-source", nil, "remote source key=value")
		cmd.Flags().String("description", "", "volume description")
		cmd.Flags().String("availability-zone", "", "availability zone")
		cmd.Flags().String("consistency-group", "", "consistency group")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
		cmd.Flags().StringArray("hint", nil, "scheduler hint key=value")
		cmd.Flags().Bool("bootable", false, "mark bootable")
		cmd.Flags().Bool("non-bootable", false, "mark non-bootable")
		cmd.Flags().Bool("read-only", false, "set read-only")
		cmd.Flags().Bool("read-write", false, "set read-write")
		cmd.Flags().String("host", "", "manage source host")
		cmd.Flags().String("cluster", "", "manage source cluster")
	case "volume delete":
		cmd.Flags().Bool("force", false, "force delete")
		cmd.Flags().Bool("purge", false, "delete snapshots with volume")
		cmd.Flags().Bool("cascade", false, "delete snapshots with volume")
		cmd.Flags().Bool("remote", false, "unmanage remote volume")
	case "volume set":
		cmd.Flags().String("name", "", "volume name")
		cmd.Flags().String("description", "", "volume description")
		cmd.Flags().Int("size", 0, "new volume size")
		cmd.Flags().String("type", "", "new volume type")
		cmd.Flags().String("migration-policy", "", "migration policy")
		cmd.Flags().String("state", "", "volume state")
		cmd.Flags().Bool("attached", false, "set attached")
		cmd.Flags().Bool("detached", false, "set detached")
		cmd.Flags().Bool("bootable", false, "mark bootable")
		cmd.Flags().Bool("non-bootable", false, "mark non-bootable")
		cmd.Flags().Bool("read-only", false, "set read-only")
		cmd.Flags().Bool("read-write", false, "set read-write")
		cmd.Flags().Bool("no-property", false, "clear metadata")
		cmd.Flags().StringArray("property", nil, "metadata key=value")
		cmd.Flags().StringArray("image-property", nil, "image metadata key=value")
	case "volume unset":
		cmd.Flags().StringArray("property", nil, "metadata key")
		cmd.Flags().StringArray("image-property", nil, "image metadata key")
	case "versions show":
		cmd.Flags().Bool("all-interfaces", false, "show all interfaces")
		cmd.Flags().String("interface", "", "show a specific interface")
		cmd.Flags().String("region-name", "", "show a specific region")
		cmd.Flags().String("service", "", "show a specific service")
		cmd.Flags().String("status", "", "show a specific version status")
	case "volume transfer request list":
		cmd.Flags().Bool("all-projects", false, "include all projects")
	case "volume transfer request create":
		cmd.Flags().String("name", "", "transfer request name")
		cmd.Flags().Bool("snapshots", false, "allow snapshots")
		cmd.Flags().Bool("no-snapshots", false, "disallow snapshots")
	case "volume transfer request accept":
		cmd.Flags().String("auth-key", "", "transfer auth key")
	case "volume qos create":
		cmd.Flags().String("consumer", "", "QoS consumer")
		cmd.Flags().StringArray("property", nil, "property key=value")
	case "volume qos delete":
		cmd.Flags().Bool("force", false, "force delete")
	case "volume qos set":
		cmd.Flags().Bool("no-property", false, "clear properties")
		cmd.Flags().StringArray("property", nil, "property key=value")
	case "volume qos unset":
		cmd.Flags().StringArray("property", nil, "property key")
	case "volume qos disassociate":
		cmd.Flags().String("volume-type", "", "volume type")
		cmd.Flags().Bool("all", false, "disassociate all")
	case "volume type create":
		cmd.Flags().String("description", "", "volume type description")
		cmd.Flags().Bool("public", false, "public type")
		cmd.Flags().Bool("private", false, "private type")
		cmd.Flags().StringArray("property", nil, "property key=value")
		cmd.Flags().Bool("multiattach", false, "enable multiattach")
		cmd.Flags().Bool("cacheable", false, "enable cacheable")
		cmd.Flags().Bool("replicated", false, "enable replication")
		cmd.Flags().StringArray("availability-zone", nil, "availability zone")
		cmd.Flags().String("project", "", "project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("encryption-provider", "", "encryption provider")
		cmd.Flags().String("encryption-cipher", "", "encryption cipher")
		cmd.Flags().Int("encryption-key-size", 0, "encryption key size")
		cmd.Flags().String("encryption-control-location", "", "encryption control location")
	case "volume type set":
		cmd.Flags().String("name", "", "volume type name")
		cmd.Flags().String("description", "", "volume type description")
		cmd.Flags().StringArray("property", nil, "property key=value")
		cmd.Flags().Bool("multiattach", false, "enable multiattach")
		cmd.Flags().Bool("cacheable", false, "enable cacheable")
		cmd.Flags().Bool("replicated", false, "enable replication")
		cmd.Flags().StringArray("availability-zone", nil, "availability zone")
		cmd.Flags().String("project", "", "project")
		cmd.Flags().Bool("public", false, "public type")
		cmd.Flags().Bool("private", false, "private type")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("encryption-provider", "", "encryption provider")
		cmd.Flags().String("encryption-cipher", "", "encryption cipher")
		cmd.Flags().Int("encryption-key-size", 0, "encryption key size")
		cmd.Flags().String("encryption-control-location", "", "encryption control location")
	case "volume type unset":
		cmd.Flags().StringArray("property", nil, "property key")
		cmd.Flags().String("project", "", "project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("encryption-type", false, "remove encryption type")
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
	case "address group create", "address group delete", "address group list", "address group set", "address group show", "address group unset",
		"address scope create", "address scope delete", "address scope list", "address scope set", "address scope show",
		"aggregate list", "aggregate show",
		"allocation candidate list",
		"availability zone list",
		"block storage cleanup",
		"block storage cluster list", "block storage cluster set", "block storage cluster show",
		"block storage log level list", "block storage log level set",
		"block storage snapshot manageable list", "block storage volume manageable list",
		"cached image clear", "cached image delete",
		"cached image list", "cached image queue",
		"compute agent list",
		"compute service list",
		"consistency group add volume", "consistency group create", "consistency group delete", "consistency group list",
		"consistency group remove volume", "consistency group set", "consistency group show",
		"consistency group snapshot create", "consistency group snapshot delete", "consistency group snapshot list", "consistency group snapshot show",
		"console connection show",
		"console log show", "console url show",
		"container create", "container delete", "container list", "container set", "container show", "container unset",
		"extension list", "extension show",
		"flavor list", "flavor show",
		"floating ip create", "floating ip delete", "floating ip list", "floating ip pool list",
		"floating ip port forwarding create", "floating ip port forwarding delete", "floating ip port forwarding list", "floating ip port forwarding set", "floating ip port forwarding show",
		"floating ip set", "floating ip show", "floating ip unset",
		"host list", "host show",
		"hypervisor list", "hypervisor show", "hypervisor stats show",
		"image add project",
		"image create", "image delete",
		"image import", "image import info", "image list",
		"image member get", "image member list", "image show",
		"image remove project",
		"image save", "image set", "image stage", "image unset",
		"image metadef namespace create", "image metadef namespace delete",
		"image metadef namespace list", "image metadef namespace set",
		"image metadef namespace show",
		"image metadef object create", "image metadef object delete",
		"image metadef object list", "image metadef object property show",
		"image metadef object show", "image metadef object update",
		"image metadef property create", "image metadef property delete",
		"image metadef property list", "image metadef property set",
		"image metadef property show",
		"image metadef resource type association create",
		"image metadef resource type association delete",
		"image metadef resource type association list",
		"image metadef resource type list",
		"image stores list", "image task list", "image task show",
		"ip availability list", "ip availability show",
		"keypair create", "keypair delete", "keypair list", "keypair show",
		"limits show",
		"network agent list", "network agent show",
		"network create", "network delete", "network list", "network set", "network show", "network unset",
		"network service provider list",
		"network qos policy create", "network qos policy delete", "network qos policy list", "network qos policy set", "network qos policy show",
		"network qos rule create", "network qos rule delete", "network qos rule list", "network qos rule set", "network qos rule show",
		"network qos rule type list", "network qos rule type show",
		"network rbac create", "network rbac delete", "network rbac list", "network rbac set", "network rbac show",
		"network segment create", "network segment delete", "network segment list", "network segment set", "network segment show",
		"network subport list",
		"network trunk create", "network trunk delete", "network trunk list", "network trunk set", "network trunk show", "network trunk unset",
		"object create", "object delete", "object list", "object save", "object set", "object show", "object unset",
		"object store account show",
		"port create", "port delete", "port list", "port set", "port show", "port unset",
		"quota delete", "quota list", "quota set", "quota show",
		"resource class list", "resource class show",
		"resource provider aggregate list",
		"resource provider allocation show",
		"resource provider inventory list", "resource provider inventory show",
		"resource provider list", "resource provider show",
		"resource provider trait list",
		"resource provider usage show",
		"router add gateway", "router add port", "router add route", "router add subnet",
		"router create", "router delete", "router list",
		"router remove gateway", "router remove port", "router remove route", "router remove subnet",
		"router set", "router show", "router unset",
		"security group create", "security group delete", "security group list", "security group set", "security group show", "security group unset",
		"security group rule create", "security group rule delete", "security group rule list", "security group rule show",
		"server event list", "server event show",
		"server list", "server show",
		"server group create", "server group delete", "server group list", "server group show",
		"server migration list",
		"server volume list",
		"subnet create", "subnet delete", "subnet list", "subnet set", "subnet show", "subnet unset",
		"subnet pool create", "subnet pool delete", "subnet pool list", "subnet pool set", "subnet pool show", "subnet pool unset",
		"trait list", "trait show",
		"usage list", "usage show",
		"versions show",
		"volume attachment complete", "volume attachment create", "volume attachment delete",
		"volume attachment list", "volume attachment set", "volume attachment show",
		"volume backend capability show",
		"volume backend pool list",
		"volume backup create", "volume backup delete", "volume backup list",
		"volume backup record export", "volume backup record import", "volume backup restore",
		"volume backup set", "volume backup show", "volume backup unset",
		"volume group create", "volume group delete", "volume group failover", "volume group list", "volume group set", "volume group show",
		"volume group snapshot create", "volume group snapshot delete", "volume group snapshot list", "volume group snapshot show",
		"volume group type create", "volume group type delete", "volume group type list", "volume group type set", "volume group type show",
		"volume host set",
		"volume create", "volume delete", "volume list", "volume set", "volume show", "volume unset",
		"volume message delete", "volume message list", "volume message show",
		"volume migrate",
		"volume qos associate", "volume qos create", "volume qos delete", "volume qos disassociate",
		"volume qos list", "volume qos set", "volume qos show", "volume qos unset",
		"volume revert",
		"volume service list", "volume service set",
		"volume snapshot create", "volume snapshot delete", "volume snapshot list",
		"volume snapshot set", "volume snapshot show", "volume snapshot unset",
		"volume summary",
		"volume transfer request accept", "volume transfer request create", "volume transfer request delete",
		"volume transfer request list", "volume transfer request show",
		"volume type create", "volume type delete", "volume type list", "volume type set", "volume type show", "volume type unset":
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
