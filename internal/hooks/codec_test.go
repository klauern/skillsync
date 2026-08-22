package hooks

import (
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
			if len(got) != 1 || got[0].Name != want.Name || got[0].Event != want.Event || got[0].Command != want.Command {
				t.Fatalf("round trip = %+v", got)
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

func TestDecodeDoesNotEchoCommandOnError(t *testing.T) {
	t.Parallel()
	secret := "sensitive-value"
	_, err := DecodeConfig(model.Gemini, []byte(`{"hooks":{"BeforeTool":[{"hooks":[{"name":"bad","command":"`+secret+`","timeout":-1}]}]}}`))
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
}
