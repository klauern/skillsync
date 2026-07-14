package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"config show default": {
			args:       []string{"skillsync", "config"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show subcommand": {
			args:       []string{"skillsync", "config", "show"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show yaml format": {
			args:       []string{"skillsync", "config", "show", "--format", "yaml"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show json format": {
			args:       []string{"skillsync", "config", "show", "--format", "json"},
			wantErr:    false,
			wantOutput: "Platforms",
		},
		"config show short flag": {
			args:       []string{"skillsync", "config", "show", "-f", "json"},
			wantErr:    false,
			wantOutput: "Platforms",
		},
		"config show invalid format": {
			args:    []string{"skillsync", "config", "show", "--format", "invalid"},
			wantErr: true,
		},
		"config path subcommand": {
			args:       []string{"skillsync", "config", "path"},
			wantErr:    false,
			wantOutput: "Configuration paths",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var err error
			output := captureStdout(t, func() {
				err = Run(context.Background(), tt.args)
			})

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestConfigInitCommand(t *testing.T) {
	tests := map[string]struct {
		setup      func(t *testing.T) string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"init creates config in temp dir": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				return tempDir
			},
			args:       []string{"skillsync", "config", "init"},
			wantErr:    false,
			wantOutput: "Created config file",
		},
		"init fails without force when config exists": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				// Create existing config
				configDir := filepath.Join(tempDir, ".skillsync")
				if err := os.MkdirAll(configDir, 0o750); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}
				configPath := filepath.Join(configDir, "config.yaml")
				if err := os.WriteFile(configPath, []byte("existing: config"), 0o600); err != nil {
					t.Fatalf("failed to write existing config: %v", err)
				}
				return tempDir
			},
			args:    []string{"skillsync", "config", "init"},
			wantErr: true,
		},
		"init with force overwrites existing config": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				// Create existing config
				configDir := filepath.Join(tempDir, ".skillsync")
				if err := os.MkdirAll(configDir, 0o750); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}
				configPath := filepath.Join(configDir, "config.yaml")
				if err := os.WriteFile(configPath, []byte("existing: config"), 0o600); err != nil {
					t.Fatalf("failed to write existing config: %v", err)
				}
				return tempDir
			},
			args:       []string{"skillsync", "config", "init", "--force"},
			wantErr:    false,
			wantOutput: "Created config file",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_ = tt.setup(t)

			var err error
			output := captureStdout(t, func() {
				err = Run(context.Background(), tt.args)
			})

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestConfigEditCommand(t *testing.T) {
	tests := map[string]struct {
		setup      func(t *testing.T)
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"edit with EDITOR set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "echo")
			},
			args:    []string{"skillsync", "config", "edit"},
			wantErr: false,
		},
		"edit with VISUAL set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "")
				t.Setenv("VISUAL", "echo")
			},
			args:    []string{"skillsync", "config", "edit"},
			wantErr: false,
		},
		"edit without editor set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "")
				t.Setenv("VISUAL", "")
			},
			args:    []string{"skillsync", "config", "edit"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.setup(t)

			var err error
			output := captureStdout(t, func() {
				err = Run(context.Background(), tt.args)
			})

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}
