package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNativePackageValidate(t *testing.T) {
	t.Parallel()

	valid := NativePackage{
		Name:       "commits@klauern-skills",
		Kind:       NativePackagePlugin,
		Platform:   ClaudeCode,
		Provenance: NativeProvenanceMarketplace,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name    string
		change  func(*NativePackage)
		wantErr string
	}{
		{name: "missing name", change: func(p *NativePackage) { p.Name = " " }, wantErr: "name is required"},
		{name: "unsupported kind", change: func(p *NativePackage) { p.Kind = "theme" }, wantErr: "kind"},
		{name: "unsupported platform", change: func(p *NativePackage) { p.Platform = "other" }, wantErr: "platform"},
		{name: "unsupported provenance", change: func(p *NativePackage) { p.Provenance = "unknown" }, wantErr: "provenance"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := valid
			tt.change(&got)
			err := got.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestNativePackageKey(t *testing.T) {
	t.Parallel()

	pkg := NativePackage{Name: "  example/tool  ", Kind: NativePackageExtension, Platform: Gemini}
	if got, want := pkg.Key(), "gemini:extension:example/tool"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
}

func TestNativePackageJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := NativePackage{
		Name:       "example/tool",
		Kind:       NativePackagePackage,
		Platform:   Pi,
		Provenance: NativeProvenanceRepository,
		Version:    "v1.2.3",
		Source:     "https://example.com/tool.git",
		Path:       "/tmp/packages/tool",
		Scope:      "user",
		Enabled:    true,
		MappingKey: "example-tool",
		Metadata:   map[string]string{"revision": "abc123"},
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got NativePackage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}
