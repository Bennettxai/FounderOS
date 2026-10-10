package example

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Example function demonstrating Docker SDK usage
// This is not called anywhere - it's just for reference
func dockerClientExample() error {
	// Create a new Docker client
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Example 1: Ping Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping docker daemon: %w", err)
	}
	log.Println("Successfully connected to Docker daemon")

	// Example 2: List containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	log.Printf("Found %d containers\n", len(containers))

	// Example 3: List images
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	log.Printf("Found %d images\n", len(images))

	return nil
}
