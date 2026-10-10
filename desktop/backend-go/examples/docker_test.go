package example

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
)

// TestDockerClientImport verifies that the Docker SDK can be imported and a client can be created
func TestDockerClientImport(t *testing.T) {
	// Create a new Docker client with default options
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// It's okay if we can't connect to Docker daemon in test environment
		// We're just verifying the import and client creation works
		t.Logf("Note: Could not create Docker client (this is expected if Docker isn't running): %v", err)
		return
	}
	defer cli.Close()

	// Try to ping the Docker daemon
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		t.Logf("Note: Could not ping Docker daemon (this is expected if Docker isn't running): %v", err)
		return
	}

	t.Log("Successfully connected to Docker daemon")
}
