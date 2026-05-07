package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestLifecycleSuiteHelpIncludesAllSuites(t *testing.T) {
	help := lifecycleSuiteHelp()
	for _, suite := range append(append([]string(nil), lifecycleSuites...), "all") {
		if !strings.Contains(help, suite) {
			t.Fatalf("expected lifecycle suite help to include %q, got %q", suite, help)
		}
	}
}

func TestRunLifecycleSuiteRejectsUnknownSuite(t *testing.T) {
	if _, err := runLifecycleSuite("bogus", "cloud", "prefix"); err == nil {
		t.Fatalf("expected unknown lifecycle suite to fail")
	}
}

func TestServerSSHPassThroughArgs(t *testing.T) {
	got := serverSSHPassThroughArgs("/tmp/id_rsa", "cirros", "whoami")
	want := []string{
		"--",
		"-i", "/tmp/id_rsa",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "LogLevel=ERROR",
		"-l", "cirros",
		"whoami",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pass-through args mismatch:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestServerSSHUserForImage(t *testing.T) {
	tests := map[string]string{
		"cirros-0.6.2":          "cirros",
		"Ubuntu 24.04":          "ubuntu",
		"Debian 12":             "debian",
		"openSUSE Leap":         "opensuse",
		"FreeBSD 14":            "freebsd",
		"Arch Linux":            "arch",
		"Rocky Linux 9 Generic": "rocky",
	}
	for image, want := range tests {
		if got := serverSSHUserForImage(image); got != want {
			t.Fatalf("serverSSHUserForImage(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestServerSSHUserCandidatesForImage(t *testing.T) {
	got := serverSSHUserCandidatesForImage("rocky10")
	want := []string{"rocky", "cloud-user"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("serverSSHUserCandidatesForImage() = %#v, want %#v", got, want)
	}
}

func TestServerLifecycleNetworkScorePrefersReachableNonTestNetwork(t *testing.T) {
	if serverLifecycleNetworkScore("os6-lan") <= serverLifecycleNetworkScore("golang-osc-test-deadbeef-net") {
		t.Fatal("expected os6-lan to score higher than test-created network")
	}
	if serverLifecycleNetworkScore("testNet") <= serverLifecycleNetworkScore("golang-osc-test-deadbeef-oracle-net") {
		t.Fatal("expected named project network to score higher than leftover oracle test network")
	}
}

func TestServerLifecycleImageScorePrefersSSHCapableImage(t *testing.T) {
	if serverLifecycleImageScore("rocky9") <= serverLifecycleImageScore("cirros") {
		t.Fatal("expected rocky image to score higher than cirros for SSH lifecycle")
	}
}
