// Package trust classifies behavior-bearing artifacts before synchronization.
package trust

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// Risk identifies a category of behavior-bearing artifact content.
type Risk string

const (
	// RiskExecutable identifies files with executable permission bits.
	RiskExecutable Risk = "executable"
	// RiskExternalReference identifies URLs, escaping paths, and symbolic links.
	RiskExternalReference Risk = "external-reference"
	// RiskNativeConfig identifies metadata that configures runtime behavior.
	RiskNativeConfig Risk = "native-config"
)

// Decision records an auditable allow or block decision for one risk.
type Decision struct {
	Artifact string `json:"artifact"`
	Risk     Risk   `json:"risk"`
	Allowed  bool   `json:"allowed"`
	Reason   string `json:"reason"`
}

// Policy contains the risk categories that an explicit override allows.
type Policy struct{ Allowed map[Risk]bool }

// ParseAllowed parses a comma-separated list of allowed risk categories.
func ParseAllowed(value string) (Policy, error) {
	p := Policy{Allowed: make(map[Risk]bool)}
	for item := range strings.SplitSeq(value, ",") {
		risk := Risk(strings.TrimSpace(item))
		if risk == "" {
			continue
		}
		switch risk {
		case RiskExecutable, RiskExternalReference, RiskNativeConfig:
			p.Allowed[risk] = true
		default:
			return Policy{}, fmt.Errorf("unknown trust category %q", item)
		}
	}
	return p, nil
}

// Evaluate inspects an artifact and applies the policy to each detected risk.
func (p Policy) Evaluate(skill model.Skill, root string) ([]Decision, error) {
	findings := make(map[Risk]string)
	if len(skill.Scripts) > 0 {
		findings[RiskExecutable] = "artifact declares executable scripts"
	}
	for _, ref := range skill.References {
		if isExternalReference(ref, root) {
			findings[RiskExternalReference] = "reference points outside the artifact"
		}
	}
	for key := range skill.Metadata {
		switch strings.ToLower(key) {
		case "hooks", "mcp-servers", "mcp_servers", "command", "commands", "approval_policy", "sandbox_mode":
			findings[RiskNativeConfig] = "metadata contains native runtime configuration"
		}
	}
	info, err := os.Lstat(root)
	if err != nil {
		if os.IsNotExist(err) && skill.Content != "" {
			return decisionsFromFindings(findings, p, root), nil
		}
		return nil, fmt.Errorf("inspect artifact %q: %w", root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		findings[RiskExternalReference] = "artifact is a symbolic link"
	} else if info.IsDir() {
		err = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			entryInfo, infoErr := entry.Info()
			if infoErr != nil {
				return fmt.Errorf("inspect artifact entry: %w", infoErr)
			}
			if entryInfo.Mode()&os.ModeSymlink != 0 {
				findings[RiskExternalReference] = "artifact contains a symbolic link"
			}
			if entryInfo.Mode().IsRegular() && entryInfo.Mode().Perm()&0o111 != 0 {
				findings[RiskExecutable] = "artifact contains an executable file"
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("inspect artifact tree %q: %w", root, err)
		}
	} else if info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
		findings[RiskExecutable] = "artifact file is executable"
	}
	return decisionsFromFindings(findings, p, root), nil
}

func decisionsFromFindings(findings map[Risk]string, p Policy, root string) []Decision {
	decisions := make([]Decision, 0, len(findings))
	for _, risk := range []Risk{RiskExecutable, RiskExternalReference, RiskNativeConfig} {
		reason, found := findings[risk]
		if !found {
			continue
		}
		allowed := p.Allowed[risk]
		if allowed {
			reason += "; allowed by explicit override"
		}
		decisions = append(decisions, Decision{Artifact: root, Risk: risk, Allowed: allowed, Reason: reason})
	}
	return decisions
}

func isExternalReference(value, root string) bool {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" {
		return true
	}
	clean := filepath.Clean(value)
	if filepath.IsAbs(value) {
		base := root
		if info, statErr := os.Stat(root); statErr == nil && !info.IsDir() {
			base = filepath.Dir(root)
		}
		rel, relErr := filepath.Rel(base, value)
		return relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator))
	}
	return clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator))
}
