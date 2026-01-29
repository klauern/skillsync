// Package security provides sensitive data detection and security validation for skills.
package security

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/klauern/skillsync/internal/validation"
)

// SensitivePattern represents a pattern to detect sensitive data
type SensitivePattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
	Severity    string // "error" or "warning"
}

// Detector performs sensitive data detection with configurable patterns.
type Detector struct {
	patterns []SensitivePattern
}

// DefaultPatterns returns the default built-in sensitive data patterns.
func DefaultPatterns() []SensitivePattern {
	return []SensitivePattern{
		{
			Name:        "API Key",
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*['\"]?[a-zA-Z0-9_\-]{16,}['\"]?`),
			Description: "API key pattern detected",
			Severity:    "warning",
		},
		{
			Name:        "Token",
			Pattern:     regexp.MustCompile(`(?i)(token|access[_-]?token|auth[_-]?token)\s*[:=]\s*['\"]?[a-zA-Z0-9_\-\.]{16,}['\"]?`),
			Description: "Authentication token pattern detected",
			Severity:    "warning",
		},
		{
			Name:        "Password",
			Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['\"]?[a-zA-Z0-9_\-@!#$%^&*()]{8,}['\"]?`),
			Description: "Password pattern detected",
			Severity:    "warning",
		},
		{
			Name:        "AWS Access Key",
			Pattern:     regexp.MustCompile(`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?key)\s*[:=]\s*['\"]?AKIA[A-Z0-9]{16}['\"]?`),
			Description: "AWS access key detected",
			Severity:    "error",
		},
		{
			Name:        "AWS Secret Key",
			Pattern:     regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?access[_-]?key|aws[_-]?secret)\s*[:=]\s*['\"]?[a-zA-Z0-9\/\+]{40}['\"]?`),
			Description: "AWS secret key detected",
			Severity:    "error",
		},
		{
			Name:        "GitHub Token",
			Pattern:     regexp.MustCompile(`(?i)(github[_-]?token|gh[_-]?token)\s*[:=]\s*['\"]?ghp_[a-zA-Z0-9]{36,}['\"]?`),
			Description: "GitHub personal access token detected",
			Severity:    "error",
		},
		{
			Name:        "Private Key",
			Pattern:     regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
			Description: "Private key detected",
			Severity:    "error",
		},
		{
			Name:        "Generic Secret",
			Pattern:     regexp.MustCompile(`(?i)(secret|secret[_-]?key)\s*[:=]\s*['\"]?[a-zA-Z0-9_\-]{16,}['\"]?`),
			Description: "Generic secret pattern detected",
			Severity:    "warning",
		},
		{
			Name:        "Bearer Token",
			Pattern:     regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`),
			Description: "Bearer token detected",
			Severity:    "warning",
		},
		{
			Name:        "Database Connection String",
			Pattern:     regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis):\/\/[^:]+:[^@]+@`),
			Description: "Database connection string with credentials detected",
			Severity:    "error",
		},
	}
}

// NewDetector creates a new detector with the given patterns.
// If patterns is nil or empty, uses DefaultPatterns().
func NewDetector(patterns []SensitivePattern) *Detector {
	if len(patterns) == 0 {
		patterns = DefaultPatterns()
	}
	return &Detector{
		patterns: patterns,
	}
}

// NewDetectorDefault creates a new detector with default patterns.
func NewDetectorDefault() *Detector {
	return NewDetector(nil)
}

// Detection represents a single detection of sensitive data
type Detection struct {
	Pattern     string
	Line        int
	Column      int
	Content     string
	Severity    string
	Description string
}

// ScanContent scans content for sensitive data patterns using this detector's patterns.
func (d *Detector) ScanContent(content string) *validation.Result {
	result := &validation.Result{Valid: true}

	if content == "" {
		return result
	}

	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Skip common false positives
		if isFalsePositive(line) {
			continue
		}

		for _, pattern := range d.patterns {
			if pattern.Pattern.MatchString(line) {
				detection := Detection{
					Pattern:     pattern.Name,
					Line:        lineNum + 1, // 1-indexed for human readability
					Column:      pattern.Pattern.FindStringIndex(line)[0] + 1,
					Content:     truncateLine(line, 80),
					Severity:    pattern.Severity,
					Description: pattern.Description,
				}

				// Add to result based on severity
				msg := fmt.Sprintf(
					"%s at line %d: %s",
					pattern.Description,
					detection.Line,
					detection.Content,
				)

				if pattern.Severity == "error" {
					result.AddError(&validation.Error{
						Field:   "content",
						Message: msg,
					})
				} else {
					result.AddWarning(msg)
				}
			}
		}
	}

	return result
}

// isFalsePositive checks if a line is likely a false positive
func isFalsePositive(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Skip comments
	if strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") {
		return true
	}

	// Skip documentation examples - but only if the value part looks like a placeholder
	// Check for common placeholder patterns in the value (after : or =)
	if strings.Contains(trimmed, ":") || strings.Contains(trimmed, "=") {
		// Split by : or = to get the value part
		parts := strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == ':' || r == '='
		})
		if len(parts) >= 2 {
			valuePart := strings.ToLower(strings.TrimSpace(parts[1]))
			// Only consider false positive if value looks like a placeholder
			if strings.Contains(valuePart, "your_") ||
				strings.Contains(valuePart, "<your") ||
				strings.Contains(valuePart, "placeholder") ||
				strings.Contains(valuePart, "example_") ||
				strings.HasPrefix(valuePart, "\"xxx") ||
				strings.HasPrefix(valuePart, "'xxx") ||
				valuePart == "xxxxxxxxxxxxx" {
				return true
			}
		}
	}

	return false
}

// truncateLine truncates a line to the specified length with ellipsis
func truncateLine(line string, maxLen int) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen-3] + "..."
}

// ScanContent scans content for sensitive data patterns using default patterns.
// This is a convenience function for backward compatibility.
func ScanContent(content string) *validation.Result {
	detector := NewDetectorDefault()
	return detector.ScanContent(content)
}

// ValidateSkillContent is a convenience function to validate skill content using default patterns.
func ValidateSkillContent(content string, skillName string) *validation.Result {
	detector := NewDetectorDefault()
	result := detector.ScanContent(content)

	if !result.Valid {
		// Add context about which skill has the issue
		for i, e := range result.Errors {
			var validationErr *validation.Error
			if errors.As(e, &validationErr) {
				result.Errors[i] = &validation.Error{
					Field:   fmt.Sprintf("skill:%s:%s", skillName, validationErr.Field),
					Message: validationErr.Message,
					Err:     validationErr.Err,
				}
			}
		}
	}

	return result
}
