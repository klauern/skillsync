package hooks

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/klauern/skillsync/internal/model"
)

type commandHook struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []commandHook `json:"hooks"`
}

type codexCommandHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type codexHookGroup struct {
	Matcher string             `json:"matcher,omitempty"`
	Hooks   []codexCommandHook `json:"hooks"`
}

type geminiHookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []commandHook `json:"hooks"`
}

// DecodeConfig reads the owned hook section without executing commands.
func DecodeConfig(platform model.Platform, data []byte) ([]model.HookConfig, error) {
	if !supported(platform) {
		return nil, fmt.Errorf("hook config codec does not support %s", platform)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode %s hook config: %w", platform, err)
	}
	raw := root["hooks"]
	if raw == nil {
		return nil, nil
	}
	var events map[string][]hookGroup
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, fmt.Errorf("decode %s hooks: %w", platform, err)
	}
	var out []model.HookConfig
	seen := make(map[string]bool)
	for event, groups := range events {
		for groupIndex, group := range groups {
			for hookIndex, hook := range group.Hooks {
				if hook.Type != "command" {
					return nil, fmt.Errorf("decode hook %q: unsupported hook type %q", hook.Name, hook.Type)
				}
				name := hook.Name
				if name == "" && platform == model.Codex {
					// Codex command hooks have no name field. Use a stable, non-sensitive
					// identity for the in-memory declaration without echoing its command.
					name = fmt.Sprintf("%s-hook-%d-%d", event, groupIndex, hookIndex)
				}
				if name == "" {
					return nil, fmt.Errorf("decode hook: name is required")
				}
				h := model.HookConfig{Name: name, Platform: platform, Event: event, Matcher: group.Matcher, Command: hook.Command, Timeout: hook.Timeout}
				if err := validateTarget(h); err != nil {
					return nil, fmt.Errorf("decode hook %q: %w", name, err)
				}
				if seen[h.Key()] {
					return nil, fmt.Errorf("decode duplicate hook %q", name)
				}
				seen[h.Key()] = true
				out = append(out, h)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out, nil
}

// EncodeConfig validates the full batch before it emits the owned hook section.
func EncodeConfig(platform model.Platform, hooks []model.HookConfig) ([]byte, error) {
	if !supported(platform) {
		return nil, fmt.Errorf("hook config codec does not support %s", platform)
	}
	seen := make(map[string]bool)
	if platform == model.Codex {
		events := make(map[string][]codexHookGroup)
		for _, hook := range hooks {
			if err := validateHookForPlatform(hook, platform, seen); err != nil {
				return nil, err
			}
			events[hook.Event] = append(events[hook.Event], codexHookGroup{Matcher: hook.Matcher, Hooks: []codexCommandHook{{Type: "command", Command: hook.Command, Timeout: hook.Timeout}}})
		}
		return encodePayload(platform, map[string]any{"hooks": events})
	}

	events := make(map[string][]geminiHookGroup)
	for _, hook := range hooks {
		if err := validateHookForPlatform(hook, platform, seen); err != nil {
			return nil, err
		}
		events[hook.Event] = append(events[hook.Event], geminiHookGroup{Matcher: hook.Matcher, Hooks: []commandHook{{Name: hook.Name, Type: "command", Command: hook.Command, Timeout: hook.Timeout}}})
	}
	return encodePayload(platform, map[string]any{"hooks": events})
}

func validateHookForPlatform(hook model.HookConfig, platform model.Platform, seen map[string]bool) error {
	if hook.Platform != platform {
		return fmt.Errorf("hook %q has platform %s, want %s", hook.Name, hook.Platform, platform)
	}
	if err := validateTarget(hook); err != nil {
		return err
	}
	if seen[hook.Key()] {
		return fmt.Errorf("duplicate hook %q", hook.Name)
	}
	seen[hook.Key()] = true
	return nil
}

func encodePayload(platform model.Platform, payload any) ([]byte, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s hooks: %w", platform, err)
	}
	return append(data, '\n'), nil
}
