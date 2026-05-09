package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/allocationcandidates"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits"
)

func allocationCandidateList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	resources := placementResourceQueryValues(opts)
	if resources == "" {
		return fmt.Errorf("At least one --resource must be specified.")
	}
	page, err := allocationcandidates.List(client, allocationcandidates.ListOpts{
		Resources:   resources,
		Required:    compactStrings([]string{placementRequiredQuery(opts)}),
		MemberOf:    placementMemberOfValues(opts),
		GroupPolicy: flagValue(opts, "group-policy"),
		Limit:       intFlag(opts, "limit"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	item, err := allocationcandidates.ExtractAllocationCandidates(page)
	if err != nil {
		return err
	}
	providerSummaries := placementProviderSummaries(item.ProviderSummaries)
	rows := make([]outputRow, 0, len(item.AllocationRequests))
	for i, request := range item.AllocationRequests {
		for provider, allocation := range request.Allocations {
			row := outputRow{
				"#":                       i + 1,
				"allocation":              placementResourceAmounts(allocation.Resources),
				"resource provider":       provider,
				"inventory used/capacity": providerSummaries[provider].Resources,
			}
			if microversionAtLeast(client.Microversion, "1.17") {
				row["traits"] = providerSummaries[provider].Traits
			}
			rows = append(rows, row)
		}
	}
	columns := []string{"#", "allocation", "resource provider", "inventory used/capacity"}
	if microversionAtLeast(client.Microversion, "1.17") {
		columns = append(columns, "traits")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

type placementProviderSummary struct {
	Resources string
	Traits    string
}

func placementProviderSummaries(values map[string]allocationcandidates.ProviderSummary) map[string]placementProviderSummary {
	result := map[string]placementProviderSummary{}
	for provider, summary := range values {
		resources := make([]string, 0, len(summary.Resources))
		for resourceClass, resource := range summary.Resources {
			resources = append(resources, fmt.Sprintf("%s=%d/%d", resourceClass, resource.Used, resource.Capacity))
		}
		sort.Strings(resources)
		traits := []string{}
		if summary.Traits != nil {
			traits = append(traits, (*summary.Traits)...)
		}
		sort.Strings(traits)
		result[provider] = placementProviderSummary{Resources: strings.Join(resources, ","), Traits: strings.Join(traits, ",")}
	}
	return result
}

func placementResourceAmounts(values map[string]int) string {
	parts := make([]string, 0, len(values))
	for resourceClass, amount := range values {
		parts = append(parts, fmt.Sprintf("%s=%d", resourceClass, amount))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func resourceClassList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := resourceclasses.List(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := resourceclasses.ExtractResourceClasses(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"name": item.Name})
	}
	return renderListOutput(stdout, opts, []string{"name"}, rows)
}

func resourceClassCreate(ctx context.Context, _ *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource class create requires <name>")
	}
	return resourceclasses.Create(ctx, client, resourceclasses.CreateOpts{Name: args[0]}).ExtractErr()
}

func resourceClassSet(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource class set requires <name>")
	}
	return resourceclasses.Update(ctx, client, args[0]).ExtractErr()
}

func resourceClassDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource class delete requires <name>")
	}
	failures := 0
	for _, name := range args {
		if err := resourceclasses.Delete(ctx, client, name).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d resource classes failed to delete.", failures, len(args))
	}
	return nil
}

func resourceClassShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource class show requires <name>")
	}
	item, err := resourceclasses.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"name", item.Name},
	})
}

func resourceProviderList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := resourceproviders.List(client, resourceproviders.ListOpts{
		UUID:      flagValue(opts, "uuid"),
		Name:      flagValue(opts, "name"),
		Resources: placementResourceQueryValues(opts),
		InTree:    flagValue(opts, "in-tree"),
		Required:  placementRequiredQuery(opts),
		MemberOf:  strings.Join(placementMemberOfValues(opts), ","),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := resourceproviders.ExtractResourceProviders(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	columns := resourceProviderColumns(client)
	for _, item := range items {
		rows = append(rows, resourceProviderRow(&item, columns))
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func resourceProviderCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider create requires <name>")
	}
	item, err := resourceproviders.Create(ctx, client, resourceproviders.CreateOpts{
		Name:               args[0],
		UUID:               flagValue(opts, "uuid"),
		ParentProviderUUID: flagValue(opts, "parent-provider"),
	}).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, resourceProviderFields(client, item))
}

func resourceProviderSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider set requires <uuid>")
	}
	name := flagValue(opts, "name")
	if name == "" {
		return fmt.Errorf("resource provider set requires --name")
	}
	update := resourceproviders.UpdateOpts{Name: &name}
	if parent := flagValue(opts, "parent-provider"); parent != "" {
		update.ParentProviderUUID = &parent
	}
	item, err := resourceproviders.Update(ctx, client, args[0], update).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, resourceProviderFields(client, item))
}

func resourceProviderDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider delete requires <uuid>")
	}
	failures := 0
	for _, id := range args {
		if err := resourceproviders.Delete(ctx, client, id).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d resource providers failed to delete.", failures, len(args))
	}
	return nil
}

func resourceProviderShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider show requires <uuid>")
	}
	item, err := findResourceProvider(ctx, client, args[0])
	if err != nil {
		return err
	}
	fields := resourceProviderFields(client, item)
	if boolFlag(opts, "allocations") {
		allocations, err := resourceproviders.GetAllocations(ctx, client, item.UUID).Extract()
		if err != nil {
			return err
		}
		fields = append(fields, outputField{"allocations", allocations.Allocations})
	}
	return renderShowOutput(stdout, opts, fields)
}

func resourceProviderColumns(client *gophercloud.ServiceClient) []string {
	columns := []string{"uuid", "name", "generation"}
	if microversionAtLeast(client.Microversion, "1.14") {
		columns = append(columns, "root_provider_uuid", "parent_provider_uuid")
	}
	return columns
}

func resourceProviderRow(item *resourceproviders.ResourceProvider, columns []string) outputRow {
	row := outputRow{
		"uuid":       item.UUID,
		"name":       item.Name,
		"generation": item.Generation,
	}
	for _, column := range columns {
		switch column {
		case "root_provider_uuid":
			row[column] = nilIfEmpty(item.RootProviderUUID)
		case "parent_provider_uuid":
			row[column] = nilIfEmpty(item.ParentProviderUUID)
		}
	}
	return row
}

func resourceProviderFields(client *gophercloud.ServiceClient, item *resourceproviders.ResourceProvider) []outputField {
	fields := []outputField{
		{"uuid", item.UUID},
		{"name", item.Name},
		{"generation", item.Generation},
	}
	if microversionAtLeast(client.Microversion, "1.14") {
		fields = append(fields,
			outputField{"root_provider_uuid", nilIfEmpty(item.RootProviderUUID)},
			outputField{"parent_provider_uuid", nilIfEmpty(item.ParentProviderUUID)},
		)
	}
	return fields
}

func resourceProviderInventoryList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider inventory list requires <uuid>")
	}
	item, err := resourceproviders.GetInventories(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	usages, err := resourceproviders.GetUsages(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Inventories))
	for resourceClass, inventory := range item.Inventories {
		rows = append(rows, inventoryRow(resourceClass, inventory, usages.Usages[resourceClass], true))
	}
	return renderListOutput(stdout, opts, append([]string{"resource_class"}, append(placementInventoryFields(), "used")...), rows)
}

func resourceProviderInventoryShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("resource provider inventory show requires <uuid> <resource_class>")
	}
	item, err := resourceproviders.GetInventory(ctx, client, args[0], args[1]).Extract()
	if err != nil {
		return err
	}
	usages, err := resourceproviders.GetUsages(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	fields := inventoryFields(item.Inventory)
	fields = append(fields, outputField{"used", usages.Usages[args[1]]})
	return renderShowOutput(stdout, opts, fields)
}

func resourceProviderInventoryClassSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("resource provider inventory class set requires <uuid> <class>")
	}
	provider, err := resourceproviders.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	inventory, err := placementInventoryValuesFromFlags(opts, true)
	if err != nil {
		return err
	}
	payload := map[string]any{"resource_provider_generation": provider.Generation}
	for key, value := range inventory {
		payload[key] = value
	}
	var result map[string]any
	resp, err := client.Put(ctx, client.ServiceURL("resource_providers", url.PathEscape(args[0]), "inventories", url.PathEscape(args[1])), payload, &result, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return renderShowOutput(stdout, opts, placementInventoryFieldsFromMap(result))
}

func resourceProviderInventoryDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider inventory delete requires <uuid>")
	}
	if resourceClass := flagValue(opts, "resource-class"); resourceClass != "" {
		return resourceproviders.DeleteInventory(ctx, client, args[0], resourceClass).ExtractErr()
	}
	return resourceproviders.DeleteInventories(ctx, client, args[0]).ExtractErr()
}

