package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
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

func quotaList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	compute, volume, network := selectedQuotaListService(opts)
	if !compute && !volume && !network {
		return fmt.Errorf("one of the arguments --compute --volume --network is required")
	}

	var rows []outputRow
	if compute {
		client, err := clients.computeV2()
		if err != nil {
			return err
		}
		serviceRows, err := quotaListRows(ctx, client, "os-quota-sets", "compute")
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}
	if volume {
		client, err := clients.blockStorageV3()
		if err != nil {
			return err
		}
		serviceRows, err := quotaListRows(ctx, client, "os-quota-sets", "volume")
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}
	if network {
		client, err := clients.networkV2()
		if err != nil {
			return err
		}
		serviceRows, err := quotaListRows(ctx, client, "quotas", "network")
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}
	return renderListOutput(stdout, opts, []string{"Project ID", "Service", "Resource", "Limit"}, rows)
}

func quotaShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, args []string) error {
	projectID, err := quotaProjectID(ctx, opts, clients, args)
	if err != nil {
		return err
	}
	compute, volume, network := selectedQuotaShowServices(opts)
	usage := boolFlag(opts, "usage")
	if boolFlag(opts, "default") && usage {
		return fmt.Errorf("argument --default: not allowed with argument --usage")
	}

	var rows []outputRow
	if compute {
		client, err := clients.computeV2()
		if err != nil {
			return err
		}
		serviceRows, err := computeQuotaRows(ctx, client, projectID, usage)
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}
	if volume {
		client, err := clients.blockStorageV3()
		if err != nil {
			return err
		}
		serviceRows, err := volumeQuotaRows(ctx, client, projectID, usage, boolFlag(opts, "default"))
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}
	if network {
		client, err := clients.networkV2()
		if err != nil {
			return err
		}
		serviceRows, err := networkQuotaRows(ctx, client, projectID, usage)
		if err != nil {
			return err
		}
		rows = append(rows, serviceRows...)
	}

	columns := []string{"Resource", "Limit"}
	if usage {
		columns = append(columns, "In Use", "Reserved")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func versionsShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients) error {
	catalog, err := currentServiceCatalog(clients)
	if err != nil {
		return err
	}
	var rows []outputRow
	seen := map[string]bool{}
	for _, entry := range catalog.Entries {
		if flagValue(opts, "service") == "" && entry.Type == "volumev3" && catalogHasServiceType(catalog, "block-storage") {
			continue
		}
		if !versionServiceSelected(opts, entry) {
			continue
		}
		for _, endpoint := range entry.Endpoints {
			if !versionEndpointSelected(opts, endpoint) {
				continue
			}
			key := entry.Type + "|" + endpoint.Region + "|" + endpoint.Interface + "|" + versionRootURL(endpoint.URL)
			if seen[key] && !boolFlag(opts, "all-interfaces") {
				continue
			}
			seen[key] = true
			serviceRows, err := versionRowsForEndpoint(ctx, clients.Provider, entry.Type, endpoint)
			if err != nil || len(serviceRows) == 0 {
				serviceRows = []outputRow{versionFallbackRow(entry.Type, endpoint)}
			}
			rows = append(rows, serviceRows...)
		}
	}
	status := strings.ToUpper(flagValue(opts, "status"))
	if status != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if strings.ToUpper(valueString(row["Status"])) == status {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	return renderListOutput(stdout, opts, []string{"Region Name", "Service Type", "Version", "Status", "Endpoint", "Min Microversion", "Max Microversion"}, rows)
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

func selectedQuotaListService(opts *Options) (bool, bool, bool) {
	return boolFlag(opts, "compute"), boolFlag(opts, "volume"), boolFlag(opts, "network")
}

func selectedQuotaShowServices(opts *Options) (bool, bool, bool) {
	compute := boolFlag(opts, "compute")
	volume := boolFlag(opts, "volume")
	network := boolFlag(opts, "network")
	if boolFlag(opts, "all") || (!compute && !volume && !network) {
		return true, true, true
	}
	return compute, volume, network
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

func quotaProjectID(ctx context.Context, opts *Options, clients *openStackClients, args []string) (string, error) {
	if len(args) > 0 {
		identityClient, err := clients.identityV3()
		if err != nil {
			return "", err
		}
		project, err := findProject(ctx, identityClient, args[0])
		if err != nil {
			return "", err
		}
		return project.ID, nil
	}
	if projectID := currentProjectID(clients); projectID != "" {
		return projectID, nil
	}
	if projectID := os.Getenv("OS_PROJECT_ID"); projectID != "" {
		return projectID, nil
	}
	if projectID := os.Getenv("OS_TENANT_ID"); projectID != "" {
		return projectID, nil
	}
	return "", fmt.Errorf("quota show requires a project when the current token is not project-scoped")
}

func currentProjectID(clients *openStackClients) string {
	if clients == nil || clients.Provider == nil {
		return ""
	}
	extractor, ok := clients.Provider.GetAuthResult().(interface {
		ExtractProject() (*tokens.Project, error)
	})
	if !ok {
		return ""
	}
	project, err := extractor.ExtractProject()
	if err != nil || project == nil {
		return ""
	}
	return project.ID
}

func currentServiceCatalog(clients *openStackClients) (*tokens.ServiceCatalog, error) {
	if clients == nil || clients.Provider == nil {
		return nil, fmt.Errorf("service catalog is not available")
	}
	extractor, ok := clients.Provider.GetAuthResult().(interface {
		ExtractServiceCatalog() (*tokens.ServiceCatalog, error)
	})
	if !ok {
		return nil, fmt.Errorf("service catalog is not available")
	}
	return extractor.ExtractServiceCatalog()
}

func versionServiceSelected(opts *Options, entry tokens.CatalogEntry) bool {
	service := flagValue(opts, "service")
	if service == "" {
		return true
	}
	return entry.Type == service || entry.Name == service
}

func versionEndpointSelected(opts *Options, endpoint tokens.Endpoint) bool {
	region := flagValue(opts, "region-name")
	if region != "" && endpoint.Region != region && endpoint.RegionID != region {
		return false
	}
	if boolFlag(opts, "all-interfaces") {
		return true
	}
	if requested := flagValue(opts, "interface"); requested != "" {
		return endpoint.Interface == requested
	}
	availability := string(availabilityFromInterface(firstNonEmpty(opts.Interface, os.Getenv("OS_INTERFACE"))))
	if availability == "" {
		availability = "public"
	}
	return endpoint.Interface == availability
}

func versionRowsForEndpoint(ctx context.Context, provider *gophercloud.ProviderClient, serviceType string, endpoint tokens.Endpoint) ([]outputRow, error) {
	var response map[string]any
	opts := &gophercloud.RequestOpts{
		JSONResponse: &response,
		OkCodes:      []int{http.StatusOK, http.StatusMultipleChoices},
	}
	root := versionRootURL(endpoint.URL)
	_, err := provider.Request(ctx, http.MethodGet, root, opts)
	if err != nil {
		return nil, err
	}
	versions := extractVersionMaps(response)
	sort.SliceStable(versions, func(i, j int) bool {
		return versionLess(versionIDString(versions[i]), versionIDString(versions[j]))
	})
	rows := make([]outputRow, 0, len(versions))
	for _, version := range versions {
		rows = append(rows, outputRow{
			"Region Name":      firstNonEmpty(endpoint.Region, endpoint.RegionID),
			"Service Type":     serviceType,
			"Version":          versionID(version),
			"Status":           versionStatus(version),
			"Endpoint":         versionEndpoint(version, endpoint.URL),
			"Min Microversion": versionValue(version, "min_version", "min_version_id", "min_microversion"),
			"Max Microversion": versionValue(version, "version", "max_version", "max_version_id", "max_microversion"),
		})
	}
	return rows, nil
}

func versionFallbackRow(serviceType string, endpoint tokens.Endpoint) outputRow {
	return outputRow{
		"Region Name":      firstNonEmpty(endpoint.Region, endpoint.RegionID),
		"Service Type":     serviceType,
		"Version":          versionFromEndpoint(endpoint.URL),
		"Status":           "CURRENT",
		"Endpoint":         versionedEndpointURL(endpoint.URL),
		"Min Microversion": nil,
		"Max Microversion": nil,
	}
}

func catalogHasServiceType(catalog *tokens.ServiceCatalog, serviceType string) bool {
	if catalog == nil {
		return false
	}
	for _, entry := range catalog.Entries {
		if entry.Type == serviceType {
			return true
		}
	}
	return false
}

func extractVersionMaps(response map[string]any) []map[string]any {
	var values []any
	if raw, ok := response["versions"].([]any); ok {
		values = raw
	} else if versions, ok := response["versions"].(map[string]any); ok {
		if raw, ok := versions["values"].([]any); ok {
			values = raw
		}
	} else if version, ok := response["version"].(map[string]any); ok {
		return []map[string]any{version}
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			result = append(result, item)
		}
	}
	return result
}

func versionRootURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	parts := strings.Split(trimmed, "/")
	for len(parts) > 0 {
		last := parts[len(parts)-1]
		lower := strings.ToLower(last)
		if strings.HasPrefix(lower, "v") || strings.Count(last, "-") >= 4 {
			parts = parts[:len(parts)-1]
			continue
		}
		break
	}
	return strings.Join(parts, "/") + "/"
}

func versionID(version map[string]any) any {
	id := versionIDString(version)
	if id != "" {
		return id
	}
	return nil
}

func versionIDString(version map[string]any) string {
	id := quotaString(version["id"])
	id = strings.TrimPrefix(id, "v")
	if id != "" {
		return id
	}
	return quotaString(versionValue(version, "version"))
}

func versionStatus(version map[string]any) string {
	status := strings.ToUpper(quotaString(version["status"]))
	if status == "STABLE" {
		return "CURRENT"
	}
	return status
}

func versionFromEndpoint(endpoint string) any {
	trimmed := strings.TrimRight(endpoint, "/")
	parts := strings.Split(trimmed, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if strings.HasPrefix(strings.ToLower(part), "v") {
			version := strings.TrimPrefix(part, "v")
			if !strings.Contains(version, ".") {
				version += ".0"
			}
			return version
		}
	}
	return nil
}

func versionedEndpointURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	parts := strings.Split(trimmed, "/")
	for i, part := range parts {
		if strings.HasPrefix(strings.ToLower(part), "v") {
			return strings.Join(parts[:i+1], "/") + "/"
		}
	}
	return endpoint
}

func versionLess(left string, right string) bool {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}
	for i := 0; i < maxParts; i++ {
		leftPart := 0
		rightPart := 0
		if i < len(leftParts) {
			fmt.Sscanf(leftParts[i], "%d", &leftPart)
		}
		if i < len(rightParts) {
			fmt.Sscanf(rightParts[i], "%d", &rightPart)
		}
		if leftPart != rightPart {
			return leftPart < rightPart
		}
	}
	return left < right
}

