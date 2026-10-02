package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIsSafe(t *testing.T) {
	d := Default()
	if d.Network {
		t.Fatal("default must not enable network")
	}
	if d.Snapshot.Enabled {
		t.Fatal("snapshots must default to off")
	}
	if d.Resources.MemoryMB <= 0 {
		t.Fatal("default memory limit should be positive")
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	body := "version: 1\npreset: offline-test\nnetwork: true\nresources:\n  memory_mb: 256\nsnapshot:\n  enabled: true\n  dir: /tmp/snaps\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Network || f.Resources.MemoryMB != 256 || !f.Snapshot.Enabled || f.Snapshot.Dir != "/tmp/snaps" {
		t.Fatalf("unexpected config: %+v", f)
	}
	p := f.Policy()
	if !p.Network || p.MemoryMB != 256 || !p.Snapshot || p.SnapshotDir != "/tmp/snaps" {
		t.Fatalf("unexpected policy: %+v", p)
	}
}

func TestLoadMissingFileKeepsDefaults(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if f.Network {
		t.Fatal("missing file must not enable network")
	}
}

func TestFindPresetFallback(t *testing.T) {
	if FindPreset("does-not-exist").ID != Presets()[0].ID {
		t.Fatal("unknown preset should fall back to the first preset")
	}
}
