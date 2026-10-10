package container

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

const (
	// WorkspaceRoot is the enforced root directory for all container file operations
	WorkspaceRoot = "/workspace"
	// MaxFileSize is the maximum file size allowed for read/write operations (100MB)
	MaxFileSize = 100 * 1024 * 1024
)

// FileSizeError is returned when a file exceeds the maximum allowed size
type FileSizeError struct {
	Size    int64
	MaxSize int64
}

func (e *FileSizeError) Error() string {
	return fmt.Sprintf("file too large: %d bytes (max %d)", e.Size, e.MaxSize)
}

// ErrFileTooLarge is a sentinel error for file size violations
var ErrFileTooLarge = errors.New("file too large")

// FileInfo represents metadata about a file or directory in a container
type FileInfo struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mod_time"`
	IsDir   bool        `json:"is_dir"`
}

// ReadFileFromContainer reads a file from a container using CopyFromContainer
func (m *ContainerManager) ReadFileFromContainer(containerID, path string) ([]byte, error) {
	ctx := context.Background()

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Copy file from container (returns tar archive)
	reader, stat, err := m.cli.CopyFromContainer(ctx, containerID, safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy from container: %w", err)
	}
	defer reader.Close()

	// Check file size before reading
	if stat.Size > MaxFileSize {
		return nil, &FileSizeError{Size: stat.Size, MaxSize: MaxFileSize}
	}

	// Extract file content from tar archive
	content, err := extractFileFromTar(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract file from tar: %w", err)
	}

	return content, nil
}

// WriteFileToContainer writes a file to a container using CopyToContainer
func (m *ContainerManager) WriteFileToContainer(containerID, path string, content []byte, mode os.FileMode) error {
	ctx := context.Background()

	// Validate content size
	if len(content) > MaxFileSize {
		return fmt.Errorf("content too large: %d bytes (max %d)", len(content), MaxFileSize)
	}

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Extract directory and filename
	dir := filepath.Dir(safePath)
	filename := filepath.Base(safePath)

	// Create tar archive with the file
	tarBuffer, err := createTarArchive(filename, content, mode)
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	// Copy tar archive to container (will extract automatically)
	err = m.cli.CopyToContainer(ctx, containerID, dir, tarBuffer, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                false,
	})
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}

	return nil
}

// StatPathInContainer gets file/directory metadata using ContainerStatPath
func (m *ContainerManager) StatPathInContainer(containerID, path string) (*FileInfo, error) {
	ctx := context.Background()

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Get path stat from container
	stat, err := m.cli.ContainerStatPath(ctx, containerID, safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	return &FileInfo{
		Name:    stat.Name,
		Path:    safePath,
		Size:    stat.Size,
		Mode:    os.FileMode(stat.Mode),
		ModTime: stat.Mtime,
		IsDir:   stat.Mode&(1<<31) != 0, // Check if directory bit is set
	}, nil
}

// ListDirectoryInContainer lists directory contents using CopyFromContainer
func (m *ContainerManager) ListDirectoryInContainer(containerID, path string) ([]FileInfo, error) {
	ctx := context.Background()

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Verify it's a directory
	stat, err := m.StatPathInContainer(containerID, safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}
	if !stat.IsDir {
		return nil, fmt.Errorf("path is not a directory: %s", safePath)
	}

	// Copy directory from container (returns tar archive with all contents)
	reader, _, err := m.cli.CopyFromContainer(ctx, containerID, safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy directory from container: %w", err)
	}
	defer reader.Close()

	// Extract file list from tar archive
	files, err := listFilesFromTar(reader, safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list files from tar: %w", err)
	}

	return files, nil
}

// DeletePathInContainer deletes a file or directory using exec
func (m *ContainerManager) DeletePathInContainer(containerID, path string) error {
	ctx := context.Background()

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Prevent deletion of workspace root
	if safePath == WorkspaceRoot || safePath == WorkspaceRoot+"/" {
		return fmt.Errorf("cannot delete workspace root")
	}

	// Execute rm command in container
	cmd := []string{"rm", "-rf", safePath}
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := m.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// Start exec
	err = m.cli.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	// Wait for exec to complete and check exit code
	inspectResp, err := m.cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("delete command failed with exit code %d", inspectResp.ExitCode)
	}

	return nil
}