func resourceProviderInventorySet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider inventory set requires <uuid>")
	}
	providers := []map[string]any{}
	if boolFlag(opts, "aggregate") {
		page, err := resourceproviders.List(client, resourceproviders.ListOpts{MemberOf: args[0]}).AllPages(ctx)
		if err != nil {
			return err
		}
		items, err := resourceproviders.ExtractResourceProviders(page)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return fmt.Errorf("No resource providers found in aggregate with uuid %s.", args[0])
		}
		for _, item := range items {
			providers = append(providers, map[string]any{"uuid": item.UUID, "generation": item.Generation})
		}
	} else {
		provider, err := resourceproviders.Get(ctx, client, args[0]).Extract()
		if err != nil {
			return err
		}
		providers = append(providers, map[string]any{"uuid": provider.UUID, "generation": provider.Generation})
	}
	rows := []outputRow{}
	failures := 0
	for _, provider := range providers {
		providerUUID := valueString(provider["uuid"])
		payload, err := placementInventorySetPayload(ctx, opts, client, providerUUID, placementIntValue(provider["generation"]))
		if err != nil {
			return err
		}
		result := payload
		if !boolFlag(opts, "dry-run") {
			var body map[string]any
			resp, err := client.Put(ctx, client.ServiceURL("resource_providers", url.PathEscape(providerUUID), "inventories"), payload, &body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
			_, _, err = gophercloud.ParseResponse(resp, err)
			if err != nil {
				if boolFlag(opts, "aggregate") {
					failures++
					continue
				}
				return oscHTTPException(err)
			}
			result = body
		}
		rows = append(rows, placementInventoryRowsFromPayload(result, providerUUID, boolFlag(opts, "aggregate"))...)
	}
	if failures > 0 {
		return fmt.Errorf("Failed to set inventory for %d of %d resource providers.", failures, len(providers))
	}
	columns := append([]string{"resource_class"}, placementInventoryFields()...)
	if boolFlag(opts, "aggregate") {
		columns = append([]string{"resource_provider"}, columns...)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func resourceProviderAllocationShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider allocation show requires <consumer_uuid>")
	}
	body, err := placementGetConsumerAllocations(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderListOutput(stdout, opts, allocationColumns(client), allocationRows(client, body))
}

func resourceProviderAllocationDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider allocation delete requires <consumer_uuid>")
	}
	resp, err := client.Delete(ctx, client.ServiceURL("allocations", url.PathEscape(args[0])), &gophercloud.RequestOpts{OkCodes: []int{http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func resourceProviderAllocationSet(ctx context.Context, stdout io.Writer, stderr io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider allocation set requires <consumer_uuid>")
	}
	allocations, err := parsePlacementAllocations(flagValues(opts, "allocation"))
	if err != nil {
		return err
	}
	if len(allocations) == 0 {
		return fmt.Errorf("At least one resource allocation must be specified")
	}
	payload := map[string]any{"allocations": placementAllocationPayload(client, allocations)}
	if microversionAtLeast(client.Microversion, "1.28") {
		current, err := placementGetConsumerAllocations(ctx, client, args[0])
		if err != nil {
			return err
		}
		payload["consumer_generation"] = current["consumer_generation"]
	}
	if microversionAtLeast(client.Microversion, "1.8") {
		projectID, userID := flagValue(opts, "project-id"), flagValue(opts, "user-id")
		if projectID == "" || userID == "" {
			return fmt.Errorf("--project-id and --user-id are required")
		}
		payload["project_id"] = projectID
		payload["user_id"] = userID
	}
	if microversionAtLeast(client.Microversion, "1.38") {
		consumerType := flagValue(opts, "consumer-type")
		if consumerType == "" {
			return fmt.Errorf("--consumer-type is required")
		}
		payload["consumer_type"] = consumerType
	} else if flagChanged(opts, "consumer-type") && flagValue(opts, "consumer-type") != "" && stderr != nil {
		_, _ = fmt.Fprintln(stderr, "--consumer-type option does not affect allocation for --os-placement-api-version less than 1.38")
	}
	resp, err := client.Put(ctx, client.ServiceURL("allocations", url.PathEscape(args[0])), payload, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	body, err := placementGetConsumerAllocations(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderListOutput(stdout, opts, allocationColumns(client), allocationRows(client, body))
}

func resourceProviderAllocationUnset(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider allocation unset requires <consumer_uuid>")
	}
	if !microversionAtLeast(client.Microversion, "1.12") {
		return fmt.Errorf("resource provider allocation unset requires --os-placement-api-version 1.12 or greater")
	}
	body, err := placementGetConsumerAllocations(ctx, client, args[0])
	if err != nil {
		return err
	}
	allocations := placementAllocationsMap(body["allocations"])
	providers := stringSet(flagValues(opts, "provider"))
	resourceClasses := flagValues(opts, "resource-class")
	if len(resourceClasses) > 0 {
		for providerID, allocation := range allocations {
			if len(providers) > 0 && !providers[providerID] {
				continue
			}
			resources := mapValue(allocation["resources"])
			for _, resourceClass := range resourceClasses {
				delete(resources, resourceClass)
			}
			if len(resources) == 0 {
				delete(allocations, providerID)
			} else {
				allocation["resources"] = resources
			}
		}
	} else if len(providers) > 0 {
		for providerID := range providers {
			delete(allocations, providerID)
		}
	} else {
		allocations = map[string]map[string]any{}
	}
	body["allocations"] = allocations
	if len(allocations) > 0 || microversionAtLeast(client.Microversion, "1.28") {
		resp, err := client.Put(ctx, client.ServiceURL("allocations", url.PathEscape(args[0])), body, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusNoContent}})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
	} else {
		resp, err := client.Delete(ctx, client.ServiceURL("allocations", url.PathEscape(args[0])), &gophercloud.RequestOpts{OkCodes: []int{http.StatusNoContent}})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
	}
	updated, err := placementGetConsumerAllocations(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderListOutput(stdout, opts, allocationColumns(client), allocationRows(client, updated))
}

func resourceProviderAggregateList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider aggregate list requires <uuid>")
	}
	item, err := resourceproviders.GetAggregates(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Aggregates))
	for _, aggregate := range item.Aggregates {
		rows = append(rows, outputRow{"uuid": aggregate})
	}
	return renderListOutput(stdout, opts, []string{"uuid"}, rows)
}

func resourceProviderAggregateSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider aggregate set requires <uuid>")
	}
	var generation *int
	if value := flagValue(opts, "generation"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid --generation %q", value)
		}
		generation = &parsed
	} else if microversionAtLeast(client.Microversion, "1.19") {
		return fmt.Errorf("A generation must be specified.")
	}
	item, err := resourceproviders.UpdateAggregates(ctx, client, args[0], resourceproviders.UpdateAggregatesOpts{
		Aggregates:                 flagValues(opts, "aggregate"),
		ResourceProviderGeneration: generation,
	}).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Aggregates))
	for _, aggregate := range item.Aggregates {
		rows = append(rows, outputRow{"uuid": aggregate})
	}
	return renderListOutput(stdout, opts, []string{"uuid"}, rows)
}

func resourceProviderTraitList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider trait list requires <uuid>")
	}
	item, err := resourceproviders.GetTraits(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Traits))
	for _, trait := range item.Traits {
		rows = append(rows, outputRow{"name": trait})
	}
	return renderListOutput(stdout, opts, []string{"name"}, rows)
}

func resourceProviderTraitSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider trait set requires <uuid>")
	}
	provider, err := resourceproviders.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	item, err := resourceproviders.UpdateTraits(ctx, client, args[0], resourceproviders.ResourceProviderTraits{
		ResourceProviderGeneration: provider.Generation,
		Traits:                     flagValues(opts, "trait"),
	}).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Traits))
	for _, trait := range item.Traits {
		rows = append(rows, outputRow{"name": trait})
	}
	return renderListOutput(stdout, opts, []string{"name"}, rows)
}

func resourceProviderTraitDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider trait delete requires <uuid>")
	}
	return resourceproviders.DeleteTraits(ctx, client, args[0]).ExtractErr()
}

func resourceProviderUsageShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider usage show requires <uuid>")
	}
	item, err := resourceproviders.GetUsages(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Usages))
	for resourceClass, usage := range item.Usages {
		rows = append(rows, outputRow{"resource_class": resourceClass, "usage": usage})
	}
	return renderListOutput(stdout, opts, []string{"resource_class", "usage"}, rows)
}

func resourceUsageShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource usage show requires <project-uuid>")
	}
	query := url.Values{}
	query.Set("project_id", args[0])
	if userID := flagValue(opts, "user-id"); userID != "" {
		query.Set("user_id", userID)
	}
	var body struct {
		Usages map[string]any `json:"usages"`
	}
	requestURL := client.ServiceURL("usages") + "?" + query.Encode()
	resp, err := client.Get(ctx, requestURL, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	rows := make([]outputRow, 0, len(body.Usages))
	for resourceClass, usage := range body.Usages {
		rows = append(rows, outputRow{"resource_class": resourceClass, "usage": usage})
	}
	return renderListOutput(stdout, opts, []string{"resource_class", "usage"}, rows)
}

func traitList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := traits.ListOpts{Name: flagValue(opts, "name")}
	if boolFlag(opts, "associated") {
		associated := true
		listOpts.Associated = &associated
	}
	page, err := traits.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := traits.ExtractTraits(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"name": item})
	}
	return renderListOutput(stdout, opts, []string{"name"}, rows)
}

func traitShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("trait show requires <name>")
	}
	if err := traits.Get(ctx, client, args[0]).ExtractErr(); err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"name", args[0]},
	})
}

func traitCreate(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("trait create requires <name>")
	}
	return traits.Create(ctx, client, args[0]).ExtractErr()
}

func traitDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("trait delete requires <name>")
	}
	failures := 0
	for _, name := range args {
		if err := traits.Delete(ctx, client, name).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d traits failed to delete.", failures, len(args))
	}
	return nil
}

func findResourceProvider(ctx context.Context, client *gophercloud.ServiceClient, value string) (*resourceproviders.ResourceProvider, error) {
	result := resourceproviders.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := resourceproviders.List(client, resourceproviders.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := resourceproviders.ExtractResourceProviders(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item resourceproviders.ResourceProvider) string { return item.Name })
}

func placementResourceQuery(value string) string {
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, "=", ":")
}

func placementResourceQueryValues(opts *Options) string {
	values := flagValues(opts, "resource")
	if len(values) == 0 && flagValue(opts, "resource") != "" {
		values = []string{flagValue(opts, "resource")}
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			parts = append(parts, placementResourceQuery(value))
		}
	}
	return strings.Join(parts, ",")
}

func placementRequiredQuery(opts *Options) string {
	required := append([]string(nil), flagValues(opts, "required")...)
	if len(required) == 0 && flagValue(opts, "required") != "" {
		required = append(required, flagValue(opts, "required"))
	}
	forbidden := flagValues(opts, "forbidden")
	if len(forbidden) == 0 && flagValue(opts, "forbidden") != "" {
		forbidden = []string{flagValue(opts, "forbidden")}
	}
	for _, value := range forbidden {
		if value != "" {
			required = append(required, "!"+value)
		}
	}
	return strings.Join(required, ",")
}

func placementMemberOfValues(opts *Options) []string {
	values := flagValues(opts, "member-of")
	if len(values) == 0 && flagValue(opts, "member-of") != "" {
		values = []string{flagValue(opts, "member-of")}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "in:") {
			result = append(result, value)
		} else {
			result = append(result, "in:"+value)
		}
	}
	return result
}

func allocationRequestResources(allocations map[string]allocationcandidates.AllocationRequestResource) map[string]map[string]int {
	values := make(map[string]map[string]int, len(allocations))
	for provider, allocation := range allocations {
		values[provider] = allocation.Resources
	}
	return values
}

