package model

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// MCPTransport identifies how a harness starts or contacts an MCP server.
type MCPTransport string

const (
	// MCPTransportStdio starts a local process over standard input and output.
	MCPTransportStdio MCPTransport = "stdio"
	// MCPTransportHTTP contacts a remote HTTP endpoint.
	MCPTransportHTTP MCPTransport = "http"
	// MCPTransportSSE contacts a remote server-sent events endpoint.
	MCPTransportSSE MCPTransport = "sse"
)

// MCPServer is a secret-safe, harness-independent MCP server declaration.
// Environment and header values must be references. Literal credentials are
// rejected before a declaration can reach a writer.
type MCPServer struct {
	Name       string            `json:"name"`
	Platform   Platform          `json:"platform"`
	Transport  MCPTransport      `json:"transport"`
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	URL        string            `json:"url,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	SourcePath string            `json:"source_path,omitempty"`
	Scope      string            `json:"scope,omitempty"`
	MappingKey string            `json:"mapping_key,omitempty"`
}

var mcpReference = regexp.MustCompile(`^(?:\$[A-Za-z_][A-Za-z0-9_]*|\$\{[^{}]+\})$`)

// Validate checks transport requirements and rejects embedded secrets.
func (s MCPServer) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("MCP server name is required")
	}
	if !s.Platform.IsValid() {
		return fmt.Errorf("unsupported MCP server platform %q", s.Platform)
	}
	switch s.Transport {
	case MCPTransportStdio:
		if strings.TrimSpace(s.Command) == "" || s.URL != "" {
			return fmt.Errorf("stdio MCP server %q requires command and forbids URL", s.Name)
		}
	case MCPTransportHTTP, MCPTransportSSE:
		if strings.TrimSpace(s.URL) == "" || s.Command != "" || len(s.Args) != 0 {
			return fmt.Errorf("remote MCP server %q requires URL and forbids command and args", s.Name)
		}
		parsed, err := url.Parse(s.URL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("MCP server %q has invalid URL", s.Name)
		}
		if parsed.User != nil || parsed.RawQuery != "" {
			return fmt.Errorf("MCP server %q URL contains credentials or query data", s.Name)
		}
	default:
		return fmt.Errorf("unsupported MCP transport %q", s.Transport)
	}
	for key, value := range s.Env {
		if strings.TrimSpace(key) == "" || !mcpReference.MatchString(value) {
			return fmt.Errorf("MCP server %q environment value for %q must be a variable reference", s.Name, key)
		}
	}
	for key, value := range s.Headers {
		if strings.TrimSpace(key) == "" || !mcpReference.MatchString(value) {
			return fmt.Errorf("MCP server %q header value for %q must be a variable reference", s.Name, key)
		}
	}
	return nil
}

// Key returns a stable native identity.
func (s MCPServer) Key() string {
	return string(s.Platform) + ":mcp:" + strings.TrimSpace(s.Name)
}
