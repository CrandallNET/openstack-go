package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestComputeAggregateCreateSetCacheAndDelete(t *testing.T) {
	t.Setenv("OS_COMPUTE_API_VERSION", "2.81")
	var sawCreate, sawMetadata, sawCache, sawDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/os-aggregates":
			sawCreate = true
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode aggregate create: %v", err)
			}
			if body["aggregate"]["name"] != "test-agg" || body["aggregate"]["availability_zone"] != "test-zone" {
				t.Fatalf("unexpected aggregate create body: %#v", body)
			}
			writeAggregateTestResponse(t, w, map[string]any{"id": 7, "name": "test-agg", "availability_zone": "test-zone", "metadata": map[string]string{}})
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/os-aggregates/7/action":
			sawMetadata = true
			var body map[string]map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode aggregate metadata: %v", err)
			}
			if body["set_metadata"]["metadata"]["role"] != "test" {
				t.Fatalf("unexpected aggregate metadata body: %#v", body)
			}
			writeAggregateTestResponse(t, w, map[string]any{"id": 7, "name": "test-agg", "availability_zone": "test-zone", "metadata": map[string]string{"role": "test"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/os-aggregates/7":
			writeAggregateTestResponse(t, w, map[string]any{"id": 7, "name": "test-agg", "availability_zone": "test-zone", "metadata": map[string]string{"role": "test"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/images/image-id":
			if err := json.NewEncoder(w).Encode(map[string]any{"id": "image-id", "name": "test-image"}); err != nil {
				t.Fatalf("encode image: %v", err)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/os-aggregates/7/images":
			sawCache = true
			var body map[string][]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode cache body: %v", err)
			}
			if len(body["cache"]) != 1 || body["cache"][0]["id"] != "image-id" {
				t.Fatalf("unexpected cache body: %#v", body)
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodDelete && r.URL.Path == "/v2.1/project/os-aggregates/7":
			sawDelete = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	computeClient := testComputeClient(server.URL)
	imageClient := testImageClient(server.URL)
	opts := &Options{
		Format:       "json",
		CommandFlags: map[string]string{"zone": "test-zone"},
		CommandFlagList: map[string][]string{
			"property": {"role=test"},
		},
	}
	var stdout bytes.Buffer
	if err := aggregateCreate(context.Background(), &stdout, opts, computeClient, []string{"test-agg"}); err != nil {
		t.Fatalf("aggregate create: %v", err)
	}
	computeClient.Microversion = "2.81"
	if err := aggregateCacheImage(context.Background(), &Options{}, computeClient, imageClient, []string{"7", "image-id"}); err != nil {
		t.Fatalf("aggregate cache image: %v", err)
	}
	if err := aggregateDelete(context.Background(), computeClient, []string{"7"}); err != nil {
		t.Fatalf("aggregate delete: %v", err)
	}
	if !sawCreate || !sawMetadata || !sawCache || !sawDelete {
		t.Fatalf("missing expected aggregate requests create=%t metadata=%t cache=%t delete=%t", sawCreate, sawMetadata, sawCache, sawDelete)
	}
	if output := stdout.String(); !strings.Contains(output, `"properties":`) || !strings.Contains(output, `"role": "test"`) {
		t.Fatalf("unexpected aggregate output:\n%s", output)
	}
}

func TestComputeServiceSetUsesServiceIDUpdate(t *testing.T) {
	var sawUpdate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/os-services":
			if got := r.URL.Query().Get("host"); got != "node1" {
				t.Fatalf("host query mismatch: %q", got)
			}
			if got := r.URL.Query().Get("binary"); got != "nova-compute" {
				t.Fatalf("binary query mismatch: %q", got)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"services": []map[string]any{{
				"id": "service-id", "binary": "nova-compute", "host": "node1", "status": "enabled", "state": "up", "zone": "nova", "updated_at": "2026-05-08T01:02:03.000000",
			}}}); err != nil {
				t.Fatalf("encode services: %v", err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/v2.1/project/os-services/service-id":
			sawUpdate = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update: %v", err)
			}
			if body["status"] != "disabled" || body["disabled_reason"] != "maint" || body["forced_down"] != true {
				t.Fatalf("unexpected update body: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"service": map[string]any{"id": "service-id"}}); err != nil {
				t.Fatalf("encode update: %v", err)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	opts := &Options{CommandFlags: map[string]string{"disable": "true", "disable-reason": "maint", "down": "true"}}
	client := testComputeClient(server.URL)
	client.Microversion = "2.53"
	if err := computeServiceSet(context.Background(), opts, client, []string{"node1", "nova-compute"}); err != nil {
		t.Fatalf("compute service set: %v", err)
	}
	if !sawUpdate {
		t.Fatalf("expected service update request")
	}
}

func TestComputeAgentCreateSetDeleteUsesOsAgents(t *testing.T) {
	var sawCreate, sawSet, sawDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/os-agents":
			sawCreate = true
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode agent create: %v", err)
			}
			if body["agent"]["hypervisor"] != "xen" || body["agent"]["version"] != "1.0" {
				t.Fatalf("unexpected agent create body: %#v", body)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"agent": body["agent"]}); err != nil {
				t.Fatalf("encode agent: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/os-agents":
			if err := json.NewEncoder(w).Encode(map[string]any{"agents": []map[string]any{{
				"agent_id": 1, "hypervisor": "xen", "os": "linux", "architecture": "x86_64", "version": "1.0", "url": "http://old", "md5hash": "oldmd5",
			}}}); err != nil {
				t.Fatalf("encode agents: %v", err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/v2.1/project/os-agents/1":
			sawSet = true
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode agent set: %v", err)
			}
			if body["para"]["version"] != "2.0" || body["para"]["url"] != "http://old" || body["para"]["md5hash"] != "oldmd5" {
				t.Fatalf("unexpected agent set body: %#v", body)
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/v2.1/project/os-agents/1":
			sawDelete = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := testComputeClient(server.URL)
	if err := computeAgentCreate(context.Background(), &bytes.Buffer{}, &Options{}, client, []string{"linux", "x86_64", "1.0", "http://agent", "md5", "xen"}); err != nil {
		t.Fatalf("compute agent create: %v", err)
	}
	if err := computeAgentSet(context.Background(), &Options{CommandFlags: map[string]string{"agent-version": "2.0"}}, client, []string{"1"}); err != nil {
		t.Fatalf("compute agent set: %v", err)
	}
	if err := computeAgentDelete(context.Background(), client, []string{"1"}); err != nil {
		t.Fatalf("compute agent delete: %v", err)
	}
	if !sawCreate || !sawSet || !sawDelete {
		t.Fatalf("missing expected agent requests create=%t set=%t delete=%t", sawCreate, sawSet, sawDelete)
	}
}

func TestHostSetUsesDeprecatedHostAPI(t *testing.T) {
	var sawSet bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPut || r.URL.Path != "/v2.1/project/os-hosts/node1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		sawSet = true
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode host set: %v", err)
		}
		if body["status"] != "disable" || body["maintenance_mode"] != "enable" {
			t.Fatalf("unexpected host set body: %#v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := &Options{CommandFlags: map[string]string{"disable": "true", "enable-maintenance": "true"}}
	if err := hostSet(context.Background(), opts, testComputeClient(server.URL), []string{"node1"}); err != nil {
		t.Fatalf("host set: %v", err)
	}
	if !sawSet {
		t.Fatalf("expected host set request")
	}
}

func writeAggregateTestResponse(t *testing.T, w http.ResponseWriter, aggregate map[string]any) {
	t.Helper()
	if _, ok := aggregate["hosts"]; !ok {
		aggregate["hosts"] = []string{}
	}
	if _, ok := aggregate["uuid"]; !ok {
		aggregate["uuid"] = "00000000-0000-0000-0000-000000000007"
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"aggregate": aggregate}); err != nil {
		t.Fatalf("encode aggregate: %v", err)
	}
}

func testImageClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/v2/",
		ResourceBase:   baseURL + "/v2/",
		Type:           "image",
	}
}
