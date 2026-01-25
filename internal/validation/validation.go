// Package validation provides pre-sync validation checks for skill operations.
package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// Error represents a validation failure with context.
type Error struct {
	// Field is the name of the field or component that failed validation
	Field string
	// Message describes the validation failure
	Message string
	// Err is the underlying error (if any)
	Err error
}

// Error returns a formatted validation error message.
func (ve *Error) Error() string {
	if ve.Err != nil {
		return fmt.Sprintf("validation failed for %q: %s: %v", ve.Field, ve.Message, ve.Err)
	}
	return fmt.Sprintf("validation failed for %q: %s", ve.Field, ve.Message)
}

// Unwrap returns the underlying error for errors.Is/As.
func (ve *Error) Unwrap() error {
	return ve.Err
}

// Errors collects multiple validation errors.
type Errors []error

// Error returns a formatted error message for all validation failures.
func (ve Errors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}
	return fmt.Sprintf("%d validation errors:\n- %s", len(ve), errors.Join(ve...))
}

// Options configures validation behavior.
type Options struct {
	// RequireWritePermission checks if target directory is writable
	RequireWritePermission bool
	// CheckConflicts enables conflict detection between source and target skills
	CheckConflicts bool
	// StrictMode enables additional validation checks
	StrictMode bool
}

// DefaultOptions returns the default validation options.
func DefaultOptions() Options {
	return Options{
		RequireWritePermission: true,
		CheckConflicts:         true,
		StrictMode:             false,
	}
}

// Result contains the outcome of a validation check.
type Result struct {
	// Valid indicates whether all validations passed
	Valid bool
	// Warnings contains non-fatal validation issues
	Warnings []string
	// Errors contains validation failures that prevent the operation
	Errors []error
}

// AddError adds an error to the validation result.
func (r *Result) AddError(err error) {
	r.Valid = false
	r.Errors = append(r.Errors, err)
}

// AddWarning adds a warning to the validation result.
func (r *Result) AddWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// HasErrors returns true if there are any validation errors.
func (r *Result) HasErrors() bool {
	return len(r.Errors) > 0
}

// Error returns the combined validation error message.
func (r *Result) Error() error {
	if !r.HasErrors() {
		return nil
	}
	if len(r.Errors) == 1 {
		return r.Errors[0]
	}
	return Errors(r.Errors)
}

// Summary returns a human-readable summary of the validation result.
func (r *Result) Summary() string {
	if r.Valid && len(r.Warnings) == 0 {
		return "All validations passed"
	}
	var msg string
	if r.Valid {
		msg = "Validation passed with warnings"
	} else {
		msg = "Validation failed"
	}
	if len(r.Warnings) > 0 {
		msg += fmt.Sprintf(" (%d warning(s))", len(r.Warnings))
	}
	return msg
}

// ValidateSourceTarget performs comprehensive validation before sync operations.
// Validates source and target platforms, paths, permissions, and skill formats.
func ValidateSourceTarget(source, target model.Platform, skills []model.Skill, opts Options) (*Result, error) {
	result := &Result{Valid: true}

	// Validate source platform
	if err := validatePlatform(source, "source", true); err != nil {
		result.AddError(err)
	}

	// Validate target platform
	if err := validatePlatform(target, "target", false); err != nil {
		result.AddError(err)
	}

	if result.HasErrors() {
		return result, result.Error()
	}

	// Validate skills
	for i, skill := range skills {
		if err := validateSkill(skill, i, opts); err != nil {
			result.AddError(fmt.Errorf("skill %d (%s): %w", i, skill.Name, err))
		}
	}

	// Check for conflicts if enabled
	if opts.CheckConflicts {
		if err := checkConflicts(source, target, skills); err != nil {
			result.AddError(err)
		}
	}

	// Validate target write permissions
	if opts.RequireWritePermission {
		if err := validateWritePermission(target); err != nil {
			result.AddError(err)
		}
	}

	// Add informational warnings
	if len(skills) == 0 {
		result.AddWarning("No skills found to sync")
	}

	return result, nil
}

