package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats"
	bsservices "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	computeservices "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	secgroups "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	secgrouprules "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"
)

func runCoreRead(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}

		switch path {
		case "aggregate list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return aggregateList(cmd.Context(), stdout, opts, client)
		case "aggregate show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return aggregateShow(cmd.Context(), stdout, opts, client, args)
		case "availability zone list":
			return availabilityZoneList(cmd.Context(), stdout, opts, clients)
		case "compute service list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeServiceList(cmd.Context(), stdout, opts, client)
		case "extension list":
			return extensionList(cmd.Context(), stdout, opts, clients)
		case "extension show":
			return extensionShow(cmd.Context(), stdout, opts, clients, args)
		case "flavor list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeFlavorList(cmd.Context(), stdout, opts, client)
		case "flavor show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeFlavorShow(cmd.Context(), stdout, opts, client, args)
		case "allocation candidate list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return allocationCandidateList(cmd.Context(), stdout, opts, client)
		case "container list":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerList(cmd.Context(), stdout, opts, client)
		case "container show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerShow(cmd.Context(), stdout, opts, client, args)
		case "floating ip list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPList(cmd.Context(), stdout, opts, client)
		case "floating ip show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPShow(cmd.Context(), stdout, opts, client, args)
		case "image list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageList(cmd.Context(), stdout, opts, client)
		case "image member get":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMemberGet(cmd.Context(), stdout, opts, client, args)
		case "image member list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMemberList(cmd.Context(), stdout, opts, client, args)
		case "image show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageShow(cmd.Context(), stdout, opts, client, args)
		case "image stores list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageStoresList(cmd.Context(), stdout, opts, client)
		case "image task list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageTaskList(cmd.Context(), stdout, opts, client)
		case "image task show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageTaskShow(cmd.Context(), stdout, opts, client, args)
		case "network list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkList(cmd.Context(), stdout, opts, client)
		case "network show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkShow(cmd.Context(), stdout, opts, client, args)
		case "object list":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectList(cmd.Context(), stdout, opts, client, args)
		case "object show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectShow(cmd.Context(), stdout, opts, client, args)
		case "object store account show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectStoreAccountShow(cmd.Context(), stdout, opts, client)
		case "quota list":
			return quotaList(cmd.Context(), stdout, opts, clients)
		case "quota show":
			return quotaShow(cmd.Context(), stdout, opts, clients, args)
		case "keypair list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairList(cmd.Context(), stdout, opts, client)
		case "keypair show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairShow(cmd.Context(), stdout, opts, client, args)
		case "hypervisor list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorList(cmd.Context(), stdout, opts, client)
		case "hypervisor show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorShow(cmd.Context(), stdout, opts, client, args)
		case "hypervisor stats show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorStatsShow(cmd.Context(), stdout, opts, client)
		case "limits show":
			return limitsShow(cmd.Context(), stdout, opts, clients)
		case "port list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			computeClient, _ := clients.computeV2()
			return portList(cmd.Context(), stdout, opts, client, computeClient)
		case "port show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portShow(cmd.Context(), stdout, opts, client, args)
		case "resource class list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceClassList(cmd.Context(), stdout, opts, client)
		case "resource class show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceClassShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider aggregate list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderAggregateList(cmd.Context(), stdout, opts, client, args)
		case "resource provider allocation show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderAllocationShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider inventory list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderInventoryList(cmd.Context(), stdout, opts, client, args)
		case "resource provider inventory show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderInventoryShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderList(cmd.Context(), stdout, opts, client)
		case "resource provider show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider trait list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderTraitList(cmd.Context(), stdout, opts, client, args)
		case "resource provider usage show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderUsageShow(cmd.Context(), stdout, opts, client, args)
		case "router list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerList(cmd.Context(), stdout, opts, client)
		case "router show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerShow(cmd.Context(), stdout, opts, client, args)
		case "security group list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupList(cmd.Context(), stdout, opts, client)
		case "security group show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupShow(cmd.Context(), stdout, opts, client, args)
		case "security group rule list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupRuleList(cmd.Context(), stdout, opts, client, args)
		case "security group rule show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupRuleShow(cmd.Context(), stdout, opts, client, args)
		case "server list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return computeServerList(cmd.Context(), stdout, opts, client, imageClient)
		case "server show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeServerShow(cmd.Context(), stdout, opts, client, args)
		case "server group list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverGroupList(cmd.Context(), stdout, opts, client)
		case "server group show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverGroupShow(cmd.Context(), stdout, opts, client, args)
		case "subnet list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetList(cmd.Context(), stdout, opts, client)
		case "subnet show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetShow(cmd.Context(), stdout, opts, client, args)
		case "trait list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return traitList(cmd.Context(), stdout, opts, client)
		case "trait show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return traitShow(cmd.Context(), stdout, opts, client, args)
		case "volume list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeList(cmd.Context(), stdout, opts, client)
		case "volume show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeShow(cmd.Context(), stdout, opts, client, args)
		case "volume backend pool list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackendPoolList(cmd.Context(), stdout, opts, client)
		case "volume backup list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupList(cmd.Context(), stdout, opts, client)
		case "volume backup show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupShow(cmd.Context(), stdout, opts, client, args)
		case "volume service list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeServiceList(cmd.Context(), stdout, opts, client)
		case "volume snapshot list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotList(cmd.Context(), stdout, opts, client)
		case "volume snapshot show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotShow(cmd.Context(), stdout, opts, client, args)
		case "volume type list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeList(cmd.Context(), stdout, opts, client)
		case "volume type show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeShow(cmd.Context(), stdout, opts, client, args)
		case "volume transfer request list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferList(cmd.Context(), stdout, opts, client)
		case "volume transfer request show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferShow(cmd.Context(), stdout, opts, client, args)
		case "versions show":
			return versionsShow(cmd.Context(), stdout, opts, clients)
		default:
			return fmt.Errorf("core read command %q is not wired", path)
		}
	}
}

func computeServerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient) error {
	page, err := servers.List(client, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := extractServers(page)
	if err != nil {
		return err
	}
	flavorNames := flavorNameMap(ctx, client)
	imageNames := imageNameMap(ctx, imageClient)
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":       item.ID,
			"Name":     item.Name,
			"Status":   item.Status,
			"Networks": serverNetworks(item.Addresses),
			"Image":    serverImage(item.Image, imageNames),
			"Flavor":   serverFlavor(item.Flavor, flavorNames),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Networks", "Image", "Flavor"}, rows)
}

func computeServerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server show requires <server>")
	}
	item, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"project_id", item.TenantID},
		{"user_id", item.UserID},
		{"created", item.Created},
		{"updated", item.Updated},
		{"image", serverImage(item.Image, nil)},
		{"flavor", serverFlavor(item.Flavor, nil)},
		{"addresses", serverNetworks(item.Addresses)},
		{"metadata", item.Metadata},
		{"key_name", nilIfEmpty(item.KeyName)},
	})
}

func aggregateList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := aggregates.List(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := aggregates.ExtractAggregates(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                item.ID,
			"Name":              item.Name,
			"Availability Zone": item.AvailabilityZone,
		}
		if boolFlag(opts, "long") {
			row["Hosts"] = item.Hosts
			row["Properties"] = item.Metadata
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Availability Zone"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Hosts", "Properties")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func aggregateShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("aggregate show requires <aggregate>")
	}
	item, err := findAggregate(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"availability_zone", item.AvailabilityZone},
		{"hosts", item.Hosts},
		{"metadata", item.Metadata},
		{"uuid", item.UUID},
		{"created_at", oscTime(item.CreatedAt)},
		{"updated_at", oscTime(item.UpdatedAt)},
		{"deleted", item.Deleted},
	})
}

func computeServiceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := computeservices.List(client, computeservices.ListOpts{
		Binary: flagValue(opts, "service"),
		Host:   flagValue(opts, "host"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := computeservices.ExtractServices(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":         item.ID,
			"Binary":     item.Binary,
			"Host":       item.Host,
			"Zone":       item.Zone,
			"Status":     item.Status,
			"State":      item.State,
			"Updated At": oscTime(item.UpdatedAt),
		}
		if boolFlag(opts, "long") {
			row["Disabled Reason"] = nilIfEmpty(item.DisabledReason)
			row["Forced Down"] = item.ForcedDown
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Binary", "Host", "Zone", "Status", "State", "Updated At"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Disabled Reason", "Forced Down")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func hypervisorList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := hypervisors.ListOpts{}
	if matching := flagValue(opts, "matching"); matching != "" {
		listOpts.HypervisorHostnamePattern = &matching
	}
	if boolFlag(opts, "with-servers") {
		withServers := true
		listOpts.WithServers = &withServers
	}
	page, err := hypervisors.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := hypervisors.ExtractHypervisors(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":                  item.ID,
			"Hypervisor Hostname": item.HypervisorHostname,
			"Hypervisor Type":     item.HypervisorType,
			"Host IP":             item.HostIP,
			"State":               item.State,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Hypervisor Hostname", "Hypervisor Type", "Host IP", "State"}, rows)
}

func hypervisorShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("hypervisor show requires <hypervisor>")
	}
	item, err := findHypervisor(ctx, client, args[0])
	if err != nil {
		return err
	}
	uptime, _ := hypervisors.GetUptime(ctx, client, item.ID).Extract()
	fields := []outputField{
		{"id", item.ID},
		{"hypervisor_hostname", item.HypervisorHostname},
		{"hypervisor_type", item.HypervisorType},
		{"hypervisor_version", item.HypervisorVersion},
		{"host_ip", item.HostIP},
		{"state", item.State},
		{"status", item.Status},
		{"service_host", item.Service.Host},
		{"service_id", item.Service.ID},
		{"vcpus", item.VCPUs},
		{"vcpus_used", item.VCPUsUsed},
		{"memory_mb", item.MemoryMB},
		{"memory_mb_used", item.MemoryMBUsed},
		{"local_gb", item.LocalGB},
		{"local_gb_used", item.LocalGBUsed},
		{"free_disk_gb", item.FreeDiskGB},
		{"free_ram_mb", item.FreeRamMB},
		{"running_vms", item.RunningVMs},
		{"current_workload", item.CurrentWorkload},
		{"cpu_info", item.CPUInfo},
	}
	if uptime != nil {
		fields = append(fields, outputField{"uptime", uptime.Uptime})
	}
	return renderShowOutput(stdout, opts, fields)
}

func hypervisorStatsShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	stats, err := hypervisors.GetStatistics(ctx, client).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"count", stats.Count},
		{"current_workload", stats.CurrentWorkload},
		{"disk_available_least", stats.DiskAvailableLeast},
		{"free_disk_gb", stats.FreeDiskGB},
		{"free_ram_mb", stats.FreeRamMB},
		{"local_gb", stats.LocalGB},
		{"local_gb_used", stats.LocalGBUsed},
		{"memory_mb", stats.MemoryMB},
		{"memory_mb_used", stats.MemoryMBUsed},
		{"running_vms", stats.RunningVMs},
		{"vcpus", stats.VCPUs},
		{"vcpus_used", stats.VCPUsUsed},
	})
}

func computeFlavorList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := flavors.ListDetail(client, nil).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":        item.ID,
			"Name":      item.Name,
			"RAM":       item.RAM,
			"Disk":      item.Disk,
			"Ephemeral": item.Ephemeral,
			"VCPUs":     item.VCPUs,
			"Is Public": item.IsPublic,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "RAM", "Disk", "Ephemeral", "VCPUs", "Is Public"}, rows)
}

func computeFlavorShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("flavor show requires <flavor>")
	}
	item, err := findFlavor(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"ram", item.RAM},
		{"disk", item.Disk},
		{"ephemeral", item.Ephemeral},
		{"vcpus", item.VCPUs},
		{"is_public", item.IsPublic},
		{"swap", item.Swap},
		{"rxtx_factor", item.RxTxFactor},
		{"properties", item.ExtraSpecs},
	})
}

func imageList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name, "Status": string(item.Status)})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status"}, rows)
}

func imageShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image show requires <image>")
	}
	item, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", string(item.Status)},
		{"visibility", string(item.Visibility)},
		{"protected", item.Protected},
		{"container_format", item.ContainerFormat},
		{"disk_format", item.DiskFormat},
		{"min_disk", item.MinDiskGigabytes},
		{"min_ram", item.MinRAMMegabytes},
		{"owner", item.Owner},
		{"size", item.SizeBytes},
		{"checksum", item.Checksum},
		{"tags", item.Tags},
		{"created_at", oscTime(item.CreatedAt)},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func imageMemberList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image member list requires <image>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	page, err := members.List(client, image.ID).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := members.ExtractMembers(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Image ID":  item.ImageID,
			"Member ID": item.MemberID,
			"Status":    item.Status,
		})
	}
	return renderListOutput(stdout, opts, []string{"Image ID", "Member ID", "Status"}, rows)
}

func imageMemberGet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image member get requires <image> <project>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	item, err := members.Get(ctx, client, image.ID, args[1]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"image_id", item.ImageID},
		{"member_id", item.MemberID},
		{"schema", item.Schema},
		{"status", item.Status},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

type imageTaskListOpts struct {
	Limit   int    `q:"limit"`
	Marker  string `q:"marker"`
	SortDir string `q:"sort_dir"`
	SortKey string `q:"sort_key"`
	Type    string `q:"type"`
	Status  string `q:"status"`
}

func (opts imageTaskListOpts) ToTaskListQuery() (string, error) {
	query, err := gophercloud.BuildQueryString(opts)
	return query.String(), err
}

func imageTaskList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := tasks.List(client, imageTaskListOpts{
		Limit:   intFlag(opts, "limit"),
		Marker:  flagValue(opts, "marker"),
		SortDir: flagValue(opts, "sort-dir"),
		SortKey: flagValue(opts, "sort-key"),
		Type:    flagValue(opts, "type"),
		Status:  flagValue(opts, "status"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := tasks.ExtractTasks(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Type":   item.Type,
			"Status": item.Status,
			"Owner":  item.Owner,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Type", "Status", "Owner"}, rows)
}

func imageTaskShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image task show requires <Task ID>")
	}
	item, err := tasks.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"expires_at", oscTime(item.ExpiresAt)},
		{"id", item.ID},
		{"input", item.Input},
		{"message", nilIfEmpty(item.Message)},
		{"owner_id", item.Owner},
		{"properties", map[string]any{}},
		{"result", item.Result},
		{"status", item.Status},
		{"type", item.Type},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

type imageStore struct {
	ID          string         `json:"id"`
	Description any            `json:"description"`
	Default     any            `json:"default"`
	Properties  map[string]any `json:"properties"`
}

func imageStoresList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	url := client.ServiceURL("info", "stores")
	if boolFlag(opts, "detail") {
		url = client.ServiceURL("info", "stores", "detail")
	}
	var response struct {
		Stores []imageStore `json:"stores"`
	}
	_, err := client.Get(ctx, url, &response, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return fmt.Errorf("Multi Backend support not enabled")
		}
		return err
	}
	rows := make([]outputRow, 0, len(response.Stores))
	for _, item := range response.Stores {
		row := outputRow{
			"ID":          item.ID,
			"Description": item.Description,
			"Default":     storeDefault(item.Default),
		}
		if boolFlag(opts, "detail") {
			row["Properties"] = item.Properties
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Description", "Default"}
	if boolFlag(opts, "detail") {
		columns = append(columns, "Properties")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func storeDefault(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		return strings.EqualFold(typed, "true")
	default:
		return typed
	}
}

func networkList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := networks.List(client, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := networks.ExtractNetworks(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name, "Subnets": item.Subnets})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Subnets"}, rows)
}

func networkShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network show requires <network>")
	}
	item, err := findNetwork(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"project_id", item.ProjectID},
		{"admin_state_up", item.AdminStateUp},
		{"shared", item.Shared},
		{"subnets", item.Subnets},
		{"description", item.Description},
		{"tags", item.Tags},
	})
}

func routerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := routers.ListOpts{
		Name:       flagValue(opts, "name"),
		ProjectID:  flagValue(opts, "project"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if boolFlag(opts, "enable") {
		enabled := true
		listOpts.AdminStateUp = &enabled
	}
	if boolFlag(opts, "disable") {
		disabled := false
		listOpts.AdminStateUp = &disabled
	}
	page, err := routers.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := routers.ExtractRouters(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Status":      item.Status,
			"State":       item.AdminStateUp,
			"Project":     item.ProjectID,
			"Distributed": item.Distributed,
			"HA":          false,
		}
		if boolFlag(opts, "long") {
			row["Routes"] = item.Routes
			row["External gateway info"] = item.GatewayInfo
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Status", "State", "Project", "Distributed", "HA"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Routes", "External gateway info", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func routerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router show requires <router>")
	}
	item, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"availability_zone_hints", item.AvailabilityZoneHints},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"distributed", item.Distributed},
		{"external_gateway_info", item.GatewayInfo},
		{"ha", false},
		{"id", item.ID},
		{"interfaces_info", routerInterfacesInfo(ctx, client, item.ID)},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"revision_number", item.RevisionNumber},
		{"routes", item.Routes},
		{"status", item.Status},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func routerInterfacesInfo(ctx context.Context, client *gophercloud.ServiceClient, routerID string) []map[string]string {
	page, err := ports.List(client, ports.ListOpts{DeviceID: routerID}).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return nil
	}
	var rows []map[string]string
	for _, port := range items {
		if port.DeviceOwner == "network:router_gateway" {
			continue
		}
		for _, fixedIP := range port.FixedIPs {
			rows = append(rows, map[string]string{
				"port_id":    port.ID,
				"ip_address": fixedIP.IPAddress,
				"subnet_id":  fixedIP.SubnetID,
			})
		}
	}
	return rows
}

func securityGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := secgroups.List(client, secgroups.ListOpts{
		ProjectID:  flagValue(opts, "project"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := secgroups.ExtractGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Project":     item.ProjectID,
			"Tags":        item.Tags,
			"Shared":      false,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Description", "Project", "Tags", "Shared"}, rows)
}

func securityGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group show requires <group>")
	}
	item, err := findSecurityGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"revision_number", item.RevisionNumber},
		{"rules", securityGroupRuleDetails(item.Rules)},
		{"stateful", item.Stateful},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func securityGroupRuleDetails(items []secgrouprules.SecGroupRule) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"id":                      item.ID,
			"description":             item.Description,
			"direction":               item.Direction,
			"ethertype":               item.EtherType,
			"protocol":                nilIfEmpty(item.Protocol),
			"port_range_min":          zeroNil(item.PortRangeMin),
			"port_range_max":          zeroNil(item.PortRangeMax),
			"remote_ip_prefix":        remoteIPPrefix(item),
			"remote_group_id":         nilIfEmpty(item.RemoteGroupID),
			"remote_address_group_id": nilIfEmpty(item.RemoteAddressGroupID),
			"security_group_id":       item.SecGroupID,
			"project_id":              item.ProjectID,
		})
	}
	return rows
}

func securityGroupRuleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	listOpts := secgrouprules.ListOpts{
		Protocol:  protocolFilter(opts),
		EtherType: ethertypeFilter(opts),
		ProjectID: flagValue(opts, "project"),
	}
	if boolFlag(opts, "ingress") {
		listOpts.Direction = "ingress"
	}
	if boolFlag(opts, "egress") {
		listOpts.Direction = "egress"
	}
	includeSecurityGroup := true
	if len(args) > 0 {
		group, err := findSecurityGroup(ctx, client, args[0])
		if err != nil {
			return err
		}
		listOpts.SecGroupID = group.ID
		includeSecurityGroup = false
	}
	page, err := secgrouprules.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := secgrouprules.ExtractRules(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                    item.ID,
			"IP Protocol":           nilIfEmpty(item.Protocol),
			"Ethertype":             item.EtherType,
			"IP Range":              remoteIPPrefix(item),
			"Port Range":            portRange(item),
			"Direction":             item.Direction,
			"Remote Security Group": nilIfEmpty(item.RemoteGroupID),
			"Remote Address Group":  nilIfEmpty(item.RemoteAddressGroupID),
		}
		if includeSecurityGroup {
			row["Security Group"] = item.SecGroupID
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "IP Protocol", "Ethertype", "IP Range", "Port Range", "Direction", "Remote Security Group", "Remote Address Group"}
	if includeSecurityGroup {
		columns = append(columns, "Security Group")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func securityGroupRuleShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group rule show requires <rule>")
	}
	item, err := secgrouprules.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"direction", item.Direction},
		{"ether_type", item.EtherType},
		{"id", item.ID},
		{"port_range_max", zeroNil(item.PortRangeMax)},
		{"port_range_min", zeroNil(item.PortRangeMin)},
		{"project_id", item.ProjectID},
		{"protocol", nilIfEmpty(item.Protocol)},
		{"remote_address_group_id", nilIfEmpty(item.RemoteAddressGroupID)},
		{"remote_group_id", nilIfEmpty(item.RemoteGroupID)},
		{"remote_ip_prefix", remoteIPPrefix(*item)},
		{"revision_number", item.RevisionNumber},
		{"security_group_id", item.SecGroupID},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func protocolFilter(opts *Options) string {
	value := strings.ToLower(flagValue(opts, "protocol"))
	if value == "any" {
		return ""
	}
	return value
}

func ethertypeFilter(opts *Options) string {
	value := flagValue(opts, "ethertype")
	switch strings.ToLower(value) {
	case "ipv4":
		return "IPv4"
	case "ipv6":
		return "IPv6"
	default:
		return value
	}
}

func remoteIPPrefix(rule secgrouprules.SecGroupRule) any {
	if rule.RemoteIPPrefix != "" {
		return rule.RemoteIPPrefix
	}
	if rule.EtherType == "IPv6" {
		return "::/0"
	}
	if rule.EtherType == "IPv4" {
		return "0.0.0.0/0"
	}
	return nil
}

func portRange(rule secgrouprules.SecGroupRule) string {
	if rule.PortRangeMin == 0 && rule.PortRangeMax == 0 {
		return ""
	}
	if rule.PortRangeMin == rule.PortRangeMax {
		return strconv.Itoa(rule.PortRangeMin)
	}
	return fmt.Sprintf("%d:%d", rule.PortRangeMin, rule.PortRangeMax)
}

func zeroNil(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func volumeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := volumes.List(client, volumes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := volumes.ExtractVolumes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Status":      item.Status,
			"Size":        item.Size,
			"Attached to": volumeAttachments(item.Attachments),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Size", "Attached to"}, rows)
}

func volumeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume show requires <volume>")
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"size", item.Size},
		{"availability_zone", item.AvailabilityZone},
		{"bootable", item.Bootable},
		{"encrypted", item.Encrypted},
		{"volume_type", item.VolumeType},
		{"attachments", volumeAttachments(item.Attachments)},
		{"metadata", item.Metadata},
		{"description", item.Description},
		{"created_at", item.CreatedAt},
	})
}

func volumeBackupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := backups.List(client, backups.ListOpts{
		AllTenants: boolFlag(opts, "all-projects") || flagValue(opts, "project") != "",
		Name:       flagValue(opts, "name"),
		Status:     flagValue(opts, "status"),
		TenantID:   flagValue(opts, "project"),
		VolumeID:   flagValue(opts, "volume"),
		Limit:      intFlag(opts, "limit"),
		Marker:     flagValue(opts, "marker"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := backups.ExtractBackups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Status":      item.Status,
			"Size":        item.Size,
			"Incremental": item.IsIncremental,
			"Created At":  oscTime(item.CreatedAt),
		}
		if boolFlag(opts, "long") {
			row["Availability Zone"] = stringPtrValue(item.AvailabilityZone)
			row["Volume"] = item.VolumeID
			row["Container"] = item.Container
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Description", "Status", "Size", "Incremental", "Created At"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Availability Zone", "Volume", "Container")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeBackupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup show requires <backup>")
	}
	item, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"availability_zone", stringPtrValue(item.AvailabilityZone)},
		{"container", item.Container},
		{"created_at", oscTime(item.CreatedAt)},
		{"data_timestamp", oscTime(item.DataTimestamp)},
		{"description", item.Description},
		{"encryption_key_id", nil},
		{"fail_reason", nilIfEmpty(item.FailReason)},
		{"has_dependent_backups", item.HasDependentBackups},
		{"id", item.ID},
		{"is_incremental", item.IsIncremental},
		{"metadata", mapPtrValue(item.Metadata)},
		{"name", item.Name},
		{"object_count", item.ObjectCount},
		{"project_id", nilIfEmpty(item.ProjectID)},
		{"size", item.Size},
		{"snapshot_id", nilIfEmpty(item.SnapshotID)},
		{"status", item.Status},
		{"updated_at", oscTime(item.UpdatedAt)},
		{"user_id", nil},
		{"volume_id", item.VolumeID},
	})
}

func volumeServiceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := bsservices.List(client, bsservices.ListOpts{
		Binary: flagValue(opts, "service"),
		Host:   flagValue(opts, "host"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := bsservices.ExtractServices(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"Binary":        item.Binary,
			"Host":          item.Host,
			"Zone":          item.Zone,
			"Status":        item.Status,
			"State":         item.State,
			"Updated At":    oscTime(item.UpdatedAt),
			"Cluster":       nilIfEmpty(item.Cluster),
			"Backend State": nil,
		}
		if boolFlag(opts, "long") {
			row["Disabled Reason"] = nilIfEmpty(item.DisabledReason)
		}
		rows = append(rows, row)
	}
	columns := []string{"Binary", "Host", "Zone", "Status", "State", "Updated At", "Cluster", "Backend State"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Disabled Reason")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeBackendPoolList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := schedulerstats.List(client, schedulerstats.ListOpts{Detail: boolFlag(opts, "long")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := schedulerstats.ExtractStoragePools(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"Name": item.Name}
		if boolFlag(opts, "long") {
			row["Capabilities"] = storagePoolCapabilities(item.Capabilities)
		}
		rows = append(rows, row)
	}
	columns := []string{"Name"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Capabilities")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeTransferList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := transfers.List(client, transfers.ListOpts{AllTenants: boolFlag(opts, "all-projects")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := transfers.ExtractTransfers(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Name":   item.Name,
			"Volume": item.VolumeID,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Volume"}, rows)
}

func volumeTransferShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume transfer request show requires <transfer-request>")
	}
	item, err := findVolumeTransfer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"auth_key", nilIfEmpty(item.AuthKey)},
		{"created_at", oscTime(item.CreatedAt)},
		{"id", item.ID},
		{"name", item.Name},
		{"volume_id", item.VolumeID},
	})
}

func stringPtrValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapPtrValue(value *map[string]string) any {
	if value == nil {
		return map[string]string{}
	}
	return *value
}

func storagePoolCapabilities(value schedulerstats.Capabilities) map[string]any {
	return map[string]any{
		"allocated_capacity_gb":       value.AllocatedCapacityGB,
		"driver_version":              value.DriverVersion,
		"filter_function":             value.FilterFunction,
		"free_capacity_gb":            value.FreeCapacityGB,
		"goodness_function":           value.GoodnessFunction,
		"location_info":               value.LocationInfo,
		"max_over_subscription_ratio": value.MaxOverSubscriptionRatio,
		"multiattach":                 value.Multiattach,
		"provisioned_capacity_gb":     value.ProvisionedCapacityGB,
		"qos_support":                 value.QoSSupport,
		"reserved_percentage":         value.ReservedPercentage,
		"sparse_copy_volume":          value.SparseCopyVolume,
		"storage_protocol":            value.StorageProtocol,
		"thick_provisioning_support":  value.ThickProvisioningSupport,
		"thin_provisioning_support":   value.ThinProvisioningSupport,
		"total_capacity_gb":           value.TotalCapacityGB,
		"total_volumes":               value.TotalVolumes,
		"vendor_name":                 value.VendorName,
		"volume_backend_name":         value.VolumeBackendName,
	}
}

