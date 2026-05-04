package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/catalog"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/credentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/ec2credentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/federation"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	identitylimits "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/policies"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/trusts"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"
)

func runIdentityRead(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := clients.identityV3()
		if err != nil {
			return err
		}

		switch path {
		case "access rule list":
			return identityAccessRuleList(cmd.Context(), stdout, opts, client)
		case "access rule show":
			return identityAccessRuleShow(cmd.Context(), stdout, opts, client, args)
		case "application credential list":
			return identityApplicationCredentialList(cmd.Context(), stdout, opts, client)
		case "application credential show":
			return identityApplicationCredentialShow(cmd.Context(), stdout, opts, client, args)
		case "catalog list":
			return identityCatalogList(cmd.Context(), stdout, opts, client)
		case "catalog show":
			return identityCatalogShow(cmd.Context(), stdout, opts, client, args)
		case "credential list":
			return identityCredentialList(cmd.Context(), stdout, opts, client)
		case "credential show":
			return identityCredentialShow(cmd.Context(), stdout, opts, client, args)
		case "domain list":
			return identityDomainList(cmd.Context(), stdout, opts, client)
		case "domain show":
			return identityDomainShow(cmd.Context(), stdout, opts, client, args)
		case "ec2 credentials list":
			return identityEC2CredentialList(cmd.Context(), stdout, opts, client)
		case "ec2 credentials show":
			return identityEC2CredentialShow(cmd.Context(), stdout, opts, client, args)
		case "endpoint list":
			return identityEndpointList(cmd.Context(), stdout, opts, client)
		case "endpoint show":
			return identityEndpointShow(cmd.Context(), stdout, opts, client, args)
		case "federation protocol list":
			return identityFederationProtocolList(cmd.Context(), stdout, opts, client)
		case "federation protocol show":
			return identityFederationProtocolShow(cmd.Context(), stdout, opts, client, args)
		case "group list":
			return identityGroupList(cmd.Context(), stdout, opts, client)
		case "group show":
			return identityGroupShow(cmd.Context(), stdout, opts, client, args)
		case "identity provider list":
			return identityProviderList(cmd.Context(), stdout, opts, client)
		case "identity provider show":
			return identityProviderShow(cmd.Context(), stdout, opts, client, args)
		case "implied role list":
			return identityImpliedRoleList(cmd.Context(), stdout, opts, client)
		case "limit list":
			return identityLimitList(cmd.Context(), stdout, opts, client)
		case "limit show":
			return identityLimitShow(cmd.Context(), stdout, opts, client, args)
		case "mapping list":
			return identityMappingList(cmd.Context(), stdout, opts, client)
		case "mapping show":
			return identityMappingShow(cmd.Context(), stdout, opts, client, args)
		case "policy list":
			return identityPolicyList(cmd.Context(), stdout, opts, client)
		case "policy show":
			return identityPolicyShow(cmd.Context(), stdout, opts, client, args)
		case "project list":
			return identityProjectList(cmd.Context(), stdout, opts, client)
		case "project show":
			return identityProjectShow(cmd.Context(), stdout, opts, client, args)
		case "region list":
			return identityRegionList(cmd.Context(), stdout, opts, client)
		case "region show":
			return identityRegionShow(cmd.Context(), stdout, opts, client, args)
		case "registered limit list":
			return identityRegisteredLimitList(cmd.Context(), stdout, opts, client)
		case "registered limit show":
			return identityRegisteredLimitShow(cmd.Context(), stdout, opts, client, args)
		case "role list":
			return identityRoleList(cmd.Context(), stdout, opts, client)
		case "role assignment list":
			return identityRoleAssignmentList(cmd.Context(), stdout, opts, clients, client)
		case "role show":
			return identityRoleShow(cmd.Context(), stdout, opts, client, args)
		case "service list":
			return identityServiceList(cmd.Context(), stdout, opts, client)
		case "service provider list":
			return identityServiceProviderList(cmd.Context(), stdout, opts, client)
		case "service provider show":
			return identityServiceProviderShow(cmd.Context(), stdout, opts, client, args)
		case "service show":
			return identityServiceShow(cmd.Context(), stdout, opts, client, args)
		case "trust list":
			return identityTrustList(cmd.Context(), stdout, opts, client)
		case "trust show":
			return identityTrustShow(cmd.Context(), stdout, opts, client, args)
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

func identityAccessRuleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	page, err := applicationcredentials.ListAccessRules(client, userID).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := applicationcredentials.ExtractAccessRules(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":      item.ID,
			"Service": item.Service,
			"Method":  item.Method,
			"Path":    item.Path,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Service", "Method", "Path"}, rows)
}

func identityAccessRuleShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("access rule show requires <access-rule>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	item, err := applicationcredentials.GetAccessRule(ctx, client, userID, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"method", item.Method},
		{"path", item.Path},
		{"service", item.Service},
	})
}

