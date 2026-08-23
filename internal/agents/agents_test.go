package agents

import (
	"context"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

type discovererStub struct{ agents []model.CustomAgent }

func (d discovererStub) DiscoverAgents(context.Context, model.Platform) ([]model.CustomAgent, error) {
	return d.agents, nil
}

type writerStub struct {
	calls  int
	agents []model.CustomAgent
}

func (w *writerStub) WriteAgents(_ context.Context, agents []model.CustomAgent) error {
	w.calls++
	w.agents = agents
	return nil
}

func trusted() trust.Policy {
	return trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}
}

func agent(name string, platform model.Platform) model.CustomAgent {
	return model.CustomAgent{Name: name, Description: "Test agent", Platform: platform, Content: "Act carefully.\n"}
}

func TestDiscoverIsOptInAndSorts(t *testing.T) {
	t.Parallel()
	d := discovererStub{agents: []model.CustomAgent{agent("z", model.ClaudeCode), agent("a", model.ClaudeCode)}}
	got, err := Discover(context.Background(), model.ClaudeCode, d, false)
	if err != nil || got != nil {
		t.Fatalf("disabled Discover() = %+v, %v", got, err)
	}
	got, err = Discover(context.Background(), model.ClaudeCode, d, true)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "a" {
		t.Fatalf("Discover() order = %+v", got)
	}
}

func TestPlanTrustUnsupportedAndMapping(t *testing.T) {
	t.Parallel()
	source := agent("reviewer", model.ClaudeCode)
	items, err := Plan([]model.CustomAgent{source}, model.ClaudeCode, Options{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionBlocked {
		t.Fatalf("Action = %s", items[0].Action)
	}
	items, err = Plan([]model.CustomAgent{source}, model.Codex, Options{Enabled: true, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionUnsupported || items[0].Reason == "" {
		t.Fatalf("item = %+v", items[0])
	}
	source.MappingKey = "review"
	items, err = Plan([]model.CustomAgent{source}, model.Gemini, Options{Enabled: true, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionUnmapped {
		t.Fatalf("Action = %s", items[0].Action)
	}
	registry := &Registry{}
	if err := registry.Register(Mapping{Key: "review", SourcePlatform: model.ClaudeCode, TargetPlatform: model.Gemini}); err != nil {
		t.Fatal(err)
	}
	items, err = Plan([]model.CustomAgent{source}, model.Gemini, Options{Enabled: true, Registry: registry, TrustPolicy: trusted()})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Action != ActionWrite || items[0].Target.Platform != model.Gemini || items[0].Warning == "" {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestSyncPreflightsAndWritesOnce(t *testing.T) {
	t.Parallel()
	w := &writerStub{}
	valid := agent("reviewer", model.Copilot)
	invalid := agent("broken", model.Copilot)
	invalid.Description = ""
	if _, err := Sync(context.Background(), []model.CustomAgent{valid, invalid}, model.Copilot, w, Options{Enabled: true, TrustPolicy: trusted()}); err == nil {
		t.Fatal("Sync() error = nil")
	}
	if w.calls != 0 {
		t.Fatalf("writer calls = %d", w.calls)
	}
	if _, err := Sync(context.Background(), []model.CustomAgent{valid}, model.Copilot, w, Options{Enabled: true, TrustPolicy: trusted()}); err != nil {
		t.Fatal(err)
	}
	if w.calls != 1 || len(w.agents) != 1 {
		t.Fatalf("writer = %+v", w)
	}
}

func TestPlanRejectsDuplicateTarget(t *testing.T) {
	t.Parallel()
	if _, err := Plan([]model.CustomAgent{agent("same", model.Gemini), agent("same", model.Gemini)}, model.Gemini, Options{Enabled: true, TrustPolicy: trusted()}); err == nil {
		t.Fatal("Plan() error = nil")
	}
}
