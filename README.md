# Portable-Sandboxer

**Try it. Close it. Leave no workspace behind.**

Portable-Sandboxer is an open-source desktop launcher for short-lived program sessions. Drop in an executable or script, run it in a temporary workspace with the isolation your OS actually supports, then close it. It is not a VM, not Docker, and not a hardened malware sandbox.

> **Security warning:** This is an early prototype, not a security boundary against determined malware. It does not promise to prevent advanced escapes. Do not run malware or sensitive workloads with it. Read the per-platform limits below before trusting any toggle.

## Preview

Add a real screenshot at `docs/screenshot.png` after running the app on a desktop. No mock image is shipped.

## Quick start

Requirements: Go 1.23+ and the native desktop toolchain that Fyne needs for your OS (OpenGL/X11 on Linux, Xcode command line tools on macOS).

```sh
git clone https://github.com/wangzi5151/Portable-Sandboxer.git
cd Portable-Sandboxer
go run .
```

1. Pick a preset on the left.
2. Click **Choose program** and select an executable or script.
3. Click **Run isolated**. The temporary workspace is removed when the process exits.

Drag-and-drop is **not** implemented yet; use the file button. A local `portable-sandboxer.yaml` next to the binary pre-fills the form.

## What each platform actually enforces

| Platform | Implemented | Not implemented |
| --- | --- | --- |
| **Windows** | Job Object with active-process limit, kill-on-close process-tree teardown, optional per-process memory limit | Restricted token, filesystem/registry virtualization, network isolation |
| **Linux** | Unprivileged user, mount and PID namespaces (Go native clone); network namespace when networking is off; cgroup v2 memory limit best effort | Read-only filesystem policy, CPU limits, guaranteed cgroup access without delegation |
| **macOS** | Generated Seatbelt (`sandbox-exec`) profile: system reads allowed, writes confined to the workspace, network deny by default | Resource limits; Apple considers this interface deprecated and behavior varies by OS version |

Every session gets its own temporary directory. On Windows the Job Object is closed after the foreground process exits, which terminates any surviving children. `Clear temporary sessions` removes stale workspaces after a crash.

The network control is hidden on Windows because that backend cannot enforce it. Presets set only options a backend honors.

## Presets

| Preset | Network | Memory | Snapshot |
| --- | --- | --- | --- |
| Try unknown portable app | off | 1 GiB | off |
| Build script test | off | 2 GiB | off |
| Temporary browser profile | on | 2 GiB | off |
| Offline test | off | 512 MiB | on |

## Configuration

`portable-sandboxer.yaml` is loaded at startup from the executable's directory. Omitted keys keep safe defaults (network off, snapshots off).

```yaml
version: 1
preset: try-unknown-portable-app
network: false
resources:
  memory_mb: 1024
snapshot:
  enabled: false
  dir: ""
```

## No install, no daemon, no telemetry

No service, no autostart, no background daemon, no API keys, no runtime download, no telemetry. Fyne is linked into the binary. The app never asks for administrator privileges; Linux namespace support must already be enabled on the host.

## Build

```sh
go build .            # native desktop binary
go test ./...         # unit tests for policy, config and cleanup
```

`build-all.sh` / `build-all.bat` attempt cross builds. Fyne's desktop driver needs native GL/X11 toolchains, so working GUI binaries should be built on (or for) their target OS. The CI workflow builds on Windows, Linux and macOS runners.

## Known limitations

- No drag-and-drop.
- Windows has no filesystem/registry/network isolation in this prototype.
- Snapshot copies the temporary workspace only; it does not roll back writes made outside it.
- Child processes are only reliably contained on Windows via the Job Object; containment elsewhere is best effort.
- macOS `sandbox-exec` is deprecated by Apple.

## Comparison

| | Portable-Sandboxer | Docker | Windows Sandbox | Sandboxie |
| --- | --- | --- | --- | --- |
| Focus | Small temporary-session launcher | Containers/deployment | Disposable Windows VM | Windows app sandbox |
| Platforms | Windows/macOS/Linux | Linux kernel containers | Windows Pro/Enterprise/Education | Windows |
| Install/service | Single app | Engine + tooling | Windows feature | Install required |
| Isolation maturity | Experimental | Mature | VM-backed | Established |
| Current caveat | Windows has process-tree containment only | Heavier operationally | Not on all editions | Platform-specific |

## Disclaimer

MIT, no warranty. You are responsible for the programs you run and for verifying the actual isolation properties on your OS. The maintainers do not warrant prevention of malware infection, data loss, or sandbox escape.
