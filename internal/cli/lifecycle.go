package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const lifecycleDefaultPrefix = "golang-osc-test"

type lifecycleRun struct {
	ID             string                       `json:"id"`
	Prefix         string                       `json:"prefix"`
	StartedAt      string                       `json:"started_at"`
	Fixtures       map[string]any               `json:"fixtures,omitempty"`
	CleanupResults []lifecycleCleanupResult     `json:"cleanup_results,omitempty"`
	cleanups       []registeredLifecycleCleanup `json:"-"`
}

type registeredLifecycleCleanup struct {
	Label string
	Fn    func(context.Context) error
}

type lifecycleCleanupResult struct {
	Label string `json:"label"`
	Error string `json:"error,omitempty"`
}

func newLifecycleRun(prefix string) (*lifecycleRun, error) {
	if strings.TrimSpace(prefix) == "" {
		prefix = lifecycleDefaultPrefix
	}
	suffix, err := randomHex(8)
	if err != nil {
		return nil, err
	}
	id := prefix + "-" + suffix
	return &lifecycleRun{
		ID:        id,
		Prefix:    prefix,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures:  map[string]any{},
	}, nil
}

func randomHex(byteCount int) (string, error) {
	if byteCount <= 0 {
		byteCount = 8
	}
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (run *lifecycleRun) resourceName(parts ...string) string {
	values := []string{run.ID}
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "-")
		if part != "" {
			values = append(values, part)
		}
	}
	return strings.Join(values, "-")
}

func (run *lifecycleRun) recordFixture(name string, value any) {
	if run.Fixtures == nil {
		run.Fixtures = map[string]any{}
	}
	run.Fixtures[name] = value
}

func (run *lifecycleRun) addCleanup(label string, fn func(context.Context) error) {
	if fn == nil {
		return
	}
	run.cleanups = append(run.cleanups, registeredLifecycleCleanup{Label: label, Fn: fn})
}

func (run *lifecycleRun) cleanup(ctx context.Context) error {
	var failures []error
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		cleanup := run.cleanups[i]
		result := lifecycleCleanupResult{Label: cleanup.Label}
		if err := cleanup.Fn(ctx); err != nil {
			result.Error = err.Error()
			failures = append(failures, fmt.Errorf("%s: %w", cleanup.Label, err))
		}
		run.CleanupResults = append(run.CleanupResults, result)
	}
	return errors.Join(failures...)
}

func (run *lifecycleRun) writeDiagnostics(path string) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
