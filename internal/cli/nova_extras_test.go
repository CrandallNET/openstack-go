package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"golang.org/x/crypto/ssh"
)

func TestNovaExtrasCommandsAreMarkedImplemented(t *testing.T) {
	stdout, stderr, err := executeForTest("command", "list", "-f", "json", "--group", "openstack.compute.v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"server ssh"`) {
		t.Fatalf("expected server ssh to be present, got:\n%s", stdout)
	}
	if strings.Contains(stdout, `"server ssh (Not Implemented Yet)"`) {
		t.Fatalf("expected server ssh without not-implemented suffix, got:\n%s", stdout)
	}
}

func TestModuleListIncludesNovaExtras(t *testing.T) {
	stdout, stderr, err := executeForTest("module", "list", "-f", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"openstack.commands.extras.nova-extras"`) {
		t.Fatalf("expected nova extras module in module list, got:\n%s", stdout)
	}
}

func TestServerSSHHelpListsPureGoPassThroughOptions(t *testing.T) {
	stdout, stderr, err := executeForTest("server", "ssh", "--help")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"usage: openstack server ssh",
		"Pure Go SSH pass-through options:",
		"-A                                      enable ssh-agent forwarding",
		"-o StrictHostKeyChecking=yes|no|ask|accept-new|off",
		"OS_SSH_USER=<login-name>",
		"<remote-command> [args ...]",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("server ssh help missing %q:\n%s", want, stdout)
		}
	}
}

func TestBuildServerSSHRequestUsesPythonAddressSelection(t *testing.T) {
	t.Setenv("OS_SSH_USER", "")
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"public": []any{
				map[string]any{"addr": "203.0.113.10", "version": float64(4), "OS-EXT-IPS:type": "fixed"},
			},
			"tenant-net": []any{
				map[string]any{"addr": "10.0.0.5", "version": float64(4), "OS-EXT-IPS:type": "fixed"},
				map[string]any{"addr": "2001:db8::5", "version": float64(6), "OS-EXT-IPS:type": "fixed"},
				map[string]any{"addr": "198.51.100.5", "version": float64(4), "OS-EXT-IPS:type": "floating"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{"private": "true", "ipv6": "true"}}
	request, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1"})
	if err != nil {
		t.Fatalf("build server ssh request: %v", err)
	}
	if got, want := request.Address, "2001:db8::5"; got != want {
		t.Fatalf("address mismatch: got %q want %q", got, want)
	}
	if got, want := request.User, "cloud"; got != want {
		t.Fatalf("user mismatch: got %q want %q", got, want)
	}
}

func TestBuildServerSSHRequestUsesOSSSHUser(t *testing.T) {
	t.Setenv("OS_SSH_USER", "rocky")
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"public": []any{
				map[string]any{"addr": "203.0.113.10", "version": float64(4), "OS-EXT-IPS:type": "floating"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{}}
	request, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1"})
	if err != nil {
		t.Fatalf("build server ssh request: %v", err)
	}
	if got, want := request.User, "rocky"; got != want {
		t.Fatalf("user mismatch: got %q want %q", got, want)
	}
}

func TestBuildServerSSHRequestPassThroughLoginOverridesOSSSHUser(t *testing.T) {
	t.Setenv("OS_SSH_USER", "rocky")
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"public": []any{
				map[string]any{"addr": "203.0.113.10", "version": float64(4), "OS-EXT-IPS:type": "floating"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{}}
	request, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1", "-l", "ubuntu"})
	if err != nil {
		t.Fatalf("build server ssh request: %v", err)
	}
	if got, want := request.User, "ubuntu"; got != want {
		t.Fatalf("user mismatch: got %q want %q", got, want)
	}
}

func TestBuildServerSSHRequestFallsBackToFixedAddressForImplicitPublic(t *testing.T) {
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"tenant-net": []any{
				map[string]any{"addr": "10.0.0.5", "version": float64(4), "OS-EXT-IPS:type": "fixed"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{}}
	request, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1"})
	if err != nil {
		t.Fatalf("build server ssh request: %v", err)
	}
	if got, want := request.Address, "10.0.0.5"; got != want {
		t.Fatalf("address mismatch: got %q want %q", got, want)
	}
}

func TestBuildServerSSHRequestDoesNotFallbackForExplicitPublic(t *testing.T) {
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"tenant-net": []any{
				map[string]any{"addr": "10.0.0.5", "version": float64(4), "OS-EXT-IPS:type": "fixed"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{"public": "true"}}
	_, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1"})
	if err == nil {
		t.Fatal("expected explicit public address selection to fail")
	}
	if !strings.Contains(err.Error(), "No public IP version [4 6] address found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildServerSSHRequestDoesNotFallbackForAddressTypePublic(t *testing.T) {
	server := &servers.Server{
		Name: "vm1",
		Addresses: map[string]any{
			"tenant-net": []any{
				map[string]any{"addr": "10.0.0.5", "version": float64(4), "OS-EXT-IPS:type": "fixed"},
			},
		},
	}
	opts := &Options{CommandFlags: map[string]string{"address-type": "public"}}
	_, err := buildServerSSHRequest(&bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, server, []string{"vm1"})
	if err == nil {
		t.Fatal("expected explicit address-type public selection to fail")
	}
	if !strings.Contains(err.Error(), "No public IP version [4 6] address found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseServerSSHPassThroughOptions(t *testing.T) {
	opts := &Options{CommandFlags: map[string]string{
		"address-type": "private",
	}}
	invocation, err := parseServerSSHInvocation(opts, gophercloud.AuthOptions{Username: "cloud"}, []string{
		"vm1",
		"-l", "ubuntu",
		"-p2222",
		"-i", "~/.ssh/id_test",
		"-o", "StrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-A",
		"uptime",
	})
	if err != nil {
		t.Fatalf("parse server ssh invocation: %v", err)
	}
	if got, want := invocation.login, "ubuntu"; got != want {
		t.Fatalf("login mismatch: got %q want %q", got, want)
	}
	if got, want := invocation.port, 2222; got != want {
		t.Fatalf("port mismatch: got %d want %d", got, want)
	}
	if got, want := invocation.addressType, "private"; got != want {
		t.Fatalf("address type mismatch: got %q want %q", got, want)
	}
	if got, want := invocation.strictHostKeyChecking, "no"; got != want {
		t.Fatalf("strict host key checking mismatch: got %q want %q", got, want)
	}
	if got, want := strings.Join(invocation.identityFiles, ","), "~/.ssh/id_test"; got != want {
		t.Fatalf("identity files mismatch: got %q want %q", got, want)
	}
	if got, want := strings.Join(invocation.knownHostsFiles, ","), "/dev/null"; got != want {
		t.Fatalf("known hosts mismatch: got %q want %q", got, want)
	}
	if !invocation.forwardAgent {
		t.Fatal("expected agent forwarding")
	}
	if got, want := strings.Join(invocation.remoteCommand, " "), "uptime"; got != want {
		t.Fatalf("remote command mismatch: got %q want %q", got, want)
	}
}

func TestServerSSHUsesInjectedPureGoRunner(t *testing.T) {
	oldRunner := runServerSSHSession
	defer func() { runServerSSHSession = oldRunner }()
	var captured serverSSHRequest
	runServerSSHSession = func(ctx context.Context, request serverSSHRequest) error {
		captured = request
		return nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v2.1/project/servers/server-id"; got != want {
			t.Fatalf("request path mismatch: got %q want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"server": map[string]any{
				"id":   "server-id",
				"name": "vm1",
				"addresses": map[string]any{
					"public": []any{map[string]any{"addr": "203.0.113.10", "version": 4}},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()
	opts := &Options{CommandFlags: map[string]string{}}
	if err := serverSSH(context.Background(), &bytes.Buffer{}, opts, gophercloud.AuthOptions{Username: "cloud"}, testComputeClient(server.URL), []string{"server-id", "-l", "ubuntu"}); err != nil {
		t.Fatalf("server ssh: %v", err)
	}
	if got, want := captured.Address, "203.0.113.10"; got != want {
		t.Fatalf("captured address mismatch: got %q want %q", got, want)
	}
	if got, want := captured.User, "ubuntu"; got != want {
		t.Fatalf("captured user mismatch: got %q want %q", got, want)
	}
}

func testComputeClient(baseURL string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       baseURL + "/v2.1/project/",
		ResourceBase:   baseURL + "/v2.1/project/",
		Type:           "compute",
	}
}

func TestRunPureGoSSHSessionExecutesRemoteCommand(t *testing.T) {
	clientKeyPath := writeTestSSHPrivateKey(t)
	address, closeServer := startTestSSHServer(t)
	defer closeServer()
	host, portValue, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}
	var stdout bytes.Buffer
	request := serverSSHRequest{
		Address:               host,
		Port:                  port,
		User:                  "test-user",
		IdentityFiles:         []string{clientKeyPath},
		StrictHostKeyChecking: "no",
		RemoteCommand:         []string{"echo", "ok"},
		Stdin:                 &bytes.Buffer{},
		Stdout:                &stdout,
		Stderr:                &bytes.Buffer{},
	}
	if err := runPureGoSSHSession(context.Background(), request); err != nil {
		t.Fatalf("run pure go ssh session: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "ran: echo ok"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

func writeTestSSHPrivateKey(t *testing.T) string {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write client key: %v", err)
	}
	return path
}

func startTestSSHServer(t *testing.T) (string, func()) {
	t.Helper()
	_, hostPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPrivateKey)
	if err != nil {
		t.Fatalf("create host signer: %v", err)
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
	}
	config.AddHostKey(hostSigner)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for test SSH server: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, channels, requests, err := ssh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		go ssh.DiscardRequests(requests)
		for newChannel := range channels {
			if newChannel.ChannelType() != "session" {
				_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
				continue
			}
			channel, channelRequests, err := newChannel.Accept()
			if err != nil {
				return
			}
			handleTestSSHSession(channel, channelRequests)
			return
		}
	}()
	return listener.Addr().String(), func() {
		_ = listener.Close()
		<-done
	}
}

func handleTestSSHSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	for request := range requests {
		switch request.Type {
		case "exec":
			var payload struct {
				Command string
			}
			ssh.Unmarshal(request.Payload, &payload)
			_ = request.Reply(true, nil)
			_, _ = io.WriteString(channel, "ran: "+payload.Command+"\n")
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
			return
		default:
			_ = request.Reply(false, nil)
		}
	}
}
