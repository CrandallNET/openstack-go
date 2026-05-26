package cli

import (
	"bytes"
	"strings"
	"testing"

)

func TestDefaultListRightAlignsNumericColumns(t *testing.T) {
	var stdout bytes.Buffer
	err := renderListOutput(&stdout, &Options{Format: defaultOutputFormat}, []string{"ID", "Name", "RAM", "Disk"}, []outputRow{
		{"ID": "0", "Name": "m1.tiny", "RAM": 512, "Disk": 10},
		{"ID": "1", "Name": "m1.small", "RAM": 1024, "Disk": 10},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"| ID | Name     |  RAM | Disk |",
		"| 0  | m1.tiny  |  512 |   10 |",
		"| 1  | m1.small | 1024 |   10 |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("default list output missing numeric alignment %q:\n%s", want, output)
		}
	}
}
func TestMaxWidthWrapsTableOutput(t *testing.T) {
	stdout, stderr, err := executeForTest("module", "list", "--max-width", "52")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := maxOutputLineLength(stdout); got > 52 {
		t.Fatalf("expected all table lines to fit within 52 columns, longest was %d:\n%s", got, stdout)
	}
}
