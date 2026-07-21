package provider

import (
	"context"

	"github.com/nexus-io/nexus/pkg/engine"
)

// Provider dictates the architectural contract required by any infrastructure engine driver
type Provider interface {
	// Reconcile brings the actual target state into alignment with the desired specification
	Reconcile(ctx context.Context, name string, spec engine.Spec) (engine.Status, error)
	// Destroy completely eliminates the active running assets tied to the contract resource
	Destroy(ctx context.Context, name string, spec engine.Spec) error
}