// Package sync implements skill synchronization logic across platforms.
// It provides strategies for merging, overwriting, and smart syncing
// of agent skills with conflict resolution.
//
// # Features
//
// The sync package supports:
//   - Unidirectional sync (source -> target) via Sync() method
//   - Bidirectional sync (platform A <-> platform B) via SyncBidirectional()
//   - Real-time progress reporting through ProgressCallback
//   - Multiple merge strategies (overwrite, skip, newer, merge, three-way, interactive)
//   - Conflict detection and resolution
//   - Dry-run mode for previewing changes
//
// # Progress Reporting
//
// Progress can be tracked by providing a ProgressCallback in Options:
//
//	opts := sync.Options{
//	    Progress: func(event sync.ProgressEvent) error {
//	        fmt.Printf("Progress: %s - %d%%\n", event.Message, event.PercentComplete)
//	        return nil // Return error to cancel sync
//	    },
//	}
//
// Progress events are emitted for:
//   - Sync start (ProgressEventStart)
//   - Each skill start (ProgressEventSkillStart)
//   - Each skill completion (ProgressEventSkillComplete)
//   - Sync completion (ProgressEventComplete)
//   - Errors (ProgressEventError)
//
// The callback can return an error to cancel the synchronization operation.
//
// # Bidirectional Sync
//
// Bidirectional sync reconciles changes between two platforms:
//
//	result, err := syncer.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check results for each direction
//	if result.ResultAtoB != nil {
//	    fmt.Printf("A->B: %d created, %d updated\n",
//	        len(result.ResultAtoB.Created()), len(result.ResultAtoB.Updated()))
//	}
//	if result.ResultBtoA != nil {
//	    fmt.Printf("B->A: %d created, %d updated\n",
//	        len(result.ResultBtoA.Created()), len(result.ResultBtoA.Updated()))
//	}
//
//	// Handle conflicts
//	if result.HasConflicts() {
//	    for _, conflict := range result.Conflicts {
//	        fmt.Printf("Conflict: %s\n", conflict.Name)
//	    }
//	}
//
// Bidirectional sync behavior depends on the strategy:
//   - StrategyNewer: Uses timestamp to determine sync direction
//   - StrategyOverwrite: Prefers A->B by default
//   - StrategyThreeWay/Interactive: Requires manual resolution
//   - StrategyMerge/Skip: Treats conflicts as requiring resolution
//
// # Sync Strategies
//
// Available strategies:
//   - StrategyOverwrite: Replace target unconditionally
//   - StrategySkip: Skip existing skills
//   - StrategyNewer: Sync only if source is newer (timestamp-based)
//   - StrategyMerge: Concatenate content with headers
//   - StrategyThreeWay: Intelligent merge with conflict detection
//   - StrategyInteractive: Prompt for each conflict
package sync
