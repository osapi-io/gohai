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

// Package filesystem collects filesystems known to the host, both
// mounted and (on Linux, via lsblk) unmounted. Mounted filesystems
// carry capacity, usage, and inode counters; every filesystem carries
// UUID and label when lsblk reports them.
package filesystem

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/shirou/gopsutil/v4/disk"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds filesystem data: the active mount table plus (Linux only)
// any block devices with a filesystem but no active mountpoint, and
// (Linux only, when `zfs` is on PATH) every ZFS dataset reported by
// `zfs get -p -H all`.
type Info struct {
	Mounts      []Mount      `json:"mounts"`
	Unmounted   []Filesystem `json:"unmounted,omitempty"`
	ZFSDatasets []ZFSDataset `json:"zfs_datasets,omitempty"`
}

// ZFSDataset describes one ZFS dataset — a filesystem, volume,
// snapshot, or bookmark the kernel's ZFS module knows about.
// Populated by the Linux variant when the `zfs` binary is on PATH;
// mirrors Ohai's `zfs_properties` / `zfs_parents` / `zfs_zpool` keys.
type ZFSDataset struct {
	// Name is the full dataset path: "tank", "tank/home", "tank/home/john".
	Name string `json:"name"`
	// Mountpoint is the dataset's mount path if the `mountpoint`
	// property is set to a real path. Empty when the dataset is a
	// snapshot, a volume, or has `mountpoint=none` / `legacy`.
	Mountpoint string `json:"mountpoint,omitempty"`
	// IsPool reports whether this dataset IS a zpool root (no slash
	// in the name — no proper ancestors). Matches Ohai's
	// `zfs_zpool` boolean.
	IsPool bool `json:"is_pool,omitempty"`
	// Parents holds every proper ancestor dataset path, shallowest
	// first. E.g. "tank/a/b" → ["tank", "tank/a"]. Matches Ohai's
	// `zfs_parents`.
	Parents []string `json:"parents,omitempty"`
	// Properties is the full property set from `zfs get all`.
	// Keys are property names (e.g. "compression", "quota", "used").
	// Values carry both the property value and the source (local /
	// default / inherited from X / received / "-").
	Properties map[string]ZFSProperty `json:"properties,omitempty"`
}

// ZFSProperty is one entry from `zfs get` — a property's value plus
// its source annotation. Source is `"-"` when the property has no
// source (read-only properties), `"default"` for kernel defaults,
// `"local"` for properties set directly on the dataset, or
// `"inherited from <ancestor>"` when inherited.
type ZFSProperty struct {
	Value  string `json:"value"`
	Source string `json:"source,omitempty"`
}

// Mount represents a single mounted filesystem.
type Mount struct {
	Device            string   `json:"device"`
	Mountpoint        string   `json:"mountpoint"`
	Fstype            string   `json:"fstype"`
	Opts              []string `json:"opts,omitempty"`
	Total             uint64   `json:"total,omitempty"`
	Used              uint64   `json:"used,omitempty"`
	Free              uint64   `json:"free,omitempty"`
	UsedPercent       float64  `json:"used_percent,omitempty"`
	InodesTotal       uint64   `json:"inodes_total,omitempty"`
	InodesUsed        uint64   `json:"inodes_used,omitempty"`
	InodesFree        uint64   `json:"inodes_free,omitempty"`
	InodesUsedPercent float64  `json:"inodes_used_percent,omitempty"`
	UUID              string   `json:"uuid,omitempty"`       // filesystem UUID from lsblk
	Label             string   `json:"label,omitempty"`      // filesystem label from lsblk
	PartUUID          string   `json:"part_uuid,omitempty"`  // GPT partition UUID from lsblk
	PartLabel         string   `json:"part_label,omitempty"` // GPT partition label from lsblk

	// Btrfs is populated only for btrfs mounts on Linux when the
	// /sys/fs/btrfs/<UUID>/allocation hierarchy is readable. Mirrors
	// Ohai's btrfs sub-object.
	Btrfs *BtrfsInfo `json:"btrfs,omitempty"`
}

