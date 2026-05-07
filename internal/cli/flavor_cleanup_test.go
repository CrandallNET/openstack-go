package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComputeFlavorCreateUsesNovaCreateExtraSpecsAndShow(t *testing.T) {
	var sawCreate bool
	var sawSpecs bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/flavors":
			sawCreate = true
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			flavor := body["flavor"]
			if flavor["name"] != "test.tiny" || flavor["ram"].(float64) != 512 || flavor["vcpus"].(float64) != 2 {
				t.Fatalf("unexpected create body: %#v", flavor)
			}
			if flavor["os-flavor-access:is_public"] != false {
				t.Fatalf("expected private flavor create body, got %#v", flavor["os-flavor-access:is_public"])
			}
			writeFlavorTestResponse(t, w, map[string]any{
				"id":                         "flavor-id",
				"name":                       "test.tiny",
				"ram":                        512,
				"vcpus":                      2,
				"disk":                       1,
				"rxtx_factor":                1,
				"os-flavor-access:is_public": false,
				"OS-FLV-EXT-DATA:ephemeral":  0,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/flavors/flavor-id/os-extra_specs":
			sawSpecs = true
			var body map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode extra specs body: %v", err)
			}
			if got := body["extra_specs"]["hw:cpu_policy"]; got != "dedicated" {
				t.Fatalf("extra spec mismatch: got %q", got)
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatalf("encode extra specs: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/flavors/flavor-id":
			writeFlavorTestResponse(t, w, map[string]any{
				"id":                         "flavor-id",
				"name":                       "test.tiny",
				"ram":                        512,
				"vcpus":                      2,
				"disk":                       1,
				"rxtx_factor":                1,
				"os-flavor-access:is_public": false,
				"OS-FLV-EXT-DATA:ephemeral":  0,
				"extra_specs":                map[string]string{"hw:cpu_policy": "dedicated"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/flavors/flavor-id/os-extra_specs":
			if err := json.NewEncoder(w).Encode(map[string]any{"extra_specs": map[string]string{"hw:cpu_policy": "dedicated"}}); err != nil {
				t.Fatalf("encode extra specs list: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/flavors/flavor-id/os-flavor-access":
			if err := json.NewEncoder(w).Encode(map[string]any{"flavor_access": []map[string]string{}}); err != nil {
				t.Fatalf("encode access list: %v", err)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	opts := &Options{
		Format: "json",
		CommandFlags: map[string]string{
			"ram":     "512",
			"vcpus":   "2",
			"disk":    "1",
			"private": "true",
		},
		CommandFlagList: map[string][]string{
			"property": {"hw:cpu_policy=dedicated"},
		},
	}
	var stdout bytes.Buffer
	err := computeFlavorCreate(context.Background(), &stdout, opts, testComputeClient(server.URL), nil, []string{"test.tiny"})
	if err != nil {
		t.Fatalf("flavor create: %v", err)
	}
	if !sawCreate || !sawSpecs {
		t.Fatalf("expected create and extra spec requests, sawCreate=%t sawSpecs=%t", sawCreate, sawSpecs)
	}
	if output := stdout.String(); !strings.Contains(output, `"name": "test.tiny"`) || !strings.Contains(output, `"hw:cpu_policy": "dedicated"`) {
		t.Fatalf("unexpected flavor create output:\n%s", output)
	}
}

func TestComputeFlavorDeleteResolvesAndDeletesEachFlavor(t *testing.T) {
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/flavors/flavor-id":
			writeFlavorTestResponse(t, w, map[string]any{
				"id":                         "flavor-id",
				"name":                       "test.tiny",
				"ram":                        512,
				"vcpus":                      2,
				"disk":                       1,
				"os-flavor-access:is_public": false,
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v2.1/project/flavors/flavor-id":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	err := computeFlavorDelete(context.Background(), testComputeClient(server.URL), []string{"flavor-id"})
	if err != nil {
		t.Fatalf("flavor delete: %v", err)
	}
	if !deleted {
		t.Fatalf("expected flavor delete request")
	}
}

func TestComputeFlavorSetAndUnsetExtraSpecs(t *testing.T) {
	var sawSet bool
	var sawUnset bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/flavors/flavor-id":
			writeFlavorTestResponse(t, w, map[string]any{
				"id":                         "flavor-id",
				"name":                       "test.tiny",
				"ram":                        512,
				"vcpus":                      2,
				"disk":                       1,
				"os-flavor-access:is_public": false,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v2.1/project/flavors/flavor-id/os-extra_specs":
			sawSet = true
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"extra_specs":{"hw:cpu_policy":"shared"}}`)); err != nil {
				t.Fatalf("write set response: %v", err)
			}
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/v2.1/project/flavors/flavor-id/os-extra_specs/hw:cpu_policy"):
			sawUnset = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	setOpts := &Options{
		CommandFlagList: map[string][]string{
			"property": {"hw:cpu_policy=shared"},
		},
	}
	if err := computeFlavorSet(context.Background(), setOpts, testComputeClient(server.URL), nil, []string{"flavor-id"}); err != nil {
		t.Fatalf("flavor set: %v", err)
	}
	unsetOpts := &Options{
		CommandFlagList: map[string][]string{
			"property": {"hw:cpu_policy"},
		},
	}
	if err := computeFlavorUnset(context.Background(), unsetOpts, testComputeClient(server.URL), nil, []string{"flavor-id"}); err != nil {
		t.Fatalf("flavor unset: %v", err)
	}
	if !sawSet || !sawUnset {
		t.Fatalf("expected set and unset requests, sawSet=%t sawUnset=%t", sawSet, sawUnset)
	}
}

func TestProjectCleanupComputeResourcesDiscoversServersAndEmptyServerGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/servers/detail":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{
						"id":        "server-id",
						"name":      "vm1",
						"tenant_id": "project-id",
						"created":   "2026-01-01T00:00:00Z",
						"updated":   "2026-01-02T00:00:00Z",
					},
				},
			}); err != nil {
				t.Fatalf("encode servers: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2.1/project/os-server-groups":
			if err := json.NewEncoder(w).Encode(map[string]any{
				"server_groups": []map[string]any{
					{
						"id":         "group-id",
						"name":       "empty-group",
						"project_id": "project-id",
						"members":    []string{},
					},
				},
			}); err != nil {
				t.Fatalf("encode server groups: %v", err)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	resources, err := projectCleanupComputeResources(context.Background(), testComputeClient(server.URL), "project-id", false, projectCleanupFilters{}, nil)
	if err != nil {
		t.Fatalf("project cleanup resources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 cleanup resources, got %#v", resources)
	}
	if resources[0].Type != "Server" || resources[0].ID != "server-id" {
		t.Fatalf("server cleanup resource mismatch: %#v", resources[0])
	}
	if resources[1].Type != "ServerGroup" || resources[1].ID != "group-id" {
		t.Fatalf("server group cleanup resource mismatch: %#v", resources[1])
	}
}

func TestCommandListMarksProjectCleanupImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.common")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"project cleanup"`) {
		t.Fatalf("expected project cleanup to be present, got:\n%s", stdout)
	}
	if strings.Contains(stdout, `"project cleanup (Not Implemented Yet)"`) {
		t.Fatalf("expected project cleanup without not-implemented suffix, got:\n%s", stdout)
	}
}

func writeFlavorTestResponse(t *testing.T, w http.ResponseWriter, flavor map[string]any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{"flavor": flavor}); err != nil {
		t.Fatalf("encode flavor: %v", err)
	}
}