// validatePlatform validates a platform configuration.
func validatePlatform(platform model.Platform, role string, isSource bool) error {
	if platform == "" {
		return &Error{
			Field:   fmt.Sprintf("%s platform", role),
			Message: "platform cannot be empty",
		}
	}

	// Validate platform is supported
	if !platform.IsValid() {
		return &Error{
			Field:   fmt.Sprintf("%s platform", role),
			Message: fmt.Sprintf("unsupported platform %q", platform),
		}
	}

	// Get platform path
	path, err := GetPlatformPath(platform)
	if err != nil {
		return &Error{
			Field:   fmt.Sprintf("%s platform", role),
			Message: "cannot determine platform path",
			Err:     err,
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// For source, path must exist
			if isSource {
				return &Error{
					Field:   fmt.Sprintf("%s platform", role),
					Message: fmt.Sprintf("path does not exist: %s", path),
					Err:     err,
				}
			}
			// For target, we'll create it, so just a warning
			return nil
		}
		return &Error{
			Field:   fmt.Sprintf("%s platform", role),
			Message: fmt.Sprintf("cannot access path: %s", path),
			Err:     err,
		}
	}

	// Verify it's a directory
	if !info.IsDir() {
		return &Error{
			Field:   fmt.Sprintf("%s platform", role),
			Message: fmt.Sprintf("path is not a directory: %s", path),
		}
	}

	return nil
}

// validateSkill validates a single skill's format and content.
func validateSkill(skill model.Skill, index int, opts Options) error {
	// Validate skill name
	if skill.Name == "" {
		return &Error{
			Field:   fmt.Sprintf("skills[%d].name", index),
			Message: "skill name cannot be empty",
		}
	}

	// Validate skill path
	if skill.Path == "" {
		return &Error{
			Field:   fmt.Sprintf("skills[%d].path", index),
			Message: "skill path cannot be empty",
		}
	}

	// Verify skill file exists (for source skills)
	if _, err := os.Stat(skill.Path); err != nil {
		return &Error{
			Field:   fmt.Sprintf("skills[%d].path", index),
			Message: fmt.Sprintf("cannot access skill file: %s", skill.Path),
			Err:     err,
		}
	}

	// Validate platform matches
	if skill.Platform == "" {
		return &Error{
			Field:   fmt.Sprintf("skills[%d].platform", index),
			Message: "skill platform cannot be empty",
		}
	}

	// Strict mode: validate content format
	if opts.StrictMode {
		if skill.Content == "" {
			return &Error{
				Field:   fmt.Sprintf("skills[%d].content", index),
				Message: "skill content cannot be empty in strict mode",
			}
		}
	}

	// Validate file extension matches platform expectations
	if err := validateFileExtension(skill); err != nil {
		return err
	}

	return nil
}

// validateFileExtension checks if the skill file has a valid extension for its platform.
func validateFileExtension(skill model.Skill) error {
	ext := filepath.Ext(skill.Path)

	switch skill.Platform {
	case model.ClaudeCode:
		// Claude Code accepts .md, .txt, or no extension
		if ext != "" && ext != ".md" && ext != ".txt" {
			return &Error{
				Field:   fmt.Sprintf("skill %q", skill.Name),
				Message: fmt.Sprintf("unexpected file extension %q for Claude Code skill (expected .md, .txt, or no extension)", ext),
			}
		}
	case model.Cursor:
		// Cursor requires .md or .mdc
		if ext != ".md" && ext != ".mdc" {
			return &Error{
				Field:   fmt.Sprintf("skill %q", skill.Name),
				Message: fmt.Sprintf("invalid file extension %q for Cursor skill (expected .md or .mdc)", ext),
			}
		}
	case model.Codex:
		// Codex typically uses .json
		if ext != ".json" {
			return &Error{
				Field:   fmt.Sprintf("skill %q", skill.Name),
				Message: fmt.Sprintf("invalid file extension %q for Codex skill (expected .json)", ext),
			}
		}
	}

	return nil
}