// BtrfsInfo carries the btrfs-specific data the kernel exposes via
// sysfs at /sys/fs/btrfs/<UUID>/. Mirrors Ohai's btrfs sub-object.
type BtrfsInfo struct {
	// RAID is the detected RAID profile name from
	// /sys/fs/btrfs/<UUID>/allocation/<bg_type>/<profile>/. Common
	// values: "single", "dup", "raid0", "raid1", "raid5", "raid6",
	// "raid10", "raid1c3", "raid1c4". When multiple block-group types
	// use the same profile (the typical case) the value is shared;
	// when they diverge the most recent observed profile wins (Ohai
	// has the same last-write-wins behaviour).
	RAID string `json:"raid,omitempty"`
	// Allocation is the per-block-group-type allocation map, keyed by
	// the block-group type name ("data", "metadata", "system").
	Allocation map[string]BtrfsAllocation `json:"allocation,omitempty"`
}

// BtrfsAllocation reports the bytes allocated and used for one
// block-group type. Sourced from
// /sys/fs/btrfs/<UUID>/allocation/<bg_type>/{total_bytes,bytes_used}.
type BtrfsAllocation struct {
	TotalBytes uint64 `json:"total_bytes"`
	BytesUsed  uint64 `json:"bytes_used"`
}

// Filesystem describes a block device with a filesystem that isn't
// currently mounted — LUKS containers, inactive LVs, btrfs device
// members, etc. Populated from lsblk on Linux when the filesystem has
// no mountpoint. Capacity/usage are omitted; statfs requires a mount.
type Filesystem struct {
	Device    string `json:"device"`
	Fstype    string `json:"fstype"`
	UUID      string `json:"uuid,omitempty"`
	Label     string `json:"label,omitempty"`
	PartUUID  string `json:"part_uuid,omitempty"`
	PartLabel string `json:"part_label,omitempty"`
}

// Collector is the public interface every filesystem variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "filesystem" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the filesystem variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// partitionsFn is the injection seam for gopsutil's
// disk.PartitionsWithContext. Kept private so importers don't
// transitively need gopsutil. Swapped via SetPartitionsFn.
var partitionsFn = disk.PartitionsWithContext

// usageFn is the injection seam for gopsutil's disk.UsageWithContext.
// Kept private alongside partitionsFn. Swapped via SetUsageFn.
var usageFn = disk.UsageWithContext

// listMounts is the production bridge to gopsutil. Enumerates
// partitions and fetches usage (capacity + inodes) for each. Per-mount
// usage failures (permission denied, stale NFS, etc.) skip usage
// fields but keep the mount in the output.
func listMounts(
	ctx context.Context,
) ([]Mount, error) {
	parts, err := partitionsFn(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]Mount, 0, len(parts))
	for _, p := range parts {
		m := Mount{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Opts:       p.Opts,
		}
		if u, err := usageFn(ctx, p.Mountpoint); err == nil {
			m.Total = u.Total
			m.Used = u.Used
			m.Free = u.Free
			m.UsedPercent = u.UsedPercent
			m.InodesTotal = u.InodesTotal
			m.InodesUsed = u.InodesUsed
			m.InodesFree = u.InodesFree
			m.InodesUsedPercent = u.InodesUsedPercent
		}
		out = append(out, m)
	}
	return out, nil
}

// lsblkJSON mirrors `lsblk -J` output. Only the subset of fields we
// consume is declared — extra columns are ignored by json.Unmarshal.
type lsblkJSON struct {
	BlockDevices []lsblkNode `json:"blockdevices"`
}

// lsblkNode is a single node in lsblk's tree. Any node with a
// non-empty Fstype represents a filesystem we care about; children are
// recursively flattened.
type lsblkNode struct {
	Name       string      `json:"name"`
	Fstype     string      `json:"fstype"`
	UUID       string      `json:"uuid"`
	Label      string      `json:"label"`
	Mountpoint string      `json:"mountpoint"`
	PartUUID   string      `json:"partuuid"`
	PartLabel  string      `json:"partlabel"`
	Children   []lsblkNode `json:"children"`
}

