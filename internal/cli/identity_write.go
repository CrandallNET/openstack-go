package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/credentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/ec2credentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/federation"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	identitylimits "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/oauth1"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/osinherit"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/policies"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projectendpoints"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/trusts"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
)

type identityRawResource struct {
	Singular string
	Plural   string
	Path     []string
	Columns  []string
	Keys     []string
}

func identityAccessRuleDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("access rule delete requires <access-rule>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	for _, id := range args {
		if err := applicationcredentials.DeleteAccessRule(ctx, client, userID, id).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityApplicationCredentialCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("application credential create requires <name>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	createOpts := applicationcredentials.CreateOpts{
		Name:         args[0],
		Description:  flagValue(opts, "description"),
		Secret:       flagValue(opts, "secret"),
		Unrestricted: boolFlag(opts, "unrestricted"),
	}
	if expires := flagValue(opts, "expiration"); expires != "" {
		parsed, err := parseIdentityTime(expires)
		if err != nil {
			return err
		}
		createOpts.ExpiresAt = &parsed
	}
	for _, roleValue := range flagValues(opts, "role") {
		role, err := findRole(ctx, client, roleValue)
		if err != nil {
			return err
		}
		createOpts.Roles = append(createOpts.Roles, applicationcredentials.Role{ID: role.ID})
	}
	item, err := applicationcredentials.Create(ctx, client, userID, createOpts).Extract()
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
		{"secret", item.Secret},
		{"unrestricted", item.Unrestricted},
	})
}

func identityApplicationCredentialDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("application credential delete requires <application-credential>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	for _, value := range args {
		item, err := findApplicationCredential(ctx, client, userID, value)
		if err != nil {
			return err
		}
		if err := applicationcredentials.Delete(ctx, client, userID, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityConsumerCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	item, err := oauth1.CreateConsumer(ctx, client, oauth1.CreateConsumerOpts{Description: flagValue(opts, "description")}).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"id", item.ID},
		{"secret", item.Secret},
	})
}

func identityConsumerDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consumer delete requires <consumer>")
	}
	for _, value := range args {
		item, err := findConsumer(ctx, client, value)
		if err != nil {
			return err
		}
		if err := oauth1.DeleteConsumer(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityConsumerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := oauth1.ListConsumers(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := oauth1.ExtractConsumers(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Description": item.Description})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Description"}, rows)
}

func identityConsumerSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consumer set requires <consumer>")
	}
	item, err := findConsumer(ctx, client, args[0])
	if err != nil {
		return err
	}
	_, err = oauth1.UpdateConsumer(ctx, client, item.ID, oauth1.UpdateConsumerOpts{Description: flagValue(opts, "description")}).Extract()
	return err
}

func identityConsumerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consumer show requires <consumer>")
	}
	item, err := findConsumer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"id", item.ID},
		{"secret", item.Secret},
	})
}

func identityCredentialCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("credential create requires <type> <data>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	projectID, err := optionalProjectID(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	item, err := credentials.Create(ctx, client, credentials.CreateOpts{
		Type:      args[0],
		Blob:      args[1],
		UserID:    userID,
		ProjectID: projectID,
	}).Extract()
	if err != nil {
		return err
	}
	return renderCredential(stdout, opts, item)
}

func identityCredentialDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("credential delete requires <credential-id>")
	}
	for _, id := range args {
		if err := credentials.Delete(ctx, client, id).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityCredentialSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("credential set requires <credential-id>")
	}
	update := credentials.UpdateOpts{
		Type: flagValue(opts, "type"),
		Blob: flagValue(opts, "data"),
	}
	if flagChanged(opts, "user") {
		user, err := findUserWithDomain(ctx, client, flagValue(opts, "user"), flagValue(opts, "user-domain"))
		if err != nil {
			return err
		}
		update.UserID = user.ID
	}
	if flagChanged(opts, "project") {
		projectID, err := optionalProjectID(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		update.ProjectID = projectID
	}
	_, err := credentials.Update(ctx, client, args[0], update).Extract()
	return err
}

func identityDomainCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("domain create requires <domain-name>")
	}
	createOpts := domains.CreateOpts{Name: args[0], Description: flagValue(opts, "description")}
	if enabled, ok := identityEnabledValue(opts); ok {
		createOpts.Enabled = &enabled
	}
	item, err := domains.Create(ctx, client, createOpts).Extract()
	if err != nil {
		if boolFlag(opts, "or-show") {
			item, err = findDomain(ctx, client, args[0])
		}
		if err != nil {
			return err
		}
	}
	return renderDomain(stdout, opts, item)
}

func identityDomainDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("domain delete requires <domain>")
	}
	for _, value := range args {
		item, err := findDomain(ctx, client, value)
		if err != nil {
			return err
		}
		if err := domains.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityDomainSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("domain set requires <domain>")
	}
	item, err := findDomain(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := domains.UpdateOpts{Name: flagValue(opts, "name")}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		update.Enabled = &enabled
	}
	_, err = domains.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityEC2CredentialCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	projectID, err := optionalProjectID(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	if projectID == "" {
		return fmt.Errorf("ec2 credentials create requires --project <project>")
	}
	item, err := ec2credentials.Create(ctx, client, userID, ec2credentials.CreateOpts{TenantID: projectID}).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"access", item.Access},
		{"project_id", item.TenantID},
		{"secret", item.Secret},
		{"trust_id", nilIfEmpty(item.TrustID)},
		{"user_id", item.UserID},
	})
}

func identityEC2CredentialDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ec2 credentials delete requires <access-key>")
	}
	userID, err := identityUserIDForScopedCommand(ctx, client, opts)
	if err != nil {
		return err
	}
	for _, access := range args {
		if err := ec2credentials.Delete(ctx, client, userID, access).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityEndpointAddProject(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("endpoint add project requires <endpoint> <project>")
	}
	endpoint, err := findEndpoint(ctx, client, args[0])
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, client, args[1], flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	return projectendpoints.Create(ctx, client, project.ID, endpoint.ID).ExtractErr()
}

