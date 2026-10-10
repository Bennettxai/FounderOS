package collect

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// NeedsFullDiskAccess is what a skipped protected source reports instead of
// opening its files (which would raise a macOS privacy prompt every tick).
const NeedsFullDiskAccess = "needs Full Disk Access: System Settings → Privacy & Security → Full Disk Access → turn on founderos-collector"

// HasFullDiskAccess says whether this process may read macOS-protected
// files, by opening the user's TCC database read-only: macOS refuses it
// (EPERM) without Full Disk Access and never prompts for it. No database
// (tests, non-macOS) means nothing is protected.
func HasFullDiskAccess(home string) bool {
	f, err := os.Open(filepath.Join(home, "Library", "Application Support", "com.apple.TCC", "TCC.db"))
	if err == nil {
		f.Close()
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return !(errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || errors.Is(err, fs.ErrPermission))
}

// protectedPath reports whether a path sits in a folder macOS guards with a
// per-app privacy prompt (Documents, Desktop, Downloads, iCloud and other
// apps' group containers).
func protectedPath(path, home string) bool {
	for _, dir := range []string{"Documents", "Desktop", "Downloads", filepath.Join("Library", "Mobile Documents"), filepath.Join("Library", "CloudStorage"), filepath.Join("Library", "Group Containers"), filepath.Join("Library", "Containers")} {
		root := filepath.Join(home, dir)
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func needsAccess(id string) connectors.Status {
	m := devicepush.Metas[id]
	return connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind, State: connectors.StateError, Detail: NeedsFullDiskAccess}
}

// SkippedForAccess lists the sources a payload skipped for want of Full Disk
// Access, sorted, for the collector's log line.
func SkippedForAccess(p devicepush.Payload) []string {
	var out []string
	for _, s := range p.Statuses {
		if s.Detail == NeedsFullDiskAccess {
			out = append(out, s.ID)
		}
	}
	sort.Strings(out)
	return out
}