func subnetList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	enableDHCP, hasDHCPFilter := subnetDHCPFilter(opts)
	listOpts := subnets.ListOpts{
		Name:         flagValue(opts, "name"),
		GatewayIP:    flagValue(opts, "gateway"),
		CIDR:         flagValue(opts, "subnet-range"),
		SubnetPoolID: flagValue(opts, "subnet-pool"),
		ProjectID:    flagValue(opts, "project"),
		Tags:         firstFlag(opts, "tags"),
		TagsAny:      firstFlag(opts, "any-tags", "tags-any"),
		NotTags:      firstFlag(opts, "not-tags"),
		NotTagsAny:   firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if value := intFlag(opts, "ip-version"); value != 0 {
		listOpts.IPVersion = value
	}
	if hasDHCPFilter {
		listOpts.EnableDHCP = &enableDHCP
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.NetworkID = resolveNetworkID(ctx, client, network)
	}

	page, err := subnets.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := subnets.ExtractSubnets(page)
	if err != nil {
		return err
	}
	serviceType := flagValue(opts, "service-type")
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		if serviceType != "" && !stringSliceContains(item.ServiceTypes, serviceType) {
			continue
		}
		row := outputRow{
			"ID":      item.ID,
			"Name":    item.Name,
			"Network": item.NetworkID,
			"Subnet":  item.CIDR,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["DHCP"] = item.EnableDHCP
			row["Gateway"] = item.GatewayIP
			row["IP Version"] = item.IPVersion
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Network", "Subnet"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "DHCP", "Gateway", "IP Version", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func subnetShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet show requires <subnet>")
	}
	item, err := findSubnet(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"project_id", item.ProjectID},
		{"cidr", item.CIDR},
		{"ip_version", item.IPVersion},
		{"gateway_ip", item.GatewayIP},
		{"enable_dhcp", item.EnableDHCP},
		{"dns_nameservers", item.DNSNameservers},
		{"allocation_pools", item.AllocationPools},
		{"host_routes", item.HostRoutes},
		{"service_types", item.ServiceTypes},
		{"subnetpool_id", item.SubnetPoolID},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func portList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) error {
	listOpts := ports.ListOpts{
		Name:           flagValue(opts, "name"),
		Status:         flagValue(opts, "status"),
		DeviceOwner:    flagValue(opts, "device-owner"),
		MACAddress:     flagValue(opts, "mac-address"),
		ProjectID:      flagValue(opts, "project"),
		SecurityGroups: compactStrings([]string{flagValue(opts, "security-group")}),
		Tags:           firstFlag(opts, "tags"),
		TagsAny:        firstFlag(opts, "any-tags", "tags-any"),
		NotTags:        firstFlag(opts, "not-tags"),
		NotTagsAny:     firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.NetworkID = resolveNetworkID(ctx, client, network)
	}
	if deviceID := firstFlag(opts, "device-id", "router"); deviceID != "" {
		listOpts.DeviceID = deviceID
	}
	if server := flagValue(opts, "server"); server != "" {
		listOpts.DeviceID = resolveServerID(ctx, computeClient, server)
	}
	if fixedIP := flagValue(opts, "fixed-ip"); fixedIP != "" {
		listOpts.FixedIPs = append(listOpts.FixedIPs, fixedIPFilter(ctx, client, fixedIP))
	}

	page, err := ports.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                 item.ID,
			"Name":               item.Name,
			"MAC Address":        item.MACAddress,
			"Fixed IP Addresses": portFixedIPs(item.FixedIPs),
			"Status":             item.Status,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["Network"] = item.NetworkID
			row["Device Owner"] = item.DeviceOwner
			row["Device ID"] = item.DeviceID
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "MAC Address", "Fixed IP Addresses", "Status"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "Network", "Device Owner", "Device ID", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func portShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port show requires <port>")
	}
	item, err := findPort(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"project_id", item.ProjectID},
		{"status", item.Status},
		{"admin_state_up", item.AdminStateUp},
		{"mac_address", item.MACAddress},
		{"fixed_ips", item.FixedIPs},
		{"device_id", item.DeviceID},
		{"device_owner", item.DeviceOwner},
		{"security_group_ids", item.SecurityGroups},
		{"allowed_address_pairs", item.AllowedAddressPairs},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func floatingIPList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := floatingips.ListOpts{
		FixedIP:    flagValue(opts, "fixed-ip-address"),
		FloatingIP: flagValue(opts, "floating-ip-address"),
		Status:     flagValue(opts, "status"),
		ProjectID:  flagValue(opts, "project"),
		RouterID:   flagValue(opts, "router"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.FloatingNetworkID = resolveNetworkID(ctx, client, network)
	}
	if port := flagValue(opts, "port"); port != "" {
		listOpts.PortID = resolvePortID(ctx, client, port)
	}
	page, err := floatingips.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                  item.ID,
			"Floating IP Address": item.FloatingIP,
			"Fixed IP Address":    item.FixedIP,
			"Port":                item.PortID,
			"Floating Network":    item.FloatingNetworkID,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["Router"] = item.RouterID
			row["Status"] = item.Status
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Floating IP Address", "Fixed IP Address", "Port", "Floating Network"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "Router", "Status", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func floatingIPShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip show requires <floating-ip>")
	}
	item, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"floating_ip_address", item.FloatingIP},
		{"fixed_ip_address", item.FixedIP},
		{"floating_network_id", item.FloatingNetworkID},
		{"port_id", item.PortID},
		{"router_id", item.RouterID},
		{"project_id", item.ProjectID},
		{"status", item.Status},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func keypairList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := keypairs.List(client, keypairs.ListOpts{UserID: flagValue(opts, "user")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := keypairs.ExtractKeyPairs(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Name":        item.Name,
			"Fingerprint": item.Fingerprint,
			"Type":        item.Type,
		})
	}
	return renderListOutput(stdout, opts, []string{"Name", "Fingerprint", "Type"}, rows)
}

func keypairShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("keypair show requires <key>")
	}
	item, err := keypairs.Get(ctx, client, args[0], keypairs.GetOpts{UserID: flagValue(opts, "user")}).Extract()
	if err != nil {
		return err
	}
	if boolFlag(opts, "public-key") {
		_, err := fmt.Fprintln(stdout, item.PublicKey)
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"name", item.Name},
		{"fingerprint", item.Fingerprint},
		{"public_key", item.PublicKey},
		{"user_id", item.UserID},
		{"type", item.Type},
	})
}

func serverGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := servergroups.ListOpts{
		AllProjects: boolFlag(opts, "all-projects"),
		Limit:       intFlag(opts, "limit"),
		Offset:      intFlag(opts, "offset"),
	}
	page, err := servergroups.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := servergroups.ExtractServerGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name, "Policies": item.Policies}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["User"] = item.UserID
			row["Members"] = item.Members
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Policies"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "User", "Members")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func serverGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server group show requires <server-group>")
	}
	item, err := findServerGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"policies", item.Policies},
		{"members", item.Members},
		{"project_id", item.ProjectID},
		{"user_id", item.UserID},
		{"metadata", item.Metadata},
		{"policy", item.Policy},
		{"rules", item.Rules},
	})
}

func volumeTypeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := volumetypes.ListOpts{Name: flagValue(opts, "name")}
	if boolFlag(opts, "public") {
		listOpts.IsPublic = volumetypes.VisibilityPublic
	}
	if boolFlag(opts, "private") {
		listOpts.IsPublic = volumetypes.VisibilityPrivate
	}
	page, err := volumetypes.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := volumetypes.ExtractVolumeTypes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		if !volumeTypeMatches(item, opts) {
			continue
		}
		row := outputRow{
			"ID":        item.ID,
			"Name":      item.Name,
			"Is Public": volumeTypeIsPublic(item),
		}
		if boolFlag(opts, "long") {
			row["Description"] = item.Description
			row["Properties"] = item.ExtraSpecs
			row["Qos Spec ID"] = item.QosSpecID
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Is Public"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Description", "Properties", "Qos Spec ID")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeTypeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type show requires <volume-type>")
	}
	item, err := findVolumeType(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"is_public", volumeTypeIsPublic(*item)},
		{"properties", item.ExtraSpecs},
		{"qos_specs_id", item.QosSpecID},
	})
}