func identityApplicationCredentialList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	page, err := applicationcredentials.List(client, userID, applicationcredentials.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := applicationcredentials.ExtractApplicationCredentials(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":           item.ID,
			"Name":         item.Name,
			"Description":  item.Description,
			"Project ID":   item.ProjectID,
			"Roles":        applicationCredentialRoles(item.Roles),
			"Unrestricted": item.Unrestricted,
			"Access Rules": applicationCredentialAccessRules(item.AccessRules),
			"Expires At":   oscTime(item.ExpiresAt),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Description", "Project ID", "Roles", "Unrestricted", "Access Rules", "Expires At"}, rows)
}

func identityApplicationCredentialShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("application credential show requires <application-credential>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	item, err := findApplicationCredential(ctx, client, userID, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"access_rules", accessRuleDetails(item.AccessRules)},
		{"description", item.Description},
		{"expires_at", oscTime(item.ExpiresAt)},
		{"id", item.ID},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"roles", applicationCredentialRoles(item.Roles)},
		{"unrestricted", item.Unrestricted},
	})
}

func identityCredentialList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := credentials.ListOpts{
		Type: flagValue(opts, "type"),
	}
	if user := flagValue(opts, "user"); user != "" {
		item, err := findUser(ctx, client, user)
		if err != nil {
			return err
		}
		listOpts.UserID = item.ID
	}
	page, err := credentials.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := credentials.ExtractCredentials(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":         item.ID,
			"Type":       item.Type,
			"User ID":    item.UserID,
			"Data":       item.Blob,
			"Project ID": nilIfEmpty(item.ProjectID),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Type", "User ID", "Data", "Project ID"}, rows)
}

func identityCredentialShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("credential show requires <credential-id>")
	}
	item, err := credentials.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"blob", item.Blob},
		{"id", item.ID},
		{"links", item.Links},
		{"project_id", nilIfEmpty(item.ProjectID)},
		{"type", item.Type},
		{"user_id", item.UserID},
	})
}

func identityEC2CredentialList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	page, err := ec2credentials.List(client, userID).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := ec2credentials.ExtractCredentials(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Access":     item.Access,
			"Secret":     item.Secret,
			"Project ID": item.TenantID,
			"User ID":    item.UserID,
		})
	}
	return renderListOutput(stdout, opts, []string{"Access", "Secret", "Project ID", "User ID"}, rows)
}

func identityEC2CredentialShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ec2 credentials show requires <access-key>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	item, err := ec2credentials.Get(ctx, client, userID, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"access", item.Access},
		{"links", item.Links},
		{"project_id", item.TenantID},
		{"secret", item.Secret},
		{"trust_id", nilIfEmpty(item.TrustID)},
		{"user_id", item.UserID},
	})
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
		{"id", item.ID},
		{"name", item.Name},
		{"domain_id", nilIfEmpty(item.DomainID)},
		{"description", nilIfEmpty(item.Description)},
	})
}

type roleAssignmentListOpts struct {
	GroupID        string `q:"group.id"`
	RoleID         string `q:"role.id"`
	ScopeDomainID  string `q:"scope.domain.id"`
	ScopeProjectID string `q:"scope.project.id"`
	ScopeSystem    string `q:"scope.system"`
	UserID         string `q:"user.id"`
	Effective      *bool  `q:"effective"`
	IncludeNames   *bool  `q:"include_names"`
	InheritedTo    string `q:"scope.OS-INHERIT:inherited_to"`
}

