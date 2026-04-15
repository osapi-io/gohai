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

package filesystem

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects filesystem data on Linux. gopsutil's `disk.Partitions`
// provides the mount table (from `/proc/mounts`); gopsutil's
// `disk.Usage` provides capacity + inodes per mount. When `lsblk` is
// on PATH we layer UUID / label / partition-UUID / partition-label per
// mount and surface block devices with a filesystem but no mountpoint
// as `Info.Unmounted`. For btrfs mounts we additionally read
// /sys/fs/btrfs/<uuid>/allocation/* via the injected avfs.VFS for
// allocation totals + RAID profile per Ohai parity.
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to production dependencies:
// the real OS filesystem for sysfs reads (btrfs allocation), and a
// real `os/exec` wrapper for `lsblk` / `zfs`. The gopsutil base is
// reached through package-level partitionsFn / usageFn, which tests
// swap via SetPartitionsFn / SetUsageFn.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect returns filesystem Info. Optional lsblk enrichment is
// attempted when Exec is configured; any lsblk error (binary missing,
// malformed JSON) silently skips the enrichment and leaves
// gopsutil-sourced fields intact.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	mounts, err := listMounts(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{Mounts: mounts}
	if l.Exec == nil {
		return info, nil
	}

	// lsblk enrichment — best-effort; non-fatal on failure.
	out, err := l.Exec.Execute(ctx,
		"lsblk", "-J", "-o", "NAME,UUID,LABEL,FSTYPE,MOUNTPOINT,PARTUUID,PARTLABEL")
	if err == nil {
		if entries, err := parseLsblk(out); err == nil {
			merged, unmounted := mergeLsblkIntoMounts(info.Mounts, entries)
			info.Mounts = merged
			info.Unmounted = unmounted
		}
	}

	// ZFS enumeration — independent of lsblk; only fires when the
	// `zfs` binary is on PATH. Any exec failure (binary missing,
	// permission denied, module not loaded) silently skips this
	// section, so non-ZFS hosts return zero ZFS datasets with no error.
	if out, err := l.Exec.Execute(ctx, "zfs", "get", "-p", "-H", "all"); err == nil {
		info.ZFSDatasets = parseZFSGetAll(out)
	}

	// Btrfs allocation enrichment — only mounts identified as btrfs
	// AND carrying a UUID (lsblk-derived) are eligible. Pure sysfs
	// reads via FS; nil FS or missing /sys/fs/btrfs paths skip cleanly.
	if l.FS != nil {
		for i := range info.Mounts {
			m := &info.Mounts[i]
			if m.Fstype == "btrfs" && m.UUID != "" {
				m.Btrfs = readBtrfsInfo(l.FS, m.UUID)
			}
		}
	}
	return info, nil
}
