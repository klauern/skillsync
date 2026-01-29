package model

import "time"

// PluginInfo contains metadata about a plugin-installed skill.
// This tracks whether a skill was installed via Claude Code's plugin system
// and from which marketplace/repository it originated.
type PluginInfo struct {
	// PluginName is the full plugin identifier (e.g., "commits@klauern-skills")
	PluginName string `json:"plugin_name,omitempty"`
	// Marketplace is the marketplace/repository name (e.g., "klauern-skills")
	Marketplace string `json:"marketplace,omitempty"`
	// Version is the installed version (e.g., "1.1.0")
	Version string `json:"version,omitempty"`
	// InstallPath is the resolved symlink target path
	InstallPath string `json:"install_path,omitempty"`
	// IsDev indicates if this is a development symlink (points outside plugin cache)
	IsDev bool `json:"is_dev,omitempty"`
	// SymlinkTarget is the raw symlink target before resolution
	SymlinkTarget string `json:"symlink_target,omitempty"`
	// InstallScope is the scope at which the plugin was installed (e.g., "user", "project")
	// This reflects where the plugin was installed, not the skill's precedence scope.
	InstallScope string `json:"install_scope,omitempty"`
}

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

	// Type indicates whether this is a regular skill or a slash command/prompt.
	// Defaults to SkillTypeSkill if not specified.
	Type SkillType `json:"type,omitempty"`

	// Trigger is the slash command trigger for prompts (e.g., "/my-command").
	// Only relevant when Type is SkillTypePrompt.
	Trigger string `json:"trigger,omitempty"`

	// Agent Skills Standard fields
	Scope                  SkillScope        `json:"scope,omitempty"`
	DisableModelInvocation bool              `json:"disable_model_invocation,omitempty"`
	License                string            `json:"license,omitempty"`
	Compatibility          map[string]string `json:"compatibility,omitempty"`
	Scripts                []string          `json:"scripts,omitempty"`
	References             []string          `json:"references,omitempty"`
	Assets                 []string          `json:"assets,omitempty"`

	// PluginInfo contains metadata if this skill was installed via a plugin symlink
	PluginInfo *PluginInfo `json:"plugin_info,omitempty"`
}

// IsHigherPrecedence returns true if this skill's scope has higher precedence than other.
func (s Skill) IsHigherPrecedence(other Skill) bool {
	return s.Scope.IsHigherPrecedence(other.Scope)
}

// DisplayScope returns a formatted scope string for table output.
// For user/repo scopes, shows the platform-specific path (~/.claude, .cursor, etc).
// For plugin scope, shows plugin:<name> using metadata.
// For symlinked skills:
//   - Plugin symlinks show: ~/.claude/skills (plugin: name@marketplace)
//   - Dev symlinks show: ~/.claude/skills (dev: marketplace) or (dev) if unknown
func (s Skill) DisplayScope() string {
	platformDir := s.Platform.ConfigDir()

	// Check for plugin info from symlink detection
	if s.PluginInfo != nil {
		base := "~/." + platformDir + "/skills"
		if s.PluginInfo.IsDev {
			// Development symlink - points outside plugin cache
			if s.PluginInfo.Marketplace != "" {
				return base + " (dev: " + s.PluginInfo.Marketplace + ")"
			}
			return base + " (dev)"
		}
		// Installed plugin symlink
		if s.PluginInfo.PluginName != "" {
			return base + " (plugin: " + s.PluginInfo.PluginName + ")"
		}
		if s.PluginInfo.Marketplace != "" {
			return base + " (plugin: " + s.PluginInfo.Marketplace + ")"
		}
	}

	switch s.Scope {
	case ScopeUser:
		return "~/." + platformDir + "/skills"
	case ScopeRepo:
		return "." + platformDir + "/skills"
	case ScopePlugin:
		if name := s.Metadata["plugin"]; name != "" {
			return "plugin:" + name
		}
		return "plugin"
	case ScopeSystem:
		return "system"
	case ScopeAdmin:
		return "admin"
	case ScopeBuiltin:
		return "builtin"
	default:
		if s.Scope == "" {
			return "-"
		}
		return string(s.Scope)
	}
}
