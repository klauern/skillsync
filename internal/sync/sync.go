package sync

import "github.com/klauern/skillsync/internal/model"

// Options configures synchronization behavior
type Options struct {
	// DryRun enables preview mode without making actual changes
	DryRun bool
}

// Syncer defines the interface for synchronization strategies
type Syncer interface {
	// Sync performs synchronization between platforms.
	// When opts.DryRun is true, returns a preview of changes without modifying files.
	Sync(source, target model.Platform, opts Options) error
}
