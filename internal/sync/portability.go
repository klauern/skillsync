package sync

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	gosync "sync"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/validation"
)

var (
	lossyFieldOnce gosync.Once
	lossyFieldMap  map[string][]model.Platform
)

type portabilitySnapshotYAML struct {
	LossyFieldMappings []struct {
		Field         string   `yaml:"field"`
		UnsupportedBy []string `yaml:"unsupported_by"`
	} `yaml:"lossy_field_mappings"`
}

func loadLossyFieldMapFromSnapshot() (map[string][]model.Platform, error) {
	root, err := validation.FindRepoRoot()
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- reads only repo-controlled documentation paths.
	data, err := os.ReadFile(filepath.Join(root, "docs", "platforms", "portability-snapshot.yaml"))
	if err != nil {
		return nil, err
	}
	var snap portabilitySnapshotYAML
	if err := yaml.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	m := make(map[string][]model.Platform, len(snap.LossyFieldMappings))
	for _, entry := range snap.LossyFieldMappings {
		// Skip descriptive pseudo-fields (e.g. "gemini extension runtime surfaces")
		// that contain whitespace and cannot appear as frontmatter keys.
		if strings.ContainsAny(entry.Field, " \t") {
			continue
		}
		var platforms []model.Platform
		for _, name := range entry.UnsupportedBy {
			p, err := model.ParsePlatform(name)
			if err != nil {
				slog.Warn("portability: unrecognized platform in snapshot", "field", entry.Field, "platform", name)
				continue
			}
			platforms = append(platforms, p)
		}
		if len(platforms) > 0 {
			m[entry.Field] = platforms
		}
	}
	return m, nil
}

func getLossyFieldMap() map[string][]model.Platform {
	lossyFieldOnce.Do(func() {
		m, err := loadLossyFieldMapFromSnapshot()
		if err != nil {
			slog.Warn("portability: cannot load portability snapshot, warnings disabled", "error", err)
			lossyFieldMap = map[string][]model.Platform{}
			return
		}
		lossyFieldMap = m
	})
	return lossyFieldMap
}

// portabilityWarningsForSkill returns the sorted list of metadata field names that will be
// lossy or silently degraded when skill is written to target.
func portabilityWarningsForSkill(skill model.Skill, target model.Platform) []string {
	var warnings []string
	for field, unsupported := range getLossyFieldMap() {
		if _, ok := skill.Metadata[field]; !ok {
			continue
		}
		for _, p := range unsupported {
			if p == target {
				warnings = append(warnings, field)
				break
			}
		}
	}
	sort.Strings(warnings)
	return warnings
}
