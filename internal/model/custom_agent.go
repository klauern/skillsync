package model

import (
	"fmt"
	"strings"
)

// CustomAgent is one harness-owned agent definition. It is not an Agent Skill.
type CustomAgent struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    Platform       `json:"platform"`
	Content     string         `json:"content"`
	Tools       []string       `json:"tools,omitempty"`
	Model       string         `json:"model,omitempty"`
	Native      map[string]any `json:"native,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	SourcePath  string         `json:"source_path,omitempty"`
	MappingKey  string         `json:"mapping_key,omitempty"`
}

// Validate checks the native-agent boundary without invoking the agent.
func (a CustomAgent) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("custom agent name is required")
	}
	if !a.Platform.IsValid() {
		return fmt.Errorf("unsupported custom agent platform %q", a.Platform)
	}
	if strings.TrimSpace(a.Description) == "" {
		return fmt.Errorf("custom agent %q description is required", a.Name)
	}
	return nil
}

// Key returns a stable native identity.
func (a CustomAgent) Key() string {
	return string(a.Platform) + ":agent:" + strings.TrimSpace(a.Name)
}
