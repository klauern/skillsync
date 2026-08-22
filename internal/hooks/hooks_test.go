package hooks

import (
	"context"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

type discovererStub struct{ hooks []model.HookConfig }

func (d discovererStub) DiscoverHooks(context.Context, model.Platform) ([]model.HookConfig, error) {
	return d.hooks, nil
}

type writerStub struct {
	calls int
	hooks []model.HookConfig
}

func (w *writerStub) WriteHooks(_ context.Context, hooks []model.HookConfig) error {
	w.calls++
	w.hooks = hooks
	return nil
}

func trusted() trust.Policy {
	return trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true, trust.RiskExecutable: true}}
}

func TestDiscoverIsOptInAndSorts(t *testing.T) {
	t.Parallel()
	d := discovererStub{hooks: []model.HookConfig{
		{Name: "z", Platform: model.Codex, Event: "Stop", Command: "true"},
		{Name: "a", Platform: model.Codex, Event: "PreToolUse", Command: "true"},
	}}
	got, err := Discover(context.Background(), model.Codex, d, false)
	if err != nil || got != nil {
		t.Fatalf("disabled Discover() = %+v, %v", got, err)
	}
	got, err = Discover(context.Background(), model.Codex, d, true)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "a" {
		t.Fatalf("Discover() order = %+v", got)
	}
}

func TestPlanRequiresBothTrustCategories(t *testing.T) {
	t.Parallel()
	hook := model.HookConfig{Name: "audit", Platform: model.Codex, Event: "PreToolUse", Command: "true"}
	items, err := Plan([]model.HookConfig{hook}, model.Codex, Options{Enabled: true, TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionBlocked {
		t.Fatalf("Action = %s", items[0].Action)
	}
}

func TestPlanRecordsUnsupportedHarness(t *testing.T) {
	t.Parallel()
	hook := model.HookConfig{Name: "audit", Platform: model.ClaudeCode, Event: "PreToolUse", Command: "true"}
	items, err := Plan([]model.HookConfig{hook}, model.Cursor, Options{Enabled: true, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionUnsupported || items[0].Reason == "" {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestPlanRequiresExactMappingAndWarns(t *testing.T) {
	t.Parallel()
	hook := model.HookConfig{Name: "audit", Platform: model.Codex, Event: "PreToolUse", Command: "true", MappingKey: "tool-before"}
	items, err := Plan([]model.HookConfig{hook}, model.Gemini, Options{Enabled: true, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionUnmapped {
		t.Fatalf("Action = %s", items[0].Action)
	}
	registry := &Registry{}
	if err := registry.Register(Mapping{Key: "tool-before", SourcePlatform: model.Codex, SourceEvent: "PreToolUse", TargetPlatform: model.Gemini, TargetEvent: "BeforeTool"}); err != nil {
		t.Fatal(err)
	}
	items, err = Plan([]model.HookConfig{hook}, model.Gemini, Options{Enabled: true, Registry: registry, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionWrite || items[0].Target.Event != "BeforeTool" || items[0].Warning == "" {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestSyncPreflightsBatchAndWritesOnce(t *testing.T) {
	t.Parallel()
	writer := &writerStub{}
	valid := model.HookConfig{Name: "audit", Platform: model.Codex, Event: "PreToolUse", Command: "true"}
	invalid := model.HookConfig{Name: "broken", Platform: model.Codex, Event: "Stop"}
	if _, err := Sync(context.Background(), []model.HookConfig{valid, invalid}, model.Codex, writer, Options{Enabled: true, TrustPolicy: trusted()}); err == nil {
		t.Fatal("Sync() error = nil")
	}
	if writer.calls != 0 {
		t.Fatalf("writer calls = %d", writer.calls)
	}
	if _, err := Sync(context.Background(), []model.HookConfig{valid}, model.Codex, writer, Options{Enabled: true, TrustPolicy: trusted()}); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || len(writer.hooks) != 1 {
		t.Fatalf("writer = %+v", writer)
	}
}

func TestPlanRejectsUnsupportedTargetEvent(t *testing.T) {
	t.Parallel()
	hook := model.HookConfig{Name: "audit", Platform: model.Codex, Event: "BeforeModel", Command: "true"}
	if _, err := Plan([]model.HookConfig{hook}, model.Codex, Options{Enabled: true, TrustPolicy: trusted()}); err == nil {
		t.Fatal("Plan() error = nil")
	}
}