func sortedAllocationProviders(allocations map[string]allocationcandidates.AllocationRequestResource) []string {
	providers := make([]string, 0, len(allocations))
	for provider := range allocations {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func placementInventoryFields() []string {
	return []string{"allocation_ratio", "min_unit", "max_unit", "reserved", "step_size", "total"}
}

func inventoryFields(inventory resourceproviders.Inventory) []outputField {
	return []outputField{
		{"allocation_ratio", inventory.AllocationRatio},
		{"min_unit", inventory.MinUnit},
		{"max_unit", inventory.MaxUnit},
		{"reserved", inventory.Reserved},
		{"step_size", inventory.StepSize},
		{"total", inventory.Total},
	}
}

func placementInventoryFieldsFromMap(inventory map[string]any) []outputField {
	fields := make([]outputField, 0, len(placementInventoryFields()))
	for _, field := range placementInventoryFields() {
		fields = append(fields, outputField{field, inventory[field]})
	}
	return fields
}

func inventoryRow(resourceClass string, inventory resourceproviders.Inventory, used int, includeClass bool) outputRow {
	row := outputRow{
		"allocation_ratio": inventory.AllocationRatio,
		"min_unit":         inventory.MinUnit,
		"max_unit":         inventory.MaxUnit,
		"reserved":         inventory.Reserved,
		"step_size":        inventory.StepSize,
		"total":            inventory.Total,
		"used":             used,
	}
	if includeClass {
		row["resource_class"] = resourceClass
	}
	return row
}

func placementInventoryValuesFromFlags(opts *Options, requireTotal bool) (map[string]any, error) {
	fields := []struct {
		flag string
		key  string
		kind string
	}{
		{"allocation-ratio", "allocation_ratio", "float"},
		{"min-unit", "min_unit", "int"},
		{"max-unit", "max_unit", "int"},
		{"reserved", "reserved", "int"},
		{"step-size", "step_size", "int"},
		{"total", "total", "int"},
	}
	values := map[string]any{}
	for _, field := range fields {
		if !flagChanged(opts, field.flag) {
			continue
		}
		switch field.kind {
		case "float":
			parsed, err := strconv.ParseFloat(flagValue(opts, field.flag), 64)
			if err != nil {
				return nil, fmt.Errorf("invalid --%s %q", field.flag, flagValue(opts, field.flag))
			}
			values[field.key] = parsed
		default:
			parsed, err := strconv.Atoi(flagValue(opts, field.flag))
			if err != nil {
				return nil, fmt.Errorf("invalid --%s %q", field.flag, flagValue(opts, field.flag))
			}
			values[field.key] = parsed
		}
	}
	if requireTotal {
		if _, ok := values["total"]; !ok {
			return nil, fmt.Errorf("resource provider inventory class set requires --total")
		}
	}
	return values, nil
}

func placementInventorySetPayload(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, providerUUID string, generation int) (map[string]any, error) {
	var inventories map[string]any
	if boolFlag(opts, "amend") {
		current, err := placementGetInventoriesRaw(ctx, client, providerUUID)
		if err != nil {
			return nil, err
		}
		inventories = mapValue(current["inventories"])
	} else {
		inventories = map[string]any{}
	}
	for _, resource := range flagValues(opts, "resource") {
		name, field, value, err := parsePlacementInventoryResource(resource)
		if err != nil {
			return nil, err
		}
		entry := mapValue(inventories[name])
		if entry == nil {
			entry = map[string]any{}
			inventories[name] = entry
		}
		entry[field] = value
	}
	return map[string]any{
		"inventories":                  inventories,
		"resource_provider_generation": generation,
	}, nil
}

func parsePlacementInventoryResource(resource string) (string, string, any, error) {
	parts := strings.Split(resource, "=")
	if len(parts) != 2 {
		return "", "", nil, fmt.Errorf("Resource argument must have \"name=value\" format")
	}
	namePart, rawValue := parts[0], parts[1]
	nameParts := strings.Split(namePart, ":")
	name, field := "", "total"
	switch len(nameParts) {
	case 1:
		name = nameParts[0]
	case 2:
		name, field = nameParts[0], nameParts[1]
	default:
		return "", "", nil, fmt.Errorf("Resource argument can contain only one colon")
	}
	if name == "" || field == "" || rawValue == "" {
		return "", "", nil, fmt.Errorf("Name, field and value must be not empty")
	}
	valid := map[string]string{
		"allocation_ratio": "float",
		"min_unit":         "int",
		"max_unit":         "int",
		"reserved":         "int",
		"step_size":        "int",
		"total":            "int",
	}
	kind, ok := valid[field]
	if !ok {
		return "", "", nil, fmt.Errorf("Unknown inventory field %s", field)
	}
	if kind == "float" {
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return "", "", nil, err
		}
		return name, field, value, nil
	}
	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return "", "", nil, err
	}
	return name, field, value, nil
}

