package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	BackupDirName     = ".backups"
	MaxBackupsPerFile = 10
)

// BackupManager handles file backups before edits
type BackupManager struct {
	workspaceRoot string
}

// NewBackupManager creates a new backup manager
func NewBackupManager(workspaceRoot string) *BackupManager {
	return &BackupManager{
		workspaceRoot: workspaceRoot,
	}
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	OriginalPath string    `json:"original_path"`
	BackupPath   string    `json:"backup_path"`
	Timestamp    time.Time `json:"timestamp"`
	Size         int64     `json:"size"`
}

// CreateBackup creates a backup of a file before editing
// Returns the backup path or error
func (m *BackupManager) CreateBackup(filePath string) (*BackupInfo, error) {
	// Validate path is under workspace
	absPath, err := m.resolveAndValidatePath(filePath)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("cannot backup directory: %s", filePath)
	}

	// Create backup directory structure
	timestamp := time.Now()
	timestampStr := timestamp.Format("2006-01-02_15-04-05")

	// Get relative path from workspace
	relPath, err := filepath.Rel(m.workspaceRoot, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}

	// Backup path: .backups/<timestamp>/<relative_path>
	backupDir := filepath.Join(m.workspaceRoot, BackupDirName, timestampStr, filepath.Dir(relPath))
	backupPath := filepath.Join(backupDir, filepath.Base(relPath))

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy file to backup location
	if err := copyFile(absPath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to copy file to backup: %w", err)
	}

	// Cleanup old backups for this file
	m.cleanupOldBackups(relPath)

	return &BackupInfo{
		OriginalPath: filePath,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         info.Size(),
	}, nil
}

// RestoreBackup restores a file from a backup
func (m *BackupManager) RestoreBackup(backupPath string) error {
	// Validate backup path is under .backups
	absBackupPath, err := m.resolveAndValidatePath(backupPath)
	if err != nil {
		return err
	}

	if !strings.Contains(absBackupPath, BackupDirName) {
		return fmt.Errorf("invalid backup path: must be under %s", BackupDirName)
	}

	// Check backup exists
	if _, err := os.Stat(absBackupPath); err != nil {
		return fmt.Errorf("backup not found: %s", backupPath)
	}

	// Extract original path from backup path
	// Format: .backups/<timestamp>/<relative_path>
	parts := strings.Split(absBackupPath, BackupDirName+string(filepath.Separator))
	if len(parts) != 2 {
		return fmt.Errorf("invalid backup path format")
	}

	// Remove timestamp prefix to get relative path
	afterBackups := parts[1]
	pathParts := strings.SplitN(afterBackups, string(filepath.Separator), 2)
	if len(pathParts) != 2 {
		return fmt.Errorf("invalid backup path format: missing timestamp")
	}

	relPath := pathParts[1]
	originalPath := filepath.Join(m.workspaceRoot, relPath)

	// Create backup of current file before restoring (safety)
	if _, err := os.Stat(originalPath); err == nil {
		_, _ = m.CreateBackup(relPath) // Ignore error, best effort
	}

	// Restore file
	if err := copyFile(absBackupPath, originalPath); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// ListBackups returns all backups for a specific file
func (m *BackupManager) ListBackups(filePath string) ([]BackupInfo, error) {
	// Get relative path
	absPath, err := m.resolveAndValidatePath(filePath)
	if err != nil {
		return nil, err
	}

	relPath, err := filepath.Rel(m.workspaceRoot, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}

	backupsDir := filepath.Join(m.workspaceRoot, BackupDirName)
	var backups []BackupInfo

	// Walk through backup directories
	err = filepath.Walk(backupsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Check if this backup is for our file
		if strings.HasSuffix(path, relPath) || strings.HasSuffix(path, filepath.Base(relPath)) {
			// Extract timestamp from path
			relToBackups, _ := filepath.Rel(backupsDir, path)
			parts := strings.SplitN(relToBackups, string(filepath.Separator), 2)
			if len(parts) >= 1 {
				timestamp, _ := time.Parse("2006-01-02_15-04-05", parts[0])
				backups = append(backups, BackupInfo{
					OriginalPath: filePath,
					BackupPath:   path,
					Timestamp:    timestamp,
					Size:         info.Size(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// ListAllBackups returns all backups in the workspace
func (m *BackupManager) ListAllBackups() ([]BackupInfo, error) {
	backupsDir := filepath.Join(m.workspaceRoot, BackupDirName)
	var backups []BackupInfo

	// Check if backups directory exists
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		return backups, nil
	}

	err := filepath.Walk(backupsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relToBackups, _ := filepath.Rel(backupsDir, path)
		parts := strings.SplitN(relToBackups, string(filepath.Separator), 2)
		if len(parts) >= 2 {
			timestamp, _ := time.Parse("2006-01-02_15-04-05", parts[0])
			backups = append(backups, BackupInfo{
				OriginalPath: parts[1],
				BackupPath:   path,
				Timestamp:    timestamp,
				Size:         info.Size(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// cleanupOldBackups removes old backups for a file, keeping only MaxBackupsPerFile
func (m *BackupManager) cleanupOldBackups(relPath string) {
	backups, err := m.ListBackups(relPath)
	if err != nil {
		return
	}

	// Remove backups beyond the limit
	if len(backups) > MaxBackupsPerFile {
		for _, backup := range backups[MaxBackupsPerFile:] {
			os.Remove(backup.BackupPath)
			// Try to remove empty parent directories
			dir := filepath.Dir(backup.BackupPath)
			for dir != filepath.Join(m.workspaceRoot, BackupDirName) {
				if err := os.Remove(dir); err != nil {
					break // Directory not empty or other error
				}
				dir = filepath.Dir(dir)
			}
		}
	}
}

// CleanupAllOldBackups removes all backups older than the specified duration
func (m *BackupManager) CleanupAllOldBackups(maxAge time.Duration) (int, error) {
	backupsDir := filepath.Join(m.workspaceRoot, BackupDirName)
	cutoff := time.Now().Add(-maxAge)
	removed := 0

	err := filepath.Walk(backupsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Check if directory is a timestamp directory
			if path != backupsDir {
				dirName := filepath.Base(path)
				timestamp, parseErr := time.Parse("2006-01-02_15-04-05", dirName)
				if parseErr == nil && timestamp.Before(cutoff) {
					// Remove entire timestamp directory
					os.RemoveAll(path)
					removed++
					return filepath.SkipDir
				}
			}
			return nil
		}

		return nil
	})

	return removed, err
}

// resolveAndValidatePath resolves a path and validates it's under workspace
func (m *BackupManager) resolveAndValidatePath(path string) (string, error) {
	// Handle relative paths
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(m.workspaceRoot, path))
	}

	// Validate path is under workspace
	if !strings.HasPrefix(absPath, m.workspaceRoot) {
		return "", fmt.Errorf("path must be under workspace: %s", path)
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	return absPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Preserve file permissions
	sourceInfo, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, sourceInfo.Mode())
	}

	return nil
}
