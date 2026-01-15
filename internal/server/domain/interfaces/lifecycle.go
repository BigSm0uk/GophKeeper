package interfaces

import "context"

// Lifecycle defines the interface for components that can be started and stopped.
type Lifecycle interface {
	// Start initializes and starts the component.
	// It should be non-blocking if the component runs continuously.
	Start(ctx context.Context) error

	// Stop gracefully stops the component.
	// It should handle cleanup and release resources.
	Stop(ctx context.Context) error

	// Name returns the component name for logging purposes.
	Name() string
}
