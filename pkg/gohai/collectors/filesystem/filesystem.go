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
	"sync"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds filesystem data: the active mount table plus (Linux only)
// any block devices with a filesystem but no active mountpoint, and
// (Linux only, when `zfs` is on PATH) every ZFS dataset reported by
// `zfs get -p -H -t filesystem,volume all`.
type Info struct {
	Mounts      []Mount      `json:"mounts"`
	Unmounted   []Filesystem `json:"unmounted,omitempty"`
	ZFSDatasets []ZFSDataset `json:"zfs_datasets,omitempty"`
}

// ZFSDataset describes one ZFS dataset: a filesystem, volume,
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
	// IsPool reports whether this dataset IS a zpool root: no `/`, `@`
	// or `#` in the name, so no proper ancestors and not a snapshot or a
	// bookmark. Matches Ohai's `zfs_zpool` boolean.
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

// ZFSProperty is one entry from `zfs get`: a property's value plus
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
	Type              string   `json:"type"`
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
// currently mounted, such as LUKS containers, inactive LVs, btrfs device
// members, etc. Populated from lsblk on Linux when the filesystem has
// no mountpoint. Capacity/usage are omitted; statfs requires a mount.
type Filesystem struct {
	Device    string `json:"device"`
	Type      string `json:"type"`
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

func (base) Name() string                 { return "filesystem" }
func (base) Category() collector.Category { return collector.CategoryHardware }
func (base) DefaultEnabled() bool         { return true }
func (base) Dependencies() []string       { return nil }

// New returns the filesystem variant for the host OS.
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

// usage fills m's sizes from a df line's sizes in kB: total, used and
// available. Used is what the filesystem holds, Free is what a user can
// still write, and UsedPercent is Used of Used and Free, as statfs gives
// them.
func usage(
	m *Mount,
	total, used, avail uint64,
) {
	m.Total, m.Used, m.Free = total*1024, used*1024, avail*1024
	if m.Used+m.Free > 0 {
		m.UsedPercent = float64(m.Used) / float64(m.Used+m.Free) * 100
	}
}

// inodes fills m's inode counts from df's used and free inodes.
func inodes(
	m *Mount,
	used, free uint64,
) {
	m.InodesUsed, m.InodesFree, m.InodesTotal = used, free, used+free
	if m.InodesTotal > 0 {
		m.InodesUsedPercent = float64(used) / float64(m.InodesTotal) * 100
	}
}

// networkTypes are the filesystem types that a server holds. A df of one
// of them waits for the server, and a server that does not answer makes
// df wait until its timeout.
var networkTypes = map[string]bool{
	"9p": true, "afpfs": true, "afs": true, "ceph": true, "cifs": true,
	"davfs": true, "fuse.davfs2": true, "fuse.glusterfs": true,
	"fuse.sshfs": true, "glusterfs": true, "lustre": true, "nfs": true,
	"nfs4": true, "smb3": true, "smbfs": true, "webdav": true,
}

// maxNetworkDF is the most df commands of network mounts that run at the
// same time.
const maxNetworkDF = 8

// df is the rows of df for mounts, by mountpoint, with the n fields that
// dfRows takes. With no network mount, it is one `df flags`. Else the
// local mounts are one `df flags -- mountpoint...`, each network mount is
// a command of its own, and they all run at the same time as the local
// one. Thus a server that does not
// answer costs the rows of its own mount only, and the collector waits
// one command timeout at most, not one for each df.
//
// df exits non-zero when one mount cannot be read, and still prints the
// others. Thus its error is not used: the rows that it printed are.
func df(
	ctx context.Context,
	exec source.Runner,
	mounts []Mount,
	n int,
	flags string,
) map[string][]uint64 {
	var local, network []string
	for _, m := range mounts {
		if networkTypes[m.Type] {
			network = append(network, m.Mountpoint)
		} else {
			local = append(local, m.Mountpoint)
		}
	}
	// With no network mount, one df of all the mounts cannot wait for a
	// server. It is the command line that a fake host answers.
	if len(network) == 0 {
		out, _ := exec.Execute(ctx, "df", flags)

		return dfRows(out, local, n)
	}
	groups := [][]string{}
	if len(local) > 0 {
		groups = append(groups, local)
	}
	for _, p := range network {
		groups = append(groups, []string{p})
	}

	// The local command takes no slot, so that hung network mounts never
	// make it wait.
	outs := make([][]byte, len(groups))
	slots := make(chan struct{}, maxNetworkDF)
	var wg sync.WaitGroup
	for i, points := range groups {
		isNetwork := len(local) == 0 || i > 0
		wg.Add(1)
		go func() {
			defer wg.Done()
			if isNetwork {
				slots <- struct{}{}
				defer func() { <-slots }()
			}
			args := append([]string{flags, "--"}, points...)
			outs[i], _ = exec.Execute(ctx, "df", args...)
		}()
	}
	wg.Wait()

	rows := map[string][]uint64{}
	for i, out := range outs {
		for mp, r := range dfRows(out, groups[i], n) {
			rows[mp] = r
		}
	}

	return rows
}

// dfRows is the numbers of each line of df output, by mountpoint: the n
// fields before the mountpoint, with % signs dropped. A line belongs to
// the longest of mountpoints it ends with, because a device or a
// mountpoint can hold spaces. A line that ends with none, or has a field
// that is not a number, is left out.
func dfRows(
	out []byte,
	mountpoints []string,
	n int,
) map[string][]uint64 {
	rows := map[string][]uint64{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, " ")
		mp := ""
		for _, m := range mountpoints {
			if strings.HasSuffix(line, " "+m) && len(m) > len(mp) {
				mp = m
			}
		}
		if mp == "" {
			continue
		}
		f := strings.Fields(strings.TrimSuffix(line, " "+mp))
		if len(f) < n {
			continue
		}
		nums := make([]uint64, 0, n)
		for _, v := range f[len(f)-n:] {
			x, err := strconv.ParseUint(strings.TrimSuffix(v, "%"), 10, 64)
			if err != nil {
				break
			}
			nums = append(nums, x)
		}
		if len(nums) == n {
			rows[mp] = nums
		}
	}

	return rows
}

