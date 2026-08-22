package model

import (
	"fmt"
	"strings"
)

// HookConfig is one harness-owned lifecycle hook declaration.
// Commands are behavior-bearing native configuration and are never executed by SkillSync.
type HookConfig struct {
	Name       string   `json:"name"`
	Platform   Platform `json:"platform"`
	Event      string   `json:"event"`
	Matcher    string   `json:"matcher,omitempty"`
	Command    string   `json:"command"`
	Timeout    int      `json:"timeout,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	SourcePath string   `json:"source_path,omitempty"`
	MappingKey string   `json:"mapping_key,omitempty"`
}

// Validate checks the portable declaration boundary without executing the hook.
func (h HookConfig) Validate() error {
	if strings.TrimSpace(h.Name) == "" {
		return fmt.Errorf("hook name is required")
	}
	if !h.Platform.IsValid() {
		return fmt.Errorf("unsupported hook platform %q", h.Platform)
	}
	if strings.TrimSpace(h.Event) == "" {
		return fmt.Errorf("hook %q event is required", h.Name)
	}
	if strings.TrimSpace(h.Command) == "" {
		return fmt.Errorf("hook %q command is required", h.Name)
	}
	if h.Timeout < 0 {
		return fmt.Errorf("hook %q timeout must not be negative", h.Name)
	}
	return nil
}

// Key returns a stable native identity.
func (h HookConfig) Key() string {
	return string(h.Platform) + ":hook:" + strings.TrimSpace(h.Event) + ":" + strings.TrimSpace(h.Name)
}
