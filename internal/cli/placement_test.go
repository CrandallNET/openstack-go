package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestPlacementResourceClassAndTraitWrites(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/resource_classes":
			seen["class-create"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode resource class create: %v", err)
			}
			if body["name"] != "CUSTOM_TEST" {
				t.Fatalf("unexpected resource class create body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && r.URL.Path == "/resource_classes/CUSTOM_TEST":
			seen["class-set"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/resource_classes/CUSTOM_TEST":
			seen["class-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/traits/CUSTOM_TEST_TRAIT":
			seen["trait-create"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/traits/CUSTOM_TEST_TRAIT":
			seen["trait-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := testPlacementClient(server.URL)
	if err := resourceClassCreate(context.Background(), &Options{}, client, []string{"CUSTOM_TEST"}); err != nil {
		t.Fatalf("resource class create: %v", err)
	}
	if err := resourceClassSet(context.Background(), client, []string{"CUSTOM_TEST"}); err != nil {
		t.Fatalf("resource class set: %v", err)
	}
	if err := resourceClassDelete(context.Background(), client, []string{"CUSTOM_TEST"}); err != nil {
		t.Fatalf("resource class delete: %v", err)
	}
	if err := traitCreate(context.Background(), client, []string{"CUSTOM_TEST_TRAIT"}); err != nil {
		t.Fatalf("trait create: %v", err)
	}
	if err := traitDelete(context.Background(), client, []string{"CUSTOM_TEST_TRAIT"}); err != nil {
		t.Fatalf("trait delete: %v", err)
	}
	for _, key := range []string{"class-create", "class-set", "class-delete", "trait-create", "trait-delete"} {
		if !seen[key] {
			t.Fatalf("missing request %s", key)
		}
	}
}

func TestPlacementResourceProviderInventoryAggregateAndTraitWrites(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/resource_providers":
			seen["provider-create"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode provider create: %v", err)
			}
			if body["name"] != "rp-name" || body["uuid"] != "rp-id" || body["parent_provider_uuid"] != "parent-id" {
				t.Fatalf("unexpected provider create body: %#v", body)
			}
			writePlacementProvider(t, w, "rp-id", "rp-name", 1)
		case r.Method == http.MethodPut && r.URL.Path == "/resource_providers/rp-id":
			seen["provider-set"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode provider set: %v", err)
			}
			if body["name"] != "rp-new" || body["parent_provider_uuid"] != "parent-id" {
				t.Fatalf("unexpected provider set body: %#v", body)
			}
			writePlacementProvider(t, w, "rp-id", "rp-new", 2)
		case r.Method == http.MethodGet && r.URL.Path == "/resource_providers/rp-id":
			writePlacementProvider(t, w, "rp-id", "rp-name", 7)
		case r.Method == http.MethodPut && r.URL.Path == "/resource_providers/rp-id/aggregates":
			seen["aggregate-set"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode aggregate set: %v", err)
			}
			if got := body["resource_provider_generation"]; got != float64(7) {
				t.Fatalf("unexpected aggregate generation: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"aggregates": []string{"agg-id"}, "resource_provider_generation": 8}); err != nil {
				t.Fatalf("encode aggregate set: %v", err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/resource_providers/rp-id/traits":
			seen["trait-set"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode trait set: %v", err)
			}
			if got := body["resource_provider_generation"]; got != float64(7) {
				t.Fatalf("unexpected trait generation: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"traits": []string{"CUSTOM_TEST_TRAIT"}, "resource_provider_generation": 8}); err != nil {
				t.Fatalf("encode trait set: %v", err)
			}
		case r.Method == http.MethodDelete && r.URL.Path == "/resource_providers/rp-id/traits":
			seen["trait-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/resource_providers/rp-id/inventories/VCPU":
			seen["inventory-class-set"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode inventory class set: %v", err)
			}
			if body["total"] != float64(16) || body["resource_provider_generation"] != float64(7) {
				t.Fatalf("unexpected inventory class set body: %#v", body)
			}
			if _, ok := body["max_unit"]; ok {
				t.Fatalf("unset inventory field should not be sent: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"allocation_ratio": 1.0, "min_unit": 1, "max_unit": 16, "reserved": 0, "step_size": 1, "total": 16}); err != nil {
				t.Fatalf("encode inventory class set: %v", err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/resource_providers/rp-id/inventories":
			seen["inventory-set"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode inventory set: %v", err)
			}
			inventories := mapValue(body["inventories"])
			vcpu := mapValue(inventories["VCPU"])
			if vcpu["total"] != float64(8) || body["resource_provider_generation"] != float64(7) {
				t.Fatalf("unexpected inventory set body: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatalf("encode inventory set: %v", err)
			}
		case r.Method == http.MethodDelete && r.URL.Path == "/resource_providers/rp-id/inventories/VCPU":
			seen["inventory-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/resource_providers/rp-id":
			seen["provider-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := testPlacementClient(server.URL)
	var stdout bytes.Buffer
	if err := resourceProviderCreate(context.Background(), &stdout, &Options{
		Format:       "json",
		CommandFlags: map[string]string{"uuid": "rp-id", "parent-provider": "parent-id"},
	}, client, []string{"rp-name"}); err != nil {
		t.Fatalf("resource provider create: %v", err)
	}
	if err := resourceProviderSet(context.Background(), &stdout, &Options{CommandFlags: map[string]string{"name": "rp-new", "parent-provider": "parent-id"}}, client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider set: %v", err)
	}
	if err := resourceProviderAggregateSet(context.Background(), &stdout, &Options{CommandFlags: map[string]string{"generation": "7"}, CommandFlagList: map[string][]string{"aggregate": {"agg-id"}}}, client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider aggregate set: %v", err)
	}
	if err := resourceProviderTraitSet(context.Background(), &stdout, &Options{CommandFlagList: map[string][]string{"trait": {"CUSTOM_TEST_TRAIT"}}}, client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider trait set: %v", err)
	}
	if err := resourceProviderTraitDelete(context.Background(), client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider trait delete: %v", err)
	}
	if err := resourceProviderInventoryClassSet(context.Background(), &stdout, &Options{CommandFlags: map[string]string{"total": "16"}}, client, []string{"rp-id", "VCPU"}); err != nil {
		t.Fatalf("resource provider inventory class set: %v", err)
	}
	if err := resourceProviderInventorySet(context.Background(), &stdout, &Options{CommandFlagList: map[string][]string{"resource": {"VCPU=8"}}}, client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider inventory set: %v", err)
	}
	if err := resourceProviderInventoryDelete(context.Background(), &Options{CommandFlags: map[string]string{"resource-class": "VCPU"}}, client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider inventory delete: %v", err)
	}
	if err := resourceProviderDelete(context.Background(), client, []string{"rp-id"}); err != nil {
		t.Fatalf("resource provider delete: %v", err)
	}
	for _, key := range []string{"provider-create", "provider-set", "aggregate-set", "trait-set", "trait-delete", "inventory-class-set", "inventory-set", "inventory-delete", "provider-delete"} {
		if !seen[key] {
			t.Fatalf("missing request %s", key)
		}
	}
	if output := stdout.String(); !strings.Contains(output, `"uuid": "rp-id"`) || !strings.Contains(output, "VCPU") {
		t.Fatalf("unexpected placement output:\n%s", output)
	}
}

func TestPlacementAllocationAndUsageRawRequests(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/allocations/consumer-id":
			seen["allocation-get"] = true
			writePlacementAllocations(t, w, map[string]any{
				"allocations":         map[string]any{},
				"consumer_generation": nil,
				"project_id":          "project-id",
				"user_id":             "user-id",
				"consumer_type":       "INSTANCE",
			})
		case r.Method == http.MethodPut && r.URL.Path == "/allocations/consumer-id":
			seen["allocation-put"] = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode allocation set: %v", err)
			}
			allocations := mapValue(body["allocations"])
			rp := mapValue(allocations["rp-id"])
			resources := mapValue(rp["resources"])
			if resources["VCPU"] != float64(2) || body["project_id"] != "project-id" || body["consumer_type"] != "INSTANCE" {
				t.Fatalf("unexpected allocation set body: %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/allocations/consumer-id":
			seen["allocation-delete"] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/usages":
			seen["usage-show"] = true
			if r.URL.Query().Get("project_id") != "project-id" || r.URL.Query().Get("user_id") != "user-id" {
				t.Fatalf("unexpected usage query: %s", r.URL.RawQuery)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"usages": map[string]any{"unknown": map[string]int{"VCPU": 2, "consumer_count": 1}}}); err != nil {
				t.Fatalf("encode usage: %v", err)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := testPlacementClient(server.URL)
	var stdout bytes.Buffer
	opts := &Options{
		Format:       "json",
		CommandFlags: map[string]string{"project-id": "project-id", "user-id": "user-id", "consumer-type": "INSTANCE"},
		CommandFlagList: map[string][]string{
			"allocation": {"rp=rp-id,VCPU=2"},
		},
	}
	if err := resourceProviderAllocationSet(context.Background(), &stdout, io.Discard, opts, client, []string{"consumer-id"}); err != nil {
		t.Fatalf("allocation set: %v", err)
	}
	if err := resourceProviderAllocationDelete(context.Background(), client, []string{"consumer-id"}); err != nil {
		t.Fatalf("allocation delete: %v", err)
	}
	if err := resourceUsageShow(context.Background(), &stdout, &Options{Format: "json", CommandFlags: map[string]string{"user-id": "user-id"}}, client, []string{"project-id"}); err != nil {
		t.Fatalf("resource usage show: %v", err)
	}
	for _, key := range []string{"allocation-get", "allocation-put", "allocation-delete", "usage-show"} {
		if !seen[key] {
			t.Fatalf("missing request %s", key)
		}
	}
	if output := stdout.String(); !strings.Contains(output, `"resource_provider": "rp-id"`) || !strings.Contains(output, `"resource_class": "unknown"`) {
		t.Fatalf("unexpected placement allocation/usage output:\n%s", output)
	}
}

func TestPlacementAllocationConsumerTypeWarningBefore138(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/allocations/consumer-id":
			writePlacementAllocations(t, w, map[string]any{
				"allocations":         map[string]any{},
				"consumer_generation": nil,
				"project_id":          "project-id",
				"user_id":             "user-id",
			})
		case r.Method == http.MethodPut && r.URL.Path == "/allocations/consumer-id":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode allocation set: %v", err)
			}
			if _, ok := body["consumer_type"]; ok {
				t.Fatalf("consumer_type should not be sent before 1.38: %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := testPlacementClient(server.URL)
	client.Microversion = "1.29"
	opts := &Options{
		Format:       "json",
		CommandFlags: map[string]string{"project-id": "project-id", "user-id": "user-id", "consumer-type": "INSTANCE"},
		CommandFlagList: map[string][]string{
			"allocation": {"rp=rp-id,VCPU=2"},
		},
	}
	var stdout, stderr bytes.Buffer
	if err := resourceProviderAllocationSet(context.Background(), &stdout, &stderr, opts, client, []string{"consumer-id"}); err != nil {
		t.Fatalf("allocation set: %v", err)
	}
	if !strings.Contains(stderr.String(), "--consumer-type option does not affect allocation for --os-placement-api-version less than 1.38") {
		t.Fatalf("missing consumer type warning: %q", stderr.String())
	}
}

func writePlacementProvider(t *testing.T, w http.ResponseWriter, id string, name string, generation int) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"uuid":                 id,
		"name":                 name,
		"generation":           generation,
		"root_provider_uuid":   id,
		"parent_provider_uuid": nil,
	}); err != nil {
		t.Fatalf("encode placement provider: %v", err)
	}
}

func writePlacementAllocations(t *testing.T, w http.ResponseWriter, body map[string]any) {
	t.Helper()
	if _, ok := body["allocations"].(map[string]any)["rp-id"]; !ok {
		body["allocations"] = map[string]any{
			"rp-id": map[string]any{"generation": 1, "resources": map[string]int{"VCPU": 2}},
		}
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode placement allocations: %v", err)
	}
}

func testPlacementClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/",
		ResourceBase:   baseURL + "/",
		Type:           "placement",
		Microversion:   "1.39",
	}
}
