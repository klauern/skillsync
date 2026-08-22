package hooks

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/klauern/skillsync/internal/model"
)

type commandHook struct {
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type hookGroup struct {
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
	raw := data
	if platform == model.Gemini {
		raw = root["hooks"]
		if raw == nil {
			return nil, nil
		}
	}
	var events map[string][]hookGroup
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, fmt.Errorf("decode %s hooks: %w", platform, err)
	}
	var out []model.HookConfig
	for event, groups := range events {
		for _, group := range groups {
			for _, hook := range group.Hooks {
				h := model.HookConfig{Name: hook.Name, Platform: platform, Event: event, Matcher: group.Matcher, Command: hook.Command, Timeout: hook.Timeout}
				if err := validateTarget(h); err != nil {
					return nil, fmt.Errorf("decode hook %q: %w", hook.Name, err)
				}
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
	events := make(map[string][]hookGroup)
	seen := make(map[string]bool)
	for _, hook := range hooks {
		if hook.Platform != platform {
			return nil, fmt.Errorf("hook %q has platform %s, want %s", hook.Name, hook.Platform, platform)
		}
		if err := validateTarget(hook); err != nil {
			return nil, err
		}
		if seen[hook.Key()] {
			return nil, fmt.Errorf("duplicate hook %q", hook.Name)
		}
		seen[hook.Key()] = true
		events[hook.Event] = append(events[hook.Event], hookGroup{Matcher: hook.Matcher, Hooks: []commandHook{{Name: hook.Name, Type: "command", Command: hook.Command, Timeout: hook.Timeout}}})
	}
	var payload any = events
	if platform == model.Gemini {
		payload = map[string]any{"hooks": events}
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s hooks: %w", platform, err)
	}
	return append(data, '\n'), nil
}
