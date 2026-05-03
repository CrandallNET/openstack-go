package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/allocationcandidates"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits"
)

func allocationCandidateList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := allocationcandidates.List(client, allocationcandidates.ListOpts{
		Resources:   placementResourceQuery(flagValue(opts, "resource")),
		Required:    compactStrings([]string{placementRequiredQuery(opts)}),
		MemberOf:    compactStrings([]string{flagValue(opts, "member-of")}),
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
	rows := make([]outputRow, 0, len(item.AllocationRequests))
	for i, request := range item.AllocationRequests {
		rows = append(rows, outputRow{
			"#":                      i + 1,
			"Allocation":             allocationRequestResources(request.Allocations),
			"Resource Provider":      sortedAllocationProviders(request.Allocations),
			"Provider Summaries":     item.ProviderSummaries,
			"Request Group Mappings": request.Mappings,
		})
	}
	return renderListOutput(stdout, opts, []string{"#", "Allocation", "Resource Provider", "Provider Summaries", "Request Group Mappings"}, rows)
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
		rows = append(rows, outputRow{"Name": item.Name})
	}
	return renderListOutput(stdout, opts, []string{"Name"}, rows)
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
		{"links", item.Links},
	})
}

func resourceProviderList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := resourceproviders.List(client, resourceproviders.ListOpts{
		UUID:      flagValue(opts, "uuid"),
		Name:      flagValue(opts, "name"),
		Resources: placementResourceQuery(flagValue(opts, "resource")),
		InTree:    flagValue(opts, "in-tree"),
		Required:  placementRequiredQuery(opts),
		MemberOf:  flagValue(opts, "member-of"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := resourceproviders.ExtractResourceProviders(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"UUID":                 item.UUID,
			"Name":                 item.Name,
			"Generation":           item.Generation,
			"Root Provider UUID":   item.RootProviderUUID,
			"Parent Provider UUID": item.ParentProviderUUID,
		})
	}
	return renderListOutput(stdout, opts, []string{"UUID", "Name", "Generation", "Root Provider UUID", "Parent Provider UUID"}, rows)
}

func resourceProviderShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider show requires <uuid>")
	}
	item, err := findResourceProvider(ctx, client, args[0])
	if err != nil {
		return err
	}
	fields := []outputField{
		{"uuid", item.UUID},
		{"name", item.Name},
		{"generation", item.Generation},
		{"root_provider_uuid", item.RootProviderUUID},
		{"parent_provider_uuid", item.ParentProviderUUID},
		{"links", item.Links},
	}
	if boolFlag(opts, "allocations") {
		allocations, err := resourceproviders.GetAllocations(ctx, client, item.UUID).Extract()
		if err != nil {
			return err
		}
		fields = append(fields, outputField{"allocations", allocations.Allocations})
	}
	return renderShowOutput(stdout, opts, fields)
}

func resourceProviderInventoryList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider inventory list requires <uuid>")
	}
	item, err := resourceproviders.GetInventories(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(item.Inventories))
	for resourceClass, inventory := range item.Inventories {
		rows = append(rows, inventoryRow(resourceClass, inventory))
	}
	return renderListOutput(stdout, opts, []string{"Resource Class", "Total", "Reserved", "Min Unit", "Max Unit", "Step Size", "Allocation Ratio"}, rows)
}

func resourceProviderInventoryShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("resource provider inventory show requires <uuid> <resource_class>")
	}
	item, err := resourceproviders.GetInventory(ctx, client, args[0], args[1]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"resource_class", args[1]},
		{"resource_provider_generation", item.ResourceProviderGeneration},
		{"total", item.Total},
		{"reserved", item.Reserved},
		{"min_unit", item.MinUnit},
		{"max_unit", item.MaxUnit},
		{"step_size", item.StepSize},
		{"allocation_ratio", item.AllocationRatio},
	})
}

func resourceProviderAllocationShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource provider allocation show requires <uuid>")
	}
	item, err := resourceproviders.GetAllocations(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"resource_provider_generation", item.ResourceProviderGeneration},
		{"allocations", item.Allocations},
	})
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
		rows = append(rows, outputRow{"UUID": aggregate})
	}
	return renderListOutput(stdout, opts, []string{"UUID"}, rows)
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
		rows = append(rows, outputRow{"Name": trait})
	}
	return renderListOutput(stdout, opts, []string{"Name"}, rows)
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
		rows = append(rows, outputRow{"Resource Class": resourceClass, "Usage": usage})
	}
	return renderListOutput(stdout, opts, []string{"Resource Class", "Usage"}, rows)
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
		rows = append(rows, outputRow{"Name": item})
	}
	return renderListOutput(stdout, opts, []string{"Name"}, rows)
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
		{"exists", true},
	})
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

func placementRequiredQuery(opts *Options) string {
	if forbidden := flagValue(opts, "forbidden"); forbidden != "" {
		return "!" + forbidden
	}
	return flagValue(opts, "required")
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

func inventoryRow(resourceClass string, inventory resourceproviders.Inventory) outputRow {
	return outputRow{
		"Resource Class":   resourceClass,
		"Total":            inventory.Total,
		"Reserved":         inventory.Reserved,
		"Min Unit":         inventory.MinUnit,
		"Max Unit":         inventory.MaxUnit,
		"Step Size":        inventory.StepSize,
		"Allocation Ratio": inventory.AllocationRatio,
	}
}
