package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDefaultShowsANSIAndASCIITable(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, nil, func(string) string { return "" }); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "ANSI 16") {
		t.Fatalf("default output missing ANSI 16 header:\n%s", text)
	}
	if !strings.Contains(text, "ANSI 256 (16 columns)") {
		t.Fatalf("default output missing ANSI 256 header:\n%s", text)
	}
	if strings.Contains(text, "ANSI 256 + HEX") {
		t.Fatalf("default output should not include hex listing:\n%s", text)
	}
}

func TestRunHexOnlyFromEnv(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, nil, func(name string) string {
		if name == "COLORS_HEX" {
			return "1"
		}
		return ""
	}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	text := out.String()
	if strings.Contains(text, "ANSI 16") || strings.Contains(text, "ANSI 256 (16 columns)") {
		t.Fatalf("hex-only env output included unexpected modes:\n%s", text)
	}
	if !strings.Contains(text, "ANSI 256 + HEX") {
		t.Fatalf("hex-only env output missing hex header:\n%s", text)
	}
}

func TestXterm256HexKnownValues(t *testing.T) {
	cases := map[int]string{
		0:   "#000000",
		15:  "#ffffff",
		16:  "#000000",
		21:  "#0000ff",
		46:  "#00ff00",
		196: "#ff0000",
		231: "#ffffff",
		232: "#080808",
		255: "#eeeeee",
	}
	for index, want := range cases {
		got := xterm256Hex(index)
		if got != want {
			t.Fatalf("xterm256Hex(%d) = %s, want %s", index, got, want)
		}
	}
}