func (opts roleAssignmentListOpts) ToRolesListAssignmentsQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

type roleAssignmentEntity struct {
	ID     string                `json:"id,omitempty"`
	Name   string                `json:"name,omitempty"`
	Domain *roleAssignmentEntity `json:"domain,omitempty"`
}

type roleAssignmentScope struct {
	Domain      *roleAssignmentEntity `json:"domain,omitempty"`
	Project     *roleAssignmentEntity `json:"project,omitempty"`
	System      map[string]any        `json:"system,omitempty"`
	InheritedTo string                `json:"OS-INHERIT:inherited_to,omitempty"`
}

type roleAssignmentRecord struct {
	Role  *roleAssignmentEntity `json:"role,omitempty"`
	Scope roleAssignmentScope   `json:"scope,omitempty"`
	User  *roleAssignmentEntity `json:"user,omitempty"`
	Group *roleAssignmentEntity `json:"group,omitempty"`
}

func identityRoleAssignmentList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	if flagValue(opts, "user") != "" && flagValue(opts, "group") != "" {
		return fmt.Errorf("argument --group: not allowed with argument --user")
	}
	scopeFilters := 0
	for _, name := range []string{"domain", "project", "system"} {
		if flagValue(opts, name) != "" {
			scopeFilters++
		}
	}
	if scopeFilters > 1 {
		return fmt.Errorf("arguments --domain, --project, and --system are mutually exclusive")
	}

	listOpts := roleAssignmentListOpts{}
	if boolFlag(opts, "effective") {
		listOpts.Effective = boolPointer(true)
	}
	includeNames := boolFlag(opts, "names")
	if includeNames {
		listOpts.IncludeNames = boolPointer(true)
	}
	if boolFlag(opts, "inherited") {
		listOpts.InheritedTo = "projects"
	}
	if value := flagValue(opts, "role"); value != "" {
		item, err := findRoleWithDomain(ctx, client, value, flagValue(opts, "role-domain"))
		if err != nil {
			return err
		}
		listOpts.RoleID = item.ID
	}
	if value := flagValue(opts, "user"); value != "" {
		item, err := findUserWithDomain(ctx, client, value, flagValue(opts, "user-domain"))
		if err != nil {
			return err
		}
		listOpts.UserID = item.ID
	} else if boolFlag(opts, "auth-user") {
		userID, err := currentTokenUserID(ctx, client)
		if err != nil {
			return err
		}
		listOpts.UserID = userID
	}
	if value := flagValue(opts, "group"); value != "" {
		item, err := findGroupWithDomain(ctx, client, value, flagValue(opts, "group-domain"))
		if err != nil {
			return err
		}
		listOpts.GroupID = item.ID
	}
	if value := flagValue(opts, "domain"); value != "" {
		item, err := findDomain(ctx, client, value)
		if err != nil {
			return err
		}
		listOpts.ScopeDomainID = item.ID
	}
	if value := flagValue(opts, "project"); value != "" {
		item, err := findProjectWithDomain(ctx, client, value, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		listOpts.ScopeProjectID = item.ID
	} else if boolFlag(opts, "auth-project") {
		listOpts.ScopeProjectID = currentProjectID(clients)
	}
	if value := flagValue(opts, "system"); value != "" {
		listOpts.ScopeSystem = value
	}

	page, err := roles.ListAssignments(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := extractRoleAssignmentRecords(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Role":      roleAssignmentValue(item.Role, includeNames),
			"User":      roleAssignmentValue(item.User, includeNames),
			"Group":     roleAssignmentValue(item.Group, includeNames),
			"Project":   roleAssignmentValue(item.Scope.Project, includeNames),
			"Domain":    roleAssignmentValue(item.Scope.Domain, includeNames),
			"System":    roleAssignmentSystem(item.Scope.System),
			"Inherited": item.Scope.InheritedTo == "projects",
		})
	}
	return renderListOutput(stdout, opts, []string{"Role", "User", "Group", "Project", "Domain", "System", "Inherited"}, rows)
}

