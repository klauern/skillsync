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
	filenameName, err := validatePath(platform, path)
	if err != nil {
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
	if value, ok := metadata["name"]; ok {
		a.Name, err = stringField("name", value)
		if err != nil {
			return model.CustomAgent{}, err
		}
	} else {
		a.Name = filenameName
	}
	if value, ok := metadata["description"]; ok {
		a.Description, err = stringField("description", value)
		if err != nil {
			return model.CustomAgent{}, err
		}
	} else {
		return model.CustomAgent{}, fmt.Errorf("custom agent frontmatter field %q is required", "description")
	}
	if value, ok := metadata["model"]; ok {
		a.Model, err = stringField("model", value)
		if err != nil {
			return model.CustomAgent{}, err
		}
	}
	if value, ok := metadata["mapping-key"]; ok {
		a.MappingKey, err = stringField("mapping-key", value)
		if err != nil {
			return model.CustomAgent{}, err
		}
	}
	if value, ok := metadata["tools"]; ok {
		a.Tools, err = stringSlice(value)
		if err != nil {
			return model.CustomAgent{}, err
		}
	}
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
		if commonFields[key] {
			return nil, fmt.Errorf("custom agent native field %q is reserved", key)
		}
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
	if err := validateName(name); err != nil {
		return "", err
	}
	root := agentRoot(platform)
	path := filepath.Clean(filepath.Join(root, name+agentSuffix(platform)))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.Contains(rel, string(filepath.Separator)) {
		return "", fmt.Errorf("custom agent name %q escapes canonical root %q", name, root)
	}
	return path, nil
}

func validatePath(platform model.Platform, path string) (string, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", fmt.Errorf("custom agent path must be nonempty and NUL-free")
	}
	base := filepath.Base(filepath.Clean(path))
	suffix := agentSuffix(platform)
	if !strings.HasSuffix(base, suffix) {
		return "", fmt.Errorf("custom agent path %q has invalid suffix for %s", path, platform)
	}
	name := strings.TrimSuffix(base, suffix)
	if err := validateName(name); err != nil {
		return "", fmt.Errorf("custom agent filename %q: %w", base, err)
	}
	return name, nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("custom agent name %q has leading or trailing whitespace", name)
	}
	if name == "" {
		return fmt.Errorf("custom agent name is required")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("custom agent name %q is not a safe filename", name)
	}
	return nil
}

func agentRoot(platform model.Platform) string {
	return map[model.Platform]string{model.ClaudeCode: ".claude/agents", model.Copilot: ".github/agents", model.Gemini: ".gemini/agents"}[platform]
}

func agentSuffix(platform model.Platform) string {
	if platform == model.Copilot {
		return ".agent.md"
	}
	return ".md"
}

func stringField(field string, value any) (string, error) {
	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("custom agent frontmatter field %q must be a string, got %T", field, value)
	}
	return result, nil
}

func stringSlice(value any) ([]string, error) {
	var out []string
	switch values := value.(type) {
	case []any:
		out = make([]string, len(values))
		for i, value := range values {
			var ok bool
			out[i], ok = value.(string)
			if !ok {
				return nil, fmt.Errorf("custom agent frontmatter field %q must contain only strings, got %T", "tools", value)
			}
		}
	case []string:
		out = append(out, values...)
	default:
		return nil, fmt.Errorf("custom agent frontmatter field %q must be a string list, got %T", "tools", value)
	}
	sort.Strings(out)
	return out, nil
}
