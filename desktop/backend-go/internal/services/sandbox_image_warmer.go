package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ImageStatus represents the status of a Docker image
type ImageStatus struct {
	Image     string    `json:"image"`
	Available bool      `json:"available"`
	Size      int64     `json:"size"`
	PulledAt  time.Time `json:"pulled_at"`
	Error     string    `json:"error,omitempty"`
}

// ImageWarmerService pre-pulls commonly used Docker images for faster sandbox deployment
type ImageWarmerService struct {
	cli               *client.Client
	logger            *slog.Logger
	mu                sync.RWMutex
	imageStatus       map[string]*ImageStatus
	stopCh            chan struct{}
	refreshInterval   time.Duration
	commonImages      []string
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	startOnce         sync.Once
	stopOnce          sync.Once
}

// NewImageWarmerService creates a new image warmer service
func NewImageWarmerService(logger *slog.Logger, refreshInterval time.Duration) (*ImageWarmerService, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create Docker client with version negotiation
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify Docker daemon is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker daemon not available: %w", err)
	}

	ctx, cancel = context.WithCancel(context.Background())

	service := &ImageWarmerService{
		cli:             cli,
		logger:          logger,
		imageStatus:     make(map[string]*ImageStatus),
		stopCh:          make(chan struct{}),
		refreshInterval: refreshInterval,
		ctx:             ctx,
		cancel:          cancel,
		commonImages: []string{
			"node:20-alpine",
			"node:18-alpine",
			"python:3.11-alpine",
			"python:3.12-alpine",
			"golang:1.22-alpine",
			"nginx:alpine",
		},
	}

	// Initialize image status map
	for _, img := range service.commonImages {
		service.imageStatus[img] = &ImageStatus{
			Image:     img,
			Available: false,
		}
	}

	logger.Info("image warmer service created",
		"image_count", len(service.commonImages),
		"refresh_interval", refreshInterval)

	return service, nil
}

// Start begins the image warming process
func (s *ImageWarmerService) Start(ctx context.Context) error {
	var startErr error
	s.startOnce.Do(func() {
		s.logger.Info("starting image warmer service")

		// Initial warm-up
		if err := s.WarmAllImages(ctx); err != nil {
			s.logger.Error("initial image warming failed", "error", err)
			startErr = err
			return
		}

		// Start background refresh goroutine
		s.wg.Add(1)
		go s.refreshLoop()

		s.logger.Info("image warmer service started successfully")
	})

	return startErr
}

// Stop stops the image warming service
func (s *ImageWarmerService) Stop() {
	s.stopOnce.Do(func() {
		s.logger.Info("stopping image warmer service")

		// Signal stop
		close(s.stopCh)
		s.cancel()

		// Wait for goroutines to finish
		s.wg.Wait()

		// Close Docker client
		if s.cli != nil {
			if err := s.cli.Close(); err != nil {
				s.logger.Error("failed to close Docker client", "error", err)
			}
		}

		s.logger.Info("image warmer service stopped")
	})
}

// refreshLoop periodically refreshes images
func (s *ImageWarmerService) refreshLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("refresh loop stopping")
			return
		case <-s.ctx.Done():
			s.logger.Info("refresh loop context cancelled")
			return
		case <-ticker.C:
			s.logger.Info("starting periodic image refresh")
			if err := s.WarmAllImages(s.ctx); err != nil {
				s.logger.Error("periodic image refresh failed", "error", err)
			}
		}
	}
}

// WarmImage pulls a specific Docker image
func (s *ImageWarmerService) WarmImage(ctx context.Context, imageName string) error {
	s.logger.Info("warming image", "image", imageName)

	status := &ImageStatus{
		Image:     imageName,
		Available: false,
	}

	// Pull the image
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	reader, err := s.cli.ImagePull(pullCtx, imageName, image.PullOptions{})
	if err != nil {
		status.Error = err.Error()
		s.updateStatus(imageName, status)
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Consume the pull output
	if _, err := io.Copy(io.Discard, reader); err != nil {
		status.Error = err.Error()
		s.updateStatus(imageName, status)
		return fmt.Errorf("failed to read pull output for %s: %w", imageName, err)
	}

	// Inspect the image to get size
	inspectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	inspect, _, err := s.cli.ImageInspectWithRaw(inspectCtx, imageName)
	if err != nil {
		s.logger.Warn("failed to inspect image", "image", imageName, "error", err)
		// Don't fail the warm operation if inspect fails
	} else {
		status.Size = inspect.Size
	}

	status.Available = true
	status.PulledAt = time.Now()
	status.Error = ""

	s.updateStatus(imageName, status)
	s.logger.Info("image warmed successfully",
		"image", imageName,
		"size_mb", status.Size/(1024*1024))

	return nil
}

// WarmAllImages pulls all configured images
func (s *ImageWarmerService) WarmAllImages(ctx context.Context) error {
	s.logger.Info("warming all images", "count", len(s.commonImages))

	var wg sync.WaitGroup
	errCh := make(chan error, len(s.commonImages))

	// Pull images concurrently
	for _, img := range s.commonImages {
		wg.Add(1)
		go func(imageName string) {
			defer wg.Done()
			if err := s.WarmImage(ctx, imageName); err != nil {
				errCh <- err
			}
		}(img)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		s.logger.Warn("some images failed to warm",
			"failed_count", len(errors),
			"total_count", len(s.commonImages))
		return fmt.Errorf("failed to warm %d/%d images", len(errors), len(s.commonImages))
	}

	s.logger.Info("all images warmed successfully", "count", len(s.commonImages))
	return nil
}

// IsImageAvailable checks if an image is cached locally
func (s *ImageWarmerService) IsImageAvailable(ctx context.Context, imageName string) bool {
	s.mu.RLock()
	status, exists := s.imageStatus[imageName]
	s.mu.RUnlock()

	if exists && status.Available {
		return true
	}

	// Double-check with Docker daemon
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _, err := s.cli.ImageInspectWithRaw(inspectCtx, imageName)
	available := err == nil

	// Update status if it's one of our common images
	if exists {
		s.mu.Lock()
		status.Available = available
		if available && status.PulledAt.IsZero() {
			status.PulledAt = time.Now()
		}
		s.mu.Unlock()
	}

	return available
}

// GetImageStatus returns the status of all tracked images
func (s *ImageWarmerService) GetImageStatus() map[string]ImageStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]ImageStatus, len(s.imageStatus))
	for name, status := range s.imageStatus {
		result[name] = *status
	}

	return result
}

// updateStatus updates the status of an image
func (s *ImageWarmerService) updateStatus(imageName string, status *ImageStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imageStatus[imageName] = status
}

// GetCommonImages returns the list of commonly used images
func (s *ImageWarmerService) GetCommonImages() []string {
	return append([]string{}, s.commonImages...)
}

// AddCommonImage adds a new image to the common images list
func (s *ImageWarmerService) AddCommonImage(imageName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, img := range s.commonImages {
		if img == imageName {
			return
		}
	}

	s.commonImages = append(s.commonImages, imageName)
	s.imageStatus[imageName] = &ImageStatus{
		Image:     imageName,
		Available: false,
	}

	s.logger.Info("added new common image", "image", imageName)
}

// RemoveCommonImage removes an image from the common images list
func (s *ImageWarmerService) RemoveCommonImage(imageName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from commonImages
	for i, img := range s.commonImages {
		if img == imageName {
			s.commonImages = append(s.commonImages[:i], s.commonImages[i+1:]...)
			break
		}
	}

	// Remove from imageStatus
	delete(s.imageStatus, imageName)

	s.logger.Info("removed common image", "image", imageName)
}
