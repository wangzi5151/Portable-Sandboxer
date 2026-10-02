//go:build !linux && !darwin && !windows

package sandbox

import (
	"context"
	"fmt"
	"runtime"
)

// start has no isolation backend on this platform. It fails closed instead of
// pretending to isolate the process.
func start(ctx context.Context, target string, args []string, p Policy, dir string) (*Session, error) {
	return nil, fmt.Errorf("Portable-Sandboxer has no isolation backend for %s/%s", runtime.GOOS, runtime.GOARCH)
}