func volumeSnapshotList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := snapshots.ListOpts{
		AllTenants: boolFlag(opts, "all-projects"),
		Name:       flagValue(opts, "name"),
		Status:     flagValue(opts, "status"),
		TenantID:   flagValue(opts, "project"),
		Limit:      intFlag(opts, "limit"),
		Marker:     flagValue(opts, "marker"),
	}
	if volume := flagValue(opts, "volume"); volume != "" {
		listOpts.VolumeID = resolveVolumeID(ctx, client, volume)
	}
	page, err := snapshots.ListDetail(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := snapshots.ExtractSnapshots(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Status":      item.Status,
			"Size":        item.Size,
		}
		if boolFlag(opts, "long") {
			row["Volume"] = item.VolumeID
			row["Created At"] = item.CreatedAt
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Description", "Status", "Size"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Volume", "Created At")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeSnapshotShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot show requires <snapshot>")
	}
	item, err := findVolumeSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"status", item.Status},
		{"size", item.Size},
		{"volume_id", item.VolumeID},
		{"metadata", item.Metadata},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func findServer(ctx context.Context, client *gophercloud.ServiceClient, value string) (*servers.Server, error) {
	result := servers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := servers.List(client, servers.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := extractServers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item servers.Server) string { return item.Name })
}

func findAggregate(ctx context.Context, client *gophercloud.ServiceClient, value string) (*aggregates.Aggregate, error) {
	if id, err := strconv.Atoi(value); err == nil {
		result := aggregates.Get(ctx, client, id)
		if result.Err == nil {
			return result.Extract()
		}
	}
	page, err := aggregates.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := aggregates.ExtractAggregates(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item aggregates.Aggregate) string { return item.Name })
}

func findHypervisor(ctx context.Context, client *gophercloud.ServiceClient, value string) (*hypervisors.Hypervisor, error) {
	result := hypervisors.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := hypervisors.List(client, hypervisors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := hypervisors.ExtractHypervisors(page)
	if err != nil {
		return nil, err
	}
	var matches []hypervisors.Hypervisor
	for _, item := range items {
		if item.HypervisorHostname == value || item.ID == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func findFlavor(ctx context.Context, client *gophercloud.ServiceClient, value string) (*flavors.Flavor, error) {
	result := flavors.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := flavors.ListDetail(client, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item flavors.Flavor) string { return item.Name })
}

func findImage(ctx context.Context, client *gophercloud.ServiceClient, value string) (*images.Image, error) {
	result := images.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := images.List(client, images.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item images.Image) string { return item.Name })
}

func findNetwork(ctx context.Context, client *gophercloud.ServiceClient, value string) (*networks.Network, error) {
	result := networks.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := networks.List(client, networks.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := networks.ExtractNetworks(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item networks.Network) string { return item.Name })
}

func findRouter(ctx context.Context, client *gophercloud.ServiceClient, value string) (*routers.Router, error) {
	result := routers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := routers.List(client, routers.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := routers.ExtractRouters(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item routers.Router) string { return item.Name })
}

func findSecurityGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*secgroups.SecGroup, error) {
	result := secgroups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := secgroups.List(client, secgroups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := secgroups.ExtractGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item secgroups.SecGroup) string { return item.Name })
}

func findVolume(ctx context.Context, client *gophercloud.ServiceClient, value string) (*volumes.Volume, error) {
	result := volumes.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := volumes.List(client, volumes.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := volumes.ExtractVolumes(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item volumes.Volume) string { return item.Name })
}

func findVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*backups.Backup, error) {
	result := backups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := backups.List(client, backups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := backups.ExtractBackups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item backups.Backup) string { return item.Name })
}

func findVolumeTransfer(ctx context.Context, client *gophercloud.ServiceClient, value string) (*transfers.Transfer, error) {
	result := transfers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := transfers.List(client, transfers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := transfers.ExtractTransfers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item transfers.Transfer) string { return item.Name })
}

func findSubnet(ctx context.Context, client *gophercloud.ServiceClient, value string) (*subnets.Subnet, error) {
	result := subnets.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := subnets.List(client, subnets.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := subnets.ExtractSubnets(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item subnets.Subnet) string { return item.Name })
}

func findPort(ctx context.Context, client *gophercloud.ServiceClient, value string) (*ports.Port, error) {
	result := ports.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := ports.List(client, ports.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item ports.Port) string { return item.Name })
}

func findFloatingIP(ctx context.Context, client *gophercloud.ServiceClient, value string) (*floatingips.FloatingIP, error) {
	result := floatingips.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := floatingips.List(client, floatingips.ListOpts{FloatingIP: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return nil, err
	}
	return singleMatch(value, items)
}

func findServerGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*servergroups.ServerGroup, error) {
	result := servergroups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := servergroups.List(client, servergroups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := servergroups.ExtractServerGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item servergroups.ServerGroup) string { return item.Name })
}

func findVolumeType(ctx context.Context, client *gophercloud.ServiceClient, value string) (*volumetypes.VolumeType, error) {
	result := volumetypes.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := volumetypes.List(client, volumetypes.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := volumetypes.ExtractVolumeTypes(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item volumetypes.VolumeType) string { return item.Name })
}

func findVolumeSnapshot(ctx context.Context, client *gophercloud.ServiceClient, value string) (*snapshots.Snapshot, error) {
	result := snapshots.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := snapshots.ListDetail(client, snapshots.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := snapshots.ExtractSnapshots(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item snapshots.Snapshot) string { return item.Name })
}

func extractServers(page pagination.Page) ([]servers.Server, error) {
	var items []servers.Server
	err := servers.ExtractServersInto(page, &items)
	return items, err
}

