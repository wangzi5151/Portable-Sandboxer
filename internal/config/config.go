package config

import (
	"os"

	"github.com/portable-sandboxer/portable-sandboxer/internal/sandbox"
	"gopkg.in/yaml.v3"
)

// File mirrors portable-sandboxer.yaml. Unknown fields are ignored so older
// builds keep working, but every field that is loaded maps to a real behavior.
type File struct {
	Version   int       `yaml:"version"`
	Preset    string    `yaml:"preset"`
	Network   bool      `yaml:"network"`
	Resources Resources `yaml:"resources"`
	Snapshot  Snapshot  `yaml:"snapshot"`
}

type Resources struct {
	MemoryMB int `yaml:"memory_mb"`
}

type Snapshot struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
}

// Default returns the safe starting configuration: no network, modest memory
// limit, snapshots off.
func Default() File {
	return File{
		Version:   1,
		Preset:    "try-unknown-portable-app",
		Network:   false,
		Resources: Resources{MemoryMB: 1024},
		Snapshot:  Snapshot{Enabled: false},
	}
}

// Load reads a YAML file, starting from defaults so omitted keys stay safe.
func Load(path string) (File, error) {
	f := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	if err := yaml.Unmarshal(b, &f); err != nil {
		return f, err
	}
	return f, nil
}

// Policy converts the loaded configuration into a sandbox policy.
func (f File) Policy() sandbox.Policy {
	snapDir := f.Snapshot.Dir
	if f.Snapshot.Enabled && snapDir == "" {
		snapDir = sandbox.DefaultSnapshotDir()
	}
	return sandbox.Policy{
		Network:     f.Network,
		MemoryMB:    f.Resources.MemoryMB,
		Snapshot:    f.Snapshot.Enabled,
		SnapshotDir: snapDir,
	}
}

// Preset is a named, ready-to-run policy. Presets expose only options that at
// least one supported platform actually enforces.
type Preset struct {
	ID          string
	Name        string
	Description string
	Network     bool
	MemoryMB    int
	Snapshot    bool
}

// Presets returns the built-in starter policies shown in the UI.
func Presets() []Preset {
	return []Preset{
		{ID: "try-unknown-portable-app", Name: "Try unknown portable app", Description: "No network, 1 GiB limit, snapshots off", Network: false, MemoryMB: 1024, Snapshot: false},
		{ID: "build-script-test", Name: "Build script test", Description: "No network, 2 GiB limit", Network: false, MemoryMB: 2048, Snapshot: false},
		{ID: "temporary-browser-profile", Name: "Temporary browser profile", Description: "Network on, 2 GiB limit", Network: true, MemoryMB: 2048, Snapshot: false},
		{ID: "offline-test", Name: "Offline test", Description: "No network, 512 MiB limit", Network: false, MemoryMB: 512, Snapshot: true},
	}
}

// FindPreset returns the preset with the given ID, or the first preset.
func FindPreset(id string) Preset {
	ps := Presets()
	for _, p := range ps {
		if p.ID == id {
			return p
		}
	}
	return ps[0]
}
