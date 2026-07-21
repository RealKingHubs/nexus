package provider

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/nexus-io/nexus/pkg/engine"
)

// DockerProvider implements the Provider interface using the official Moby/Docker Client SDK
type DockerProvider struct {
	cli *client.Client
}

// NewDockerProvider establishes an authenticated session with the local host Docker daemon
func NewDockerProvider() (*DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to wire host docker engine connection: %w", err)
	}
	return &DockerProvider{cli: cli}, nil
}

// Reconcile ensures the target container is downloaded, configured, and actively running
func (p *DockerProvider) Reconcile(ctx context.Context, name string, spec engine.Spec) (engine.Status, error) {
	targetImage := "nginx:alpine"

	// 1. Ensure image exists locally without being killed by short CLI context deadlines
	pullCtx := context.Background()
	reader, err := p.cli.ImagePull(pullCtx, targetImage, client.ImagePullOptions{})
	if err != nil {
		return engine.Status{}, fmt.Errorf("failed to pull image layer from registry: %w", err)
	}
	
	// Stream progress to completion so Docker finishes fetching all layers before proceeding
	_, _ = io.Copy(io.Discard, reader)
	_ = reader.Close()

	// 2. Interrogate the daemon to discover if the container already exists (Idempotency)
	containers, err := p.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err == nil {
		for _, c := range containers.Items {
			for _, containerName := range c.Names {
				if containerName == "/"+name || containerName == name {
					// Resource exists: Ensure it is running to fix environmental configuration drift
					if c.State != "running" {
						fmt.Printf("🔄 Container '%s' stopped down out of sync. Restarting engine...\n", name)
						_, err = p.cli.ContainerStart(ctx, c.ID, client.ContainerStartOptions{})
						if err != nil {
							return engine.Status{}, fmt.Errorf("failed to restart stalled container: %w", err)
						}
					}
					return p.compileRuntimeStatus("Running", c.ID, targetImage), nil
				}
			}
		}
	}

	// 3. Asset absent: Structural creation execution
	resp, err := p.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: name,
		Config: &container.Config{
			Image: targetImage,
		},
		HostConfig: &container.HostConfig{},
	})
	if err != nil {
		return engine.Status{}, fmt.Errorf("failed to initialize infrastructure configuration container: %w", err)
	}

	// 4. Trigger active workload execution loop
	_, err = p.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		return engine.Status{}, fmt.Errorf("failed to boot infrastructure runtime container: %w", err)
	}

	return p.compileRuntimeStatus("Deployed", resp.ID, targetImage), nil
}

// Destroy completely terminates the container instance and strips tracking resources
func (p *DockerProvider) Destroy(ctx context.Context, name string, spec engine.Spec) error {
	stopTimeout := 10

	// Stop the workload execution sequence gracefully
	_, err := p.cli.ContainerStop(ctx, name, client.ContainerStopOptions{Timeout: &stopTimeout})
	if err != nil && !isNotFoundErr(err) {
		return fmt.Errorf("failed to command worker container stop sequence: %w", err)
	}

	// Purge the tracking resource allocation mappings entirely
	_, err = p.cli.ContainerRemove(ctx, name, client.ContainerRemoveOptions{Force: true})
	if err != nil && !isNotFoundErr(err) {
		return fmt.Errorf("failed to strip runtime layer tracking mapping storage: %w", err)
	}

	return nil
}

// Helper function to safely ignore missing resource errors during destruction
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such") || strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

func (p *DockerProvider) compileRuntimeStatus(phase, containerID, img string) engine.Status {
	shorthandID := containerID
	if len(containerID) >= 12 {
		shorthandID = containerID[:12]
	}

	liveOutputs := make(map[string]string)
	liveOutputs["container_id"] = shorthandID
	liveOutputs["target_image"] = img
	liveOutputs["engine_host"]  = "localhost"

	return engine.Status{
		Phase:     phase,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Outputs:   liveOutputs,
	}
}