func serverImage(image map[string]any, imageNames map[string]string) any {
	if len(image) == 0 {
		return "N/A (booted from volume)"
	}
	if id, ok := image["id"].(string); ok && id != "" {
		if name := imageNames[id]; name != "" {
			return name
		}
	}
	if name, ok := image["name"].(string); ok && name != "" {
		return name
	}
	if id, ok := image["id"].(string); ok && id != "" {
		return id
	}
	return image
}

func serverFlavor(flavor map[string]any, flavorNames map[string]string) any {
	if len(flavor) == 0 {
		return ""
	}
	if id, ok := flavor["id"].(string); ok && id != "" {
		if name := flavorNames[id]; name != "" {
			return name
		}
	}
	if original, ok := flavor["original_name"].(string); ok && original != "" {
		return original
	}
	if name, ok := flavor["name"].(string); ok && name != "" {
		return name
	}
	if id, ok := flavor["id"].(string); ok && id != "" {
		return id
	}
	return flavor
}

func flavorNameMap(ctx context.Context, client *gophercloud.ServiceClient) map[string]string {
	if client == nil {
		return nil
	}
	page, err := flavors.ListDetail(client, nil).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(items))
	for _, item := range items {
		names[item.ID] = item.Name
	}
	return names
}

func imageNameMap(ctx context.Context, client *gophercloud.ServiceClient) map[string]string {
	if client == nil {
		return nil
	}
	page, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(items))
	for _, item := range items {
		names[item.ID] = item.Name
	}
	return names
}

func serverNetworks(addresses map[string]any) map[string][]string {
	networks := make(map[string][]string, len(addresses))
	for name, rawAddresses := range addresses {
		for _, rawAddress := range anySlice(rawAddresses) {
			if address, ok := rawAddress.(map[string]any); ok {
				if addr, ok := address["addr"].(string); ok && addr != "" {
					networks[name] = append(networks[name], addr)
				}
				continue
			}
			if addr, ok := rawAddress.(string); ok && addr != "" {
				networks[name] = append(networks[name], addr)
			}
		}
	}
	for _, values := range networks {
		sort.Strings(values)
	}
	return networks
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		values := make([]any, len(typed))
		for i, item := range typed {
			values[i] = item
		}
		return values
	default:
		return nil
	}
}

func volumeAttachments(attachments []volumes.Attachment) []map[string]any {
	values := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		values = append(values, map[string]any{
			"id":            item.ID,
			"attachment_id": item.AttachmentID,
			"volume_id":     item.VolumeID,
			"server_id":     item.ServerID,
			"host_name":     item.HostName,
			"device":        item.Device,
			"attached_at":   item.AttachedAt.Format("2006-01-02T15:04:05.000000"),
		})
	}
	return values
}

func subnetDHCPFilter(opts *Options) (bool, bool) {
	if boolFlag(opts, "dhcp") {
		return true, true
	}
	if boolFlag(opts, "no-dhcp") {
		return false, true
	}
	return false, false
}

func firstFlag(opts *Options, names ...string) string {
	for _, name := range names {
		if value := flagValue(opts, name); value != "" {
			return value
		}
	}
	return ""
}

func intFlag(opts *Options, name string) int {
	value := flagValue(opts, name)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func resolveNetworkID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findNetwork(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolvePortID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findPort(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveSubnetID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findSubnet(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveServerID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findServer(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveVolumeID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findVolume(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func fixedIPFilter(ctx context.Context, client *gophercloud.ServiceClient, value string) ports.FixedIPOpts {
	filter := ports.FixedIPOpts{}
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		raw = strings.TrimSpace(raw)
		switch key {
		case "subnet":
			filter.SubnetID = resolveSubnetID(ctx, client, raw)
		case "ip-address", "ip_address":
			filter.IPAddress = raw
		case "ip-substring", "ip_address_substr", "ip-substr":
			filter.IPAddressSubstr = raw
		}
	}
	return filter
}

func portFixedIPs(fixedIPs []ports.IP) []string {
	values := make([]string, 0, len(fixedIPs))
	for _, item := range fixedIPs {
		switch {
		case item.IPAddress != "" && item.SubnetID != "":
			values = append(values, fmt.Sprintf("%s, subnet_id=%s", item.IPAddress, item.SubnetID))
		case item.IPAddress != "":
			values = append(values, item.IPAddress)
		case item.SubnetID != "":
			values = append(values, fmt.Sprintf("subnet_id=%s", item.SubnetID))
		}
	}
	sort.Strings(values)
	return values
}

func volumeTypeMatches(item volumetypes.VolumeType, opts *Options) bool {
	property := flagValue(opts, "property")
	if property != "" {
		key, value, ok := strings.Cut(property, "=")
		if !ok || item.ExtraSpecs[strings.TrimSpace(key)] != strings.TrimSpace(value) {
			return false
		}
	}
	if boolFlag(opts, "multiattach") && !extraSpecTruthy(item.ExtraSpecs, "multiattach") {
		return false
	}
	if boolFlag(opts, "cacheable") && !extraSpecTruthy(item.ExtraSpecs, "cacheable") {
		return false
	}
	if boolFlag(opts, "replicated") && !extraSpecTruthy(item.ExtraSpecs, "replication_enabled") {
		return false
	}
	availabilityZone := flagValue(opts, "availability-zone")
	if availabilityZone != "" && !volumeTypeAvailabilityZoneMatches(item.ExtraSpecs, availabilityZone) {
		return false
	}
	return true
}

func volumeTypeIsPublic(item volumetypes.VolumeType) bool {
	return item.IsPublic || item.PublicAccess
}

func extraSpecTruthy(specs map[string]string, key string) bool {
	value := strings.TrimSpace(strings.ToLower(specs[key]))
	return value == "true" || value == "<is> true" || value == "yes" || value == "1"
}

func volumeTypeAvailabilityZoneMatches(specs map[string]string, availabilityZone string) bool {
	availabilityZone = strings.TrimSpace(availabilityZone)
	if availabilityZone == "" {
		return true
	}
	for key, value := range specs {
		if key == "RESKEY:availability_zones:"+availabilityZone || value == availabilityZone {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func compactStrings(values []string) []string {
	var compacted []string
	for _, value := range values {
		if value != "" {
			compacted = append(compacted, value)
		}
	}
	return compacted
}
