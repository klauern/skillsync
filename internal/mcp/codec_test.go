package mcp

import (
	"reflect"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestConfigRoundTrip(t *testing.T) {
	t.Parallel()
	for _, platform := range []model.Platform{model.ClaudeCode, model.Codex, model.Copilot, model.Gemini} {
		platform := platform
		t.Run(string(platform), func(t *testing.T) {
			t.Parallel()
			want := []model.MCPServer{{Name: "local", Platform: platform, Transport: model.MCPTransportStdio, Command: "mcp-local", Args: []string{"--quiet"}, Env: map[string]string{"TOKEN": "${MCP_TOKEN}"}}} // #nosec G101 -- reference only
			data, err := EncodeConfig(platform, want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeConfig(platform, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("round trip = %#v, want %#v\n%s", got, want, data)
			}
		})
	}
}

func TestRemoteConfigRoundTrip(t *testing.T) {
	t.Parallel()
	for _, platform := range []model.Platform{model.ClaudeCode, model.Copilot, model.Gemini} {
		platform := platform
		t.Run(string(platform), func(t *testing.T) {
			want := []model.MCPServer{{Name: "remote", Platform: platform, Transport: model.MCPTransportHTTP, URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "${AUTH_HEADER}"}}}
			data, err := EncodeConfig(platform, want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeConfig(platform, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("round trip = %#v, want %#v", got, want)
			}
		})
	}
}

func TestDecodeRejectsLiteralSecretsWithoutEchoingThem(t *testing.T) {
	t.Parallel()
	const secret = "do-not-leak-this-token"
	_, err := DecodeConfig(model.ClaudeCode, []byte(`{"mcpServers":{"bad":{"command":"server","env":{"API_TOKEN":"`+secret+`"}}}}`))
	if err == nil {
		t.Fatal("DecodeConfig() error = nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked secret: %v", err)
	}
}

func TestCodexRejectsHeadersBeforeEncoding(t *testing.T) {
	t.Parallel()
	_, err := EncodeConfig(model.Codex, []model.MCPServer{{Name: "remote", Platform: model.Codex, Transport: model.MCPTransportHTTP, URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "${AUTH_HEADER}"}}})
	if err == nil || !strings.Contains(err.Error(), "does not support headers") {
		t.Fatalf("EncodeConfig() error = %v", err)
	}
}
