package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
)

type lookupErrorKind string

const (
	lookupNotFound  lookupErrorKind = "not-found"
	lookupAmbiguous lookupErrorKind = "ambiguous"
)

type lookupError struct {
	Kind     lookupErrorKind
	Resource string
	Value    string
	Matches  int
}

func (err lookupError) Error() string {
	resource := err.Resource
	if resource == "" {
		resource = "resource"
	}
	switch err.Kind {
	case lookupAmbiguous:
		return fmt.Sprintf("multiple %s found for %q", resource, err.Value)
	default:
		return fmt.Sprintf("no %s found for %q", resource, err.Value)
	}
}

func newLookupNotFound(resource string, value string) error {
	return lookupError{Kind: lookupNotFound, Resource: resource, Value: value}
}

func newLookupAmbiguous(resource string, value string, matches int) error {
	return lookupError{Kind: lookupAmbiguous, Resource: resource, Value: value, Matches: matches}
}

func isLookupNotFound(err error) bool {
	var lookup lookupError
	return errors.As(err, &lookup) && lookup.Kind == lookupNotFound
}

func isLookupAmbiguous(err error) bool {
	var lookup lookupError
	return errors.As(err, &lookup) && lookup.Kind == lookupAmbiguous
}

type partialFailureError struct {
	Action   string
	Resource string
	Failures int
	Total    int
}

func (err partialFailureError) Error() string {
	action := strings.TrimSpace(err.Action)
	if action == "" {
		action = "process"
	}
	resource := strings.TrimSpace(err.Resource)
	if resource == "" {
		resource = "resources"
	}
	return fmt.Sprintf("Failed to %s %d of %d %s.", action, err.Failures, err.Total, resource)
}

func newPartialFailureError(action string, resource string, failures int, total int) error {
	if failures <= 0 {
		return nil
	}
	return partialFailureError{Action: action, Resource: resource, Failures: failures, Total: total}
}

func compatErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var lookup lookupError
	if errors.As(err, &lookup) {
		return lookup.Error()
	}
	var partial partialFailureError
	if errors.As(err, &partial) {
		return partial.Error()
	}
	var response gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &response) {
		body := strings.TrimSpace(string(response.Body))
		if body == "" {
			return fmt.Sprintf("HTTP %d error from OpenStack API", response.Actual)
		}
		return fmt.Sprintf("HTTP %d error from OpenStack API: %s", response.Actual, body)
	}
	var resourceNotFound gophercloud.ErrResourceNotFound
	if errors.As(err, &resourceNotFound) {
		return resourceNotFound.Error()
	}
	return err.Error()
}
