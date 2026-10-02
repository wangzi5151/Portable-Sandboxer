package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const workspacePrefix = "portable-sandbox-"

// Policy describes the isolation requested for a session. Options that a platform
// cannot enforce are reported through Session.Notes rather than silently applied.
type Policy struct {
	Network     bool
	MemoryMB    int
	Snapshot    bool
	SnapshotDir string
}

// Session is a running isolated foreground process plus its temporary workspace.
type Session struct {
	Dir         string
	Cmd         *exec.Cmd
	SnapshotDir string
	Started     time.Time
	Notes       []string
	release     func()
}

// Start validates the target, creates a fresh temporary workspace, and launches
// the platform-specific isolation backend.
func Start(ctx context.Context, target string, args []string, p Policy) (*Session, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, errors.New("target must be a file, not a directory")
	}
	dir, err := os.MkdirTemp("", workspacePrefix)
	if err != nil {
		return nil, err
	}
	s, err := start(ctx, abs, args, p, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	s.Started = time.Now()
	if p.Snapshot {
		s.SnapshotDir = p.SnapshotDir
		if s.SnapshotDir == "" {
			s.SnapshotDir = DefaultSnapshotDir()
		}
	}
	return s, nil
}

// Wait blocks until the foreground process exits, then snapshots (if requested)
// and removes the temporary workspace. Closing the release hook also tears down
// the platform isolation object (for example a Windows Job Object).
func (s *Session) Wait() error {
	err := s.Cmd.Wait()
	if s.release != nil {
		s.release()
		s.release = nil
	}
	if s.SnapshotDir != "" {
		base := filepath.Base(s.Dir)
		dst := filepath.Join(s.SnapshotDir, base)
		if copyErr := copyTree(s.Dir, dst); copyErr == nil {
			s.SnapshotDir = dst
		} else {
			s.Notes = append(s.Notes, "snapshot failed: "+copyErr.Error())
		}
	}
	_ = os.RemoveAll(s.Dir)
	return err
}

// DefaultSnapshotDir returns the per-user location for opt-in persistent snapshots.
func DefaultSnapshotDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Portable-Sandboxer", "snapshots")
}

// CleanupStale removes temporary workspaces left behind by a crash or forced exit.
// It never touches directories outside the OS temp directory with our prefix.
func CleanupStale() (int, error) {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), workspacePrefix) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(os.TempDir(), e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}
