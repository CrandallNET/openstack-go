package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/catalog"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
)

type discoveryReport struct {
	Cloud       string           `json:"cloud"`
	GeneratedAt string           `json:"generated_at"`
	Status      string           `json:"status"`
	Error       string           `json:"error,omitempty"`
	Auth        authSummary      `json:"auth"`
	Services    []serviceSummary `json:"services"`
	Fixtures    fixtureSummary   `json:"fixtures"`
	Notes       []string         `json:"notes,omitempty"`
}

type authSummary struct {
	IdentityEndpoint string `json:"identity_endpoint,omitempty"`
	Region           string `json:"region,omitempty"`
	EndpointType     string `json:"endpoint_type,omitempty"`
}

type serviceSummary struct {
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Type      string            `json:"type"`
	Endpoints []endpointSummary `json:"endpoints"`
}

type endpointSummary struct {
	Region    string `json:"region,omitempty"`
	RegionID  string `json:"region_id,omitempty"`
	Interface string `json:"interface,omitempty"`
	URL       string `json:"url"`
}

type fixtureSummary struct {
	Images      fixtureSet `json:"images"`
	Flavors     fixtureSet `json:"flavors"`
	Networks    fixtureSet `json:"networks"`
	VolumeTypes fixtureSet `json:"volume_types"`
	Roles       fixtureSet `json:"roles"`
}

type fixtureSet struct {
	Status string        `json:"status"`
	Error  string        `json:"error,omitempty"`
	Items  []fixtureItem `json:"items,omitempty"`
}

type fixtureItem struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"`
	Extra    string `json:"extra,omitempty"`
	Selected bool   `json:"selected,omitempty"`
}

