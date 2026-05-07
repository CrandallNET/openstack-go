package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestCinderExtrasCommandsAreMarkedImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.volume.v3")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{`"block storage resource filter list"`, `"block storage resource filter show"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %s to be marked implemented, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"block storage resource filter list (Not Implemented Yet)"`) {
		t.Fatalf("expected resource filter list without not-implemented suffix, got:\n%s", stdout)
	}
}

func TestModuleListIncludesCinderExtras(t *testing.T) {
	stdout, stderr, err := executeForTest("module", "list", "-f", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"openstack.commands.extras.cinder-extras"`) {
		t.Fatalf("expected cinder extras module in module list, got:\n%s", stdout)
	}
}

func TestListBlockStorageResourceFiltersUsesCinderEndpoint(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		if r.Method != http.MethodGet {
			t.Fatalf("method mismatch: got %s want GET", r.Method)
		}
		if got, want := r.Header.Get("X-OpenStack-Volume-API-Version"), "3.33"; got != want {
			t.Fatalf("volume microversion header mismatch: got %q want %q", got, want)
		}
		if got, want := r.Header.Get("OpenStack-API-Version"), "volume 3.33"; got != want {
			t.Fatalf("generic microversion header mismatch: got %q want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"resource_filters": []map[string]any{
				{"resource": "volume", "filters": []string{"name", "status"}},
				{"resource": "backup", "filters": []string{"name"}},
			},
		}
		if r.URL.Query().Get("resource") == "volume" {
			response["resource_filters"] = []map[string]any{
				{"resource": "volume", "filters": []string{"name", "status"}},
			}
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := testBlockStorageClient(server.URL)
	items, err := listBlockStorageResourceFilters(context.Background(), client, "")
	if err != nil {
		t.Fatalf("list resource filters: %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("resource filter count mismatch: got %d want %d", got, want)
	}

	filtered, err := listBlockStorageResourceFilters(context.Background(), client, "volume")
	if err != nil {
		t.Fatalf("show resource filters: %v", err)
	}
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("filtered resource filter count mismatch: got %d want %d", got, want)
	}
	if got, want := filtered[0].Resource, "volume"; got != want {
		t.Fatalf("filtered resource mismatch: got %q want %q", got, want)
	}
	if len(requests) != 2 {
		t.Fatalf("request count mismatch: got %d want 2", len(requests))
	}
	if got, want := requests[0], "/v3/project/resource_filters"; got != want {
		t.Fatalf("list request path mismatch: got %q want %q", got, want)
	}
	if got, want := requests[1], "/v3/project/resource_filters?resource=volume"; got != want {
		t.Fatalf("show request path mismatch: got %q want %q", got, want)
	}
}

func testBlockStorageClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/v3/project/",
		ResourceBase:   baseURL + "/v3/project/",
		Type:           "volumev3",
		Microversion:   "3.33",
	}
}
