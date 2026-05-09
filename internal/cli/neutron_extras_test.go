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

func TestNeutronExtrasCommandsAreMarkedImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.network.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		`"default security group rule list"`,
		`"network flavor list"`,
		`"tap service list"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %s to be marked implemented, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"network flavor list (Not Implemented Yet)"`) {
		t.Fatalf("expected network flavor list without not-implemented suffix, got:\n%s", stdout)
	}
}

func TestDefaultSecurityGroupRuleListUsesNeutronEndpoint(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		if r.Method != http.MethodGet {
			t.Fatalf("method mismatch: got %s want GET", r.Method)
		}
		if got, want := r.URL.Path, "/v2.0/default-security-group-rules"; got != want {
			t.Fatalf("path mismatch: got %q want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"default_security_group_rules": []map[string]any{
				{
					"id":                            "rule-1",
					"direction":                     "egress",
					"ethertype":                     "IPv6",
					"protocol":                      nil,
					"remote_ip_prefix":              nil,
					"remote_group_id":               nil,
					"remote_address_group_id":       nil,
					"port_range_min":                nil,
					"port_range_max":                nil,
					"used_in_default_sg":            true,
					"used_in_non_default_sg":        true,
					"should_not_be_in_list_columns": "ignored",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	opts := &Options{Format: "json"}
	err := neutronExtraList(context.Background(), &stdout, opts, nil, testNetworkClient(server.URL), neutronDefaultSecurityGroupRuleSpec(), nil)
	if err != nil {
		t.Fatalf("list default security group rules: %v", err)
	}
	if got, want := len(requests), 1; got != want {
		t.Fatalf("request count mismatch: got %d want %d", got, want)
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if got, want := rows[0]["IP Range"], "::/0"; got != want {
		t.Fatalf("IP Range mismatch: got %#v want %q", got, want)
	}
	if got, want := rows[0]["Port Range"], ""; got != want {
		t.Fatalf("Port Range mismatch: got %#v want %q", got, want)
	}
}

func TestNetworkFlavorCreateUsesNeutronEndpoint(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: got %s want POST", r.Method)
		}
		if got, want := r.URL.Path, "/v2.0/flavors"; got != want {
			t.Fatalf("path mismatch: got %q want %q", got, want)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := map[string]any{
			"flavor": map[string]any{
				"id":           "flavor-1",
				"name":         "vpn-basic",
				"enabled":      true,
				"service_type": "VPN",
				"description":  "basic",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	opts := &Options{
		Format:       "json",
		CommandFlags: map[string]string{"service-type": "VPN", "description": "basic", "enable": "true"},
	}
	var stdout bytes.Buffer
	err := neutronExtraCreate(context.Background(), &stdout, opts, nil, testNetworkClient(server.URL), neutronFlavorSpec(), networkFlavorValues(context.Background(), opts, nil, []string{"vpn-basic"}))
	if err != nil {
		t.Fatalf("create network flavor: %v", err)
	}
	flavor, ok := requestBody["flavor"].(map[string]any)
	if !ok {
		t.Fatalf("request body missing flavor object: %#v", requestBody)
	}
	if got, want := flavor["service_type"], "VPN"; got != want {
		t.Fatalf("service_type mismatch: got %#v want %q", got, want)
	}
}

func testNetworkClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/v2.0/",
		ResourceBase:   baseURL + "/v2.0/",
		Type:           "network",
	}
}
