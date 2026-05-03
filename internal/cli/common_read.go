package cli

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/gophercloud/gophercloud/v2"
	volumeaz "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones"
	volumelimits "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
	"github.com/gophercloud/gophercloud/v2/openstack/common/extensions"
	computeaz "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones"
	computelimits "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits"
)

type networkAvailabilityZone struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Resource string `json:"resource"`
}

func availabilityZoneList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	includeCompute, includeNetwork, includeVolume := selectedAvailabilityZoneServices(opts)
	long := boolFlag(opts, "long")
	var rows []outputRow

	if includeCompute {
		client, err := clients.computeV2()
		if err != nil {
			return err
		}
		computeRows, err := computeAvailabilityZoneRows(ctx, client, long)
		if err != nil {
			return err
		}
		rows = append(rows, computeRows...)
	}

	if includeVolume {
		client, err := clients.blockStorageV3()
		if err != nil {
			return err
		}
		volumeRows, err := volumeAvailabilityZoneRows(ctx, client, long)
		if err != nil {
			return err
		}
		rows = append(rows, volumeRows...)
	}

	if includeNetwork {
		client, err := clients.networkV2()
		if err != nil {
			return err
		}
		networkRows, err := networkAvailabilityZoneRows(ctx, client, long)
		if err != nil {
			return err
		}
		rows = append(rows, networkRows...)
	}

	columns := []string{"Zone Name", "Zone Status"}
	if long {
		columns = append(columns, "Zone Resource", "Host Name", "Service Name", "Service Status")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func extensionList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	includeCompute, includeIdentity, includeNetwork, includeVolume := selectedExtensionServices(opts)
	if includeIdentity {
		return fmt.Errorf("Extensions list not supported by Identity API")
	}

	long := boolFlag(opts, "long")
	var rows []outputRow
	for _, source := range []struct {
		enabled bool
		client  func() (*gophercloud.ServiceClient, error)
	}{
		{enabled: includeCompute, client: clients.computeV2},
		{enabled: includeVolume, client: clients.blockStorageV3},
		{enabled: includeNetwork, client: clients.networkV2},
	} {
		if !source.enabled {
			continue
		}
		client, err := source.client()
		if err != nil {
			return err
		}
		serviceRows, err := extensionRows(ctx, client, long)
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}

	columns := []string{"Name", "Alias", "Description"}
	if long {
		columns = append(columns, "Namespace", "Updated At", "Links")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func extensionShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("extension show requires <extension>")
	}
	client, err := clients.networkV2()
	if err != nil {
		return err
	}
	item, err := findNetworkExtension(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"alias", item.Alias},
		{"description", item.Description},
		{"name", item.Name},
		{"updated_at", item.Updated},
	})
}

func limitsShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	if boolFlag(opts, "absolute") == boolFlag(opts, "rate") {
		return fmt.Errorf("one of the arguments --absolute --rate is required")
	}
	if boolFlag(opts, "rate") {
		return limitRateShow(ctx, stdout, opts, clients)
	}
	return limitAbsoluteShow(ctx, stdout, opts, clients)
}

func selectedAvailabilityZoneServices(opts *Options) (bool, bool, bool) {
	compute := boolFlag(opts, "compute")
	network := boolFlag(opts, "network")
	volume := boolFlag(opts, "volume")
	if !compute && !network && !volume {
		return true, true, true
	}
	return compute, network, volume
}

func selectedExtensionServices(opts *Options) (bool, bool, bool, bool) {
	compute := boolFlag(opts, "compute")
	identity := boolFlag(opts, "identity")
	network := boolFlag(opts, "network")
	volume := boolFlag(opts, "volume")
	if !compute && !identity && !network && !volume {
		return true, false, true, true
	}
	return compute, identity, network, volume
}

func computeAvailabilityZoneRows(ctx context.Context, client *gophercloud.ServiceClient, long bool) ([]outputRow, error) {
	page, err := computeaz.ListDetail(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := computeaz.ExtractAvailabilityZones(page)
	if err != nil {
		return nil, err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		if !long {
			rows = append(rows, availabilityZoneRow(item.ZoneName, availabilityZoneStatus(item.ZoneState.Available), "", "", "", "", false))
			continue
		}
		if len(item.Hosts) == 0 {
			rows = append(rows, availabilityZoneRow(item.ZoneName, availabilityZoneStatus(item.ZoneState.Available), "", "", "", "", true))
			continue
		}
		hostNames := make([]string, 0, len(item.Hosts))
		for hostName := range item.Hosts {
			hostNames = append(hostNames, hostName)
		}
		sort.Strings(hostNames)
		for _, hostName := range hostNames {
			serviceNames := make([]string, 0, len(item.Hosts[hostName]))
			for serviceName := range item.Hosts[hostName] {
				serviceNames = append(serviceNames, serviceName)
			}
			sort.Strings(serviceNames)
			for _, serviceName := range serviceNames {
				state := item.Hosts[hostName][serviceName]
				rows = append(rows, availabilityZoneRow(
					item.ZoneName,
					availabilityZoneStatus(item.ZoneState.Available),
					"",
					hostName,
					serviceName,
					computeServiceStatus(state),
					true,
				))
			}
		}
	}
	return rows, nil
}

func volumeAvailabilityZoneRows(ctx context.Context, client *gophercloud.ServiceClient, long bool) ([]outputRow, error) {
	page, err := volumeaz.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := volumeaz.ExtractAvailabilityZones(page)
	if err != nil {
		return nil, err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, availabilityZoneRow(item.ZoneName, availabilityZoneStatus(item.ZoneState.Available), "", "", "", "", long))
	}
	return rows, nil
}

