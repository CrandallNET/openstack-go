package cli

import (
	"bytes"
	"context"
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

