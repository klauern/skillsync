package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/klauern/skillsync/internal/model"
)

type serverConfig struct {
	Type    string            `json:"type,omitempty" toml:"type,omitempty"`
	Command string            `json:"command,omitempty" toml:"command,omitempty"`
	Args    []string          `json:"args,omitempty" toml:"args,omitempty"`
	URL     string            `json:"url,omitempty" toml:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty" toml:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty" toml:"headers,omitempty"`
}

// DecodeConfig decodes MCP declarations without retaining unrelated settings
// or literal secret values. A secret-bearing config fails as one batch.
func DecodeConfig(platform model.Platform, data []byte) ([]model.MCPServer, error) {
	configs, err := decodeServerMap(platform, data)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)
	servers := make([]model.MCPServer, 0, len(names))
	for _, name := range names {
		cfg := configs[name]
		transport := model.MCPTransport(cfg.Type)
		if transport == "" {
			if cfg.URL != "" {
				transport = model.MCPTransportHTTP
			} else {
				transport = model.MCPTransportStdio
			}
		}
		server := model.MCPServer{Name: name, Platform: platform, Transport: transport, Command: cfg.Command, Args: cfg.Args, URL: cfg.URL, Env: cfg.Env, Headers: cfg.Headers}
		if err := validateTarget(server); err != nil {
			return nil, fmt.Errorf("decode MCP server %q: %w", name, err)
		}
		servers = append(servers, server)
	}
	return servers, nil
}

// EncodeConfig validates all declarations before it emits target config bytes.
func EncodeConfig(platform model.Platform, servers []model.MCPServer) ([]byte, error) {
	configs := make(map[string]serverConfig, len(servers))
	for _, server := range servers {
		if server.Platform != platform {
			return nil, fmt.Errorf("MCP server %q has platform %s, want %s", server.Name, server.Platform, platform)
		}
		if err := validateTarget(server); err != nil {
			return nil, err
		}
		if _, exists := configs[server.Name]; exists {
			return nil, fmt.Errorf("duplicate MCP server %q", server.Name)
		}
		cfg := serverConfig{Command: server.Command, Args: server.Args, URL: server.URL, Env: server.Env, Headers: server.Headers}
		if server.Transport != model.MCPTransportStdio {
			cfg.Type = string(server.Transport)
		}
		configs[server.Name] = cfg
	}
	if platform == model.Codex {
		var out bytes.Buffer
		if err := toml.NewEncoder(&out).Encode(struct {
			MCPServers map[string]serverConfig `toml:"mcp_servers"`
		}{configs}); err != nil {
			return nil, fmt.Errorf("encode Codex MCP config: %w", err)
		}
		return out.Bytes(), nil
	}
	key, err := jsonConfigKey(platform)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{key: configs}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s MCP config: %w", platform, err)
	}
	return append(data, '\n'), nil
}

func decodeServerMap(platform model.Platform, data []byte) (map[string]serverConfig, error) {
	if platform == model.Codex {
		var root struct {
			MCPServers map[string]serverConfig `toml:"mcp_servers"`
		}
		if _, err := toml.Decode(string(data), &root); err != nil {
			return nil, fmt.Errorf("decode Codex MCP config: %w", err)
		}
		return root.MCPServers, nil
	}
	key, err := jsonConfigKey(platform)
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode %s MCP config: %w", platform, err)
	}
	configs := make(map[string]serverConfig)
	if raw := root[key]; raw != nil {
		if err := json.Unmarshal(raw, &configs); err != nil {
			return nil, fmt.Errorf("decode %s %s: %w", platform, key, err)
		}
	}
	return configs, nil
}

func jsonConfigKey(platform model.Platform) (string, error) {
	switch platform {
	case model.ClaudeCode, model.Gemini:
		return "mcpServers", nil
	case model.Copilot:
		return "servers", nil
	default:
		return "", fmt.Errorf("MCP config codec does not support %s", platform)
	}
}