func placementGetInventoriesRaw(ctx context.Context, client *gophercloud.ServiceClient, providerUUID string) (map[string]any, error) {
	var body map[string]any
	resp, err := client.Get(ctx, client.ServiceURL("resource_providers", url.PathEscape(providerUUID), "inventories"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	return body, nil
}

func placementInventoryRowsFromPayload(payload map[string]any, providerUUID string, includeProvider bool) []outputRow {
	inventories := mapValue(payload["inventories"])
	names := make([]string, 0, len(inventories))
	for name := range inventories {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]outputRow, 0, len(names))
	for _, name := range names {
		inventory := mapValue(inventories[name])
		row := outputRow{"resource_class": name}
		if includeProvider {
			row["resource_provider"] = providerUUID
		}
		for _, field := range placementInventoryFields() {
			row[field] = inventory[field]
		}
		rows = append(rows, row)
	}
	return rows
}

func placementGetConsumerAllocations(ctx context.Context, client *gophercloud.ServiceClient, consumerUUID string) (map[string]any, error) {
	var body map[string]any
	resp, err := client.Get(ctx, client.ServiceURL("allocations", url.PathEscape(consumerUUID)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	return body, nil
}

func parsePlacementAllocations(values []string) (map[string]map[string]int, error) {
	allocations := map[string]map[string]int{}
	for _, value := range values {
		if !strings.Contains(value, "=") || !strings.Contains(value, ",") {
			return nil, fmt.Errorf("Incorrect allocation string format")
		}
		parts := strings.Split(value, ",")
		provider := ""
		resources := map[string]int{}
		for _, part := range parts {
			kv := strings.Split(part, "=")
			if len(kv) != 2 {
				return nil, fmt.Errorf("Incorrect allocation string format")
			}
			if kv[0] == "rp" {
				provider = kv[1]
				continue
			}
			amount, err := strconv.Atoi(kv[1])
			if err != nil {
				return nil, err
			}
			resources[kv[0]] = amount
		}
		if provider == "" {
			return nil, fmt.Errorf("Resource provider parameter is required for allocation string")
		}
		current := allocations[provider]
		if current == nil {
			current = map[string]int{}
			allocations[provider] = current
		}
		for resourceClass, amount := range resources {
			if existing, ok := current[resourceClass]; ok && existing != amount {
				return nil, fmt.Errorf("Conflict detected for resource provider %s resource class %s", provider, resourceClass)
			}
			current[resourceClass] = amount
		}
	}
	return allocations, nil
}

func placementAllocationPayload(client *gophercloud.ServiceClient, allocations map[string]map[string]int) any {
	if microversionAtLeast(client.Microversion, "1.12") {
		payload := map[string]map[string]map[string]int{}
		for provider, resources := range allocations {
			payload[provider] = map[string]map[string]int{"resources": resources}
		}
		return payload
	}
	payload := make([]map[string]any, 0, len(allocations))
	for provider, resources := range allocations {
		payload = append(payload, map[string]any{
			"resource_provider": map[string]string{"uuid": provider},
			"resources":         resources,
		})
	}
	return payload
}

func allocationColumns(client *gophercloud.ServiceClient) []string {
	columns := []string{"resource_provider", "generation", "resources"}
	if microversionAtLeast(client.Microversion, "1.12") {
		columns = append(columns, "project_id", "user_id")
	}
	if microversionAtLeast(client.Microversion, "1.38") {
		columns = append(columns, "consumer_type")
	}
	return columns
}

func allocationRows(client *gophercloud.ServiceClient, body map[string]any) []outputRow {
	allocations := placementAllocationsMap(body["allocations"])
	providers := make([]string, 0, len(allocations))
	for provider := range allocations {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	rows := make([]outputRow, 0, len(providers))
	for _, provider := range providers {
		allocation := allocations[provider]
		row := outputRow{
			"resource_provider": provider,
			"generation":        allocation["generation"],
			"resources":         allocation["resources"],
		}
		if microversionAtLeast(client.Microversion, "1.12") {
			row["project_id"] = body["project_id"]
			row["user_id"] = body["user_id"]
		}
		if microversionAtLeast(client.Microversion, "1.38") {
			row["consumer_type"] = body["consumer_type"]
		}
		rows = append(rows, row)
	}
	return rows
}

func placementAllocationsMap(value any) map[string]map[string]any {
	result := map[string]map[string]any{}
	switch typed := value.(type) {
	case map[string]any:
		for provider, allocation := range typed {
			result[provider] = mapValue(allocation)
		}
	case []any:
		for _, entry := range typed {
			item := mapValue(entry)
			provider := mapValue(item["resource_provider"])
			id := valueString(provider["uuid"])
			if id != "" {
				result[id] = item
			}
		}
	}
	return result
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func placementIntValue(value any) int {
	parsed, ok := intFromAny(value)
	if !ok {
		return 0
	}
	return parsed
}
