//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// start runs the target under a generated Seatbelt profile via sandbox-exec.
// Apple considers sandbox-exec a private, deprecated interface, so this backend
// is explicitly experimental and varies across macOS releases.
func start(ctx context.Context, target string, args []string, p Policy, dir string) (*Session, error) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return nil, fmt.Errorf("sandbox-exec is not available on this macOS installation")
	}
	profile := buildProfile(target, dir, p.Network)
	profilePath := filepath.Join(dir, "profile.sb")
	if err := os.WriteFile(profilePath, []byte(profile), 0o600); err != nil {
		return nil, err
	}
	argv := append([]string{"-f", profilePath, target}, args...)
	cmd := exec.CommandContext(ctx, "sandbox-exec", argv...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := &Session{Dir: dir, Cmd: cmd}
	s.Notes = append(s.Notes, "experimental Seatbelt profile active")
	return s, nil
}

// buildProfile grants read access to system locations needed to launch and write
// access only to the temporary workspace. All other reads and writes are denied.
func buildProfile(target, dir string, network bool) string {
	quoted := func(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n(allow process*)\n(allow sysctl-read)\n")
	fmt.Fprintf(&b, "(allow file-read* (subpath \"/System\") (subpath \"/usr\") (subpath \"/bin\") (subpath \"/sbin\") (subpath \"/Library\") (subpath \"/Applications\") (subpath \"%s\") (literal \"%s\"))\n", quoted(dir), quoted(target))
	fmt.Fprintf(&b, "(allow file-write* (subpath \"%s\"))\n", quoted(dir))
	if network {
		b.WriteString("(allow network*)\n")
	}
	return b.String()
}