// CreateDirectoryInContainer creates a directory using exec
func (m *ContainerManager) CreateDirectoryInContainer(containerID, path string) error {
	ctx := context.Background()

	// Sanitize and validate path
	safePath, err := sanitizeContainerPath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Execute mkdir command in container
	cmd := []string{"mkdir", "-p", safePath}
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := m.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// Start exec
	err = m.cli.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	// Wait for exec to complete and check exit code
	inspectResp, err := m.cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("mkdir command failed with exit code %d", inspectResp.ExitCode)
	}

	return nil
}

// Helper Functions

// extractFileFromTar extracts the first file from a tar archive
func extractFileFromTar(reader io.ReadCloser) ([]byte, error) {
	tarReader := tar.NewReader(reader)

	// Read first entry (should be the file we want)
	header, err := tarReader.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to read tar header: %w", err)
	}

	// Verify it's a regular file
	if header.Typeflag != tar.TypeReg {
		return nil, fmt.Errorf("expected regular file, got type %d", header.Typeflag)
	}

	// Check file size
	if header.Size > MaxFileSize {
		return nil, &FileSizeError{Size: header.Size, MaxSize: MaxFileSize}
	}

	// Read file content
	content := make([]byte, header.Size)
	_, err = io.ReadFull(tarReader, content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

// listFilesFromTar extracts file list from a tar archive
func listFilesFromTar(reader io.ReadCloser, basePath string) ([]FileInfo, error) {
	tarReader := tar.NewReader(reader)
	var files []FileInfo

	// Docker's CopyFromContainer returns tar entries relative to the PARENT of basePath
	// e.g., copying /workspace returns entries like "workspace/file.txt"
	// So we need the parent path to construct full paths correctly
	parentPath := filepath.Dir(basePath)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip the base directory itself
		if header.Name == "." || header.Name == "./" {
			continue
		}

		// Build full path using parent directory
		// header.Name is relative to parent, e.g., "workspace/file.txt"
		fullPath := filepath.Join(parentPath, header.Name)

		// Only include direct children of basePath (not the directory itself or nested items)
		relPath, err := filepath.Rel(basePath, fullPath)
		if err != nil || relPath == "." {
			continue
		}
		// Skip nested items (items with "/" in relative path)
		if strings.Contains(relPath, string(filepath.Separator)) {
			continue
		}

		files = append(files, FileInfo{
			Name:    filepath.Base(header.Name),
			Path:    fullPath,
			Size:    header.Size,
			Mode:    os.FileMode(header.Mode),
			ModTime: header.ModTime,
			IsDir:   header.Typeflag == tar.TypeDir,
		})
	}

	return files, nil
}

// createTarArchive creates a tar archive with a single file
func createTarArchive(filename string, content []byte, mode os.FileMode) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)
	defer tarWriter.Close()

	// Create tar header
	header := &tar.Header{
		Name:    filename,
		Size:    int64(len(content)),
		Mode:    int64(mode),
		ModTime: time.Now(),
	}

	// Write header
	err := tarWriter.WriteHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	// Write content
	_, err = tarWriter.Write(content)
	if err != nil {
		return nil, fmt.Errorf("failed to write tar content: %w", err)
	}

	// Ensure all data is flushed
	err = tarWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	return buffer, nil
}

// sanitizeContainerPath validates and sanitizes a container path
// CRITICAL SECURITY FUNCTION - prevents path traversal attacks
func sanitizeContainerPath(userPath string) (string, error) {
	// Remove any leading/trailing whitespace
	userPath = strings.TrimSpace(userPath)

	// Empty path check
	if userPath == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Ensure path is absolute (starts with /)
	if !filepath.IsAbs(userPath) {
		userPath = filepath.Join(WorkspaceRoot, userPath)
	}

	// Clean the path (removes .., ., and duplicate slashes)
	cleanPath := filepath.Clean(userPath)

	// CRITICAL: Verify the cleaned path is still under WorkspaceRoot
	// This prevents path traversal attacks like "../../etc/passwd"
	if !strings.HasPrefix(cleanPath, WorkspaceRoot) {
		return "", fmt.Errorf("path must be under %s (attempted: %s)", WorkspaceRoot, cleanPath)
	}

	// Additional security checks
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains invalid characters: ..")
	}

	// Block access to sensitive directories even under /workspace
	blockedPaths := []string{
		"/workspace/../",
		"/workspace/..",
	}
	for _, blocked := range blockedPaths {
		if strings.HasPrefix(cleanPath, blocked) || cleanPath == strings.TrimSuffix(blocked, "/") {
			return "", fmt.Errorf("access to path denied: %s", cleanPath)
		}
	}

	return cleanPath, nil
}
