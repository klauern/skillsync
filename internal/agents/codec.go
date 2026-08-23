package agents

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
)

var commonFields = map[string]bool{"name": true, "description": true, "tools": true, "model": true, "mapping-key": true}

// DecodeMarkdown decodes one supported native custom-agent file.
func DecodeMarkdown(platform model.Platform, path string, data []byte) (model.CustomAgent, error) {
	if !supported(platform) {
		return model.CustomAgent{}, fmt.Errorf("custom agent codec does not support %s", platform)
	}
	if err := validatePath(platform, path); err != nil {
		return model.CustomAgent{}, err
	}
	parts := parser.SplitFrontmatter(data)
	if !parts.HasFrontmatter {
		return model.CustomAgent{}, fmt.Errorf("custom agent frontmatter is required")
	}
	metadata, err := parser.ParseYAMLFrontmatter(parts.Frontmatter)
	if err != nil {
		return model.CustomAgent{}, fmt.Errorf("decode custom agent frontmatter: %w", err)
	}
	a := model.CustomAgent{Platform: platform, Content: parts.Content, SourcePath: path, Native: make(map[string]any)}
	a.Name, _ = metadata["name"].(string)
	if a.Name == "" {
		a.Name = strings.TrimSuffix(filepath.Base(path), agentSuffix(platform))
	}
	a.Description, _ = metadata["description"].(string)
	a.Model, _ = metadata["model"].(string)
	a.MappingKey, _ = metadata["mapping-key"].(string)
	a.Tools = append(a.Tools, stringSlice(metadata["tools"])...)
	for key, value := range metadata {
		if !commonFields[key] {
			a.Native[key] = value
		}
	}
	if len(a.Native) == 0 {
		a.Native = nil
	}
	if err := a.Validate(); err != nil {
		return model.CustomAgent{}, err
	}
	return a, nil
}

// EncodeMarkdown emits one supported native custom-agent file.
func EncodeMarkdown(a model.CustomAgent) ([]byte, error) {
	if !supported(a.Platform) {
		return nil, fmt.Errorf("custom agent codec does not support %s", a.Platform)
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	metadata := make(map[string]any, len(a.Native)+5)
	for key, value := range a.Native {
		metadata[key] = value
	}
	metadata["name"], metadata["description"] = a.Name, a.Description
	if len(a.Tools) > 0 {
		metadata["tools"] = a.Tools
	}
	if a.Model != "" {
		metadata["model"] = a.Model
	}
	if a.MappingKey != "" {
		metadata["mapping-key"] = a.MappingKey
	}
	frontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode custom agent frontmatter: %w", err)
	}
	return []byte("---\n" + string(frontmatter) + "---\n" + a.Content), nil
}

// CanonicalPath returns the repository-relative destination for an agent.
func CanonicalPath(platform model.Platform, name string) (string, error) {
	if !supported(platform) {
		return "", fmt.Errorf("custom agent path does not support %s", platform)
	}
	root := map[model.Platform]string{model.ClaudeCode: ".claude/agents", model.Copilot: ".github/agents", model.Gemini: ".gemini/agents"}[platform]
	return filepath.Join(root, name+agentSuffix(platform)), nil
}

func validatePath(platform model.Platform, path string) error {
	if !strings.HasSuffix(filepath.Base(path), agentSuffix(platform)) {
		return fmt.Errorf("custom agent path %q has invalid suffix for %s", path, platform)
	}
	return nil
}

func agentSuffix(platform model.Platform) string {
	if platform == model.Copilot {
		return ".agent.md"
	}
	return ".md"
}

func stringSlice(value any) []string {
	var out []string
	switch values := value.(type) {
	case []any:
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
	case []string:
		out = append(out, values...)
	}
	sort.Strings(out)
	return out
}