// parseLsblk decodes lsblk's JSON output into a flat slice of
// Filesystem entries. Includes every node with a non-empty Fstype;
// callers decide mount vs unmounted by inspecting Mountpoint on the
// corresponding node.
//
// Returns the flat list of fs-bearing nodes paired with each node's
// mountpoint so the caller can partition into mounted / unmounted.
func parseLsblk(
	raw []byte,
) ([]lsblkEntry, error) {
	var j lsblkJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, err
	}
	var out []lsblkEntry
	var walk func(nodes []lsblkNode)
	walk = func(nodes []lsblkNode) {
		for _, n := range nodes {
			if n.Fstype != "" {
				out = append(out, lsblkEntry{
					Device:     "/dev/" + n.Name,
					Fstype:     n.Fstype,
					UUID:       n.UUID,
					Label:      n.Label,
					Mountpoint: n.Mountpoint,
					PartUUID:   n.PartUUID,
					PartLabel:  n.PartLabel,
				})
			}
			walk(n.Children)
		}
	}
	walk(j.BlockDevices)
	return out, nil
}

// lsblkEntry is the internal flattened form — a filesystem-bearing
// block device with enough context to decide mounted vs unmounted.
type lsblkEntry struct {
	Device     string
	Fstype     string
	UUID       string
	Label      string
	Mountpoint string
	PartUUID   string
	PartLabel  string
}

// btrfsBlockGroupTypes is the canonical set of btrfs block-group
// types Ohai walks under /sys/fs/btrfs/<uuid>/allocation/. Order is
// stable so the resulting Allocation map JSON-marshals predictably
// when consumers serialize for diffs.
var btrfsBlockGroupTypes = []string{"data", "metadata", "system"}

// readBtrfsInfo walks /sys/fs/btrfs/<uuid>/allocation/{data,metadata,system}/
// via the injected avfs.VFS and returns BtrfsInfo populated with the
// allocation totals + detected RAID profile. Returns nil when the
// allocation directory is missing (non-btrfs filesystem, stripped /sys,
// or kernel without sysfs btrfs export).
//
// Mirrors Ohai's collect_btrfs_data: total_bytes / bytes_used parsed
// per block-group type, and a `raid` field set from the profile
// subdirectory name. We extend Ohai's `single` / `dup` only check to
// recognise every btrfs RAID profile (raid0, raid1, raid1c3, raid1c4,
// raid5, raid6, raid10) since those exist in real /sys output and
// carry the same semantic.
func readBtrfsInfo(
	fs avfs.VFS,
	uuid string,
) *BtrfsInfo {
	allocRoot := "/sys/fs/btrfs/" + uuid + "/allocation"
	if _, err := fs.Stat(allocRoot); err != nil {
		return nil
	}
	info := &BtrfsInfo{Allocation: map[string]BtrfsAllocation{}}
	for _, bg := range btrfsBlockGroupTypes {
		dir := allocRoot + "/" + bg
		if _, err := fs.Stat(dir); err != nil {
			continue
		}
		info.Allocation[bg] = BtrfsAllocation{
			TotalBytes: readUint(fs, dir+"/total_bytes"),
			BytesUsed:  readUint(fs, dir+"/bytes_used"),
		}
		if r := detectBtrfsRAID(fs, dir); r != "" {
			info.RAID = r
		}
	}
	if len(info.Allocation) == 0 && info.RAID == "" {
		return nil
	}
	return info
}

// btrfsRAIDProfiles is the full set of RAID profile subdirectory
// names btrfs may publish under <bg_type>/. Iterated in the kernel's
// own ordering — the first profile present wins for that block-group
// type. Ohai checks only "single" / "dup" but real systems also see
// raidN profiles, so we cover the lot.
var btrfsRAIDProfiles = []string{
	"single", "dup", "raid0", "raid1", "raid1c3", "raid1c4",
	"raid5", "raid6", "raid10",
}