func networkAvailabilityZoneRows(ctx context.Context, client *gophercloud.ServiceClient, long bool) ([]outputRow, error) {
	var response struct {
		AvailabilityZones []networkAvailabilityZone `json:"availability_zones"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("availability_zones"), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	rows := make([]outputRow, 0, len(response.AvailabilityZones))
	for _, item := range response.AvailabilityZones {
		rows = append(rows, availabilityZoneRow(item.Name, item.State, item.Resource, "", "", "", long))
	}
	return rows, nil
}

func availabilityZoneRow(name string, status string, resource string, host string, service string, serviceStatus string, long bool) outputRow {
	row := outputRow{
		"Zone Name":   name,
		"Zone Status": status,
	}
	if long {
		row["Zone Resource"] = resource
		row["Host Name"] = host
		row["Service Name"] = service
		row["Service Status"] = serviceStatus
	}
	return row
}

func availabilityZoneStatus(available bool) string {
	if available {
		return "available"
	}
	return "not available"
}

func computeServiceStatus(state computeaz.ServiceState) string {
	status := "disabled"
	if state.Active {
		status = "enabled"
	}
	health := ":-)"
	if !state.Available {
		health = "XXX"
	}
	if state.UpdatedAt.IsZero() {
		return status + " " + health
	}
	return status + " " + health + " " + state.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000")
}

func extensionRows(ctx context.Context, client *gophercloud.ServiceClient, long bool) ([]outputRow, error) {
	page, err := extensions.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := extensions.ExtractExtensions(page)
	if err != nil {
		return nil, err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"Name":        item.Name,
			"Alias":       item.Alias,
			"Description": item.Description,
		}
		if long {
			row["Namespace"] = item.Namespace
			row["Updated At"] = item.Updated
			row["Links"] = item.Links
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func findNetworkExtension(ctx context.Context, client *gophercloud.ServiceClient, value string) (*extensions.Extension, error) {
	result := extensions.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := extensions.List(client).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := extensions.ExtractExtensions(page)
	if err != nil {
		return nil, err
	}
	var matches []extensions.Extension
	for _, item := range items {
		if item.Alias == value || item.Name == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func limitAbsoluteShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	projectID, err := limitsProjectID(ctx, opts, clients)
	if err != nil {
		return err
	}
	var rows []outputRow

	computeClient, err := clients.computeV2()
	if err != nil {
		return err
	}
	computeOpts := computelimits.GetOpts{TenantID: projectID}
	computeLimits, err := computelimits.Get(ctx, computeClient, computeOpts).Extract()
	if err != nil {
		return err
	}
	rows = append(rows, computeLimitRows(computeLimits.Absolute)...)

	volumeClient, err := clients.blockStorageV3()
	if err != nil {
		return err
	}
	volumeLimits, err := volumelimits.Get(ctx, volumeClient).Extract()
	if err != nil {
		return err
	}
	rows = append(rows, reflectedLimitRows(volumeLimits.Absolute)...)

	sortLimitRows(rows)
	return renderListOutput(stdout, opts, []string{"Name", "Value"}, rows)
}

func limitRateShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	client, err := clients.blockStorageV3()
	if err != nil {
		return err
	}
	limits, err := volumelimits.Get(ctx, client).Extract()
	if err != nil {
		return err
	}
	var rows []outputRow
	for _, rate := range limits.Rate {
		for _, limit := range rate.Limit {
			rows = append(rows, outputRow{
				"Verb":           limit.Verb,
				"URI":            rate.URI,
				"Regex":          rate.Regex,
				"Value":          limit.Value,
				"Remaining":      limit.Remaining,
				"Unit":           limit.Unit,
				"Next Available": limit.NextAvailable,
			})
		}
	}
	return renderListOutput(stdout, opts, []string{"Verb", "URI", "Regex", "Value", "Remaining", "Unit", "Next Available"}, rows)
}

func limitsProjectID(ctx context.Context, opts *Options, clients *openStackClients) (string, error) {
	project := flagValue(opts, "project")
	if project == "" {
		return "", nil
	}
	identityClient, err := clients.identityV3()
	if err != nil {
		return "", err
	}
	item, err := findProject(ctx, identityClient, project)
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

func computeLimitRows(absolute computelimits.Absolute) []outputRow {
	values := map[string]any{}
	addComputeLimit(values, "MaxTotalCores", absolute.MaxTotalCores)
	addComputeLimit(values, "MaxImageMeta", absolute.MaxImageMeta)
	addComputeLimit(values, "MaxServerMeta", absolute.MaxServerMeta)
	addComputeLimit(values, "MaxPersonality", absolute.MaxPersonality)
	addComputeLimit(values, "MaxPersonalitySize", absolute.MaxPersonalitySize)
	addComputeLimit(values, "MaxTotalKeypairs", absolute.MaxTotalKeypairs)
	addComputeLimit(values, "MaxSecurityGroups", absolute.MaxSecurityGroups)
	addComputeLimit(values, "MaxSecurityGroupRules", absolute.MaxSecurityGroupRules)
	addComputeLimit(values, "MaxServerGroups", absolute.MaxServerGroups)
	addComputeLimit(values, "MaxServerGroupMembers", absolute.MaxServerGroupMembers)
	addComputeLimit(values, "MaxTotalFloatingIps", absolute.MaxTotalFloatingIps)
	addComputeLimit(values, "MaxTotalInstances", absolute.MaxTotalInstances)
	addComputeLimit(values, "MaxTotalRAMSize", absolute.MaxTotalRAMSize)
	addComputeLimit(values, "TotalCoresUsed", absolute.TotalCoresUsed)
	addComputeLimit(values, "TotalInstancesUsed", absolute.TotalInstancesUsed)
	addComputeLimit(values, "TotalFloatingIpsUsed", absolute.TotalFloatingIpsUsed)
	addComputeLimit(values, "TotalRAMUsed", absolute.TotalRAMUsed)
	addComputeLimit(values, "TotalSecurityGroupsUsed", absolute.TotalSecurityGroupsUsed)
	addComputeLimit(values, "TotalServerGroupsUsed", absolute.TotalServerGroupsUsed)
	return mapLimitRows(values)
}

func addComputeLimit(values map[string]any, field string, value any) {
	names := []string{camelToSnake(field)}
	switch field {
	case "MaxTotalCores":
		names = append(names, "total_cores")
	case "MaxImageMeta":
		names = append(names, "image_meta")
	case "MaxServerMeta":
		names = append(names, "server_meta")
	case "MaxPersonality":
		names = []string{"personality"}
	case "MaxPersonalitySize":
		names = []string{"personality_size"}
	case "MaxTotalKeypairs":
		names = append(names, "keypairs")
	case "MaxSecurityGroups":
		names = append(names, "security_groups")
	case "MaxSecurityGroupRules":
		names = append(names, "security_group_rules")
	case "MaxServerGroups":
		names = append(names, "server_groups")
	case "MaxServerGroupMembers":
		names = append(names, "server_group_members")
	case "MaxTotalFloatingIps":
		names = append(names, "floating_ips")
	case "MaxTotalInstances":
		names = append(names, "instances")
	case "MaxTotalRAMSize":
		names = append(names, "total_ram")
	case "TotalInstancesUsed":
		names = append(names, "instances_used")
	case "TotalFloatingIpsUsed":
		names = append(names, "floating_ips_used")
	case "TotalSecurityGroupsUsed":
		names = append(names, "security_groups_used")
	case "TotalServerGroupsUsed":
		names = append(names, "server_groups_used")
	}
	for _, name := range names {
		values[name] = value
	}
}

func reflectedLimitRows(value any) []outputRow {
	values := map[string]any{}
	rv := reflect.Indirect(reflect.ValueOf(value))
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := jsonTagName(field)
		if name == "" || name == "-" {
			name = camelToSnake(field.Name)
		} else {
			name = camelToSnake(name)
		}
		values[name] = rv.Field(i).Interface()
	}
	return mapLimitRows(values)
}

func mapLimitRows(values map[string]any) []outputRow {
	rows := make([]outputRow, 0, len(values))
	for name, value := range values {
		rows = append(rows, outputRow{"Name": name, "Value": value})
	}
	return rows
}

func sortLimitRows(rows []outputRow) {
	sort.Slice(rows, func(i, j int) bool {
		return valueString(rows[i]["Name"]) < valueString(rows[j]["Name"])
	})
}

func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

func camelToSnake(value string) string {
	var b strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			prev := rune(0)
			next := rune(0)
			if i > 0 {
				prev = runes[i-1]
			}
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if i > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && unicode.IsLower(next))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
