//go:build windows

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// start launches the target and assigns it to a Windows Job Object. The job
// enforces process-count and optional memory limits and kills the whole process
// tree when the job handle closes. This prototype does NOT implement restricted
// tokens, filesystem/registry virtualization, or network isolation; those are
// reported as unavailable and the UI hides controls it cannot honor.
func start(ctx context.Context, target string, args []string, p Policy, dir string) (*Session, error) {
	cmd := exec.CommandContext(ctx, target, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("launch failed: %w", err)
	}
	s := &Session{Dir: dir, Cmd: cmd}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		s.Notes = append(s.Notes, "Job Object unavailable; process tree is not supervised")
		return s, nil
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	info.BasicLimitInformation.ActiveProcessLimit = 64
	if p.MemoryMB > 0 {
		info.ProcessMemoryLimit = uintptr(p.MemoryMB) * 1024 * 1024
		info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		s.Notes = append(s.Notes, "Job Object limits rejected by the OS")
		return s, nil
	}
	// Assign after start; there is a short pre-assignment window where a child
	// could be spawned outside the job. Documented as a prototype limitation.
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err == nil {
		_ = windows.AssignProcessToJobObject(job, h)
		windows.CloseHandle(h)
		s.Notes = append(s.Notes, "Job Object active: process count limit, kill-on-close")
	} else {
		s.Notes = append(s.Notes, "Job Object created but process assignment failed")
	}
	s.release = func() { windows.CloseHandle(job) }
	return s, nil
}
