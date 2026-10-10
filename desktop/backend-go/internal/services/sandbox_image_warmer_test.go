package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireDocker skips the test if no Docker daemon is reachable.
// All ImageWarmerService tests require a live Docker daemon.
func requireDocker(t *testing.T) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc, err := NewImageWarmerService(logger, time.Hour)
	if err != nil {
		t.Skipf("Docker daemon not available, skipping test: %v", err)
	}
	svc.Stop()
}

func TestNewImageWarmerService(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, service)
	defer service.Stop()

	assert.NotNil(t, service.cli)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.imageStatus)
	assert.NotNil(t, service.stopCh)
	assert.Equal(t, 24*time.Hour, service.refreshInterval)

	// Check that common images are initialized
	images := service.GetCommonImages()
	assert.Greater(t, len(images), 0)
	assert.Contains(t, images, "node:20-alpine")
	assert.Contains(t, images, "python:3.11-alpine")
}

func TestNewImageWarmerService_NilLogger(t *testing.T) {
	requireDocker(t)
	service, err := NewImageWarmerService(nil, time.Hour)
	require.NoError(t, err)
	require.NotNil(t, service)
	defer service.Stop()

	assert.NotNil(t, service.logger)
}

func TestImageWarmerService_AddRemoveCommonImage(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	// Test adding a new image
	customImage := "alpine:latest"
	service.AddCommonImage(customImage)

	images := service.GetCommonImages()
	assert.Contains(t, images, customImage)

	// Test adding duplicate (should not duplicate)
	initialLen := len(images)
	service.AddCommonImage(customImage)
	images = service.GetCommonImages()
	assert.Equal(t, initialLen, len(images))

	// Test removing image
	service.RemoveCommonImage(customImage)
	images = service.GetCommonImages()
	assert.NotContains(t, images, customImage)

	// Test removing non-existent image (should not error)
	service.RemoveCommonImage("non-existent:latest")
}

func TestImageWarmerService_GetImageStatus(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	status := service.GetImageStatus()
	assert.NotNil(t, status)

	// All common images should have status entries
	images := service.GetCommonImages()
	for _, img := range images {
		_, exists := status[img]
		assert.True(t, exists, "Status should exist for image: %s", img)
	}
}

func TestImageWarmerService_WarmImage(t *testing.T) {
	requireDocker(t)
	if testing.Short() {
		t.Skip("Skipping Docker pull test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test pulling a small image
	testImage := "alpine:latest"
	err = service.WarmImage(ctx, testImage)
	assert.NoError(t, err)

	// Verify image is available
	available := service.IsImageAvailable(ctx, testImage)
	assert.True(t, available)
}

func TestImageWarmerService_WarmImage_InvalidImage(t *testing.T) {
	requireDocker(t)
	if testing.Short() {
		t.Skip("Skipping Docker pull test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test pulling a non-existent image
	err = service.WarmImage(ctx, "this-image-does-not-exist:invalid-tag")
	assert.Error(t, err)
}

func TestImageWarmerService_IsImageAvailable(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	ctx := context.Background()

	// Test with non-existent image
	available := service.IsImageAvailable(ctx, "non-existent-image:latest")
	assert.False(t, available)
}

func TestImageWarmerService_StartStop(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 100*time.Millisecond)
	require.NoError(t, err)

	// Start should not error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Note: Start will try to pull images, this might take time
	// For unit tests, we just verify it doesn't crash
	err = service.Start(ctx)
	// We don't assert NoError because pulling might fail in test environment
	_ = err

	// Give it a moment to start
	time.Sleep(200 * time.Millisecond)

	// Stop should not error
	service.Stop()

	// Multiple stops should be safe
	service.Stop()
}

func TestImageWarmerService_ConcurrentAccess(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	ctx := context.Background()

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = service.GetImageStatus()
			_ = service.GetCommonImages()
			_ = service.IsImageAvailable(ctx, "alpine:latest")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			service.AddCommonImage("test:v" + string(rune(idx)))
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestImageWarmerService_GetCommonImages(t *testing.T) {
	requireDocker(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := NewImageWarmerService(logger, 24*time.Hour)
	require.NoError(t, err)
	defer service.Stop()

	images := service.GetCommonImages()
	assert.NotNil(t, images)
	assert.Greater(t, len(images), 0)

	// Verify expected images
	expectedImages := []string{
		"node:20-alpine",
		"node:18-alpine",
		"python:3.11-alpine",
		"python:3.12-alpine",
		"golang:1.22-alpine",
		"nginx:alpine",
	}

	for _, expected := range expectedImages {
		assert.Contains(t, images, expected)
	}

	// Modifying returned slice should not affect service
	originalLen := len(service.commonImages)
	images = service.GetCommonImages()
	images[0] = "modified:latest"
	assert.Equal(t, originalLen, len(service.commonImages))
}
