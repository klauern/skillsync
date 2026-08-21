package model

import "testing"

func TestMCPServerValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		server  MCPServer
		wantErr bool
	}{
		{"stdio reference", MCPServer{Name: "local", Platform: Codex, Transport: MCPTransportStdio, Command: "server", Env: map[string]string{"TOKEN": "${MCP_TOKEN}"}}, false}, // #nosec G101 -- reference only
		{"http header reference", MCPServer{Name: "remote", Platform: ClaudeCode, Transport: MCPTransportHTTP, URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "${AUTH_HEADER}"}}, false},
		{"literal environment secret", MCPServer{Name: "local", Platform: Codex, Transport: MCPTransportStdio, Command: "server", Env: map[string]string{"TOKEN": "secret-value"}}, true},
		{"URL userinfo", MCPServer{Name: "remote", Platform: Gemini, Transport: MCPTransportHTTP, URL: "https://user:pass@example.com/mcp"}, true}, // #nosec G101 -- rejection fixture
		{"URL query", MCPServer{Name: "remote", Platform: Gemini, Transport: MCPTransportHTTP, URL: "https://example.com/mcp?token=secret"}, true},
		{"mixed transport", MCPServer{Name: "mixed", Platform: Copilot, Transport: MCPTransportStdio, Command: "server", URL: "https://example.com"}, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.server.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
