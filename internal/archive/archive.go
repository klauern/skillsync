package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

// Manifest represents the metadata for an archive
type Manifest struct {
	Version    string           `json:"version"`
	CreatedAt  time.Time        `json:"created_at"`
	Platform   string           `json:"platform,omitempty"`
	SkillCount int              `json:"skill_count"`
	Skills     []ManifestSkill  `json:"skills"`
}

// ManifestSkill represents a skill entry in the manifest
type ManifestSkill struct {
	Name       string            `json:"name"`
	Platform   string            `json:"platform"`
	Scope      string            `json:"scope,omitempty"`
	ModifiedAt time.Time         `json:"modified_at"`
	Filename   string            `json:"filename"`
	Size       int64             `json:"size"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// CreateOptions configures archive creation
type CreateOptions struct {
	Platform   model.Platform   // Filter by platform (empty = all)
	Since      time.Time        // Filter skills modified after this date
	Before     time.Time        // Filter skills modified before this date
	IncludeMeta bool            // Include detailed metadata
}

// ExtractOptions configures archive extraction
type ExtractOptions struct {
	TargetDir  string           // Target directory for extraction
	Platform   model.Platform   // Filter by platform during extraction
	DryRun     bool            // Preview without extraction
}

// Create creates a tar.gz archive from skills
func Create(skills []model.Skill, w io.Writer, opts CreateOptions) error {
	// Filter skills based on options
	filtered := filterSkills(skills, opts)
	if len(filtered) == 0 {
		return fmt.Errorf("no skills match the specified filters")
	}

	// Create gzip writer
	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Build manifest
	manifest := Manifest{
		Version:    "1.0",
		CreatedAt:  time.Now(),
		SkillCount: len(filtered),
		Skills:     make([]ManifestSkill, 0, len(filtered)),
	}
	if opts.Platform != "" {
		manifest.Platform = string(opts.Platform)
	}

	// Add each skill to archive
	for _, skill := range filtered {
		// Serialize skill to JSON
		skillData, err := json.MarshalIndent(skill, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize skill %s: %w", skill.Name, err)
		}

		// Generate filename
		filename := fmt.Sprintf("skills/%s-%s.json", skill.Platform, sanitizeFilename(skill.Name))

		// Add to manifest
		manifest.Skills = append(manifest.Skills, ManifestSkill{
			Name:       skill.Name,
			Platform:   string(skill.Platform),
			Scope:      string(skill.Scope),
			ModifiedAt: skill.ModifiedAt,
			Filename:   filename,
			Size:       int64(len(skillData)),
			Metadata:   skill.Metadata,
		})

		// Write skill to tar
		header := &tar.Header{
			Name:    filename,
			Mode:    0o644,
			Size:    int64(len(skillData)),
			ModTime: skill.ModifiedAt,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", skill.Name, err)
		}
		if _, err := tarWriter.Write(skillData); err != nil {
			return fmt.Errorf("failed to write skill data for %s: %w", skill.Name, err)
		}
	}

	// Write manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize manifest: %w", err)
	}

	manifestHeader := &tar.Header{
		Name:    "manifest.json",
		Mode:    0o644,
		Size:    int64(len(manifestData)),
		ModTime: time.Now(),
	}
	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return fmt.Errorf("failed to write manifest header: %w", err)
	}
	if _, err := tarWriter.Write(manifestData); err != nil {
		return fmt.Errorf("failed to write manifest data: %w", err)
	}

	return nil
}

// Extract extracts skills from a tar.gz archive
func Extract(r io.Reader, opts ExtractOptions) ([]model.Skill, *Manifest, error) {
	// Create gzip reader
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	var manifest *Manifest
	skills := make([]model.Skill, 0)

	// Read archive entries
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Read entry data
		data, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read entry %s: %w", header.Name, err)
		}

		// Handle manifest
		if header.Name == "manifest.json" {
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
			}
			continue
		}

		// Handle skills
		if filepath.Dir(header.Name) == "skills" {
			var skill model.Skill
			if err := json.Unmarshal(data, &skill); err != nil {
				return nil, nil, fmt.Errorf("failed to parse skill from %s: %w", header.Name, err)
			}

			// Apply platform filter
			if opts.Platform != "" && skill.Platform != opts.Platform {
				continue
			}

			// Write to target directory if specified and not dry-run
			if opts.TargetDir != "" && !opts.DryRun {
				if err := writeSkillToDir(skill, opts.TargetDir); err != nil {
					return nil, nil, fmt.Errorf("failed to write skill %s: %w", skill.Name, err)
				}
			}

			skills = append(skills, skill)
		}
	}

	if manifest == nil {
		return nil, nil, fmt.Errorf("archive missing manifest.json")
	}

	return skills, manifest, nil
}

// filterSkills filters skills based on create options
func filterSkills(skills []model.Skill, opts CreateOptions) []model.Skill {
	filtered := make([]model.Skill, 0, len(skills))
	for _, skill := range skills {
		// Platform filter
		if opts.Platform != "" && skill.Platform != opts.Platform {
			continue
		}

		// Date filters
		// Since: include skills modified at or after this time
		if !opts.Since.IsZero() && skill.ModifiedAt.Before(opts.Since) {
			continue
		}
		// Before: include skills modified strictly before this time
		if !opts.Before.IsZero() && !skill.ModifiedAt.Before(opts.Before) {
			continue
		}

		filtered = append(filtered, skill)
	}
	return filtered
}

// writeSkillToDir writes a skill to the target directory
func writeSkillToDir(skill model.Skill, targetDir string) error {
	// Determine target path based on platform
	var platformDir string
	switch skill.Platform {
	case model.ClaudeCode:
		platformDir = "claude-code"
	case model.Cursor:
		platformDir = "cursor"
	case model.Codex:
		platformDir = "codex"
	default:
		platformDir = string(skill.Platform)
	}

	targetPath := filepath.Join(targetDir, platformDir)
	if err := os.MkdirAll(targetPath, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
	}

	// Write skill file
	filename := sanitizeFilename(skill.Name) + ".json"
	filePath := filepath.Join(targetPath, filename)

	data, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize skill: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	result := make([]rune, 0, len(name))
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			result = append(result, '_')
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
