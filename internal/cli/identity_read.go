package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/catalog"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"github.com/spf13/cobra"
)

func runIdentityRead(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		client, err := identityClient(cmd.Context(), opts)
		if err != nil {
			return err
		}

		switch path {
		case "catalog list":
			return identityCatalogList(cmd.Context(), stdout, opts, client)
		case "catalog show":
			return identityCatalogShow(cmd.Context(), stdout, opts, client, args)
		case "domain list":
			return identityDomainList(cmd.Context(), stdout, opts, client)
		case "domain show":
			return identityDomainShow(cmd.Context(), stdout, opts, client, args)
		case "endpoint list":
			return identityEndpointList(cmd.Context(), stdout, opts, client)
		case "endpoint show":
			return identityEndpointShow(cmd.Context(), stdout, opts, client, args)
		case "group list":
			return identityGroupList(cmd.Context(), stdout, opts, client)
		case "group show":
			return identityGroupShow(cmd.Context(), stdout, opts, client, args)
		case "project list":
			return identityProjectList(cmd.Context(), stdout, opts, client)
		case "project show":
			return identityProjectShow(cmd.Context(), stdout, opts, client, args)
		case "region list":
			return identityRegionList(cmd.Context(), stdout, opts, client)
		case "region show":
			return identityRegionShow(cmd.Context(), stdout, opts, client, args)
		case "role list":
			return identityRoleList(cmd.Context(), stdout, opts, client)
		case "role show":
			return identityRoleShow(cmd.Context(), stdout, opts, client, args)
		case "service list":
			return identityServiceList(cmd.Context(), stdout, opts, client)
		case "service show":
			return identityServiceShow(cmd.Context(), stdout, opts, client, args)
		case "user list":
			return identityUserList(cmd.Context(), stdout, opts, client)
		case "user show":
			return identityUserShow(cmd.Context(), stdout, opts, client, args)
		default:
			return fmt.Errorf("identity command %q is not wired", path)
		}
	}
}

func identityClient(ctx context.Context, opts *Options) (*gophercloud.ServiceClient, error) {
	clients, err := newOpenStackClients(ctx, opts)
	if err != nil {
		return nil, err
	}
	return clients.identityV3()
}

func identityDomainList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	enabled, hasEnabled := enabledFilter(opts)
	var listOpts domains.ListOpts
	if hasEnabled {
		listOpts.Enabled = &enabled
	}
	page, err := domains.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := domains.ExtractDomains(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name, "Enabled": item.Enabled, "Description": item.Description})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Enabled", "Description"}, rows)
}

func identityDomainShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("domain show requires <domain>")
	}
	item, err := findDomain(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"enabled", item.Enabled},
		{"description", item.Description},
		{"options", map[string]any{}},
	})
}

func identityProjectList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	enabled, hasEnabled := enabledFilter(opts)
	listOpts := projects.ListOpts{
		DomainID: flagValue(opts, "domain"),
		Tags:     flagValue(opts, "tags"),
		TagsAny:  flagValue(opts, "tags-any"),
		NotTags:  flagValue(opts, "not-tags"),
	}
	if value := flagValue(opts, "not-tags-any"); value != "" {
		listOpts.NotTagsAny = value
	}
	if hasEnabled {
		listOpts.Enabled = &enabled
	}
	page, err := projects.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := projects.ExtractProjects(page)
	if err != nil {
		return err
	}
	long := boolFlag(opts, "long")
	columns := []string{"ID", "Name"}
	if long {
		columns = []string{"ID", "Name", "Domain ID", "Description", "Enabled", "Parent ID", "Tags"}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name}
		if long {
			row["Domain ID"] = item.DomainID
			row["Description"] = item.Description
			row["Enabled"] = item.Enabled
			row["Parent ID"] = item.ParentID
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func identityProjectShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project show requires <project>")
	}
	item, err := findProject(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", item.DomainID},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"is_domain", item.IsDomain},
		{"name", item.Name},
		{"options", item.Options},
		{"parent_id", item.ParentID},
		{"tags", item.Tags},
	})
}