func versionEndpoint(version map[string]any, fallback string) string {
	links, ok := version["links"].([]any)
	if !ok {
		return fallback
	}
	for _, raw := range links {
		link, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rel := quotaString(link["rel"])
		if rel != "" && rel != "self" {
			continue
		}
		href := quotaString(link["href"])
		if href != "" {
			return href
		}
	}
	return fallback
}

func versionValue(version map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := version[key]
		if !ok || value == nil || value == "" {
			continue
		}
		return value
	}
	return nil
}

func quotaListRows(ctx context.Context, client *gophercloud.ServiceClient, resource string, service string) ([]outputRow, error) {
	var response struct {
		Quotas []map[string]any `json:"quotas"`
	}
	resp, err := client.Get(ctx, client.ServiceURL(resource), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return []outputRow{}, nil
		}
		return nil, err
	}
	var rows []outputRow
	for _, quota := range response.Quotas {
		projectID := quotaString(quota["id"])
		if projectID == "" {
			projectID = quotaString(quota["tenant_id"])
		}
		for key, value := range quota {
			if key == "id" || key == "tenant_id" || key == "project_id" {
				continue
			}
			rows = append(rows, outputRow{
				"Project ID": projectID,
				"Service":    service,
				"Resource":   quotaResourceName(service, key),
				"Limit":      quotaLimit(value),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		left := valueString(rows[i]["Project ID"]) + valueString(rows[i]["Service"]) + valueString(rows[i]["Resource"])
		right := valueString(rows[j]["Project ID"]) + valueString(rows[j]["Service"]) + valueString(rows[j]["Resource"])
		return left < right
	})
	return rows, nil
}

func computeQuotaRows(ctx context.Context, client *gophercloud.ServiceClient, projectID string, usage bool) ([]outputRow, error) {
	path := client.ServiceURL("os-quota-sets", projectID)
	if usage {
		path = client.ServiceURL("os-quota-sets", projectID, "detail")
	}
	quota, err := quotaResponse(ctx, client, path, "quota_set")
	if err != nil {
		return nil, err
	}
	if usage {
		return quotaUsageRows(quota, []quotaField{
			{Key: "cores", Name: "cores"},
			{Key: "instances", Name: "instances"},
			{Key: "ram", Name: "ram"},
			{Key: "__osc_missing_fixed_ips", Name: "fixed_ips", DefaultLimit: 0},
			{Key: "injected_file_content_bytes", Name: "injected-file-size"},
			{Key: "injected_file_path_bytes", Name: "injected-path-size"},
			{Key: "injected_files", Name: "injected-files"},
			{Key: "key_pairs", Name: "key-pairs"},
			{Key: "metadata_items", Name: "properties"},
			{Key: "server_group_members", Name: "server-group-members"},
			{Key: "server_groups", Name: "server-groups"},
		}), nil
	}
	return quotaLimitRows(quota, []quotaField{
		{Key: "cores", Name: "cores"},
		{Key: "instances", Name: "instances"},
		{Key: "ram", Name: "ram"},
		{Key: "__osc_missing_fixed_ips", Name: "fixed_ips"},
		{Key: "__osc_missing_floating_ips", Name: "floating_ips"},
		{Key: "__osc_missing_networks", Name: "networks"},
		{Key: "__osc_missing_security_group_rules", Name: "security_group_rules"},
		{Key: "__osc_missing_security_groups", Name: "security_groups"},
		{Key: "injected_file_content_bytes", Name: "injected-file-size"},
		{Key: "injected_file_path_bytes", Name: "injected-path-size"},
		{Key: "injected_files", Name: "injected-files"},
		{Key: "key_pairs", Name: "key-pairs"},
		{Key: "metadata_items", Name: "properties"},
		{Key: "server_group_members", Name: "server-group-members"},
		{Key: "server_groups", Name: "server-groups"},
	}), nil
}

func volumeQuotaRows(ctx context.Context, client *gophercloud.ServiceClient, projectID string, usage bool, defaults bool) ([]outputRow, error) {
	path := client.ServiceURL("os-quota-sets", projectID)
	if usage {
		path += "?usage=true"
	} else if defaults {
		path = client.ServiceURL("os-quota-sets", projectID, "defaults")
	}
	quota, err := quotaResponse(ctx, client, path, "quota_set")
	if err != nil {
		return nil, err
	}

	fields := []quotaField{
		{Key: "volumes", Name: "volumes"},
		{Key: "snapshots", Name: "snapshots"},
		{Key: "gigabytes", Name: "gigabytes"},
		{Key: "backups", Name: "backups"},
	}
	extras := quotaExtraFields(quota, []string{"volumes_", "gigabytes_", "snapshots_"})
	fields = append(fields, extras...)
	fields = append(fields,
		quotaField{Key: "groups", Name: "groups"},
		quotaField{Key: "backup_gigabytes", Name: "backup-gigabytes"},
		quotaField{Key: "per_volume_gigabytes", Name: "per-volume-gigabytes"},
	)
	if usage {
		return quotaUsageRows(quota, fields), nil
	}
	return quotaLimitRows(quota, fields), nil
}

func networkQuotaRows(ctx context.Context, client *gophercloud.ServiceClient, projectID string, usage bool) ([]outputRow, error) {
	path := client.ServiceURL("quotas", projectID)
	if usage {
		path = client.ServiceURL("quotas", projectID, "details")
	}
	quota, err := quotaResponse(ctx, client, path, "quota")
	if err != nil {
		return nil, err
	}
	if usage {
		return quotaUsageRows(quota, []quotaField{
			{Key: "network", Name: "networks"},
			{Key: "port", Name: "ports"},
			{Key: "rbac_policy", Name: "rbac_policies"},
			{Key: "router", Name: "routers"},
			{Key: "subnet", Name: "subnets"},
			{Key: "subnetpool", Name: "subnet_pools"},
			{Key: "floatingip", Name: "floating-ips"},
			{Key: "security_group_rule", Name: "secgroup-rules"},
			{Key: "security_group", Name: "secgroups"},
		}), nil
	}
	return quotaLimitRows(quota, []quotaField{
		{Name: "check_limit"},
		{Name: "health_monitors"},
		{Name: "listeners"},
		{Name: "load_balancers"},
		{Name: "l7_policies"},
		{Key: "network", Name: "networks"},
		{Name: "pools"},
		{Key: "port", Name: "ports"},
		{Name: "project_id"},
		{Key: "rbac_policy", Name: "rbac_policies"},
		{Key: "router", Name: "routers"},
		{Key: "subnet", Name: "subnets"},
		{Key: "subnetpool", Name: "subnet_pools"},
		{Key: "floatingip", Name: "floating-ips"},
		{Key: "security_group_rule", Name: "secgroup-rules"},
		{Key: "security_group", Name: "secgroups"},
	}), nil
}

type quotaField struct {
	Key          string
	Name         string
	DefaultLimit any
}

func quotaResponse(ctx context.Context, client *gophercloud.ServiceClient, url string, key string) (map[string]any, error) {
	var response map[string]any
	resp, err := client.Get(ctx, url, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	raw, ok := response[key].(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return raw, nil
}

func quotaLimitRows(quota map[string]any, fields []quotaField) []outputRow {
	rows := make([]outputRow, 0, len(fields))
	for _, field := range fields {
		key := field.Key
		if key == "" {
			key = field.Name
		}
		value, ok := quota[key]
		limit := field.DefaultLimit
		if ok {
			limit = quotaLimit(value)
		}
		rows = append(rows, outputRow{"Resource": field.Name, "Limit": limit})
	}
	return rows
}

func quotaUsageRows(quota map[string]any, fields []quotaField) []outputRow {
	rows := make([]outputRow, 0, len(fields))
	for _, field := range fields {
		key := field.Key
		if key == "" {
			key = field.Name
		}
		value, ok := quota[key]
		if !ok {
			rows = append(rows, outputRow{"Resource": field.Name, "Limit": field.DefaultLimit, "In Use": 0, "Reserved": 0})
			continue
		}
		detail, ok := value.(map[string]any)
		if !ok {
			rows = append(rows, outputRow{"Resource": field.Name, "Limit": quotaLimit(value), "In Use": 0, "Reserved": 0})
			continue
		}
		inUse := quotaLimit(detail["in_use"])
		if inUse == nil {
			inUse = quotaLimit(detail["used"])
		}
		rows = append(rows, outputRow{
			"Resource": field.Name,
			"Limit":    quotaLimit(detail["limit"]),
			"In Use":   inUse,
			"Reserved": quotaLimit(detail["reserved"]),
		})
	}
	return rows
}

func quotaExtraFields(quota map[string]any, prefixes []string) []quotaField {
	var keys []string
	for key := range quota {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				keys = append(keys, key)
				break
			}
		}
	}
	sort.Strings(keys)
	fields := make([]quotaField, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, quotaField{Key: key, Name: key})
	}
	return fields
}

func quotaResourceName(service string, key string) string {
	switch service {
	case "compute":
		switch key {
		case "injected_file_content_bytes":
			return "injected-file-size"
		case "injected_file_path_bytes":
			return "injected-path-size"
		case "injected_files":
			return "injected-files"
		case "key_pairs":
			return "key-pairs"
		case "metadata_items":
			return "properties"
		case "server_group_members":
			return "server-group-members"
		case "server_groups":
			return "server-groups"
		}
	case "network":
		switch key {
		case "floatingip":
			return "floating-ips"
		case "security_group_rule":
			return "secgroup-rules"
		case "security_group":
			return "secgroups"
		case "subnetpool":
			return "subnet_pools"
		}
	case "volume":
		switch key {
		case "backup_gigabytes":
			return "backup-gigabytes"
		case "per_volume_gigabytes":
			return "per-volume-gigabytes"
		}
	}
	return key
}

func quotaLimit(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case int:
		return typed
	case int64:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return int64(typed)
		}
		return typed
	case jsonNumber:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		return typed
	default:
		return typed
	}
}

type jsonNumber interface {
	String() string
}

func quotaString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
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
