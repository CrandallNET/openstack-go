package main

import (
	"os"
	"strings"
	"testing"
)

func TestCompareResultsRequiresExitStdoutAndStderrParity(t *testing.T) {
	base := commandResult{Stdout: "out\n", Stderr: "err\n", ExitCode: 2}
	if matched, diff := compareResults(base, base); !matched || diff != "" {
		t.Fatalf("expected identical results to match, matched=%v diff=%q", matched, diff)
	}

	changedExit := base
	changedExit.ExitCode = 1
	if matched, diff := compareResults(base, changedExit); matched || !strings.Contains(diff, "exit code") {
		t.Fatalf("expected exit code mismatch, matched=%v diff=%q", matched, diff)
	}

	changedStdout := base
	changedStdout.Stdout = "different\n"
	if matched, diff := compareResults(base, changedStdout); matched || !strings.Contains(diff, "stdout line 1") {
		t.Fatalf("expected stdout mismatch, matched=%v diff=%q", matched, diff)
	}

	changedStderr := base
	changedStderr.Stderr = "different\n"
	if matched, diff := compareResults(base, changedStderr); matched || !strings.Contains(diff, "stderr line 1") {
		t.Fatalf("expected stderr mismatch, matched=%v diff=%q", matched, diff)
	}
}

func TestRequiredFailuresIgnoresKnownGaps(t *testing.T) {
	results := []checkResult{
		{Case: checkCase{Name: "pass"}, Matched: true},
		{Case: checkCase{Name: "known", KnownGap: true}, Matched: false},
		{Case: checkCase{Name: "skip", Skip: true}, Skipped: true},
		{Case: checkCase{Name: "required"}, Matched: false},
	}
	if got := requiredFailures(results); got != 1 {
		t.Fatalf("expected one required failure, got %d", got)
	}
}

func TestSplitCommaTrimsEmptyParts(t *testing.T) {
	got := splitComma("flavor list, image list ,, server list")
	want := []string{"flavor list", "image list", "server list"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("split mismatch: got %v want %v", got, want)
	}
}

func TestReadCommandsSortsFlattenedCatalog(t *testing.T) {
	path := t.TempDir() + "/commands.json"
	data := `[
  {"Command Group": "g2", "Commands": ["server list"]},
  {"Command Group": "g1", "Commands": ["command list", "aggregate list"]}
]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	commands, err := readCommands(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"aggregate list", "command list", "server list"}
	if strings.Join(commands, "|") != strings.Join(want, "|") {
		t.Fatalf("commands mismatch: got %v want %v", commands, want)
	}
}

func TestFirstFixtureIDUsesFirstIDOrName(t *testing.T) {
	got, err := firstFixtureID(`[{"Name":"alpha"},{"ID":"server-id","Name":"beta"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "alpha" {
		t.Fatalf("expected first available name, got %q", got)
	}

	got, err = firstFixtureID(`[{"ID":"server-id","Name":"beta"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "server-id" {
		t.Fatalf("expected ID, got %q", got)
	}
}

func TestFixtureResolverReplacesAliases(t *testing.T) {
	resolver := &fixtureResolver{cache: map[string]fixtureLookup{
		"server": {Value: "server-id"},
	}}
	args, skip := resolver.resolveArgs([]string{"server", "show", "{server_id}"})
	if skip != "" {
		t.Fatalf("unexpected skip: %s", skip)
	}
	if strings.Join(args, " ") != "server show server-id" {
		t.Fatalf("resolved args mismatch: %v", args)
	}
}

func TestFixtureResolverSkipsUnsupportedPlaceholders(t *testing.T) {
	resolver := &fixtureResolver{cache: map[string]fixtureLookup{}}
	_, skip := resolver.resolveArgs([]string{"server", "show", "{unsupported}"})
	if !strings.Contains(skip, "unsupported live fixture placeholder") {
		t.Fatalf("expected unsupported placeholder skip, got %q", skip)
	}
}
