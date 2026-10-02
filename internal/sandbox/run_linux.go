//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// start launches the target inside unprivileged user, mount and PID namespaces,
// plus a network namespace when networking is disabled. It uses the Go runtime's
// native clone support instead of shelling out to util-linux. Hosts that forbid
// unprivileged user namespaces fail closed: no privileged fallback is attempted.
func start(ctx context.Context, target string, args []string, p Policy, dir string) (*Session, error) {
	cmd := exec.CommandContext(ctx, target, args...)
	cmd.Dir = dir
	flags := uintptr(syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID)
	if !p.Network {
		flags |= syscall.CLONE_NEWNET
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: flags,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("namespace launch failed (unprivileged user namespaces may be disabled): %w", err)
	}
	s := &Session{Dir: dir, Cmd: cmd}
	s.Notes = append(s.Notes, "user/mount/PID"+netNote(p.Network)+" namespaces active")
	if p.MemoryMB > 0 {
		if err := applyMemoryLimit(cmd.Process.Pid, p.MemoryMB); err != nil {
			s.Notes = append(s.Notes, "memory limit not applied: "+err.Error())
		} else {
			s.Notes = append(s.Notes, fmt.Sprintf("memory limit %d MiB active", p.MemoryMB))
		}
	}
	return s, nil
}

func netNote(network bool) string {
	if network {
		return ""
	}
	return "/network"
}

// applyMemoryLimit is best effort. It only works when the current user has a
// delegated cgroup v2 subtree; otherwise the caller is told the limit failed.
func applyMemoryLimit(pid, mb int) error {
	base := "/sys/fs/cgroup"
	if _, err := os.Stat(filepath.Join(base, "cgroup.controllers")); err != nil {
		return fmt.Errorf("cgroup v2 not available")
	}
	dir := filepath.Join(base, "portable-sandboxer", "session")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "memory.max"), []byte(strconv.Itoa(mb*1024*1024)), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644)
}