// lsblkJSON mirrors `lsblk -J` output. Only the subset of fields we
// consume is declared. json.Unmarshal ignores the extra columns.
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
					Device:     n.Name,
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

// lsblkEntry is the internal flattened form: a filesystem-bearing
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
// own order, and the first profile present wins for that block-group
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
// content, as the silent-on-miss convention elsewhere in
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

// parseZFSGetAll parses tab-separated `zfs get -p -H -t
// filesystem,volume all` output into
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
//   - IsPool is true when the dataset name has no `/`, `@` or `#`: a
//     pool root, not a dataset under it, a snapshot or a bookmark (ohai
//     zfs_zpool).
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
				IsPool:     !strings.ContainsAny(name, "/@#"),
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
// shallowest first. "tank/a/b" → ["tank", "tank/a"]; "tank" → nil. A
// snapshot or a bookmark has its dataset as a parent too:
// "tank/a@snap" → ["tank", "tank/a"]. Matches Ohai's zfs_parents
// computation (parents.pop at the end).
func zfsParentsOf(
	name string,
) []string {
	if i := strings.IndexAny(name, "@#"); i >= 0 {
		return append(zfsParentsOf(name[:i]), name[:i])
	}
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
			// mountpoint exists in lsblk but not in the mount table;
			// treat as mounted-but-unobserved rather than unmounted.
			continue
		}
		unmounted = append(unmounted, Filesystem{
			Device:    e.Device,
			Type:      e.Fstype,
			UUID:      e.UUID,
			Label:     e.Label,
			PartUUID:  e.PartUUID,
			PartLabel: e.PartLabel,
		})
	}
	return mounts, unmounted
}