func identityEndpointRemoveProject(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("endpoint remove project requires <endpoint> <project>")
	}
	endpoint, err := findEndpoint(ctx, client, args[0])
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, client, args[1], flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	return projectendpoints.Delete(ctx, client, project.ID, endpoint.ID).ExtractErr()
}

func identityEndpointCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("endpoint create requires <service> <interface> <url>")
	}
	service, err := findService(ctx, client, args[0])
	if err != nil {
		return err
	}
	createOpts := endpoints.CreateOpts{
		ServiceID:    service.ID,
		Availability: availabilityFromInterface(args[1]),
		URL:          args[2],
		Region:       flagValue(opts, "region"),
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		createOpts.Enabled = &enabled
	}
	item, err := endpoints.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderEndpoint(stdout, opts, item, service)
}

func identityEndpointDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("endpoint delete requires <endpoint-id>")
	}
	for _, value := range args {
		item, err := findEndpoint(ctx, client, value)
		if err != nil {
			return err
		}
		if err := endpoints.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityEndpointSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("endpoint set requires <endpoint>")
	}
	item, err := findEndpoint(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := endpoints.UpdateOpts{
		URL:    flagValue(opts, "url"),
		Region: flagValue(opts, "region"),
	}
	if iface := flagValue(opts, "interface"); iface != "" {
		update.Availability = availabilityFromInterface(iface)
	}
	if serviceValue := flagValue(opts, "service"); serviceValue != "" {
		service, err := findService(ctx, client, serviceValue)
		if err != nil {
			return err
		}
		update.ServiceID = service.ID
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		update.Enabled = &enabled
	}
	_, err = endpoints.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityEndpointGroupSpec() identityRawResource {
	return identityRawResource{
		Singular: "endpoint_group",
		Plural:   "endpoint_groups",
		Path:     []string{"OS-EP-FILTER", "endpoint_groups"},
		Columns:  []string{"ID", "Name", "Description"},
		Keys:     []string{"id", "name", "description"},
	}
}

func identityEndpointGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("endpoint group create requires <name> <filters-file>")
	}
	filters, err := readJSONMapFile(args[1])
	if err != nil {
		return err
	}
	values := map[string]any{"name": args[0], "filters": filters}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	item, err := identityRawCreate(ctx, client, identityEndpointGroupSpec(), values)
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, item)
}

func identityEndpointGroupDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	return identityRawDelete(ctx, client, identityEndpointGroupSpec(), args)
}

func identityEndpointGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	if value := flagValue(opts, "endpointgroup"); value != "" {
		group, err := identityRawFind(ctx, client, identityEndpointGroupSpec(), value)
		if err != nil {
			return err
		}
		items, err := identityRawListAt(ctx, client, []string{"OS-EP-FILTER", "endpoint_groups", valueString(group["id"]), "projects"}, "projects")
		if err != nil {
			return err
		}
		return renderIdentityRawList(stdout, opts, []string{"ID", "Name", "Description"}, []string{"id", "name", "description"}, items)
	}
	if projectValue := flagValue(opts, "project"); projectValue != "" {
		project, err := findProjectWithDomain(ctx, client, projectValue, flagValue(opts, "domain"))
		if err != nil {
			return err
		}
		items, err := identityRawListAt(ctx, client, []string{"OS-EP-FILTER", "projects", project.ID, "endpoint_groups"}, "endpoint_groups")
		if err != nil {
			return err
		}
		return renderIdentityRawList(stdout, opts, []string{"ID", "Name", "Description"}, []string{"id", "name", "description"}, items)
	}
	return identityRawList(ctx, stdout, opts, client, identityEndpointGroupSpec())
}

func identityEndpointGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	return identityRawShow(ctx, stdout, opts, client, identityEndpointGroupSpec(), args)
}

func identityEndpointGroupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("endpoint group set requires <endpoint-group>")
	}
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if filtersFile := flagValue(opts, "filters"); filtersFile != "" {
		filters, err := readJSONMapFile(filtersFile)
		if err != nil {
			return err
		}
		values["filters"] = filters
	}
	return identityRawSet(ctx, client, identityEndpointGroupSpec(), args, values)
}

func identityEndpointGroupProjectAssociation(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string, add bool) error {
	if len(args) < 2 {
		return fmt.Errorf("endpoint group project association requires <endpoint-group> <project>")
	}
	group, err := identityRawFind(ctx, client, identityEndpointGroupSpec(), args[0])
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, client, args[1], flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	url := client.ServiceURL("OS-EP-FILTER", "endpoint_groups", valueString(group["id"]), "projects", project.ID)
	var resp *http.Response
	if add {
		resp, err = client.Put(ctx, url, nil, nil, &gophercloud.RequestOpts{OkCodes: []int{204}})
	} else {
		resp, err = client.Delete(ctx, url, &gophercloud.RequestOpts{OkCodes: []int{204}})
	}
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func identityFederationDomainList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := identityRawListAt(ctx, client, []string{"OS-FEDERATION", "domains"}, "domains")
	if err != nil {
		return err
	}
	return renderIdentityRawList(stdout, opts, []string{"ID", "Name", "Enabled", "Description"}, []string{"id", "name", "enabled", "description"}, items)
}

func identityFederationProjectList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := identityRawListAt(ctx, client, []string{"OS-FEDERATION", "projects"}, "projects")
	if err != nil {
		return err
	}
	return renderIdentityRawList(stdout, opts, []string{"ID", "Domain ID", "Enabled", "Name"}, []string{"id", "domain_id", "enabled", "name"}, items)
}

func identityFederationProtocolCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("federation protocol create requires <federation-protocol>")
	}
	idpID := flagValue(opts, "identity-provider")
	if idpID == "" {
		return fmt.Errorf("federation protocol create requires --identity-provider <identity-provider>")
	}
	mappingID := flagValue(opts, "mapping")
	if mappingID == "" {
		return fmt.Errorf("federation protocol create requires --mapping <mapping>")
	}
	var body struct {
		Protocol map[string]any `json:"protocol"`
	}
	values := map[string]any{"mapping_id": mappingID}
	resp, err := client.Put(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", idpID, "protocols", args[0]), map[string]any{"protocol": values}, &body, &gophercloud.RequestOpts{OkCodes: []int{201, 200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, body.Protocol)
}

func identityFederationProtocolDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	idpID := flagValue(opts, "identity-provider")
	if idpID == "" {
		return fmt.Errorf("federation protocol delete requires --identity-provider <identity-provider>")
	}
	if len(args) < 1 {
		return fmt.Errorf("federation protocol delete requires <federation-protocol>")
	}
	for _, value := range args {
		resp, err := client.Delete(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", idpID, "protocols", value), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func identityFederationProtocolSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	idpID := flagValue(opts, "identity-provider")
	if idpID == "" {
		return fmt.Errorf("federation protocol set requires --identity-provider <identity-provider>")
	}
	if len(args) < 1 {
		return fmt.Errorf("federation protocol set requires <federation-protocol>")
	}
	mappingID := flagValue(opts, "mapping")
	if mappingID == "" {
		return nil
	}
	resp, err := client.Patch(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", idpID, "protocols", args[0]), map[string]any{"protocol": map[string]any{"mapping_id": mappingID}}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func identityGroupAddUser(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("group add user requires <group> <user>")
	}
	group, err := findGroupWithDomain(ctx, client, args[0], flagValue(opts, "group-domain"))
	if err != nil {
		return err
	}
	for _, userValue := range args[1:] {
		user, err := findUserWithDomain(ctx, client, userValue, flagValue(opts, "user-domain"))
		if err != nil {
			return err
		}
		if err := users.AddToGroup(ctx, client, group.ID, user.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityGroupContainsUser(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("group contains user requires <group> <user>")
	}
	group, err := findGroupWithDomain(ctx, client, args[0], flagValue(opts, "group-domain"))
	if err != nil {
		return err
	}
	user, err := findUserWithDomain(ctx, client, args[1], flagValue(opts, "user-domain"))
	if err != nil {
		return err
	}
	result := users.IsMemberOfGroup(ctx, client, group.ID, user.ID)
	if result.Err != nil {
		return result.Err
	}
	member, err := result.Extract()
	if err != nil {
		return err
	}
	if member {
		_, err = fmt.Fprintf(stdout, "%s in group %s\n", args[1], args[0])
	} else {
		_, err = fmt.Fprintf(stdout, "%s not in group %s\n", args[1], args[0])
	}
	return err
}

func identityGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("group create requires <group-name>")
	}
	domainID, err := optionalDomainID(ctx, client, flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	item, err := groups.Create(ctx, client, groups.CreateOpts{Name: args[0], DomainID: domainID, Description: flagValue(opts, "description")}).Extract()
	if err != nil {
		if boolFlag(opts, "or-show") {
			item, err = findGroupWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
		}
		if err != nil {
			return err
		}
	}
	return renderGroup(stdout, opts, item)
}

func identityGroupDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("group delete requires <group>")
	}
	for _, value := range args {
		item, err := findGroupWithDomain(ctx, client, value, flagValue(opts, "group-domain"))
		if err != nil {
			return err
		}
		if err := groups.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityGroupRemoveUser(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("group remove user requires <group> <user>")
	}
	group, err := findGroupWithDomain(ctx, client, args[0], flagValue(opts, "group-domain"))
	if err != nil {
		return err
	}
	for _, userValue := range args[1:] {
		user, err := findUserWithDomain(ctx, client, userValue, flagValue(opts, "user-domain"))
		if err != nil {
			return err
		}
		if err := users.RemoveFromGroup(ctx, client, group.ID, user.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityGroupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("group set requires <group>")
	}
	item, err := findGroupWithDomain(ctx, client, args[0], flagValue(opts, "group-domain"))
	if err != nil {
		return err
	}
	update := groups.UpdateOpts{Name: flagValue(opts, "name")}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	_, err = groups.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityProviderCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("identity provider create requires <name>")
	}
	values := map[string]any{
		"description": flagValue(opts, "description"),
		"remote_ids":  identityRemoteIDs(opts),
	}
	if domainID, err := optionalDomainID(ctx, client, flagValue(opts, "domain")); err != nil {
		return err
	} else if domainID != "" {
		values["domain_id"] = domainID
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		values["enabled"] = enabled
	} else {
		values["enabled"] = true
	}
	if flagChanged(opts, "authorization-ttl") {
		values["authorization_ttl"] = intFlag(opts, "authorization-ttl")
	}
	var body struct {
		IdentityProvider map[string]any `json:"identity_provider"`
	}
	resp, err := client.Put(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", args[0]), map[string]any{"identity_provider": values}, &body, &gophercloud.RequestOpts{OkCodes: []int{201, 200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, body.IdentityProvider)
}

func identityProviderDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("identity provider delete requires <identity-provider>")
	}
	for _, value := range args {
		resp, err := client.Delete(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", value), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func identityProviderSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("identity provider set requires <identity-provider>")
	}
	values := map[string]any{}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if ids := identityRemoteIDs(opts); ids != nil {
		values["remote_ids"] = ids
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		values["enabled"] = enabled
	}
	if flagChanged(opts, "authorization-ttl") {
		values["authorization_ttl"] = intFlag(opts, "authorization-ttl")
	}
	resp, err := client.Patch(ctx, client.ServiceURL("OS-FEDERATION", "identity_providers", args[0]), map[string]any{"identity_provider": values}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func identityImpliedRoleCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("implied role create requires <role> <implied-role>")
	}
	prior, implied, err := findImpliedRoles(ctx, client, opts, args[0], args[1])
	if err != nil {
		return err
	}
	result, err := roles.CreateRoleInferenceRule(ctx, client, prior.ID, implied.ID).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"implied", result.RoleInference.ImpliedRole},
		{"prior", result.RoleInference.PriorRole},
	})
}

func identityImpliedRoleDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("implied role delete requires <role> <implied-role>")
	}
	prior, implied, err := findImpliedRoles(ctx, client, opts, args[0], args[1])
	if err != nil {
		return err
	}
	return roles.DeleteRoleInferenceRule(ctx, client, prior.ID, implied.ID).ExtractErr()
}

func identityLimitCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("limit create requires <resource-name>")
	}
	service, err := findService(ctx, client, flagValue(opts, "service"))
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	result, err := identitylimits.BatchCreate(ctx, client, identitylimits.BatchCreateOpts{{
		ProjectID:     project.ID,
		ServiceID:     service.ID,
		RegionID:      flagValue(opts, "region"),
		ResourceName:  args[0],
		ResourceLimit: intFlag(opts, "resource-limit"),
		Description:   flagValue(opts, "description"),
	}}).Extract()
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return nil
	}
	return renderLimit(stdout, opts, &result[0])
}

func identityLimitDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("limit delete requires <limit>")
	}
	for _, value := range args {
		if err := identitylimits.Delete(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityLimitSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("limit set requires <limit>")
	}
	update := identitylimits.UpdateOpts{}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if flagChanged(opts, "resource-limit") {
		limit := intFlag(opts, "resource-limit")
		update.ResourceLimit = &limit
	}
	_, err := identitylimits.Update(ctx, client, args[0], update).Extract()
	return err
}

func identityMappingCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mapping create requires <mapping>")
	}
	rules, err := readMappingRules(flagValue(opts, "rules"))
	if err != nil {
		return err
	}
	item, err := federation.CreateMapping(ctx, client, args[0], federation.CreateMappingOpts{Rules: rules}).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{{"id", item.ID}, {"rules", item.Rules}})
}

func identityMappingDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mapping delete requires <mapping>")
	}
	for _, value := range args {
		if err := federation.DeleteMapping(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityMappingSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mapping set requires <mapping>")
	}
	if flagValue(opts, "rules") == "" {
		return nil
	}
	rules, err := readMappingRules(flagValue(opts, "rules"))
	if err != nil {
		return err
	}
	_, err = federation.UpdateMapping(ctx, client, args[0], federation.UpdateMappingOpts{Rules: rules}).Extract()
	return err
}

func identityPolicyCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("policy create requires <filename> <type>")
	}
	blob, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	item, err := policies.Create(ctx, client, policies.CreateOpts{Blob: blob, Type: args[1]}).Extract()
	if err != nil {
		return err
	}
	return renderPolicy(stdout, opts, item)
}

func identityPolicyDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("policy delete requires <policy>")
	}
	for _, value := range args {
		if err := policies.Delete(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityPolicySet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("policy set requires <policy>")
	}
	update := policies.UpdateOpts{Type: flagValue(opts, "type")}
	if file := flagValue(opts, "rules"); file != "" {
		blob, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		update.Blob = blob
	}
	_, err := policies.Update(ctx, client, args[0], update).Extract()
	return err
}

func identityProjectCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project create requires <project-name>")
	}
	domainID, err := optionalDomainID(ctx, client, flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	parentID := ""
	if parent := flagValue(opts, "parent"); parent != "" {
		project, err := findProjectWithDomain(ctx, client, parent, flagValue(opts, "domain"))
		if err != nil {
			return err
		}
		parentID = project.ID
	}
	createOpts := projects.CreateOpts{
		Name:        args[0],
		DomainID:    domainID,
		ParentID:    parentID,
		Description: flagValue(opts, "description"),
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		createOpts.Enabled = &enabled
	}
	if tags := flagValues(opts, "tag"); len(tags) > 0 {
		createOpts.Tags = tags
	}
	item, err := projects.Create(ctx, client, createOpts).Extract()
	if err != nil {
		if boolFlag(opts, "or-show") {
			item, err = findProjectWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
		}
		if err != nil {
			return err
		}
	}
	return renderProject(stdout, opts, item)
}

func identityProjectDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project delete requires <project>")
	}
	for _, value := range args {
		item, err := findProjectWithDomain(ctx, client, value, flagValue(opts, "domain"))
		if err != nil {
			return err
		}
		if err := projects.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityProjectSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project set requires <project>")
	}
	item, err := findProjectWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	update := projects.UpdateOpts{Name: flagValue(opts, "name")}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		update.Enabled = &enabled
	}
	if flagChanged(opts, "tag") {
		tags := flagValues(opts, "tag")
		update.Tags = &tags
	}
	_, err = projects.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityRegionCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("region create requires <region-id>")
	}
	item, err := regions.Create(ctx, client, regions.CreateOpts{
		ID:             args[0],
		Description:    flagValue(opts, "description"),
		ParentRegionID: flagValue(opts, "parent-region"),
	}).Extract()
	if err != nil {
		return err
	}
	return renderRegion(stdout, opts, item)
}

func identityRegionDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("region delete requires <region>")
	}
	for _, value := range args {
		if err := regions.Delete(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityRegionSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("region set requires <region>")
	}
	update := regions.UpdateOpts{ParentRegionID: flagValue(opts, "parent-region")}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	_, err := regions.Update(ctx, client, args[0], update).Extract()
	return err
}

func identityRegisteredLimitCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("registered limit create requires <resource-name>")
	}
	service, err := findService(ctx, client, flagValue(opts, "service"))
	if err != nil {
		return err
	}
	result, err := registeredlimits.BatchCreate(ctx, client, registeredlimits.BatchCreateOpts{{
		ServiceID:    service.ID,
		RegionID:     flagValue(opts, "region"),
		ResourceName: args[0],
		DefaultLimit: intFlag(opts, "default-limit"),
		Description:  flagValue(opts, "description"),
	}}).Extract()
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return nil
	}
	return renderRegisteredLimit(stdout, opts, &result[0])
}

func identityRegisteredLimitDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("registered limit delete requires <registered-limit>")
	}
	for _, value := range args {
		if err := registeredlimits.Delete(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityRegisteredLimitSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("registered limit set requires <registered-limit>")
	}
	update := registeredlimits.UpdateOpts{
		RegionID:     flagValue(opts, "region"),
		ResourceName: flagValue(opts, "resource-name"),
	}
	if serviceValue := flagValue(opts, "service"); serviceValue != "" {
		service, err := findService(ctx, client, serviceValue)
		if err != nil {
			return err
		}
		update.ServiceID = service.ID
	}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if flagChanged(opts, "default-limit") {
		limit := intFlag(opts, "default-limit")
		update.DefaultLimit = &limit
	}
	_, err := registeredlimits.Update(ctx, client, args[0], update).Extract()
	return err
}

func identityRoleCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("role create requires <role-name>")
	}
	domainID, err := optionalDomainID(ctx, client, flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	createOpts := roles.CreateOpts{Name: args[0], DomainID: domainID, Description: flagValue(opts, "description")}
	if flagChanged(opts, "immutable") {
		createOpts.Options = map[roles.Option]any{roles.Immutable: boolFlag(opts, "immutable")}
	}
	item, err := roles.Create(ctx, client, createOpts).Extract()
	if err != nil {
		if boolFlag(opts, "or-show") {
			item, err = findRoleWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
		}
		if err != nil {
			return err
		}
	}
	return renderRole(stdout, opts, item)
}

func identityRoleDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("role delete requires <role>")
	}
	for _, value := range args {
		item, err := findRoleWithDomain(ctx, client, value, flagValue(opts, "role-domain"))
		if err != nil {
			return err
		}
		if err := roles.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityRoleSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("role set requires <role>")
	}
	item, err := findRoleWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	update := roles.UpdateOpts{Name: flagValue(opts, "name")}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if flagChanged(opts, "immutable") {
		update.Options = map[roles.Option]any{roles.Immutable: boolFlag(opts, "immutable")}
	}
	_, err = roles.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityRoleAssignment(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string, add bool) error {
	if len(args) < 1 {
		return fmt.Errorf("role assignment requires <role>")
	}
	role, err := findRoleWithDomain(ctx, client, args[0], flagValue(opts, "role-domain"))
	if err != nil {
		return err
	}
	assignment, err := identityRoleAssignmentOpts(ctx, opts, client)
	if err != nil {
		return err
	}
	if boolFlag(opts, "inherited") {
		if add {
			return osinherit.Assign(ctx, client, role.ID, osinherit.AssignOpts{
				UserID: assignment.UserID, GroupID: assignment.GroupID, ProjectID: assignment.ProjectID, DomainID: assignment.DomainID,
			}).ExtractErr()
		}
		return osinherit.Unassign(ctx, client, role.ID, osinherit.UnassignOpts{
			UserID: assignment.UserID, GroupID: assignment.GroupID, ProjectID: assignment.ProjectID, DomainID: assignment.DomainID,
		}).ExtractErr()
	}
	if assignment.System != "" {
		return identitySystemRoleAssignment(ctx, client, role.ID, assignment, add)
	}
	if add {
		return roles.Assign(ctx, client, role.ID, roles.AssignOpts{
			UserID: assignment.UserID, GroupID: assignment.GroupID, ProjectID: assignment.ProjectID, DomainID: assignment.DomainID,
		}).ExtractErr()
	}
	return roles.Unassign(ctx, client, role.ID, roles.UnassignOpts{
		UserID: assignment.UserID, GroupID: assignment.GroupID, ProjectID: assignment.ProjectID, DomainID: assignment.DomainID,
	}).ExtractErr()
}

func identityServiceCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service create requires <type>")
	}
	createOpts := services.CreateOpts{
		Type:        args[0],
		Name:        flagValue(opts, "name"),
		Description: flagValue(opts, "description"),
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		createOpts.Enabled = &enabled
	}
	item, err := services.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderService(stdout, opts, item)
}

func identityServiceDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service delete requires <service>")
	}
	for _, value := range args {
		item, err := findService(ctx, client, value)
		if err != nil {
			return err
		}
		if err := services.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityServiceSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service set requires <service>")
	}
	item, err := findService(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := services.UpdateOpts{Type: flagValue(opts, "type")}
	if flagChanged(opts, "name") {
		name := flagValue(opts, "name")
		update.Name = &name
	}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		update.Enabled = &enabled
	}
	_, err = services.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityServiceProviderCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service provider create requires <name>")
	}
	values := map[string]any{
		"auth_url":    flagValue(opts, "auth-url"),
		"description": flagValue(opts, "description"),
		"sp_url":      flagValue(opts, "service-provider-url"),
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		values["enabled"] = enabled
	} else {
		values["enabled"] = true
	}
	var body struct {
		ServiceProvider map[string]any `json:"service_provider"`
	}
	resp, err := client.Put(ctx, client.ServiceURL("OS-FEDERATION", "service_providers", args[0]), map[string]any{"service_provider": values}, &body, &gophercloud.RequestOpts{OkCodes: []int{201, 200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, body.ServiceProvider)
}

func identityServiceProviderDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service provider delete requires <service-provider>")
	}
	for _, value := range args {
		resp, err := client.Delete(ctx, client.ServiceURL("OS-FEDERATION", "service_providers", value), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func identityServiceProviderSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service provider set requires <service-provider>")
	}
	values := map[string]any{}
	for flag, key := range map[string]string{"auth-url": "auth_url", "description": "description", "service-provider-url": "sp_url"} {
		if flagChanged(opts, flag) {
			values[key] = flagValue(opts, flag)
		}
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		values["enabled"] = enabled
	}
	resp, err := client.Patch(ctx, client.ServiceURL("OS-FEDERATION", "service_providers", args[0]), map[string]any{"service_provider": values}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func identityTokenRevoke(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("token revoke requires <token>")
	}
	_, err := tokens.Revoke(ctx, client, args[0]).Extract()
	return err
}

func identityTrustCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("trust create requires <trustor-user> <trustee-user>")
	}
	trustor, err := findUserWithDomain(ctx, client, args[0], flagValue(opts, "trustor-domain"))
	if err != nil {
		return err
	}
	trustee, err := findUserWithDomain(ctx, client, args[1], flagValue(opts, "trustee-domain"))
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	createOpts := trusts.CreateOpts{
		TrustorUserID: trustor.ID,
		TrusteeUserID: trustee.ID,
		ProjectID:     project.ID,
		Impersonation: boolFlag(opts, "impersonate"),
	}
	for _, roleValue := range flagValues(opts, "role") {
		role, err := findRole(ctx, client, roleValue)
		if err != nil {
			return err
		}
		createOpts.Roles = append(createOpts.Roles, trusts.Role{ID: role.ID})
	}
	if expires := flagValue(opts, "expiration"); expires != "" {
		parsed, err := parseIdentityTime(expires)
		if err != nil {
			return err
		}
		createOpts.ExpiresAt = &parsed
	}
	item, err := trusts.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderTrust(stdout, opts, item)
}

func identityTrustDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("trust delete requires <trust>")
	}
	for _, value := range args {
		if err := trusts.Delete(ctx, client, value).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityUserCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("user create requires <user-name>")
	}
	domainID, err := optionalDomainID(ctx, client, flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	projectID, err := optionalProjectID(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	createOpts := users.CreateOpts{
		Name:             args[0],
		DomainID:         domainID,
		DefaultProjectID: projectID,
		Description:      flagValue(opts, "description"),
		Password:         flagValue(opts, "password"),
	}
	if email := flagValue(opts, "email"); email != "" {
		createOpts.Extra = map[string]any{"email": email}
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		createOpts.Enabled = &enabled
	}
	item, err := users.Create(ctx, client, createOpts).Extract()
	if err != nil {
		if boolFlag(opts, "or-show") {
			item, err = findUserWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
		}
		if err != nil {
			return err
		}
	}
	return renderUser(stdout, opts, item)
}

func identityUserDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("user delete requires <user>")
	}
	for _, value := range args {
		item, err := findUserWithDomain(ctx, client, value, flagValue(opts, "domain"))
		if err != nil {
			return err
		}
		if err := users.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func identityUserPasswordSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) error {
	userID, err := currentTokenUserID(ctx, client)
	if err != nil {
		return err
	}
	if !flagChanged(opts, "password") || !flagChanged(opts, "original-password") {
		return fmt.Errorf("user password set requires --password and --original-password")
	}
	return users.ChangePassword(ctx, client, userID, users.ChangePasswordOpts{
		OriginalPassword: flagValue(opts, "original-password"),
		Password:         flagValue(opts, "password"),
	}).ExtractErr()
}

func identityUserSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("user set requires <user>")
	}
	item, err := findUserWithDomain(ctx, client, args[0], flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	projectID, err := optionalProjectID(ctx, client, flagValue(opts, "project"), flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	update := users.UpdateOpts{
		Name:             flagValue(opts, "name"),
		DefaultProjectID: projectID,
		Password:         flagValue(opts, "password"),
	}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		update.Description = &description
	}
	if flagChanged(opts, "email") {
		update.Extra = map[string]any{"email": flagValue(opts, "email")}
	}
	if enabled, ok := identityEnabledValue(opts); ok {
		update.Enabled = &enabled
	}
	_, err = users.Update(ctx, client, item.ID, update).Extract()
	return err
}

func identityRequestTokenAuthorize(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	requestKey := flagValue(opts, "request-key")
	if requestKey == "" {
		return fmt.Errorf("request token authorize requires --request-key <request-key>")
	}
	authOpts := oauth1.AuthorizeTokenOpts{}
	for _, roleValue := range flagValues(opts, "role") {
		role, err := findRole(ctx, client, roleValue)
		if err != nil {
			return err
		}
		authOpts.Roles = append(authOpts.Roles, oauth1.Role{ID: role.ID})
	}
	result, err := oauth1.AuthorizeToken(ctx, client, requestKey, authOpts).Extract()
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, map[string]any{"oauth_verifier": result.OAuthVerifier})
}

func identityRequestTokenCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	project, err := findProjectWithDomain(ctx, client, flagValue(opts, "project"), flagValue(opts, "domain"))
	if err != nil {
		return err
	}
	result := oauth1.RequestToken(ctx, client, oauth1.RequestTokenOpts{
		OAuthConsumerKey:     flagValue(opts, "consumer-key"),
		OAuthConsumerSecret:  flagValue(opts, "consumer-secret"),
		OAuthSignatureMethod: oauth1.HMACSHA1,
		RequestedProjectID:   project.ID,
	})
	if result.Err != nil {
		return result.Err
	}
	return renderOAuthTokenBody(stdout, opts, result.Body)
}

func identityAccessTokenCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	result := oauth1.CreateAccessToken(ctx, client, oauth1.CreateAccessTokenOpts{
		OAuthConsumerKey:     flagValue(opts, "consumer-key"),
		OAuthConsumerSecret:  flagValue(opts, "consumer-secret"),
		OAuthToken:           flagValue(opts, "request-key"),
		OAuthTokenSecret:     flagValue(opts, "request-secret"),
		OAuthVerifier:        flagValue(opts, "verifier"),
		OAuthSignatureMethod: oauth1.HMACSHA1,
	})
	if result.Err != nil {
		return result.Err
	}
	return renderOAuthTokenBody(stdout, opts, result.Body)
}