// detectBtrfsRAID returns the first RAID profile subdirectory present
// under dir (e.g. "/sys/fs/btrfs/<uuid>/allocation/data/raid1"). Empty
// when none of the known profile names exist.
func detectBtrfsRAID(
	fs avfs.VFS,
	dir string,
) string {
	for _, p := range btrfsRAIDProfiles {
		if _, err := fs.Stat(dir + "/" + p); err == nil {
			return p
		}
	}
	return ""
}

// readUint reads path through fs and returns the leading decimal
// integer value. Returns 0 on missing file, read error, or unparseable
// content — matches the silent-on-miss convention used elsewhere in
// the package.
func readUint(
	fs avfs.VFS,
	path string,
) uint64 {
	b, err := fs.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// parseZFSGetAll parses tab-separated `zfs get -p -H all` output into
// a flat slice of ZFSDataset. Format per line:
//
//	<name>\t<property>\t<value>\t<source>
//
// Lines that don't match are skipped. Datasets are emitted in the
// order they first appeared in the output, so consumers iterating the
// slice see pool roots before their children (zfs get's natural
// output order).
//
// The parser preserves Ohai's computed fields:
//   - Mountpoint is lifted from the `mountpoint` property when the
//     value is an absolute path (ignores `none` / `legacy` / `-`).
//   - Parents enumerates proper ancestor paths (ohai zfs_parents).
//   - IsPool is true when the dataset name has no `/` (ohai zfs_zpool).
func parseZFSGetAll(
	raw []byte,
) []ZFSDataset {
	byName := map[string]*ZFSDataset{}
	order := []string{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(fields) != 4 {
			continue
		}
		name, property, value, source := fields[0], fields[1], fields[2], fields[3]
		if name == "" || property == "" {
			continue
		}
		ds, ok := byName[name]
		if !ok {
			ds = &ZFSDataset{
				Name:       name,
				Properties: map[string]ZFSProperty{},
				IsPool:     !strings.Contains(name, "/"),
				Parents:    zfsParentsOf(name),
			}
			byName[name] = ds
			order = append(order, name)
		}
		ds.Properties[property] = ZFSProperty{Value: value, Source: source}
		if property == "mountpoint" && strings.HasPrefix(value, "/") {
			ds.Mountpoint = value
		}
	}
	out := make([]ZFSDataset, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}

// zfsParentsOf returns the proper-ancestor dataset paths of name,
// shallowest first. "tank/a/b" → ["tank", "tank/a"]; "tank" → nil.
// Matches Ohai's zfs_parents computation (parents.pop at the end).
func zfsParentsOf(
	name string,
) []string {
	parts := strings.Split(name, "/")
	if len(parts) <= 1 {
		return nil
	}
	out := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		out = append(out, strings.Join(parts[:i], "/"))
	}
	return out
}

// mergeLsblkIntoMounts enriches the mount table with UUID/label from
// lsblk (matched by device path) and returns the set of lsblk entries
// that had no mountpoint so the caller can surface them as
// Info.Unmounted.
func mergeLsblkIntoMounts(
	mounts []Mount,
	entries []lsblkEntry,
) ([]Mount, []Filesystem) {
	byDevice := map[string]lsblkEntry{}
	for _, e := range entries {
		byDevice[e.Device] = e
	}
	for i, m := range mounts {
		if e, ok := byDevice[m.Device]; ok {
			mounts[i].UUID = e.UUID
			mounts[i].Label = e.Label
			mounts[i].PartUUID = e.PartUUID
			mounts[i].PartLabel = e.PartLabel
			delete(byDevice, m.Device)
		}
	}
	var unmounted []Filesystem
	for _, e := range entries {
		if _, still := byDevice[e.Device]; !still {
			continue
		}
		if e.Mountpoint != "" {
			// mountpoint exists in lsblk but not in gopsutil's view;
			// treat as mounted-but-unobserved rather than unmounted.
			continue
		}
		unmounted = append(unmounted, Filesystem{
			Device:    e.Device,
			Fstype:    e.Fstype,
			UUID:      e.UUID,
			Label:     e.Label,
			PartUUID:  e.PartUUID,
			PartLabel: e.PartLabel,
		})
	}
	return mounts, unmounted
}
