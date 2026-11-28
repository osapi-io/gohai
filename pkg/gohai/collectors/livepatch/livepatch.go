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
// loaded patch module. On Darwin the collector returns nil gracefully,
// kernel livepatching is a Linux-only feature.
package livepatch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// livepatchDir is the sysfs path containing loaded livepatch modules.
const livepatchDir = "/sys/kernel/livepatch"

// Info holds the livepatch status for this host.
type Info struct {
	// Patches is the map of loaded livepatch modules keyed by patch name.
	// An empty map means no livepatches are loaded (but the subsystem
	// exists). When the livepatch sysfs directory does not exist, because
	// the kernel has no livepatch support, the collector gives no fact, so
	// Patches is never nil.
	Patches map[string]Patch `json:"patches"`
}

// Patch describes a single loaded livepatch module.
type Patch struct {
	// Enabled is true when the patch is active (sysfs "enabled" == "1").
	Enabled bool `json:"enabled"`
	// Transition is true when the patch is mid-transition (sysfs
	// "transition" == "1"): the kernel is still patching live tasks.
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
func (base) Category() collector.Category { return collector.CategoryLinux }

// DefaultEnabled returns false, because livepatch is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the livepatch collector variant appropriate to the
// detected host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// collectPatches reads /sys/kernel/livepatch/ entries. It returns nil
// when the directory does not exist, because the kernel has no livepatch
// support, and an empty map when the directory exists but no patches are
// loaded. A directory or file that is there and can't be read is an
// error.
func collectPatches(
	fs avfs.VFS,
) (map[string]Patch, error) {
	entries, err := fs.ReadDir(livepatchDir)
	if collector.Absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", livepatchDir, err)
	}

	var errs []error
	patches := map[string]Patch{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		enabled, err := readSysBool(fs, livepatchDir+"/"+name+"/enabled")
		errs = append(errs, err)
		transition, err := readSysBool(fs, livepatchDir+"/"+name+"/transition")
		errs = append(errs, err)
		patches[name] = Patch{Enabled: enabled, Transition: transition}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return patches, nil
}

// readSysBool reads a sysfs file that contains "0" or "1" and returns
// the boolean value. A missing file is false, because absent sysfs files
// are normal for optional kernel features. A file that is there and
// can't be read is an error.
func readSysBool(
	fs avfs.VFS,
	path string,
) (bool, error) {
	b, err := fs.ReadFile(path)
	if collector.Absent(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	return strings.TrimSpace(string(b)) == "1", nil
}