func extractRoleAssignmentRecords(page pagination.Page) ([]roleAssignmentRecord, error) {
	var body struct {
		RoleAssignments []roleAssignmentRecord `json:"role_assignments"`
	}
	err := page.(roles.RoleAssignmentPage).ExtractInto(&body)
	return body.RoleAssignments, err
}

func roleAssignmentValue(entity *roleAssignmentEntity, includeNames bool) string {
	if entity == nil {
		return ""
	}
	if !includeNames {
		return entity.ID
	}
	if entity.Name == "" {
		return ""
	}
	if entity.Domain != nil && entity.Domain.Name != "" {
		return entity.Name + "@" + entity.Domain.Name
	}
	return entity.Name
}

func roleAssignmentSystem(system map[string]any) string {
	if len(system) == 0 {
		return ""
	}
	return "all"
}

func boolPointer(value bool) *bool {
	return &value
}

type mappingRecord struct {
	ID            string                   `json:"id,omitempty"`
	Links         map[string]any           `json:"links,omitempty"`
	Rules         []federation.MappingRule `json:"rules,omitempty"`
	SchemaVersion any                      `json:"schema_version,omitempty"`
}

func identityMappingList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := federation.ListMappings(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := extractMappingRecords(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":             item.ID,
			"schema_version": item.SchemaVersion,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "schema_version"}, rows)
}

func identityMappingShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mapping show requires <mapping>")
	}
	var body struct {
		Mapping mappingRecord `json:"mapping"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "mappings", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", body.Mapping.ID},
		{"rules", body.Mapping.Rules},
		{"schema_version", body.Mapping.SchemaVersion},
	})
}

func extractMappingRecords(page pagination.Page) ([]mappingRecord, error) {
	var body struct {
		Mappings []mappingRecord `json:"mappings"`
	}
	err := page.(federation.MappingsPage).ExtractInto(&body)
	return body.Mappings, err
}

type identityProviderRecord struct {
	ID          string         `json:"id,omitempty"`
	Enabled     bool           `json:"enabled,omitempty"`
	DomainID    string         `json:"domain_id,omitempty"`
	Description string         `json:"description,omitempty"`
	RemoteIDs   []string       `json:"remote_ids,omitempty"`
	Links       map[string]any `json:"links,omitempty"`
}

func identityProviderList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	var body struct {
		IdentityProviders []identityProviderRecord `json:"identity_providers"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(body.IdentityProviders))
	for _, item := range body.IdentityProviders {
		if id := flagValue(opts, "id"); id != "" && item.ID != id {
			continue
		}
		if boolFlag(opts, "enabled") && !item.Enabled {
			continue
		}
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Enabled":     item.Enabled,
			"Domain ID":   item.DomainID,
			"Description": item.Description,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Enabled", "Domain ID", "Description"}, rows)
}

func identityProviderShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("identity provider show requires <identity-provider>")
	}
	item, err := getIdentityProvider(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", item.DomainID},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"remote_ids", item.RemoteIDs},
	})
}

func getIdentityProvider(ctx context.Context, client *gophercloud.ServiceClient, id string) (*identityProviderRecord, error) {
	var body struct {
		IdentityProvider identityProviderRecord `json:"identity_provider"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", id), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return &body.IdentityProvider, nil
}

type serviceProviderRecord struct {
	ID                 string `json:"id,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`
	Description        string `json:"description,omitempty"`
	AuthURL            string `json:"auth_url,omitempty"`
	ServiceProviderURL string `json:"sp_url,omitempty"`
	RelayStatePrefix   string `json:"relay_state_prefix,omitempty"`
}

func identityServiceProviderList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	var body struct {
		ServiceProviders []serviceProviderRecord `json:"service_providers"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "service_providers"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(body.ServiceProviders))
	for _, item := range body.ServiceProviders {
		rows = append(rows, outputRow{
			"ID":                   item.ID,
			"Enabled":              item.Enabled,
			"Description":          item.Description,
			"Auth URL":             item.AuthURL,
			"Service Provider URL": item.ServiceProviderURL,
			"Relay State Prefix":   item.RelayStatePrefix,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Enabled", "Description", "Auth URL", "Service Provider URL", "Relay State Prefix"}, rows)
}

func identityServiceProviderShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service provider show requires <service-provider>")
	}
	var body struct {
		ServiceProvider serviceProviderRecord `json:"service_provider"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "service_providers", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", body.ServiceProvider.ID},
		{"enabled", body.ServiceProvider.Enabled},
		{"description", body.ServiceProvider.Description},
		{"auth_url", body.ServiceProvider.AuthURL},
		{"sp_url", body.ServiceProvider.ServiceProviderURL},
		{"relay_state_prefix", body.ServiceProvider.RelayStatePrefix},
	})
}

