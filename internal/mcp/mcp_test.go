package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

type recordingWriter struct {
	calls   int
	servers []model.MCPServer
}

func (w *recordingWriter) WriteMCPServers(_ context.Context, servers []model.MCPServer) error {
	w.calls++
	w.servers = servers
	return nil
}

func validServer() model.MCPServer {
	return model.MCPServer{Name: "files", Platform: model.ClaudeCode, Transport: model.MCPTransportStdio, Command: "mcp-files", Env: map[string]string{"TOKEN": "${MCP_TOKEN}"}, MappingKey: "files"} // #nosec G101 -- reference only
}

func TestSyncPreflightsFullBatchBeforeWrite(t *testing.T) {
	t.Parallel()
	w := &recordingWriter{}
	bad := validServer()
	bad.Name = "bad"
	bad.Env["TOKEN"] = "literal-secret"
	_, err := Sync(context.Background(), []model.MCPServer{validServer(), bad}, model.ClaudeCode, w, Options{Enabled: true, TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}})
	if err == nil || !strings.Contains(err.Error(), "variable reference") {
		t.Fatalf("Sync() error = %v", err)
	}
	if w.calls != 0 {
		t.Fatalf("writer called %d times before validation completed", w.calls)
	}
}

func TestPlanRequiresTrustAndExactMapping(t *testing.T) {
	t.Parallel()
	server := validServer()
	items, err := Plan([]model.MCPServer{server}, model.Codex, Options{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionBlocked {
		t.Fatalf("action = %s, want blocked", items[0].Action)
	}

	policy := trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}
	items, err = Plan([]model.MCPServer{server}, model.Codex, Options{Enabled: true, TrustPolicy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionUnmapped {
		t.Fatalf("action = %s, want unmapped", items[0].Action)
	}

	registry := &Registry{}
	if err := registry.Register(Mapping{Key: "files", SourcePlatform: model.ClaudeCode, SourceName: "files", TargetPlatform: model.Codex, TargetName: "filesystem"}); err != nil {
		t.Fatal(err)
	}
	items, err = Plan([]model.MCPServer{server}, model.Codex, Options{Enabled: true, TrustPolicy: policy, Registry: registry})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionWrite || items[0].Target.Name != "filesystem" {
		t.Fatalf("item = %#v", items[0])
	}
}

func TestPlanRejectsUnsupportedTargetFields(t *testing.T) {
	t.Parallel()
	server := validServer()
	server.Platform = model.Codex
	server.Headers = map[string]string{"Authorization": "${AUTH_HEADER}"}
	_, err := Plan([]model.MCPServer{server}, model.Codex, Options{Enabled: true, TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}})
	if err == nil || !strings.Contains(err.Error(), "does not support headers") {
		t.Fatalf("Plan() error = %v", err)
	}
}

func TestSyncWritesBatchOnce(t *testing.T) {
	t.Parallel()
	w := &recordingWriter{}
	_, err := Sync(context.Background(), []model.MCPServer{validServer()}, model.ClaudeCode, w, Options{Enabled: true, TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if w.calls != 1 || len(w.servers) != 1 {
		t.Fatalf("writer calls = %d, servers = %d", w.calls, len(w.servers))
	}
}
