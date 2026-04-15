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

package filesystem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	gpdisk "github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
)

var (
	_ collector.Collector = (*filesystem.Linux)(nil)
	_ collector.Collector = (*filesystem.Darwin)(nil)
)

type FilesystemPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FilesystemPublicTestSuite))
}

// btrfsBgFixture is one block-group entry for the btrfs sysfs mock.
// Either total/used (uint64) OR totalRaw (string) may be set —
// totalRaw lets tests inject unparseable content for the silent-skip
// path. raid is the profile subdirectory name to create.
type btrfsBgFixture struct {
	raid     string
	total    uint64
	used     uint64
	totalRaw string // overrides total when non-empty
}

// btrfsFS builds a memfs prepared with /sys/fs/btrfs/<uuid>/allocation/...
// matching the supplied per-block-group fixtures. Profile subdir is
// created as an empty directory so detectBtrfsRAID's Stat hits.
func btrfsFS(
	s *FilesystemPublicTestSuite,
	uuid string,
	bgs map[string]btrfsBgFixture,
) avfs.VFS {
	fs := memfs.New()
	root := "/sys/fs/btrfs/" + uuid + "/allocation"
	s.Require().NoError(fs.MkdirAll(root, 0o755))
	for bg, fix := range bgs {
		dir := root + "/" + bg
		s.Require().NoError(fs.MkdirAll(dir, 0o755))
		if fix.raid != "" {
			s.Require().NoError(fs.MkdirAll(dir+"/"+fix.raid, 0o755))
		}
		if fix.totalRaw != "" {
			s.Require().NoError(fs.WriteFile(dir+"/total_bytes", []byte(fix.totalRaw), 0o644))
		} else {
			s.Require().NoError(fs.WriteFile(
				dir+"/total_bytes",
				[]byte(strconvBytes(fix.total)),
				0o644,
			))
		}
		s.Require().NoError(fs.WriteFile(
			dir+"/bytes_used",
			[]byte(strconvBytes(fix.used)),
			0o644,
		))
	}
	return fs
}

// strconvBytes is a tiny helper to avoid importing strconv here.
func strconvBytes(
	n uint64,
) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// noLsblkExec returns a mock Executor that errors on every call —
// simulates lsblk absent from PATH or a minimal container.
func noLsblkExec(
	t *testing.T,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsblk", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("not found")).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "zfs", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("not found")).
		AnyTimes()
	return m
}

// lsblkExec returns a mock Executor that returns canned lsblk JSON.
// Any `zfs` invocation returns an error (simulates `zfs` not installed —
// the common case for Linux hosts without OpenZFS).
func lsblkExec(
	t *testing.T,
	out string,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsblk", gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]byte(out), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "zfs", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("zfs: command not found")).
		AnyTimes()
	return m
}

// lsblkZFSExec behaves like lsblkExec but additionally returns canned
// `zfs get -p -H all` output for the ZFS enumeration path.
func lsblkZFSExec(
	t *testing.T,
	lsblkOut, zfsOut string,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsblk", gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]byte(lsblkOut), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "zfs", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]byte(zfsOut), nil).
		AnyTimes()
	return m
}

