package cli

import (
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/trust"
)

func TestOutputSyncResultJSONIncludesTrustDecisions(t *testing.T) {
	output := captureStdout(t, func() {
		err := outputSyncResultJSON(&sync.Result{
			Source: model.ClaudeCode,
			Target: model.Cursor,
			Skills: []sync.SkillResult{{
				Skill:  model.Skill{Name: "blocked"},
				Action: sync.ActionFailed,
				TrustDecisions: []trust.Decision{{
					Artifact: "run.sh",
					Risk:     trust.RiskExecutable,
					Allowed:  false,
					Reason:   "artifact declares executable scripts",
				}},
			}},
		})
		if err != nil {
			t.Fatalf("outputSyncResultJSON() error = %v", err)
		}
	})

	for _, want := range []string{"\"trust_decisions\"", "\"risk\": \"executable\"", "\"allowed\": false"} {
		if !strings.Contains(output, want) {
			t.Errorf("outputSyncResultJSON() = %q, want substring %q", output, want)
		}
	}
}