type federationProtocolRecord struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IDPID     string `json:"idp_id,omitempty"`
	MappingID string `json:"mapping_id,omitempty"`
}

func identityFederationProtocolList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	idpID := flagValue(opts, "identity-provider")
	if idpID == "" {
		return fmt.Errorf("federation protocol list requires --identity-provider <identity-provider>")
	}
	var body struct {
		Protocols []federationProtocolRecord `json:"protocols"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", idpID, "protocols"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(body.Protocols))
	for _, item := range body.Protocols {
		rows = append(rows, outputRow{
			"id":      firstNonEmpty(item.Name, item.ID),
			"mapping": item.MappingID,
		})
	}
	return renderListOutput(stdout, opts, []string{"id", "mapping"}, rows)
}

func identityFederationProtocolShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	idpID := flagValue(opts, "identity-provider")
	if idpID == "" {
		return fmt.Errorf("federation protocol show requires --identity-provider <identity-provider>")
	}
	if len(args) < 1 {
		return fmt.Errorf("federation protocol show requires <federation-protocol>")
	}
	var body struct {
		Protocol federationProtocolRecord `json:"protocol"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", idpID, "protocols", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", firstNonEmpty(body.Protocol.Name, body.Protocol.ID)},
		{"identity_provider", firstNonEmpty(body.Protocol.IDPID, idpID)},
		{"mapping", body.Protocol.MappingID},
	})
}

func identityImpliedRoleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	result, err := roles.ListRoleInferenceRules(ctx, client).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0)
	for _, rule := range result.RoleInferenceRuleList {
		for _, implied := range rule.ImpliedRoles {
			rows = append(rows, outputRow{
				"Prior Role ID":     rule.PriorRole.ID,
				"Prior Role Name":   rule.PriorRole.Name,
				"Implied Role ID":   implied.ID,
				"Implied Role Name": implied.Name,
			})
		}
	}
	return renderListOutput(stdout, opts, []string{"Prior Role ID", "Prior Role Name", "Implied Role ID", "Implied Role Name"}, rows)
}

func identityLimitList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := identitylimits.ListOpts{
		RegionID:     flagValue(opts, "region"),
		ResourceName: flagValue(opts, "resource-name"),
		ProjectID:    flagValue(opts, "project"),
	}
	if service := flagValue(opts, "service"); service != "" {
		item, err := findService(ctx, client, service)
		if err != nil {
			return err
		}
		listOpts.ServiceID = item.ID
	}
	page, err := identitylimits.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := identitylimits.ExtractLimits(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":             item.ID,
			"Project ID":     item.ProjectID,
			"Service ID":     item.ServiceID,
			"Resource Name":  item.ResourceName,
			"Resource Limit": item.ResourceLimit,
			"Description":    item.Description,
			"Region ID":      nilIfEmpty(item.RegionID),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Project ID", "Service ID", "Resource Name", "Resource Limit", "Description", "Region ID"}, rows)
}

func identityLimitShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("limit show requires <limit>")
	}
	item, err := identitylimits.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", nilIfEmpty(item.DomainID)},
		{"id", item.ID},
		{"links", item.Links},
		{"project_id", item.ProjectID},
		{"region_id", nilIfEmpty(item.RegionID)},
		{"resource_limit", item.ResourceLimit},
		{"resource_name", item.ResourceName},
		{"service_id", item.ServiceID},
	})
}

