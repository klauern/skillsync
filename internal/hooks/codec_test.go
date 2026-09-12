package hooks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestCodecRoundTrip(t *testing.T) {
	t.Parallel()
	for _, platform := range []model.Platform{model.Codex, model.Gemini} {
		platform := platform
		t.Run(string(platform), func(t *testing.T) {
			t.Parallel()
			want := model.HookConfig{Name: "audit", Platform: platform, Event: "PreToolUse", Matcher: "shell", Command: "./audit.sh", Timeout: 10}
			if platform == model.Gemini {
				want.Event = "BeforeTool"
			}
			data, err := EncodeConfig(platform, []model.HookConfig{want})
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeConfig(platform, data)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Event != want.Event || got[0].Command != want.Command {
				t.Fatalf("round trip = %+v", got)
			}
			if platform == model.Gemini && got[0].Name != want.Name {
				t.Fatalf("Gemini hook name = %q, want %q", got[0].Name, want.Name)
			}
		})
	}
}

func TestCodecRejectsUnsupportedPlatformAndInvalidBatch(t *testing.T) {
	t.Parallel()
	if _, err := EncodeConfig(model.ClaudeCode, nil); err == nil {
		t.Fatal("EncodeConfig() unsupported error = nil")
	}
	hooks := []model.HookConfig{
		{Name: "ok", Platform: model.Codex, Event: "Stop", Command: "true"},
		{Name: "bad", Platform: model.Codex, Event: "Stop"},
	}
	data, err := EncodeConfig(model.Codex, hooks)
	if err == nil || data != nil {
		t.Fatalf("EncodeConfig() = %q, %v", data, err)
	}
}

func TestCodecUsesExactHarnessEnvelopes(t *testing.T) {
	t.Parallel()
	for _, platform := range []model.Platform{model.Codex, model.Gemini} {
		platform := platform
		t.Run(string(platform), func(t *testing.T) {
			hook := model.HookConfig{Name: "audit", Platform: platform, Event: "PreToolUse", Command: "./audit.sh"}
			if platform == model.Gemini {
				hook.Event = "BeforeTool"
			}
			data, err := EncodeConfig(platform, []model.HookConfig{hook})
			if err != nil {
				t.Fatal(err)
			}
			var root map[string]json.RawMessage
			if err := json.Unmarshal(data, &root); err != nil {
				t.Fatal(err)
			}
			raw, ok := root["hooks"]
			if !ok {
				t.Fatalf("encoded %s config has no hooks envelope: %s", platform, data)
			}
			var events map[string][]map[string]json.RawMessage
			if err := json.Unmarshal(raw, &events); err != nil {
				t.Fatal(err)
			}
			group := events[hook.Event][0]
			var encodedHook map[string]json.RawMessage
			var hooks []map[string]json.RawMessage
			if err := json.Unmarshal(group["hooks"], &hooks); err != nil {
				t.Fatal(err)
			}
			encodedHook = hooks[0]
			if _, ok := encodedHook["type"]; !ok {
				t.Fatal("encoded hook has no type")
			}
			if platform == model.Codex {
				if _, ok := encodedHook["name"]; ok {
					t.Fatalf("Codex hook unexpectedly contains name: %s", data)
				}
			} else if _, ok := encodedHook["name"]; !ok {
				t.Fatalf("Gemini hook has no name: %s", data)
			}
		})
	}
}

func TestDecodeRejectsNonCommandTypeWithoutEchoingCommand(t *testing.T) {
	t.Parallel()
	secret := "sensitive-value"
	_, err := DecodeConfig(model.Gemini, []byte(`{"hooks":{"BeforeTool":[{"hooks":[{"name":"bad","type":"prompt","command":"`+secret+`"}]}]}}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported hook type") || strings.Contains(err.Error(), secret) {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
}
