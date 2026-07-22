package ui

import "testing"

func TestFormatSize(t *testing.T) {
	tests := map[string]struct {
		bytes int64
		want  string
	}{
		"negative":       {bytes: -1, want: "-1 B"},
		"zero":           {bytes: 0, want: "0 B"},
		"bytes":          {bytes: 500, want: "500 B"},
		"below kilobyte": {bytes: 1023, want: "1023 B"},
		"kilobyte":       {bytes: 1024, want: "1.0 KB"},
		"kilobytes":      {bytes: 1536, want: "1.5 KB"},
		"megabyte":       {bytes: 1024 * 1024, want: "1.0 MB"},
		"megabytes":      {bytes: 1536 * 1024, want: "1.5 MB"},
		"gigabyte":       {bytes: 1024 * 1024 * 1024, want: "1.0 GB"},
		"gigabytes":      {bytes: 1536 * 1024 * 1024, want: "1.5 GB"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := FormatSize(tt.bytes); got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