func identityPolicyList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := policies.List(client, nil).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := policies.ExtractPolicies(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":   item.ID,
			"Type": item.Type,
		}
		if boolFlag(opts, "long") {
			row["Rules"] = item.Blob
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Type"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Rules")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func identityPolicyShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("policy show requires <policy>")
	}
	item, err := policies.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"blob", item.Blob},
		{"extra", item.Extra},
		{"id", item.ID},
		{"links", item.Links},
		{"type", item.Type},
	})
}

func identityRegisteredLimitList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := registeredlimits.ListOpts{
		RegionID:     flagValue(opts, "region"),
		ResourceName: flagValue(opts, "resource-name"),
	}
	if service := flagValue(opts, "service"); service != "" {
		item, err := findService(ctx, client, service)
		if err != nil {
			return err
		}
		listOpts.ServiceID = item.ID
	}
	page, err := registeredlimits.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := registeredlimits.ExtractRegisteredLimits(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":            item.ID,
			"Service ID":    item.ServiceID,
			"Resource Name": item.ResourceName,
			"Default Limit": item.DefaultLimit,
			"Description":   item.Description,
			"Region ID":     nilIfEmpty(item.RegionID),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Service ID", "Resource Name", "Default Limit", "Description", "Region ID"}, rows)
}

func identityRegisteredLimitShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("registered limit show requires <registered-limit>")
	}
	item, err := registeredlimits.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"default_limit", item.DefaultLimit},
		{"description", item.Description},
		{"id", item.ID},
		{"links", item.Links},
		{"region_id", nilIfEmpty(item.RegionID)},
		{"resource_name", item.ResourceName},
		{"service_id", item.ServiceID},
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
		{"description", item.Description},
		{"parent_region", nilIfEmpty(item.ParentRegionID)},
	})
}

func identityTrustList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := trusts.ListOpts{}
	if trustor := flagValue(opts, "trustor"); trustor != "" {
		item, err := findUser(ctx, client, trustor)
		if err != nil {
			return err
		}
		listOpts.TrustorUserID = item.ID
	}
	if trustee := flagValue(opts, "trustee"); trustee != "" {
		item, err := findUser(ctx, client, trustee)
		if err != nil {
			return err
		}
		listOpts.TrusteeUserID = item.ID
	}
	if boolFlag(opts, "auth-user") {
		userID, err := currentTokenUserID(ctx, client)
		if err != nil {
			return err
		}
		if listOpts.TrustorUserID == "" {
			listOpts.TrustorUserID = userID
		}
	}
	page, err := trusts.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := trusts.ExtractTrusts(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":              item.ID,
			"Expires At":      oscTime(item.ExpiresAt),
			"Impersonation":   item.Impersonation,
			"Project ID":      nilIfEmpty(item.ProjectID),
			"Trustee User ID": item.TrusteeUserID,
			"Trustor User ID": item.TrustorUserID,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Expires At", "Impersonation", "Project ID", "Trustee User ID", "Trustor User ID"}, rows)
}

func identityTrustShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("trust show requires <trust>")
	}
	item, err := trusts.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"allow_redelegation", item.AllowRedelegation},
		{"deleted_at", oscTime(item.DeletedAt)},
		{"expires_at", oscTime(item.ExpiresAt)},
		{"id", item.ID},
		{"impersonation", item.Impersonation},
		{"project_id", nilIfEmpty(item.ProjectID)},
		{"redelegated_trust_id", nilIfEmpty(item.RedelegatedTrustID)},
		{"redelegation_count", item.RedelegationCount},
		{"remaining_uses", item.RemainingUses},
		{"roles", trustRoles(item.Roles)},
		{"trustee_user_id", item.TrusteeUserID},
		{"trustor_user_id", item.TrustorUserID},
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

func identityUserIDForScopedCommand(ctx context.Context, client *gophercloud.ServiceClient, opts *Options) (string, error) {
	if user := flagValue(opts, "user"); user != "" {
		item, err := findUser(ctx, client, user)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	}
	return currentTokenUserID(ctx, client)
}

func currentTokenUserID(ctx context.Context, client *gophercloud.ServiceClient) (string, error) {
	result := tokens.Get(ctx, client, client.Token())
	user, err := result.ExtractUser()
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func applicationCredentialRoles(items []applicationcredentials.Role) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, firstNonEmpty(item.Name, item.ID))
	}
	return values
}

