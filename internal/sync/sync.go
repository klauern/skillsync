package sync

import "github.com/klauern/skillsync/internal/model"

// Syncer defines the interface for synchronization strategies
type Syncer interface {
	// Sync performs synchronization between platforms
	Sync(source, target model.Platform) error
}
