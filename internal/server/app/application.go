package app

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"go.uber.org/zap"
)

const (
	// DefaultShutdownTimeout is the default timeout for graceful shutdown
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultStartupTimeout is the default timeout for component startup
	DefaultStartupTimeout = 10 * time.Second
)

// Application manages the lifecycle of all application components.
// This is the Application Layer in Clean Architecture.
type Application struct {
	components        []interfaces.Lifecycle
	logger            *zap.Logger
	shutdownTimeout   time.Duration
	startupTimeout    time.Duration
	startedComponents []interfaces.Lifecycle
}

// NewApplication creates a new application instance.
func NewApplication(logger *zap.Logger) *Application {
	return &Application{
		components:      make([]interfaces.Lifecycle, 0),
		logger:          logger,
		shutdownTimeout: DefaultShutdownTimeout,
		startupTimeout:  DefaultStartupTimeout,
	}
}

// AddComponent registers a component to be managed by the application.
// Must be called before Start().
func (a *Application) AddComponent(component interfaces.Lifecycle) {
	a.components = append(a.components, component)
}

// SetShutdownTimeout sets the timeout for graceful shutdown.
func (a *Application) SetShutdownTimeout(timeout time.Duration) {
	a.shutdownTimeout = timeout
}

// SetStartupTimeout sets the timeout for component startup.
func (a *Application) SetStartupTimeout(timeout time.Duration) {
	a.startupTimeout = timeout
}

// Start starts all registered components in order.
// If any component fails to start, it stops all previously started components.
func (a *Application) Start(ctx context.Context) error {
	a.logger.Info("Starting application components",
		zap.Int("count", len(a.components)),
	)

	for _, component := range a.components {
		if err := a.startComponent(ctx, component); err != nil {
			a.logger.Error("Failed to start component",
				zap.String("component", component.Name()),
				zap.Error(err),
			)
			// Stop all previously started components
			_ = a.stopStartedComponents(ctx)
			return fmt.Errorf("failed to start %s: %w", component.Name(), err)
		}

		a.startedComponents = append(a.startedComponents, component)

		a.logger.Info("Component started successfully",
			zap.String("component", component.Name()),
		)
	}

	a.logger.Info("All components started successfully")
	return nil
}

// startComponent starts a single component with timeout.
func (a *Application) startComponent(ctx context.Context, component interfaces.Lifecycle) error {
	startCtx, cancel := context.WithTimeout(ctx, a.startupTimeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- component.Start(startCtx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-startCtx.Done():
		return fmt.Errorf("startup timeout exceeded: %w", startCtx.Err())
	}
}

// Stop gracefully stops all components in reverse order.
func (a *Application) Stop(ctx context.Context) error {
	a.logger.Info("Stopping application components")

	stopCtx, cancel := context.WithTimeout(ctx, a.shutdownTimeout)
	defer cancel()

	return a.stopStartedComponents(stopCtx)
}

// stopStartedComponents stops all started components in reverse order.
func (a *Application) stopStartedComponents(ctx context.Context) error {
	// Stop in reverse order (LIFO)
	var lastErr error
	for i := len(a.startedComponents) - 1; i >= 0; i-- {
		component := a.startedComponents[i]

		a.logger.Info("Stopping component",
			zap.String("component", component.Name()),
		)

		if err := component.Stop(ctx); err != nil {
			a.logger.Error("Failed to stop component",
				zap.String("component", component.Name()),
				zap.Error(err),
			)
			lastErr = err
			// Continue stopping other components
		} else {
			a.logger.Info("Component stopped successfully",
				zap.String("component", component.Name()),
			)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("errors occurred during shutdown: %w", lastErr)
	}

	a.logger.Info("All components stopped successfully")
	return nil
}

// Run starts the application and waits for a shutdown signal.
// This is a convenience method that combines Start, signal handling, and Stop.
func (a *Application) Run(ctx context.Context) error {
	// Start all components
	if err := a.Start(ctx); err != nil {
		return err
	}

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()
	a.logger.Info("Shutdown signal received")

	// Create a new context for shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	// Stop all components
	return a.Stop(shutdownCtx)
}