func applicationCredentialAccessRules(items []applicationcredentials.AccessRule) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.ID)
	}
	return values
}

func accessRuleDetails(items []applicationcredentials.AccessRule) []map[string]any {
	values := make([]map[string]any, 0, len(items))
	for _, item := range items {
		values = append(values, map[string]any{
			"id":      item.ID,
			"method":  item.Method,
			"path":    item.Path,
			"service": item.Service,
		})
	}
	return values
}

func trustRoles(items []trusts.Role) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, firstNonEmpty(item.Name, item.ID))
	}
	return values
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

func findProjectWithDomain(ctx context.Context, client *gophercloud.ServiceClient, value string, domainValue string) (*projects.Project, error) {
	if domainValue == "" {
		return findProject(ctx, client, value)
	}
	domain, err := findDomain(ctx, client, domainValue)
	if err != nil {
		return nil, err
	}
	result := projects.Get(ctx, client, value)
	if result.Err == nil {
		item, err := result.Extract()
		if err == nil && item.DomainID == domain.ID {
			return item, nil
		}
	}
	page, err := projects.List(client, projects.ListOpts{Name: value, DomainID: domain.ID}).AllPages(ctx)
	if err != nil {
		return nil, err
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

func findUserWithDomain(ctx context.Context, client *gophercloud.ServiceClient, value string, domainValue string) (*users.User, error) {
	if domainValue == "" {
		return findUser(ctx, client, value)
	}
	domain, err := findDomain(ctx, client, domainValue)
	if err != nil {
		return nil, err
	}
	result := users.Get(ctx, client, value)
	if result.Err == nil {
		item, err := result.Extract()
		if err == nil && item.DomainID == domain.ID {
			return item, nil
		}
	}
	page, err := users.List(client, users.ListOpts{Name: value, DomainID: domain.ID}).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := users.ExtractUsers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item users.User) string { return item.Name })
}

func findApplicationCredential(ctx context.Context, client *gophercloud.ServiceClient, userID string, value string) (*applicationcredentials.ApplicationCredential, error) {
	result := applicationcredentials.Get(ctx, client, userID, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := applicationcredentials.List(client, userID, applicationcredentials.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := applicationcredentials.ExtractApplicationCredentials(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item applicationcredentials.ApplicationCredential) string { return item.Name })
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

func findGroupWithDomain(ctx context.Context, client *gophercloud.ServiceClient, value string, domainValue string) (*groups.Group, error) {
	if domainValue == "" {
		return findGroup(ctx, client, value)
	}
	domain, err := findDomain(ctx, client, domainValue)
	if err != nil {
		return nil, err
	}
	result := groups.Get(ctx, client, value)
	if result.Err == nil {
		item, err := result.Extract()
		if err == nil && item.DomainID == domain.ID {
			return item, nil
		}
	}
	page, err := groups.List(client, groups.ListOpts{Name: value, DomainID: domain.ID}).AllPages(ctx)
	if err != nil {
		return nil, err
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

func findRoleWithDomain(ctx context.Context, client *gophercloud.ServiceClient, value string, domainValue string) (*roles.Role, error) {
	if domainValue == "" {
		return findRole(ctx, client, value)
	}
	domain, err := findDomain(ctx, client, domainValue)
	if err != nil {
		return nil, err
	}
	result := roles.Get(ctx, client, value)
	if result.Err == nil {
		item, err := result.Extract()
		if err == nil && item.DomainID == domain.ID {
			return item, nil
		}
	}
	page, err := roles.List(client, roles.ListOpts{Name: value, DomainID: domain.ID}).AllPages(ctx)
	if err != nil {
		return nil, err
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

func flagValues(opts *Options, name string) []string {
	if opts == nil || opts.CommandFlagList == nil {
		return nil
	}
	return opts.CommandFlagList[name]
}

func flagChanged(opts *Options, name string) bool {
	if opts == nil || opts.CommandFlags == nil {
		return false
	}
	_, ok := opts.CommandFlags[name]
	return ok
}

func boolFlag(opts *Options, name string) bool {
	return flagValue(opts, name) == "true"
}
