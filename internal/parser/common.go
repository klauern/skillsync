package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterResult contains the parsed frontmatter and remaining content.
type FrontmatterResult struct {
	// Frontmatter contains the raw frontmatter bytes (YAML or JSON)
	Frontmatter []byte
	// Content contains the remaining content after frontmatter
	Content string
	// HasFrontmatter indicates whether frontmatter was found
	HasFrontmatter bool
}

// SplitFrontmatter extracts YAML or JSON frontmatter from content.
// Supports both --- (YAML) and +++ (TOML/alternative) delimiters.
// Returns the frontmatter bytes, remaining content, and whether frontmatter was found.
func SplitFrontmatter(content []byte) FrontmatterResult {
	// Check for YAML frontmatter (---)
	if bytes.HasPrefix(content, []byte("---\n")) || bytes.HasPrefix(content, []byte("---\r\n")) {
		return extractFrontmatter(content, []byte("---"))
	}

	// Check for alternative frontmatter (+++)
	if bytes.HasPrefix(content, []byte("+++\n")) || bytes.HasPrefix(content, []byte("+++\r\n")) {
		return extractFrontmatter(content, []byte("+++"))
	}

	// No frontmatter found
	return FrontmatterResult{
		Frontmatter:    nil,
		Content:        string(content),
		HasFrontmatter: false,
	}
}

// extractFrontmatter extracts frontmatter between delimiters.
func extractFrontmatter(content []byte, delimiter []byte) FrontmatterResult {
	// Skip opening delimiter
	remaining := content[len(delimiter):]

	// Handle both \n and \r\n line endings
	if bytes.HasPrefix(remaining, []byte("\r\n")) {
		remaining = remaining[2:]
	} else if bytes.HasPrefix(remaining, []byte("\n")) {
		remaining = remaining[1:]
	}

	// Find closing delimiter
	// First check if it's right at the start (empty frontmatter case)
	var frontmatter []byte
	var bodyStart int
	delimFound := false

	if bytes.HasPrefix(remaining, delimiter) {
		// Empty frontmatter case: ---\n---\n
		frontmatter = []byte{}
		bodyStart = len(delimiter)
		delimFound = true
	} else {
		// Try to find closing delimiter preceded by newline
		// Try Unix line ending first
		closingDelim := append([]byte("\n"), delimiter...)
		idx := bytes.Index(remaining, closingDelim)
		if idx != -1 {
			frontmatter = remaining[:idx]
			bodyStart = idx + len(closingDelim)
			delimFound = true
		} else {
			// Try Windows line ending
			closingDelim = append([]byte("\r\n"), delimiter...)
			idx = bytes.Index(remaining, closingDelim)
			if idx != -1 {
				frontmatter = remaining[:idx]
				bodyStart = idx + len(closingDelim)
				delimFound = true
			}
		}
	}

	if !delimFound {
		// No closing delimiter found, treat entire content as no frontmatter
		return FrontmatterResult{
			Frontmatter:    nil,
			Content:        string(content),
			HasFrontmatter: false,
		}
	}

	// Normalize frontmatter by removing \r from Windows line endings
	cleanFrontmatter := bytes.ReplaceAll(frontmatter, []byte("\r\n"), []byte("\n"))
	cleanFrontmatter = bytes.TrimRight(cleanFrontmatter, "\r")

	// Skip trailing newline after closing delimiter
	if bodyStart < len(remaining) {
		if bytes.HasPrefix(remaining[bodyStart:], []byte("\r\n")) {
			bodyStart += 2
		} else if bytes.HasPrefix(remaining[bodyStart:], []byte("\n")) {
			bodyStart++
		}
	}

	var body string
	if bodyStart < len(remaining) {
		body = string(remaining[bodyStart:])
	}

	return FrontmatterResult{
		Frontmatter:    cleanFrontmatter,
		Content:        body,
		HasFrontmatter: true,
	}
}

// ParseYAMLFrontmatter parses YAML frontmatter into a map.
func ParseYAMLFrontmatter(frontmatter []byte) (map[string]interface{}, error) {
	if len(frontmatter) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(frontmatter, &result); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return result, nil
}