func main() {
	var cloudList string
	var outputDir string
	var limit int
	var timeout time.Duration

	flag.StringVar(&cloudList, "cloud", os.Getenv("OS_CLOUD"), "cloud name or comma-separated cloud names to discover")
	flag.StringVar(&outputDir, "output-dir", "compat/live-clouds", "directory for non-secret discovery reports")
	flag.IntVar(&limit, "limit", 20, "maximum fixture candidates to keep per resource type")
	flag.DurationVar(&timeout, "timeout", 2*time.Minute, "timeout per cloud")
	flag.Parse()

	cloudsToDiscover := splitClouds(cloudList)
	if len(cloudsToDiscover) == 0 {
		fmt.Fprintln(os.Stderr, "cloud-discovery: provide --cloud or OS_CLOUD")
		os.Exit(1)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cloud-discovery: %v\n", err)
		os.Exit(1)
	}

	failed := false
	for _, cloud := range cloudsToDiscover {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		report := discoverCloud(ctx, cloud, limit)
		cancel()
		path := filepath.Join(outputDir, cloud+".json")
		if err := writeJSON(path, report); err != nil {
			fmt.Fprintf(os.Stderr, "cloud-discovery: write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("%s: %s", cloud, report.Status)
		if report.Error != "" {
			fmt.Printf(" (%s)", report.Error)
		}
		fmt.Printf(" -> %s\n", path)
		if report.Status != "ok" {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func splitClouds(value string) []string {
	var clouds []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			clouds = append(clouds, part)
		}
	}
	return clouds
}

func discoverCloud(ctx context.Context, cloud string, limit int) discoveryReport {
	report := discoveryReport{
		Cloud:       cloud,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Status:      "ok",
		Notes: []string{
			"Discovery artifacts must not include tokens, passwords, application credential secrets, clouds.yaml contents, or debug logs.",
			"Fixture candidates are discovered at run time and should be re-queried before lifecycle tests because cloud state can change.",
		},
	}

	authOptions, endpointOptions, tlsConfig, err := clouds.Parse(clouds.WithCloudName(cloud))
	if err != nil {
		report.Status = "error"
		report.Error = err.Error()
		return report
	}
	report.Auth = authSummary{
		IdentityEndpoint: authOptions.IdentityEndpoint,
		Region:           endpointOptions.Region,
		EndpointType:     string(endpointOptions.Availability),
	}

	provider, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		report.Status = "error"
		report.Error = err.Error()
		return report
	}

	identity, err := openstack.NewIdentityV3(provider, endpointOptions)
	if err != nil {
		report.Status = "error"
		report.Error = err.Error()
		return report
	}
	report.Services = discoverCatalog(ctx, identity)
	report.Fixtures.Images = discoverImages(ctx, provider, endpointOptions, limit)
	report.Fixtures.Flavors = discoverFlavors(ctx, provider, endpointOptions, limit)
	report.Fixtures.Networks = discoverNetworks(ctx, provider, endpointOptions, limit)
	report.Fixtures.VolumeTypes = discoverVolumeTypes(ctx, provider, endpointOptions, limit)
	report.Fixtures.Roles = fixtureSet{Status: "not-implemented", Error: "role discovery is not implemented yet"}
	return report
}

func discoverCatalog(ctx context.Context, client *gophercloud.ServiceClient) []serviceSummary {
	page, err := catalog.List(client).AllPages(ctx)
	if err != nil {
		return nil
	}
	entries, err := catalog.ExtractServiceCatalog(page)
	if err != nil {
		return nil
	}
	services := make([]serviceSummary, 0, len(entries))
	for _, entry := range entries {
		service := serviceSummary{
			ID:   entry.ID,
			Name: entry.Name,
			Type: entry.Type,
		}
		for _, endpoint := range entry.Endpoints {
			service.Endpoints = append(service.Endpoints, endpointSummary{
				Region:    endpoint.Region,
				RegionID:  endpoint.RegionID,
				Interface: endpoint.Interface,
				URL:       endpoint.URL,
			})
		}
		sort.Slice(service.Endpoints, func(i int, j int) bool {
			return service.Endpoints[i].Interface+service.Endpoints[i].Region+service.Endpoints[i].URL <
				service.Endpoints[j].Interface+service.Endpoints[j].Region+service.Endpoints[j].URL
		})
		services = append(services, service)
	}
	sort.Slice(services, func(i int, j int) bool {
		if services[i].Type == services[j].Type {
			return services[i].Name < services[j].Name
		}
		return services[i].Type < services[j].Type
	})
	return services
}

func discoverImages(ctx context.Context, provider *gophercloud.ProviderClient, endpointOptions gophercloud.EndpointOpts, limit int) fixtureSet {
	client, err := openstack.NewImageV2(provider, endpointOptions)
	if err != nil {
		return fixtureSet{Status: "skipped", Error: err.Error()}
	}
	page, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	fixtures := make([]fixtureItem, 0, len(items))
	for _, item := range items {
		fixtures = append(fixtures, fixtureItem{ID: item.ID, Name: item.Name, Status: string(item.Status), Extra: string(item.Visibility)})
	}
	return selectedFixtureSet(fixtures, limit)
}

func discoverFlavors(ctx context.Context, provider *gophercloud.ProviderClient, endpointOptions gophercloud.EndpointOpts, limit int) fixtureSet {
	client, err := openstack.NewComputeV2(provider, endpointOptions)
	if err != nil {
		return fixtureSet{Status: "skipped", Error: err.Error()}
	}
	page, err := flavors.ListDetail(client, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	fixtures := make([]fixtureItem, 0, len(items))
	for _, item := range items {
		fixtures = append(fixtures, fixtureItem{ID: item.ID, Name: item.Name, Extra: fmt.Sprintf("ram=%d disk=%d vcpus=%d", item.RAM, item.Disk, item.VCPUs)})
	}
	return selectedFixtureSet(fixtures, limit)
}

func discoverNetworks(ctx context.Context, provider *gophercloud.ProviderClient, endpointOptions gophercloud.EndpointOpts, limit int) fixtureSet {
	client, err := openstack.NewNetworkV2(provider, endpointOptions)
	if err != nil {
		return fixtureSet{Status: "skipped", Error: err.Error()}
	}
	page, err := networks.List(client, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	items, err := networks.ExtractNetworks(page)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	fixtures := make([]fixtureItem, 0, len(items))
	for _, item := range items {
		extra := fmt.Sprintf("shared=%t subnets=%d", item.Shared, len(item.Subnets))
		fixtures = append(fixtures, fixtureItem{ID: item.ID, Name: item.Name, Status: item.Status, Extra: extra})
	}
	return selectedFixtureSet(fixtures, limit)
}

func discoverVolumeTypes(ctx context.Context, provider *gophercloud.ProviderClient, endpointOptions gophercloud.EndpointOpts, limit int) fixtureSet {
	client, err := openstack.NewBlockStorageV3(provider, endpointOptions)
	if err != nil {
		return fixtureSet{Status: "skipped", Error: err.Error()}
	}
	page, err := volumetypes.List(client, volumetypes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	items, err := volumetypes.ExtractVolumeTypes(page)
	if err != nil {
		return fixtureSet{Status: "error", Error: err.Error()}
	}
	fixtures := make([]fixtureItem, 0, len(items))
	for _, item := range items {
		fixtures = append(fixtures, fixtureItem{ID: item.ID, Name: item.Name, Extra: fmt.Sprintf("public=%t", item.IsPublic)})
	}
	return selectedFixtureSet(fixtures, limit)
}

func selectedFixtureSet(items []fixtureItem, limit int) fixtureSet {
	sort.Slice(items, func(i int, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].ID < items[j].ID
		}
		return items[i].Name < items[j].Name
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	if len(items) > 0 {
		items[0].Selected = true
	}
	return fixtureSet{Status: "ok", Items: items}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
