package model

import (
	"fmt"
	"strings"
)

// NativePackageKind identifies a harness-owned installation surface.
type NativePackageKind string

const (
	// NativePackagePlugin identifies a plugin installation.
	NativePackagePlugin NativePackageKind = "plugin"
	// NativePackagePackage identifies a package installation.
	NativePackagePackage NativePackageKind = "package"
	// NativePackageExtension identifies an extension installation.
	NativePackageExtension NativePackageKind = "extension"
)

// IsValid reports whether the kind is supported.
func (k NativePackageKind) IsValid() bool {
	switch k {
	case NativePackagePlugin, NativePackagePackage, NativePackageExtension:
		return true
	default:
		return false
	}
}

// NativePackageProvenance identifies the source of an installation.
type NativePackageProvenance string

const (
	// NativeProvenanceMarketplace identifies a harness marketplace install.
	NativeProvenanceMarketplace NativePackageProvenance = "marketplace"
	// NativeProvenanceRegistry identifies a package registry install.
	NativeProvenanceRegistry NativePackageProvenance = "registry"
	// NativeProvenanceRepository identifies a source repository install.
	NativeProvenanceRepository NativePackageProvenance = "repository"
	// NativeProvenanceLocal identifies an install from a local path.
	NativeProvenanceLocal NativePackageProvenance = "local"
	// NativeProvenanceBuiltin identifies content supplied by the harness.
	NativeProvenanceBuiltin NativePackageProvenance = "builtin"
)

// IsValid reports whether the provenance is supported.
func (p NativePackageProvenance) IsValid() bool {
	switch p {
	case NativeProvenanceMarketplace, NativeProvenanceRegistry, NativeProvenanceRepository,
		NativeProvenanceLocal, NativeProvenanceBuiltin:
		return true
	default:
		return false
	}
}

// NativePackage describes one harness-owned installation.
// It is separate from Skill because it does not imply portable skill content.
type NativePackage struct {
	Name       string                  `json:"name"`
	Kind       NativePackageKind       `json:"kind"`
	Platform   Platform                `json:"platform"`
	Provenance NativePackageProvenance `json:"provenance"`
	Version    string                  `json:"version,omitempty"`
	Source     string                  `json:"source,omitempty"`
	Path       string                  `json:"path,omitempty"`
	Scope      string                  `json:"scope,omitempty"`
	Enabled    bool                    `json:"enabled,omitempty"`
	// MappingKey permits an explicit cross-harness mapping. An empty value
	// means that the installation remains native to its source platform.
	MappingKey string            `json:"mapping_key,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Validate checks the required identity and supported enum values.
func (p NativePackage) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("native package name is required")
	}
	if !p.Kind.IsValid() {
		return fmt.Errorf("unsupported native package kind %q", p.Kind)
	}
	if !p.Platform.IsValid() {
		return fmt.Errorf("unsupported native package platform %q", p.Platform)
	}
	if !p.Provenance.IsValid() {
		return fmt.Errorf("unsupported native package provenance %q", p.Provenance)
	}
	return nil
}

// Key returns a stable key for an installation on one platform.
// MappingKey does not change this native identity.
func (p NativePackage) Key() string {
	return strings.Join([]string{string(p.Platform), string(p.Kind), strings.TrimSpace(p.Name)}, ":")
}
