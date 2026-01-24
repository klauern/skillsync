package model

import "time"

// Skill represents a unified agent skill across platforms
type Skill struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Platform    Platform          `json:"platform"`
	Path        string            `json:"path"`
	Tools       []string          `json:"tools,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Content     string            `json:"content"`
	ModifiedAt  time.Time         `json:"modified_at"`
}
