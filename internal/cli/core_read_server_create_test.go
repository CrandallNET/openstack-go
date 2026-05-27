package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestServerCreateRecoversNameFromWaitValue(t *testing.T) {
	opts := &Options{
		CommandFlags: map[string]string{
			"wait": "test-vm",
		},
	}

	err := serverCreate(context.Background(), &bytes.Buffer{}, opts, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected an error")
	}
	if !strings.Contains(err.Error(), "argument --flavor is required") {
		t.Fatalf("expected flavor-required error after wait-name recovery, got: %v", err)
	}
}

func TestServerCreateStillRequiresNameWithBooleanWait(t *testing.T) {
	opts := &Options{
		CommandFlags: map[string]string{
			"wait": "true",
		},
	}

	err := serverCreate(context.Background(), &bytes.Buffer{}, opts, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected an error")
	}
	if !strings.Contains(err.Error(), "server create requires <server-name>") {
		t.Fatalf("expected missing server-name error, got: %v", err)
	}
}

func TestRecoverPositionalFromWait(t *testing.T) {
	t.Run("keeps existing args", func(t *testing.T) {
		opts := &Options{CommandFlags: map[string]string{"wait": "name"}}
		got := recoverPositionalFromWait(opts, []string{"server-a"})
		if !reflect.DeepEqual(got, []string{"server-a"}) {
			t.Fatalf("unexpected args: %#v", got)
		}
		if opts.CommandFlags["wait"] != "name" {
			t.Fatalf("wait flag unexpectedly changed: %q", opts.CommandFlags["wait"])
		}
	})

	t.Run("recovers arg from wait value", func(t *testing.T) {
		opts := &Options{CommandFlags: map[string]string{"wait": "server-b"}}
		got := recoverPositionalFromWait(opts, nil)
		if !reflect.DeepEqual(got, []string{"server-b"}) {
			t.Fatalf("unexpected recovered args: %#v", got)
		}
		if opts.CommandFlags["wait"] != "true" {
			t.Fatalf("expected wait flag normalized to true, got %q", opts.CommandFlags["wait"])
		}
	})

	t.Run("does not recover boolean wait values", func(t *testing.T) {
		for _, waitValue := range []string{"true", "false", ""} {
			opts := &Options{CommandFlags: map[string]string{"wait": waitValue}}
			got := recoverPositionalFromWait(opts, nil)
			if len(got) != 0 {
				t.Fatalf("expected no recovery for wait=%q, got %#v", waitValue, got)
			}
		}
	})
}