// DiscoverFiles finds all files matching the given patterns in a directory.
// Patterns are glob patterns relative to the base directory.
// Supports ** for recursive matching (custom implementation).
// Returns absolute paths to matching files.
func DiscoverFiles(baseDir string, patterns []string) ([]string, error) {
	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		// Not an error if directory doesn't exist, just return empty slice
		return []string{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat directory %q: %w", baseDir, err)
	}

	var files []string
	seen := make(map[string]bool) // Deduplicate results

	for _, pattern := range patterns {
		// Check if pattern uses ** for recursive matching
		if strings.Contains(pattern, "**") {
			// Custom recursive matching
			matches, err := walkMatch(baseDir, pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to walk pattern %q: %w", pattern, err)
			}
			for _, match := range matches {
				if !seen[match] {
					seen[match] = true
					files = append(files, match)
				}
			}
		} else {
			// Standard glob matching
			fullPattern := filepath.Join(baseDir, pattern)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return nil, fmt.Errorf("failed to glob pattern %q: %w", pattern, err)
			}

			for _, match := range matches {
				// Skip directories
				info, err := os.Stat(match)
				if err != nil {
					continue // Skip files we can't stat
				}
				if info.IsDir() {
					continue
				}

				// Get absolute path
				absPath, err := filepath.Abs(match)
				if err != nil {
					return nil, fmt.Errorf("failed to get absolute path for %q: %w", match, err)
				}

				// Deduplicate
				if !seen[absPath] {
					seen[absPath] = true
					files = append(files, absPath)
				}
			}
		}
	}

	return files, nil
}

// walkMatch performs recursive file matching for patterns containing **.
// It follows symlinks to directories to support symlinked skill directories.
func walkMatch(baseDir, pattern string) ([]string, error) {
	var matches []string

	// Remove ** from pattern to get the file extension or pattern to match
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid pattern %q: only one ** supported", pattern)
	}

	// The suffix after ** (e.g., "/*.md" becomes "*.md")
	suffix := strings.TrimPrefix(parts[1], "/")

	// Use a custom walker that follows symlinks
	err := walkFollowSymlinks(baseDir, func(path string, info os.FileInfo) error {
		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from baseDir
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil // Skip on error
		}

		// Match against the suffix pattern
		matched, err := filepath.Match(suffix, filepath.Base(relPath))
		if err != nil {
			return nil // Skip on error
		}

		if matched {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil // Skip on error
			}
			matches = append(matches, absPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return matches, nil
}

// walkFollowSymlinks walks a directory tree, following symlinks to directories.
// It detects and avoids cycles by tracking visited directories.
func walkFollowSymlinks(root string, walkFn func(path string, info os.FileInfo) error) error {
	visited := make(map[string]bool)
	return walkFollowSymlinksImpl(root, visited, walkFn)
}

func walkFollowSymlinksImpl(path string, visited map[string]bool, walkFn func(path string, info os.FileInfo) error) error {
	// Resolve symlinks to get the real path for cycle detection
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil // Skip paths we can't resolve
	}

	// Check for cycles
	if visited[realPath] {
		return nil
	}
	visited[realPath] = true

	// Get info about the path (follows symlinks)
	info, err := os.Stat(path)
	if err != nil {
		return nil // Skip paths we can't stat
	}

	// Call the walk function
	if err := walkFn(path, info); err != nil {
		return err
	}

	// If it's a directory, recurse into it
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil // Skip directories we can't read
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			if err := walkFollowSymlinksImpl(childPath, visited, walkFn); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateSkillName checks if a skill name is valid.
// Valid names contain only alphanumeric characters, hyphens, and underscores.
func ValidateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	// Trim whitespace
	trimmed := strings.TrimSpace(name)
	if trimmed != name {
		return fmt.Errorf("skill name cannot have leading/trailing whitespace: %q", name)
	}

	// Check for valid characters
	for _, r := range name {
		if !isValidNameChar(r) {
			return fmt.Errorf("skill name contains invalid character %q: %q", r, name)
		}
	}

	return nil
}

// isValidNameChar returns true if the rune is valid in a skill name.
func isValidNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == ':' || r == '/'
}

// NormalizeContent trims excessive whitespace from content.
func NormalizeContent(content string) string {
	// Trim leading/trailing whitespace
	trimmed := strings.TrimSpace(content)

	// Normalize line endings to \n
	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")

	return normalized
}
