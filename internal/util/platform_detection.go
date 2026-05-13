package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// PlatformDetectionStatus describes how completely a platform was detected.
type PlatformDetectionStatus string

const (
	// PlatformDetectionMissing means no configured paths were found.
	PlatformDetectionMissing PlatformDetectionStatus = "missing"
	// PlatformDetectionPartial means some configured paths were found, but not all.
	PlatformDetectionPartial PlatformDetectionStatus = "partial"
	// PlatformDetectionPresent means all checked configured paths were found.
	PlatformDetectionPresent PlatformDetectionStatus = "present"
)

// PlatformDetection captures the detection state for one platform.
type PlatformDetection struct {
	Platform     model.Platform          `json:"platform"`
	Status       PlatformDetectionStatus `json:"status"`
	CheckedPaths []ScopedPath            `json:"checked_paths"`
	PresentPaths []ScopedPath            `json:"present_paths"`
	MissingPaths []ScopedPath            `json:"missing_paths"`
	Reason       string                  `json:"reason"`
}

// PlatformDetectionResult contains the full detection output.
type PlatformDetectionResult struct {
	Detected []model.Platform    `json:"detected"`
	Details  []PlatformDetection `json:"details"`
}

// DetectInstalledPlatforms inspects the default locations for all supported
// platforms and returns the detected platforms plus reasons for missing or
// partial matches.
func DetectInstalledPlatforms() PlatformDetectionResult {
	return DetectInstalledPlatformsWithConfig(TieredPathConfig{})
}

// DetectInstalledPlatformsWithConfig inspects a caller-supplied working tree.
// If WorkingDir is empty, it falls back to the current process directory.
func DetectInstalledPlatformsWithConfig(cfg TieredPathConfig) PlatformDetectionResult {
	if cfg.WorkingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			cfg.WorkingDir = cwd
		}
	}

	result := PlatformDetectionResult{
		Detected: make([]model.Platform, 0, len(model.AllPlatforms())),
		Details:  make([]PlatformDetection, 0, len(model.AllPlatforms())),
	}

	for _, platform := range model.AllPlatforms() {
		platformCfg := cfg
		platformCfg.Platform = platform

		checked := GetAllSearchPaths(platformCfg)
		present := FilterExistingPaths(checked)
		missing := missingScopedPaths(checked, present)

		detail := PlatformDetection{
			Platform:     platform,
			Status:       PlatformDetectionMissing,
			CheckedPaths: checked,
			PresentPaths: present,
			MissingPaths: missing,
			Reason:       fmt.Sprintf("no configured paths found; checked %s", formatScopedPaths(checked)),
		}

		switch {
		case len(present) == 0:
			// Keep the default missing status and reason.
		case len(missing) == 0:
			detail.Status = PlatformDetectionPresent
			detail.Reason = fmt.Sprintf("found %d configured path(s): %s", len(present), formatScopedPaths(present))
			result.Detected = append(result.Detected, platform)
		default:
			detail.Status = PlatformDetectionPartial
			detail.Reason = fmt.Sprintf(
				"found %d of %d configured path(s): %s; missing %s",
				len(present),
				len(checked),
				formatScopedPaths(present),
				formatScopedPaths(missing),
			)
			result.Detected = append(result.Detected, platform)
		}

		result.Details = append(result.Details, detail)
	}

	return result
}

// Detail returns the detection result for a single platform.
func (r PlatformDetectionResult) Detail(platform model.Platform) (PlatformDetection, bool) {
	for _, detail := range r.Details {
		if detail.Platform == platform {
			return detail, true
		}
	}
	return PlatformDetection{}, false
}

// HasPlatform reports whether the platform was detected at any configured path.
func (r PlatformDetectionResult) HasPlatform(platform model.Platform) bool {
	for _, detected := range r.Detected {
		if detected == platform {
			return true
		}
	}
	return false
}

func missingScopedPaths(checked, present []ScopedPath) []ScopedPath {
	if len(checked) == 0 {
		return nil
	}

	presentByPath := make(map[string]struct{}, len(present))
	for _, path := range present {
		presentByPath[path.Path] = struct{}{}
	}

	missing := make([]ScopedPath, 0, len(checked))
	for _, path := range checked {
		if _, ok := presentByPath[path.Path]; ok {
			continue
		}
		missing = append(missing, path)
	}
	return missing
}

func formatScopedPaths(paths []ScopedPath) string {
	if len(paths) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		parts = append(parts, fmt.Sprintf("%s:%s", path.Scope.String(), path.Path))
	}
	return strings.Join(parts, ", ")
}