func identityUserList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	enabled, hasEnabled := enabledFilter(opts)
	listOpts := users.ListOpts{DomainID: flagValue(opts, "domain")}
	if hasEnabled {
		listOpts.Enabled = &enabled
	}
	page, err := users.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := users.ExtractUsers(page)
	if err != nil {
		return err
	}
	long := boolFlag(opts, "long")
	columns := []string{"ID", "Name"}
	if long {
		columns = []string{"ID", "Name", "Domain ID", "Description", "Enabled", "Default Project ID"}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name}
		if long {
			row["Domain ID"] = item.DomainID
			row["Description"] = item.Description
			row["Enabled"] = item.Enabled
			row["Default Project ID"] = item.DefaultProjectID
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func identityUserShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("user show requires <user>")
	}
	item, err := findUser(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"default_project_id", nilIfEmpty(item.DefaultProjectID)},
		{"domain_id", item.DomainID},
		{"email", item.Extra["email"]},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"name", item.Name},
		{"description", nilIfEmpty(item.Description)},
		{"password_expires_at", nil},
		{"options", item.Options},
	})
}

func identityGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := groups.List(client, groups.ListOpts{DomainID: flagValue(opts, "domain")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := groups.ExtractGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name"}, rows)
}

func identityGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("group show requires <group>")
	}
	item, err := findGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", item.DomainID},
		{"id", item.ID},
		{"name", item.Name},
	})
}

func identityRoleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := roles.List(client, roles.ListOpts{DomainID: flagValue(opts, "domain")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := roles.ExtractRoles(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name"}, rows)
}

func identityRoleShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("role show requires <role>")
	}
	item, err := findRole(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"domain_id", nilIfEmpty(item.DomainID)},
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"options", item.Options},
	})
}

func identityServiceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := services.List(client, nil).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := services.ExtractServices(page)
	if err != nil {
		return err
	}
	long := boolFlag(opts, "long")
	columns := []string{"ID", "Name", "Type"}
	if long {
		columns = []string{"ID", "Name", "Type", "Enabled", "Description"}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name, "Type": item.Type}
		if long {
			row["Enabled"] = item.Enabled
			row["Description"] = item.Description
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func identityServiceShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service show requires <service>")
	}
	item, err := findService(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"type", item.Type},
		{"enabled", item.Enabled},
		{"description", item.Description},
	})
}

func identityEndpointList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	serviceByID, err := serviceByID(ctx, client)
	if err != nil {
		return err
	}
	listOpts := endpoints.ListOpts{RegionID: flagValue(opts, "region")}
	if service := flagValue(opts, "service"); service != "" {
		item, err := findService(ctx, client, service)
		if err != nil {
			return err
		}
		listOpts.ServiceID = item.ID
	}
	if iface := flagValue(opts, "interface"); iface != "" {
		listOpts.Availability = availabilityFromInterface(iface)
	}
	page, err := endpoints.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := endpoints.ExtractEndpoints(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		service := serviceByID[item.ServiceID]
		rows = append(rows, outputRow{
			"ID":           item.ID,
			"Region":       item.Region,
			"Service Name": service.Name,
			"Service Type": service.Type,
			"Enabled":      item.Enabled,
			"Interface":    string(item.Availability),
			"URL":          item.URL,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Region", "Service Name", "Service Type", "Enabled", "Interface", "URL"}, rows)
}

func identityEndpointShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("endpoint show requires <endpoint>")
	}
	item, err := findEndpoint(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"region", item.Region},
		{"service_id", item.ServiceID},
		{"interface", string(item.Availability)},
		{"url", item.URL},
		{"enabled", item.Enabled},
	})
}

func identityRegionList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := regions.List(client, regions.ListOpts{ParentRegionID: flagValue(opts, "parent-region")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := regions.ExtractRegions(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"Region": item.ID, "Parent Region": nilIfEmpty(item.ParentRegionID), "Description": item.Description})
	}
	return renderListOutput(stdout, opts, []string{"Region", "Parent Region", "Description"}, rows)
}

func identityRegionShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("region show requires <region>")
	}
	result := regions.Get(ctx, client, args[0])
	item, err := result.Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"region", item.ID},
		{"parent_region", nilIfEmpty(item.ParentRegionID)},
		{"description", item.Description},
	})
}

func identityCatalogList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := catalogEntries(ctx, client)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"Name": item.Name, "Type": item.Type, "Endpoints": item.Endpoints})
	}
	return renderListOutput(stdout, opts, []string{"Name", "Type", "Endpoints"}, rows)
}

func identityCatalogShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("catalog show requires <service>")
	}
	items, err := catalogEntries(ctx, client)
	if err != nil {
		return err
	}
	item, err := findCatalogEntry(items, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"endpoints", item.Endpoints},
		{"id", item.ID},
		{"name", item.Name},
		{"type", item.Type},
	})
}

func catalogEntries(ctx context.Context, client *gophercloud.ServiceClient) ([]tokens.CatalogEntry, error) {
	page, err := catalog.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return catalog.ExtractServiceCatalog(page)
}

func serviceByID(ctx context.Context, client *gophercloud.ServiceClient) (map[string]services.Service, error) {
	page, err := services.List(client, nil).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := services.ExtractServices(page)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]services.Service, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	return byID, nil
}

func findDomain(ctx context.Context, client *gophercloud.ServiceClient, value string) (*domains.Domain, error) {
	result := domains.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := domains.List(client, domains.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := domains.ExtractDomains(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item domains.Domain) string { return item.Name })
}

func findProject(ctx context.Context, client *gophercloud.ServiceClient, value string) (*projects.Project, error) {
	result := projects.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := projects.List(client, projects.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := projects.ExtractProjects(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item projects.Project) string { return item.Name })
}

func findUser(ctx context.Context, client *gophercloud.ServiceClient, value string) (*users.User, error) {
	result := users.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := users.List(client, users.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := users.ExtractUsers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item users.User) string { return item.Name })
}

func findGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*groups.Group, error) {
	result := groups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := groups.List(client, groups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := groups.ExtractGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item groups.Group) string { return item.Name })
}

func findRole(ctx context.Context, client *gophercloud.ServiceClient, value string) (*roles.Role, error) {
	result := roles.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := roles.List(client, roles.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := roles.ExtractRoles(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item roles.Role) string { return item.Name })
}

func findService(ctx context.Context, client *gophercloud.ServiceClient, value string) (*services.Service, error) {
	result := services.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := services.List(client, nil).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := services.ExtractServices(page)
	if err != nil {
		return nil, err
	}
	var matches []services.Service
	for _, item := range items {
		if item.Name == value || item.Type == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func findEndpoint(ctx context.Context, client *gophercloud.ServiceClient, value string) (*endpoints.Endpoint, error) {
	result := endpoints.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	service, err := findService(ctx, client, value)
	if err != nil {
		return nil, result.Err
	}
	page, err := endpoints.List(client, endpoints.ListOpts{ServiceID: service.ID}).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := endpoints.ExtractEndpoints(page)
	if err != nil {
		return nil, err
	}
	return singleMatch(value, items)
}

func findCatalogEntry(items []tokens.CatalogEntry, value string) (*tokens.CatalogEntry, error) {
	var matches []tokens.CatalogEntry
	for _, item := range items {
		if item.ID == value || item.Name == value || item.Type == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func singleByName[T any](value string, items []T, name func(T) string) (*T, error) {
	var matches []T
	for _, item := range items {
		if name(item) == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func singleMatch[T any](value string, matches []T) (*T, error) {
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no resource found for %q", value)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("multiple resources found for %q", value)
	}
}

func enabledFilter(opts *Options) (bool, bool) {
	if boolFlag(opts, "enabled") {
		return true, true
	}
	if boolFlag(opts, "disabled") {
		return false, true
	}
	return false, false
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func flagValue(opts *Options, name string) string {
	if opts == nil || opts.CommandFlags == nil {
		return ""
	}
	return opts.CommandFlags[name]
}

func boolFlag(opts *Options, name string) bool {
	return flagValue(opts, name) == "true"
}