// checkConflicts checks for potential conflicts between source and target skills.
func checkConflicts(_, target model.Platform, skills []model.Skill) error {
	targetPath, err := GetPlatformPath(target)
	if err != nil {
		return fmt.Errorf("failed to get target path: %w", err)
	}

	// Check if target directory has existing skills
	for _, skill := range skills {
		// Construct potential target path
		relPath := filepath.Base(skill.Path)
		targetFile := filepath.Join(targetPath, relPath)

		// Check if file exists in target
		if _, err := os.Stat(targetFile); err == nil {
			// File exists - potential conflict
			return &Error{
				Field:   "conflict",
				Message: fmt.Sprintf("target already has skill file %q (may be overwritten)", relPath),
			}
		}
	}

	return nil
}

// validateWritePermission checks if the target directory is writable.
func validateWritePermission(platform model.Platform) error {
	path, err := GetPlatformPath(platform)
	if err != nil {
		return fmt.Errorf("failed to get platform path: %w", err)
	}

	// If path doesn't exist, check parent directory
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Dir(path)
	}

	// Check write permission by creating a temp file
	testFile := filepath.Join(path, ".skillsync-write-test")
	// #nosec G304 - testFile is constructed from validated path
	f, err := os.Create(testFile)
	if err != nil {
		return &Error{
			Field:   "write permission",
			Message: fmt.Sprintf("target directory is not writable: %s", path),
			Err:     err,
		}
	}
	_ = f.Close()

	// Clean up test file
	_ = os.Remove(testFile)

	return nil
}

// ValidateSkillsFormat validates the format of parsed skills.
// Useful for validating skills after parsing but before sync.
func ValidateSkillsFormat(skills []model.Skill, platform model.Platform) (*Result, error) {
	result := &Result{Valid: true}

	if len(skills) == 0 {
		result.AddWarning("No skills to validate")
		return result, nil
	}

	// Track skill names for uniqueness check
	names := make(map[string]bool)

	for i, skill := range skills {
		// Validate name
		if skill.Name == "" {
			result.AddError(&Error{
				Field:   fmt.Sprintf("skills[%d].name", i),
				Message: "skill name cannot be empty",
			})
			continue
		}

		// Check for duplicate names
		if names[skill.Name] {
			result.AddError(&Error{
				Field:   fmt.Sprintf("skills[%d].name", i),
				Message: fmt.Sprintf("duplicate skill name %q", skill.Name),
			})
		}
		names[skill.Name] = true

		// Validate platform matches
		if skill.Platform != platform {
			result.AddError(&Error{
				Field:   fmt.Sprintf("skills[%d].platform", i),
				Message: fmt.Sprintf("skill platform %q does not match expected platform %q", skill.Platform, platform),
			})
		}

		// Validate content is not empty
		if skill.Content == "" {
			result.AddWarning(fmt.Sprintf("skill %q has empty content", skill.Name))
		}

		// Validate path exists
		if skill.Path != "" {
			if _, err := os.Stat(skill.Path); err != nil {
				result.AddWarning(fmt.Sprintf("skill %q path not accessible: %v", skill.Name, err))
			}
		}
	}

	return result, nil
}

// ValidatePath checks if a path is valid for the given platform.
func ValidatePath(path string, _ model.Platform) error {
	if path == "" {
		return &Error{
			Field:   "path",
			Message: "path cannot be empty",
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &Error{
			Field:   "path",
			Message: "cannot convert to absolute path",
			Err:     err,
		}
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Error{
				Field:   "path",
				Message: fmt.Sprintf("path does not exist: %s", absPath),
				Err:     err,
			}
		}
		return &Error{
			Field:   "path",
			Message: fmt.Sprintf("cannot access path: %s", absPath),
			Err:     err,
		}
	}

	// For platforms expecting directories, verify it's a directory
	if !info.IsDir() {
		return &Error{
			Field:   "path",
			Message: fmt.Sprintf("path is not a directory: %s", absPath),
		}
	}

	return nil
}

// GetPlatformPath returns the default path for a platform.
func GetPlatformPath(platform model.Platform) (string, error) {
	switch platform {
	case model.ClaudeCode:
		return util.ClaudeCodeSkillsPath(), nil
	case model.Cursor:
		return util.CursorSkillsPath(), nil
	case model.Codex:
		// Codex is project-specific, use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot get current directory: %w", err)
		}
		return util.CodexConfigPath(cwd), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}
