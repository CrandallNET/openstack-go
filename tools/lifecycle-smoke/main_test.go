package main

import (
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