func (s *FilesystemPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := filesystem.New()
			s.Equal("filesystem", c.Name())
			s.Equal("hardware", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*filesystem.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*filesystem.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *FilesystemPublicTestSuite) TestCollect() {
	baseParts := []gpdisk.PartitionStat{
		{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4"},
		{Device: "/dev/sda2", Mountpoint: "/boot", Fstype: "ext4"},
	}
	okPartitions := func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
		return baseParts, nil
	}
	okUsage := func(_ context.Context, mp string) (*gpdisk.UsageStat, error) {
		if mp == "/" {
			return &gpdisk.UsageStat{
				Total:             100,
				Used:              50,
				Free:              50,
				UsedPercent:       50,
				InodesTotal:       1000,
				InodesUsed:        250,
				InodesFree:        750,
				InodesUsedPercent: 25,
			}, nil
		}
		return &gpdisk.UsageStat{}, nil
	}
	darwinParts := []gpdisk.PartitionStat{
		{
			Device:     "/dev/disk3s1",
			Mountpoint: "/",
			Fstype:     "apfs",
			Opts:       []string{"rw"},
		},
	}
	darwinPartitions := func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
		return darwinParts, nil
	}
	darwinUsage := func(context.Context, string) (*gpdisk.UsageStat, error) {
		return &gpdisk.UsageStat{Total: 500, Used: 200, Free: 300, UsedPercent: 40}, nil
	}

	tests := []struct {
		name         string
		variant      string
		partitionsFn func(context.Context, bool) ([]gpdisk.PartitionStat, error)
		usageFn      func(context.Context, string) (*gpdisk.UsageStat, error)
		exec         func(*testing.T) executor.Executor
		fs           avfs.VFS // optional; nil → btrfs sysfs reads short-circuit
		wantErr      bool
		validate     func(*filesystem.Info)
	}{
		{
			name:         "linux: no lsblk, mounts unchanged no unmounted",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         func(t *testing.T) executor.Executor { return noLsblkExec(t) },
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
				s.Equal(uint64(1000), i.Mounts[0].InodesTotal)
				s.Equal(float64(25), i.Mounts[0].InodesUsedPercent)
			},
		},
		{
			name:         "linux: lsblk merges uuid/label into matching mounts",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":null,"children":[
						{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"part-1","partlabel":""},
						{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"EFI","mountpoint":"/boot","partuuid":"part-2","partlabel":"EFI"}
					]}
				]}`)
			},
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Equal("root-uuid", i.Mounts[0].UUID)
				s.Equal("part-1", i.Mounts[0].PartUUID)
				s.Equal("boot-uuid", i.Mounts[1].UUID)
				s.Equal("EFI", i.Mounts[1].Label)
				s.Equal("EFI", i.Mounts[1].PartLabel)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:         "linux: lsblk unmounted entry surfaces as Unmounted",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":null,"children":[
						{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"","partlabel":""},
						{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"","mountpoint":"/boot","partuuid":"","partlabel":""},
						{"name":"sdb1","fstype":"crypto_LUKS","uuid":"luks-uuid","label":"data","mountpoint":"","partuuid":"","partlabel":""}
					]}
				]}`)
			},
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Len(i.Unmounted, 1)
				s.Equal("/dev/sdb1", i.Unmounted[0].Device)
				s.Equal("crypto_LUKS", i.Unmounted[0].Fstype)
				s.Equal("luks-uuid", i.Unmounted[0].UUID)
				s.Equal("data", i.Unmounted[0].Label)
			},
		},
		{
			name:         "linux: lsblk node with empty fstype ignored (raw disk)",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":"","children":[
						{"name":"sda1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/","partuuid":"","partlabel":""}
					]}
				]}`)
			},
			validate: func(i *filesystem.Info) {
				s.Equal("u", i.Mounts[0].UUID)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:         "linux: lsblk entry with mountpoint but not in gopsutil mounts is ignored",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sdc1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/mnt/foo","partuuid":"","partlabel":""}
				]}`)
			},
			validate: func(i *filesystem.Info) {
				s.Empty(i.Unmounted)
			},
		},
		{
			name:         "linux: malformed lsblk json, extension silently skipped",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         func(t *testing.T) executor.Executor { return lsblkExec(t, `not json`) },
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
			},
		},
		{
			name:         "linux: nil Exec, extension skipped cleanly",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         func(*testing.T) executor.Executor { return nil },
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:    "linux: gopsutil partitions error propagated",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return nil, errors.New("partitions error")
			},
			usageFn: okUsage,
			exec:    func(t *testing.T) executor.Executor { return noLsblkExec(t) },
			wantErr: true,
		},
		{
			name:         "linux: usage error keeps mount without usage",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return nil, errors.New("usage failed")
			},
			exec: func(t *testing.T) executor.Executor { return noLsblkExec(t) },
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Zero(i.Mounts[0].Total)
			},
		},
		{
			name:         "linux: ZFS datasets parsed with pool + nested + mountpoints",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				// tab-separated `zfs get -p -H all` — realistic subset
				// covering pool root + nested fs + snapshot (mountpoint
				// absent) + dataset with `none` (skipped for mountpoint).
				const zfs = "tank\ttype\tfilesystem\t-\n" +
					"tank\tmountpoint\t/tank\tdefault\n" +
					"tank\tused\t12345\t-\n" +
					"tank/home\ttype\tfilesystem\t-\n" +
					"tank/home\tmountpoint\t/home\tlocal\n" +
					"tank/home\tcompression\tlz4\tlocal\n" +
					"tank/home/john\ttype\tfilesystem\t-\n" +
					"tank/home/john\tmountpoint\t/home/john\tinherited from tank/home\n" +
					"tank/vol1\ttype\tvolume\t-\n" +
					"tank/vol1\tmountpoint\tnone\tdefault\n" +
					"tank@snap1\ttype\tsnapshot\t-\n"
				return lsblkZFSExec(t, "", zfs)
			},
			validate: func(i *filesystem.Info) {
				s.Require().Len(i.ZFSDatasets, 5)

				tank := i.ZFSDatasets[0]
				s.Equal("tank", tank.Name)
				s.Equal("/tank", tank.Mountpoint)
				s.True(tank.IsPool)
				s.Empty(tank.Parents)
				s.Equal("filesystem", tank.Properties["type"].Value)
				s.Equal("default", tank.Properties["mountpoint"].Source)

				home := i.ZFSDatasets[1]
				s.Equal("tank/home", home.Name)
				s.Equal("/home", home.Mountpoint)
				s.False(home.IsPool)
				s.Equal([]string{"tank"}, home.Parents)
				s.Equal("lz4", home.Properties["compression"].Value)
				s.Equal("local", home.Properties["compression"].Source)

				john := i.ZFSDatasets[2]
				s.Equal("tank/home/john", john.Name)
				s.Equal("/home/john", john.Mountpoint)
				s.Equal([]string{"tank", "tank/home"}, john.Parents)
				s.Equal(
					"inherited from tank/home",
					john.Properties["mountpoint"].Source,
				)

				vol := i.ZFSDatasets[3]
				s.Equal("tank/vol1", vol.Name)
				s.Empty(vol.Mountpoint) // "none" is not an absolute path
				s.Equal("none", vol.Properties["mountpoint"].Value)

				snap := i.ZFSDatasets[4]
				s.Equal("tank@snap1", snap.Name)
				s.True(snap.IsPool) // no "/" → looks like a pool-level name
			},
		},
		{
			name:         "linux: zfs binary present but empty output yields no datasets",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkZFSExec(t, "", "")
			},
			validate: func(i *filesystem.Info) {
				s.Empty(i.ZFSDatasets)
			},
		},
		{
			name:         "linux: zfs malformed line skipped, well-formed line parsed",
			variant:      "linux",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: func(t *testing.T) executor.Executor {
				return lsblkZFSExec(
					t,
					"",
					"garbage\nonly two\tfields\nname\tprop\tval\tsource\n\t\t\t\n",
				)
			},
			validate: func(i *filesystem.Info) {
				s.Require().Len(i.ZFSDatasets, 1)
				s.Equal("name", i.ZFSDatasets[0].Name)
				s.Equal("val", i.ZFSDatasets[0].Properties["prop"].Value)
			},
		},
		{
			name:    "linux: btrfs mount enriched with allocation + raid profile",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "btrfs"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{Total: 1, Used: 0}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"abc-123","mountpoint":"/"}
				]}`)
			},
			fs: btrfsFS(s, "abc-123", map[string]btrfsBgFixture{
				"data":     {raid: "raid1", total: 1073741824, used: 524288000},
				"metadata": {raid: "raid1", total: 268435456, used: 134217728},
				"system":   {raid: "raid1", total: 33554432, used: 16384},
			}),
			validate: func(i *filesystem.Info) {
				s.Require().Len(i.Mounts, 1)
				m := i.Mounts[0]
				s.Equal("btrfs", m.Fstype)
				s.Equal("abc-123", m.UUID)
				s.Require().NotNil(m.Btrfs)
				s.Equal("raid1", m.Btrfs.RAID)
				s.Equal(uint64(1073741824), m.Btrfs.Allocation["data"].TotalBytes)
				s.Equal(uint64(524288000), m.Btrfs.Allocation["data"].BytesUsed)
				s.Equal(uint64(268435456), m.Btrfs.Allocation["metadata"].TotalBytes)
				s.Equal(uint64(33554432), m.Btrfs.Allocation["system"].TotalBytes)
			},
		},
		{
			name:    "linux: btrfs single profile recognised + zero-byte file tolerated",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "btrfs"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"single-fs","mountpoint":"/"}
				]}`)
			},
			fs: btrfsFS(s, "single-fs", map[string]btrfsBgFixture{
				"data":     {raid: "single", totalRaw: "not-a-number"},
				"metadata": {raid: "single", total: 1024, used: 256},
			}),
			validate: func(i *filesystem.Info) {
				m := i.Mounts[0]
				s.Require().NotNil(m.Btrfs)
				s.Equal("single", m.Btrfs.RAID)
				s.Equal(uint64(0), m.Btrfs.Allocation["data"].TotalBytes) // unparseable
				s.Equal(uint64(1024), m.Btrfs.Allocation["metadata"].TotalBytes)
			},
		},
		{
			name:    "linux: btrfs mount without /sys/fs/btrfs entry leaves Btrfs nil",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "btrfs"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"missing","mountpoint":"/"}
				]}`)
			},
			fs: memfs.New(), // empty FS — no /sys/fs/btrfs at all
			validate: func(i *filesystem.Info) {
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:    "linux: btrfs bg dirs exist but with no profile + missing files",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "btrfs"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"empty-files","mountpoint":"/"}
				]}`)
			},
			fs: func() avfs.VFS {
				fs := memfs.New()
				// bg dir exists, no profile subdirs, no total_bytes/bytes_used files —
				// exercises detectBtrfsRAID's no-match path and readUint's missing-file path.
				s.Require().NoError(
					fs.MkdirAll("/sys/fs/btrfs/empty-files/allocation/data", 0o755),
				)
				return fs
			}(),
			validate: func(i *filesystem.Info) {
				m := i.Mounts[0]
				s.Require().NotNil(m.Btrfs)
				s.Empty(m.Btrfs.RAID)
				s.Equal(uint64(0), m.Btrfs.Allocation["data"].TotalBytes)
				s.Equal(uint64(0), m.Btrfs.Allocation["data"].BytesUsed)
			},
		},
		{
			name:    "linux: btrfs allocation root exists but no bg subdirs yields nil Btrfs",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "btrfs"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"empty-bg","mountpoint":"/"}
				]}`)
			},
			fs: func() avfs.VFS {
				fs := memfs.New()
				s.Require().NoError(fs.MkdirAll("/sys/fs/btrfs/empty-bg/allocation", 0o755))
				return fs
			}(),
			validate: func(i *filesystem.Info) {
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:    "linux: non-btrfs mount left alone even with FS configured",
			variant: "linux",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return []gpdisk.PartitionStat{
					{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4"},
				}, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"ext4","uuid":"ext-uuid","mountpoint":"/"}
				]}`)
			},
			fs: memfs.New(),
			validate: func(i *filesystem.Info) {
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:         "darwin: APFS root populated",
			variant:      "darwin",
			partitionsFn: darwinPartitions,
			usageFn:      darwinUsage,
			validate: func(i *filesystem.Info) {
				s.Require().Len(i.Mounts, 1)
				s.Equal("/dev/disk3s1", i.Mounts[0].Device)
				s.Equal("apfs", i.Mounts[0].Fstype)
				s.Equal(uint64(500), i.Mounts[0].Total)
			},
		},
		{
			name:    "darwin: gopsutil error wrapped and returned",
			variant: "darwin",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return nil, errors.New("getfsstat failed")
			},
			usageFn: darwinUsage,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer filesystem.SetPartitionsFn(tt.partitionsFn)()
			defer filesystem.SetUsageFn(tt.usageFn)()
			var c filesystem.Collector
			switch tt.variant {
			case "linux":
				c = &filesystem.Linux{FS: tt.fs, Exec: tt.exec(s.T())}
			case "darwin":
				c = &filesystem.Darwin{}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*filesystem.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
