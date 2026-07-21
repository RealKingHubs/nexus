package daemon

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nexus-io/nexus/pkg/engine"
	"github.com/nexus-io/nexus/pkg/provider"
)

type ManagedResource struct {
	Name     string
	Provider string
	Spec     engine.Spec
}

type Daemon struct {
	dockerProvider *provider.DockerProvider
	interval       time.Duration
	resources      []ManagedResource
}

func NewDaemon(interval time.Duration) (*Daemon, error) {
	dockerProv, err := provider.NewDockerProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to wire docker provider driver: %w", err)
	}

	defaultResources := []ManagedResource{
		{
			Name:     "nexus-local-web",
			Provider: "docker",
			Spec:     engine.Spec{},
		},
	}

	return &Daemon{
		dockerProvider: dockerProv,
		interval:       interval,
		resources:      defaultResources,
	}, nil
}

func (d *Daemon) Start(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	log.Printf("[INFO] Nexus Continuous Reconciliation Daemon Engine Initialized (Interval: %s)\n", d.interval)

	d.reconcileAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Gracefully stopping Nexus Daemon reconciliation loop...")
			return nil
		case <-ticker.C:
			d.reconcileAll(ctx)
		}
	}
}

func (d *Daemon) reconcileAll(ctx context.Context) {
	for _, res := range d.resources {
		switch res.Provider {
		case "docker":
			status, err := d.dockerProvider.Reconcile(ctx, res.Name, res.Spec)
			if err != nil {
				log.Printf("[ERROR] [%s] Convergence failure: %v\n", res.Name, err)
				continue
			}
			log.Printf("[SYNC] [%s] Status: %s | ID: %s\n", res.Name, status.Phase, status.Outputs["container_id"])
		}
	}
}