func findConsumer(ctx context.Context, client *gophercloud.ServiceClient, value string) (*oauth1.Consumer, error) {
	result := oauth1.GetConsumer(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := oauth1.ListConsumers(client).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := oauth1.ExtractConsumers(page)
	if err != nil {
		return nil, err
	}
	var matches []oauth1.Consumer
	for _, item := range items {
		if item.ID == value || item.Description == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func optionalDomainID(ctx context.Context, client *gophercloud.ServiceClient, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	item, err := findDomain(ctx, client, value)
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

func optionalProjectID(ctx context.Context, client *gophercloud.ServiceClient, value string, domainValue string) (string, error) {
	if value == "" {
		return "", nil
	}
	item, err := findProjectWithDomain(ctx, client, value, domainValue)
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

func identityEnabledValue(opts *Options) (bool, bool) {
	if flagChanged(opts, "enable") {
		return true, true
	}
	if flagChanged(opts, "disable") {
		return false, true
	}
	if flagChanged(opts, "enabled") {
		return boolFlag(opts, "enabled"), true
	}
	return false, false
}

func parseIdentityTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

func renderCredential(stdout io.Writer, opts *Options, item *credentials.Credential) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"blob", item.Blob},
		{"id", item.ID},
		{"links", item.Links},
		{"project_id", nilIfEmpty(item.ProjectID)},
		{"type", item.Type},
		{"user_id", item.UserID},
	})
}

func renderDomain(stdout io.Writer, opts *Options, item *domains.Domain) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"name", item.Name},
		{"options", map[string]any{}},
	})
}

func renderEndpoint(stdout io.Writer, opts *Options, item *endpoints.Endpoint, service *services.Service) error {
	serviceName, serviceType := "", ""
	if service != nil {
		serviceName = service.Name
		serviceType = service.Type
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"interface", string(item.Availability)},
		{"region", item.Region},
		{"region_id", item.Region},
		{"service_id", item.ServiceID},
		{"service_name", serviceName},
		{"service_type", serviceType},
		{"url", item.URL},
	})
}

func renderGroup(stdout io.Writer, opts *Options, item *groups.Group) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", item.DomainID},
		{"id", item.ID},
		{"name", item.Name},
	})
}

func renderLimit(stdout io.Writer, opts *Options, item *identitylimits.Limit) error {
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

func renderPolicy(stdout io.Writer, opts *Options, item *policies.Policy) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"blob", item.Blob},
		{"extra", item.Extra},
		{"id", item.ID},
		{"links", item.Links},
		{"type", item.Type},
	})
}

func renderProject(stdout io.Writer, opts *Options, item *projects.Project) error {
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

func renderRegion(stdout io.Writer, opts *Options, item *regions.Region) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"parent_region", nilIfEmpty(item.ParentRegionID)},
		{"region", item.ID},
	})
}

func renderRegisteredLimit(stdout io.Writer, opts *Options, item *registeredlimits.RegisteredLimit) error {
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

func renderRole(stdout io.Writer, opts *Options, item *roles.Role) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"domain_id", nilIfEmpty(item.DomainID)},
		{"id", item.ID},
		{"name", item.Name},
	})
}

func renderService(stdout io.Writer, opts *Options, item *services.Service) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"description", item.Description},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"name", item.Name},
		{"type", item.Type},
	})
}

func renderTrust(stdout io.Writer, opts *Options, item *trusts.Trust) error {
	return renderShowOutput(stdout, opts, []outputField{
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

func renderUser(stdout io.Writer, opts *Options, item *users.User) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"default_project_id", nilIfEmpty(item.DefaultProjectID)},
		{"description", nilIfEmpty(item.Description)},
		{"domain_id", item.DomainID},
		{"email", item.Extra["email"]},
		{"enabled", item.Enabled},
		{"id", item.ID},
		{"name", item.Name},
		{"options", item.Options},
		{"password_expires_at", nil},
	})
}

func identityRawList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, spec identityRawResource) error {
	items, err := identityRawListAt(ctx, client, spec.Path, spec.Plural)
	if err != nil {
		return err
	}
	return renderIdentityRawList(stdout, opts, spec.Columns, spec.Keys, items)
}

func identityRawShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, spec identityRawResource, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s show requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), spec.Singular)
	}
	item, err := identityRawFind(ctx, client, spec, args[0])
	if err != nil {
		return err
	}
	return renderIdentityRawShow(stdout, opts, item)
}

