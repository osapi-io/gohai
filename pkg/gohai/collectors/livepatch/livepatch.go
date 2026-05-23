// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package livepatch reports the status of kernel livepatches loaded on
// a Linux host. Each entry under /sys/kernel/livepatch/ represents a
// loaded patch module. On Darwin the collector returns nil gracefully —
// kernel livepatching is a Linux-only feature.
package livepatch

import (
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// livepatchDir is the sysfs path containing loaded livepatch modules.
const livepatchDir = "/sys/kernel/livepatch"

// Info holds the livepatch status for this host.
type Info struct {
	// Patches is the map of loaded livepatch modules keyed by patch name.
	// An empty map means no livepatches are loaded (but the subsystem
	// exists). A nil Patches field means the livepatch sysfs directory
	// does not exist — either the kernel was not compiled with livepatch
	// support or no livepatches have ever been loaded.
	Patches map[string]Patch `json:"patches"`
}

// Patch describes a single loaded livepatch module.
type Patch struct {
	// Enabled is true when the patch is active (sysfs "enabled" == "1").
	Enabled bool `json:"enabled"`
	// Transition is true when the patch is mid-transition (sysfs
	// "transition" == "1") — the kernel is still patching live tasks.
	Transition bool `json:"transition"`
}

// Collector is the public interface every livepatch variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "livepatch".
func (base) Name() string { return "livepatch" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — livepatch is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the livepatch collector variant appropriate to the
// detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// collectPatches reads /sys/kernel/livepatch/ entries. Returns nil
// when the directory does not exist — livepatch kernel support is
// absent. Returns an empty map when the directory exists but no patches
// are loaded.
func collectPatches(
	fs avfs.VFS,
) map[string]Patch {
	entries, err := fs.ReadDir(livepatchDir)
	if err != nil {
		// Directory absent → livepatch not supported or no patches loaded.
		return nil
	}

	patches := map[string]Patch{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		patch := Patch{
			Enabled:    readSysBool(fs, livepatchDir+"/"+name+"/enabled"),
			Transition: readSysBool(fs, livepatchDir+"/"+name+"/transition"),
		}
		patches[name] = patch
	}
	return patches
}

// readSysBool reads a sysfs file that contains "0" or "1" and returns
// the boolean value. Returns false when the file does not exist or
// cannot be read — absent sysfs files are normal for optional kernel
// features.
func readSysBool(
	fs avfs.VFS,
	path string,
) bool {
	b, err := fs.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "1"
}
