package cli

import (
	"bytes"
	"testing"
)

func TestOutputJSONTo(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := outputJSONTo(&output, map[string]any{
		"name":    "example",
		"enabled": true,
	}); err != nil {
		t.Fatalf("outputJSONTo() error = %v", err)
	}

	want := "{\n  \"enabled\": true,\n  \"name\": \"example\"\n}\n"
	if got := output.String(); got != want {
		t.Errorf("outputJSONTo() = %q, want %q", got, want)
	}
}

func TestOutputJSONToReturnsEncodingError(t *testing.T) {
	t.Parallel()

	err := outputJSONTo(&bytes.Buffer{}, func() {})
	if err == nil {
		t.Fatal("outputJSONTo() error = nil, want encoding error")
	}
}
