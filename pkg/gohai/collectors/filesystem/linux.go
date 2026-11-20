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
	"fmt"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects filesystem data on Linux. /proc/mounts, read through
// FS, is the mount table; `df -kP` and `df -iP` give the capacity and
// inodes of each mount. When `lsblk` is
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
// real `os/exec` wrapper for `df`, `lsblk` and `zfs`.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

// Collect returns filesystem Info. Optional lsblk enrichment is
// attempted when Exec is configured; any lsblk error (binary missing,
// malformed JSON) silently skips the enrichment and leaves
// the mount table's fields intact.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	mounts, err := l.mounts(ctx)
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
			if m.Type == "btrfs" && m.UUID != "" {
				m.Btrfs = readBtrfsInfo(l.FS, m.UUID)
			}
		}
	}
	return info, nil
}

// procMounts is the kernel's mount table.
const procMounts = "/proc/mounts"

// mounts is each mount of /proc/mounts, with its sizes from df -kP and
// its inodes from df -iP. A mount that df does not list keeps no sizes.
func (l *Linux) mounts(
	ctx context.Context,
) ([]Mount, error) {
	b, err := l.FS.ReadFile(procMounts)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procMounts, err)
	}
	mounts := []Mount{}
	var points []string
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		m := Mount{
			Device: unescapeMount(f[0]), Mountpoint: unescapeMount(f[1]),
			Type: f[2], Opts: strings.Split(f[3], ","),
		}
		mounts = append(mounts, m)
		points = append(points, m.Mountpoint)
	}
	if l.Exec == nil {
		return mounts, nil
	}
	// df exits non-zero when one mount cannot be read, and still prints
	// the others.
	sizes, _ := l.Exec.Execute(ctx, "df", "-kP")
	counts, _ := l.Exec.Execute(ctx, "df", "-iP")
	sizeRows, inodeRows := dfRows(sizes, points, 4), dfRows(counts, points, 4)
	for i := range mounts {
		if r, ok := sizeRows[mounts[i].Mountpoint]; ok {
			usage(&mounts[i], r[0], r[1], r[2])
		}
		if r, ok := inodeRows[mounts[i].Mountpoint]; ok {
			inodes(&mounts[i], r[1], r[2])
		}
	}

	return mounts, nil
}

// unescapeMount undoes the octal escapes of /proc/mounts: \040 for a
// space, \011 for a tab, \012 for a newline and \134 for a backslash.
func unescapeMount(
	s string,
) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(s)
}
