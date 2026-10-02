package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/portable-sandboxer/portable-sandboxer/internal/config"
	"github.com/portable-sandboxer/portable-sandboxer/internal/sandbox"
)

// Run builds the desktop UI and blocks until the window is closed.
func Run() {
	a := app.NewWithID("dev.portablesandboxer.app")
	w := a.NewWindow("Portable Sandboxer")
	w.Resize(fyne.NewSize(1060, 720))

	target := ""

	logs := widget.NewMultiLineEntry()
	logs.SetPlaceHolder("Session events appear here.")
	logs.Disable()
	status := widget.NewLabel("Ready")
	selected := widget.NewLabel("No program selected")
	platform := widget.NewLabel(platformNote())
	platform.Wrapping = fyne.TextWrapWord

	appendLog := func(format string, args ...any) {
		line := time.Now().Format("15:04:05") + "  " + fmt.Sprintf(format, args...)
		logs.SetText(strings.TrimRight(logs.Text, "\n") + "\n" + line)
	}

	presets := config.Presets()
	ids := make([]string, 0, len(presets))
	for _, p := range presets {
		ids = append(ids, p.Name)
	}
	network := widget.NewCheck("Allow network", nil)
	memory := widget.NewEntry()
	memory.SetPlaceHolder("Memory limit (MiB)")
	snapshot := widget.NewCheck("Keep snapshot after exit", nil)

	byName := map[string]config.Preset{}
	for _, p := range presets {
		byName[p.Name] = p
	}
	applyPreset := func(name string) {
		p, ok := byName[name]
		if !ok {
			return
		}
		network.SetChecked(p.Network)
		memory.SetText(strconv.Itoa(p.MemoryMB))
		snapshot.SetChecked(p.Snapshot)
	}
	preset := widget.NewSelect(ids, applyPreset)
	preset.SetSelected(presets[0].Name)
	applyPreset(presets[0].Name)

	// Load a local config file if present so advanced users can pre-seed the form.
	loadConfig := func() {
		path := filepath.Join(executableDir(), "portable-sandboxer.yaml")
		f, err := config.Load(path)
		if err != nil {
			appendLog("No config loaded (%v); using built-in preset.", err)
			return
		}
		p := config.FindPreset(f.Preset)
		preset.SetSelected(p.Name)
		network.SetChecked(f.Network)
		if f.Resources.MemoryMB > 0 {
			memory.SetText(strconv.Itoa(f.Resources.MemoryMB))
		}
		snapshot.SetChecked(f.Snapshot.Enabled)
		appendLog("Loaded %s", path)
	}

	choose := widget.NewButton("Choose program", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil {
				appendLog("File dialog error: %v", err)
				return
			}
			if r == nil {
				return
			}
			target = r.URI().Path()
			selected.SetText(filepath.Base(target))
			status.SetText("Selected")
		}, w)
	})

	start := widget.NewButton("Run isolated", func() {
		if target == "" {
			appendLog("Select a program first.")
			return
		}
		policy := sandbox.Policy{Network: network.Checked}
		if mb, err := strconv.Atoi(strings.TrimSpace(memory.Text)); err == nil {
			policy.MemoryMB = mb
		}
		if snapshot.Checked {
			policy.Snapshot = true
			policy.SnapshotDir = sandbox.DefaultSnapshotDir()
		}
		s, err := sandbox.Start(context.Background(), target, nil, policy)
		if err != nil {
			status.SetText("Launch blocked")
			appendLog("Launch blocked: %v", err)
			return
		}
		status.SetText("Running")
		appendLog("Started %s in %s", filepath.Base(target), s.Dir)
		for _, n := range s.Notes {
			appendLog("%s", n)
		}
		go func() {
			runErr := s.Wait()
			snap := s.SnapshotDir
			fyne.Do(func() {
				if runErr != nil {
					appendLog("Process ended: %v", runErr)
				} else {
					appendLog("Process ended normally")
				}
				if snap != "" {
					appendLog("Snapshot kept at %s", snap)
				}
				status.SetText("Ready")
			})
		}()
	})

	cleanup := widget.NewButton("Clear temporary sessions", func() {
		removed, err := sandbox.CleanupStale()
		if err != nil {
			appendLog("Cleanup failed: %v", err)
			return
		}
		appendLog("Removed %d temporary workspace(s).", removed)
	})

	clearLogs := widget.NewButton("Clear log", func() { logs.SetText("") })

	// The network control is only shown where a backend can enforce it.
	policyItems := []fyne.CanvasObject{network, memory, snapshot}
	if runtime.GOOS == "windows" {
		network.Disable()
	}

	leftItems := []fyne.CanvasObject{
		widget.NewLabelWithStyle("ISOLATION", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		preset,
		widget.NewSeparator(),
	}
	leftItems = append(leftItems, policyItems...)
	leftItems = append(leftItems, widget.NewSeparator(), platform, widget.NewSeparator(), choose, selected, start, cleanup)
	left := container.NewVBox(leftItems...)

	dropText := widget.NewLabel("TEMPORARY SESSION\nChoose a program, then run it in an isolated process namespace.\nUse the button on the left; drag-and-drop is not supported yet.")
	dropText.Alignment = fyne.TextAlignCenter
	center := container.NewMax(widget.NewCard("Temporary workspace", "All changes stay here and are removed when the process exits.", dropText))

	logHeader := container.NewBorder(nil, nil, widget.NewLabel("Recent events"), clearLogs)
	snapshots := widget.NewLabel("Snapshots: " + sandbox.DefaultSnapshotDir())
	snapshots.Wrapping = fyne.TextWrapWord
	bottom := container.NewVBox(status, logHeader, container.NewScroll(logs), snapshots)

	content := container.NewBorder(nil, bottom, left, nil, center)
	w.SetContent(content)

	loadConfig()
	w.ShowAndRun()
}

// platformNote states honestly what each backend can enforce.
func platformNote() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows: Job Object process-count/memory limits and kill-on-close. No restricted token, registry, filesystem, or network isolation."
	case "linux":
		return "Linux: unprivileged user/mount/PID namespaces, plus a network namespace when networking is off. Memory limit is best effort via cgroup v2."
	case "darwin":
		return "macOS: experimental Seatbelt (sandbox-exec) profile, deprecated by Apple and OS-version dependent."
	default:
		return runtime.GOOS + ": no isolation backend; launching is refused."
	}
}

func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		dir, _ := os.Getwd()
		return dir
	}
	return filepath.Dir(exe)
}
