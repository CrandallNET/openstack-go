package cli

import (
	"bytes"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	"strings"
	"testing"
)

func TestCompatErrorMessageFormatsKnownErrors(t *testing.T) {
	notFound := newLookupNotFound("resource", "missing")
	if got := compatErrorMessage(notFound); got != `no resource found for "missing"` {
		t.Fatalf("lookup error message mismatch: %q", got)
	}
	partial := newPartialFailureError("delete", "images", 2, 3)
	if got := compatErrorMessage(partial); got != "Failed to delete 2 of 3 images." {
		t.Fatalf("partial failure message mismatch: %q", got)
	}
	if got := compatErrorMessage(errors.New("plain")); got != "plain" {
		t.Fatalf("plain error message mismatch: %q", got)
	}
}
func TestDefaultShowFormatsOSCEmptyAndNoneValues(t *testing.T) {
	type projectOption string
	var stdout bytes.Buffer
	err := renderShowOutput(&stdout, &Options{Format: defaultOutputFormat}, []outputField{
		{Name: "nil_value", Value: nil},
		{Name: "empty_slice", Value: []string{}},
		{Name: "empty_typed_map", Value: map[projectOption]any{}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"| nil_value       | None  |",
		"| empty_slice     | []    |",
		"| empty_typed_map | {}    |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("default show output missing %q:\n%s", want, output)
		}
	}
}
func TestOSCHTTPExceptionFormatsOpenStackFault(t *testing.T) {
	err := oscHTTPException(gophercloud.ErrUnexpectedResponseCode{
		URL:    "http://example.test/v2.1/os-agents",
		Method: "GET",
		Actual: 410,
		Body:   []byte(`{"computeFault":{"code":410,"message":"This resource is no longer available. No forwarding address is given."}}`),
	})
	if got, want := err.Error(), "HttpException: 410: Client Error for url: http://example.test/v2.1/os-agents, This resource is no longer available. No forwarding address is given."; got != want {
		t.Fatalf("HTTP exception mismatch: got %q want %q", got, want)
	}
}
func TestOSCResourceNotFoundFormatsSDKLookupError(t *testing.T) {
	err := oscResourceNotFoundError(gophercloud.ErrUnexpectedResponseCode{
		URL:    "http://example.test/v2.1/os-console-auth-tokens/bad-token",
		Method: "GET",
		Actual: 404,
		Body:   []byte(`{"itemNotFound":{"code":404,"message":"Token not found"}}`),
	}, "ConsoleAuthToken", "bad-token")
	if got, want := err.Error(), "No ConsoleAuthToken found for bad-token: Client Error for url: http://example.test/v2.1/os-console-auth-tokens/bad-token, Token not found"; got != want {
		t.Fatalf("resource not found mismatch: got %q want %q", got, want)
	}
}
func TestOpenStackFaultMessageFormatsFlatGlanceError(t *testing.T) {
	body := []byte(`{"message":"Caching via API is not supported at this site.<br /><br />\n\n\n","code":"404 Not Found","title":"Not Found"}`)
	if got, want := openStackFaultMessage(body), "404 Not Found: Caching via API is not supported at this site."; got != want {
		t.Fatalf("fault message mismatch: got %q want %q", got, want)
	}
}
func TestSingleMatchReturnsTypedLookupErrors(t *testing.T) {
	if _, err := singleMatch("missing", []string{}); !isLookupNotFound(err) {
		t.Fatalf("expected typed not-found lookup error, got %T %[1]v", err)
	}
	if _, err := singleMatch("duplicate", []string{"a", "b"}); !isLookupAmbiguous(err) {
		t.Fatalf("expected typed ambiguous lookup error, got %T %[1]v", err)
	}
}
