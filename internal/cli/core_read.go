package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
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
		case "image list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageList(cmd.Context(), stdout, opts, client)
		case "image show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageShow(cmd.Context(), stdout, opts, client, args)
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
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
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