func identityRawCreate(ctx context.Context, client *gophercloud.ServiceClient, spec identityRawResource, values map[string]any) (map[string]any, error) {
	var body map[string]any
	resp, err := client.Post(ctx, client.ServiceURL(spec.Path...), map[string]any{spec.Singular: values}, &body, &gophercloud.RequestOpts{OkCodes: []int{201, 200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return mapValue(body[spec.Singular]), nil
}

func identityRawDelete(ctx context.Context, client *gophercloud.ServiceClient, spec identityRawResource, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s delete requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), spec.Singular)
	}
	for _, value := range args {
		item, err := identityRawFind(ctx, client, spec, value)
		if err != nil {
			return err
		}
		resp, err := client.Delete(ctx, client.ServiceURL(append(spec.Path, valueString(item["id"]))...), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func identityRawSet(ctx context.Context, client *gophercloud.ServiceClient, spec identityRawResource, args []string, values map[string]any) error {
	if len(args) < 1 {
		return fmt.Errorf("%s set requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), spec.Singular)
	}
	if len(values) == 0 {
		return nil
	}
	item, err := identityRawFind(ctx, client, spec, args[0])
	if err != nil {
		return err
	}
	resp, err := client.Patch(ctx, client.ServiceURL(append(spec.Path, valueString(item["id"]))...), map[string]any{spec.Singular: values}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func identityRawFind(ctx context.Context, client *gophercloud.ServiceClient, spec identityRawResource, value string) (map[string]any, error) {
	var body map[string]any
	resp, err := client.Get(ctx, client.ServiceURL(append(spec.Path, value)...), &body, nil)
	_, _, parsedErr := gophercloud.ParseResponse(resp, err)
	if parsedErr == nil {
		if item := mapValue(body[spec.Singular]); len(item) > 0 {
			return item, nil
		}
	}
	items, err := identityRawListAt(ctx, client, spec.Path, spec.Plural)
	if err != nil {
		return nil, parsedErr
	}
	matches := make([]map[string]any, 0)
	for _, item := range items {
		if valueString(item["id"]) == value || valueString(item["name"]) == value {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, newLookupNotFound("resource", value)
	case 1:
		return matches[0], nil
	default:
		return nil, newLookupAmbiguous("resources", value, len(matches))
	}
}

func identityRawListAt(ctx context.Context, client *gophercloud.ServiceClient, segments []string, plural string) ([]map[string]any, error) {
	var body map[string]any
	resp, err := client.Get(ctx, client.ServiceURL(segments...), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return mapSlice(body[plural]), nil
}

func renderIdentityRawList(stdout io.Writer, opts *Options, columns []string, keys []string, items []map[string]any) error {
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{}
		for i, column := range columns {
			if i < len(keys) {
				row[column] = item[keys[i]]
			}
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func renderIdentityRawShow(stdout io.Writer, opts *Options, item map[string]any) error {
	delete(item, "links")
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(item, false))
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func mapSlice(value any) []map[string]any {
	items := anySlice(value)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped := mapValue(item); mapped != nil {
			result = append(result, mapped)
		}
	}
	return result
}

func readJSONMapFile(path string) (map[string]any, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(blob, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func readMappingRules(path string) ([]federation.MappingRule, error) {
	if path == "" {
		return nil, fmt.Errorf("mapping requires --rules <filename>")
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var direct []federation.MappingRule
	if err := json.Unmarshal(blob, &direct); err == nil {
		return direct, nil
	}
	var wrapped struct {
		Rules []federation.MappingRule `json:"rules"`
	}
	if err := json.Unmarshal(blob, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Rules, nil
}

func identityRemoteIDs(opts *Options) []string {
	if file := flagValue(opts, "remote-id-file"); file != "" {
		blob, err := os.ReadFile(file)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(blob), "\n")
		values := make([]string, 0, len(lines))
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values
	}
	if flagChanged(opts, "remote-id") {
		return flagValues(opts, "remote-id")
	}
	return nil
}

func findImpliedRoles(ctx context.Context, client *gophercloud.ServiceClient, opts *Options, priorValue string, impliedValue string) (*roles.Role, *roles.Role, error) {
	prior, err := findRoleWithDomain(ctx, client, priorValue, flagValue(opts, "role-domain"))
	if err != nil {
		return nil, nil, err
	}
	implied, err := findRoleWithDomain(ctx, client, impliedValue, flagValue(opts, "implied-role-domain"))
	if err != nil {
		return nil, nil, err
	}
	return prior, implied, nil
}

type identityRoleAssignmentValues struct {
	UserID    string
	GroupID   string
	ProjectID string
	DomainID  string
	System    string
}

func identityRoleAssignmentOpts(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (identityRoleAssignmentValues, error) {
	var values identityRoleAssignmentValues
	if userValue := flagValue(opts, "user"); userValue != "" {
		user, err := findUserWithDomain(ctx, client, userValue, flagValue(opts, "user-domain"))
		if err != nil {
			return values, err
		}
		values.UserID = user.ID
	}
	if groupValue := flagValue(opts, "group"); groupValue != "" {
		group, err := findGroupWithDomain(ctx, client, groupValue, flagValue(opts, "group-domain"))
		if err != nil {
			return values, err
		}
		values.GroupID = group.ID
	}
	if projectValue := flagValue(opts, "project"); projectValue != "" {
		project, err := findProjectWithDomain(ctx, client, projectValue, flagValue(opts, "project-domain"))
		if err != nil {
			return values, err
		}
		values.ProjectID = project.ID
	}
	if domainValue := flagValue(opts, "domain"); domainValue != "" {
		domain, err := findDomain(ctx, client, domainValue)
		if err != nil {
			return values, err
		}
		values.DomainID = domain.ID
	}
	if systemValue := flagValue(opts, "system"); systemValue != "" {
		values.System = systemValue
	}
	if (values.UserID == "") == (values.GroupID == "") {
		return values, fmt.Errorf("role assignment requires exactly one of --user or --group")
	}
	scopeCount := 0
	for _, value := range []string{values.ProjectID, values.DomainID, values.System} {
		if value != "" {
			scopeCount++
		}
	}
	if scopeCount != 1 {
		return values, fmt.Errorf("role assignment requires exactly one of --project, --domain, or --system")
	}
	return values, nil
}

func identitySystemRoleAssignment(ctx context.Context, client *gophercloud.ServiceClient, roleID string, assignment identityRoleAssignmentValues, add bool) error {
	actorType, actorID := "users", assignment.UserID
	if assignment.GroupID != "" {
		actorType, actorID = "groups", assignment.GroupID
	}
	url := client.ServiceURL("system", actorType, actorID, "roles", roleID)
	var resp *http.Response
	var err error
	if add {
		resp, err = client.Put(ctx, url, nil, nil, &gophercloud.RequestOpts{OkCodes: []int{204}})
	} else {
		resp, err = client.Delete(ctx, url, &gophercloud.RequestOpts{OkCodes: []int{204}})
	}
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func renderOAuthTokenBody(stdout io.Writer, opts *Options, body []byte) error {
	values := map[string]any{}
	for _, part := range strings.Split(string(body), "&") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	return renderIdentityRawShow(stdout, opts, values)
}

func intFromFlag(opts *Options, name string) *int {
	if !flagChanged(opts, name) {
		return nil
	}
	value, err := strconv.Atoi(flagValue(opts, name))
	if err != nil {
		return nil
	}
	return &value
}
