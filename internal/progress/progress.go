// Package progress provides progress indicators for long-running operations.
package progress

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/schollz/progressbar/v3"
)

// Bar wraps progressbar functionality with integration to skillsync's UI and logging.
type Bar struct {
	bar     *progressbar.ProgressBar
	enabled bool
	desc    string
}

// Options configures the progress bar behavior.
type Options struct {
	// Max is the maximum value for the progress bar (total steps).
	Max int64
	// Description is the prefix text shown before the progress bar.
	Description string
	// Writer is the output destination. Defaults to os.Stderr.
	Writer io.Writer
	// ShowElapsed shows elapsed time.
	ShowElapsed bool
	// ShowCount shows current/total count (e.g., "5/10").
	ShowCount bool
}

// DefaultOptions returns sensible defaults for CLI progress bars.
func DefaultOptions() Options {
	return Options{
		Max:         100,
		Description: "Processing",
		Writer:      os.Stderr,
		ShowElapsed: true,
		ShowCount:   true,
	}
}

// New creates a new progress bar with the given options.
// The bar is only shown if:
//   - Colors are enabled (respects NO_COLOR and --no-color)
//   - Output is a terminal
//   - Not in debug/verbose mode (to avoid interfering with logs)
func New(opts Options) *Bar {
	if opts.Writer == nil {
		opts.Writer = os.Stderr
	}

	// Determine if progress should be shown
	enabled := shouldShowProgress(opts.Writer)

	b := &Bar{
		enabled: enabled,
		desc:    opts.Description,
	}

	if !enabled {
		// Log start at debug level instead
		logging.Debug(fmt.Sprintf("%s started", opts.Description),
			logging.Count(int(opts.Max)))
		return b
	}

	// Create the progress bar with schollz/progressbar
	b.bar = progressbar.NewOptions64(
		opts.Max,
		progressbar.OptionSetDescription(opts.Description),
		progressbar.OptionSetWriter(opts.Writer),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(opts.Writer, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(ui.IsColorEnabled()),
	)

	return b
}

// Add increments the progress bar by n steps.
func (b *Bar) Add(n int) error {
	if !b.enabled {
		return nil
	}
	return b.bar.Add(n)
}

// Add64 increments the progress bar by n steps (int64 version).
func (b *Bar) Add64(n int64) error {
	if !b.enabled {
		return nil
	}
	return b.bar.Add64(n)
}

// Set sets the progress bar to a specific value.
func (b *Bar) Set(n int) error {
	if !b.enabled {
		return nil
	}
	return b.bar.Set(n)
}

// Set64 sets the progress bar to a specific value (int64 version).
func (b *Bar) Set64(n int64) error {
	if !b.enabled {
		return nil
	}
	return b.bar.Set64(n)
}

// Describe updates the progress bar description.
func (b *Bar) Describe(desc string) {
	b.desc = desc
	if !b.enabled {
		return
	}
	b.bar.Describe(desc)
}

// Finish completes the progress bar and logs completion.
func (b *Bar) Finish() error {
	if !b.enabled {
		logging.Debug(fmt.Sprintf("%s completed", b.desc))
		return nil
	}
	return b.bar.Finish()
}

// Clear removes the progress bar from the terminal.
func (b *Bar) Clear() error {
	if !b.enabled {
		return nil
	}
	return b.bar.Clear()
}

// IsFinished returns true if the progress bar has reached its max value.
func (b *Bar) IsFinished() bool {
	if !b.enabled {
		return false
	}
	return b.bar.IsFinished()
}

// shouldShowProgress determines if progress bars should be displayed.
// Progress is disabled if:
//   - Not outputting to a terminal
//   - Colors are disabled (NO_COLOR, --no-color)
//   - Logger is at debug level (to avoid interfering with debug output)
func shouldShowProgress(w io.Writer) bool {
	// Check if colors are enabled (respects NO_COLOR)
	if !ui.IsColorEnabled() {
		return false
	}

	// Check if we're outputting to a terminal
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		// Check if it's a terminal (CharDevice) vs pipe/file
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			return false
		}
	}

	// Disable progress if at debug level (avoid interfering with logs)
	ctx := context.Background()
	if logging.Default().Enabled(ctx, logging.LevelDebug) {
		return false
	}

	return true
}

// Simple creates a simple progress bar without needing to configure options.
// This is a convenience function for common cases.
func Simple(max int64, description string) *Bar {
	return New(Options{
		Max:         max,
		Description: description,
		Writer:      os.Stderr,
		ShowElapsed: true,
		ShowCount:   true,
	})
}
