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
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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
) source.Runner {
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
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
) source.Runner {
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
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
) source.Runner {
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
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
		name         string
		detect       string
		validateFunc func(filesystem.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c filesystem.Collector) {
				_, ok := c.(*filesystem.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c filesystem.Collector) {
				_, ok := c.(*filesystem.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c filesystem.Collector) {
				_, ok := c.(*filesystem.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c filesystem.Collector) {
				_, ok := c.(*filesystem.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c filesystem.Collector) {
				_, ok := c.(*filesystem.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := filesystem.New()
			s.Equal("filesystem", c.Name())
			s.Equal("hardware", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *FilesystemPublicTestSuite) TestCollect() {
	const dfHead = "Filesystem 1024-blocks Used Available Capacity Mounted on\n"
	baseMounts := "/dev/sda1 / ext4 rw,relatime 0 0\n/dev/sda2 /boot ext4 rw 0 0\nshort line\n"
	baseDF := map[string]string{
		"df -akP": dfHead + "/dev/sda1 100 50 50 50% /\n/dev/sda2 10 0 10 0% /boot\n",
		"df -aiP": "Filesystem Inodes IUsed IFree IUse% Mounted on\n/dev/sda1 1000 250 750 25% /\n" +
			"short /\n/dev/sda2 a b c d% /boot\n",
	}
	btrfsMount := "/dev/sda1 / btrfs rw 0 0\n"
	darwinOut := map[string]string{
		"mount": "/dev/disk3s1 on / (apfs, local, journaled)\nnot a mount line\n",
		"df -ki": "Filesystem 1024-blocks Used Available Capacity iused ifree %iused Mounted on\n" +
			"/dev/disk3s1 500 200 300 40% 10 90 10% /\n",
	}

	tests := []struct {
		name         string
		variant      string
		mounts       string            // /proc/mounts on Linux
		out          map[string]string // df on Linux; mount and df on macOS
		errs         map[string]error
		exec         func(*testing.T) source.Runner
		fs           avfs.VFS // optional; nil → btrfs sysfs reads short-circuit
		validateFunc func(any, error)
	}{
		{
			name:    "linux: no lsblk, mounts unchanged no unmounted",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec:    func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
				s.Equal(uint64(1000), i.Mounts[0].InodesTotal)
				s.Equal(float64(25), i.Mounts[0].InodesUsedPercent)
			},
		},
		{
			name:    "linux: lsblk merges uuid/label into matching mounts",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":null,"children":[
						{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"part-1","partlabel":""},
						{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"EFI","mountpoint":"/boot","partuuid":"part-2","partlabel":"EFI"}
					]}
				]}`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
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
			name:    "linux: lsblk unmounted entry surfaces as Unmounted",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":null,"children":[
						{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"","partlabel":""},
						{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"","mountpoint":"/boot","partuuid":"","partlabel":""},
						{"name":"sdb1","fstype":"crypto_LUKS","uuid":"luks-uuid","label":"data","mountpoint":"","partuuid":"","partlabel":""}
					]}
				]}`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Len(i.Mounts, 2)
				s.Len(i.Unmounted, 1)
				s.Equal("/dev/sdb1", i.Unmounted[0].Device)
				s.Equal("crypto_LUKS", i.Unmounted[0].Type)
				s.Equal("luks-uuid", i.Unmounted[0].UUID)
				s.Equal("data", i.Unmounted[0].Label)
			},
		},
		{
			name:    "linux: lsblk node with empty fstype ignored (raw disk)",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda","fstype":"","children":[
						{"name":"sda1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/","partuuid":"","partlabel":""}
					]}
				]}`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Equal("u", i.Mounts[0].UUID)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:    "linux: lsblk entry with mountpoint but not in /proc/mounts is ignored",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sdc1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/mnt/foo","partuuid":"","partlabel":""}
				]}`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:    "linux: malformed lsblk json, extension silently skipped",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec:    func(t *testing.T) source.Runner { return lsblkExec(t, `not json`) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
			},
		},
		{
			name:    "linux: nil Exec, extension skipped cleanly",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:    "linux: no /proc/mounts",
			variant: "linux",
			out:     baseDF,
			exec:    func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: df fails, mounts kept without usage",
			variant: "linux",
			mounts:  baseMounts,
			exec:    func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Len(i.Mounts, 2)
				s.Zero(i.Mounts[0].Total)
			},
		},
		{
			name:    "linux: an NFS server that does not answer costs only its own mount",
			variant: "linux",
			mounts: "/dev/sda1 / ext4 rw 0 0\nsrv:/export /mnt/nfs nfs4 rw 0 0\n" +
				"/dev/sdb1 /data ext4 rw 0 0\n",
			out: map[string]string{
				// One df of every mount stops at the hung server, so the
				// mounts after it get no row.
				"df -akP": dfHead + "/dev/sda1 100 50 50 50% /\n",
				"df -aiP": "Filesystem Inodes IUsed IFree IUse% Mounted on\n" +
					"/dev/sda1 1000 250 750 25% /\n",
				"df -akP -- / /data": dfHead + "/dev/sda1 100 50 50 50% /\n" +
					"/dev/sdb1 200 20 180 10% /data\n",
				"df -aiP -- / /data": "Filesystem Inodes IUsed IFree IUse% Mounted on\n" +
					"/dev/sda1 1000 250 750 25% /\n/dev/sdb1 2000 200 1800 10% /data\n",
			},
			errs: map[string]error{
				"df -akP":             errors.New("exec df: stopped after 30s"),
				"df -aiP":             errors.New("exec df: stopped after 30s"),
				"df -akP -- /mnt/nfs": errors.New("exec df: stopped after 30s"),
				"df -aiP -- /mnt/nfs": errors.New("exec df: stopped after 30s"),
			},
			exec: func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 3)
				s.Equal(uint64(100*1024), i.Mounts[0].Total)
				s.Zero(i.Mounts[1].Total)
				s.Equal("/data", i.Mounts[2].Mountpoint)
				s.Equal(uint64(200*1024), i.Mounts[2].Total)
				s.Equal(uint64(2000), i.Mounts[2].InodesTotal)
			},
		},
		{
			name:    "linux: an NFS mount that answers gets its sizes",
			variant: "linux",
			mounts:  "/dev/sda1 / ext4 rw 0 0\nsrv:/export /mnt/nfs nfs4 rw 0 0\n",
			out: map[string]string{
				"df -akP -- /": dfHead + "/dev/sda1 100 50 50 50% /\n",
				"df -aiP -- /": "Filesystem Inodes IUsed IFree IUse% Mounted on\n" +
					"/dev/sda1 1000 250 750 25% /\n",
				"df -akP -- /mnt/nfs": dfHead + "srv:/export 400 100 300 25% /mnt/nfs\n",
				"df -aiP -- /mnt/nfs": "Filesystem Inodes IUsed IFree IUse% Mounted on\n" +
					"srv:/export 40 10 30 25% /mnt/nfs\n",
			},
			exec: func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 2)
				s.Equal(uint64(400*1024), i.Mounts[1].Total)
				s.Equal(uint64(40), i.Mounts[1].InodesTotal)
			},
		},
		{
			name:    "linux: only network mounts",
			variant: "linux",
			mounts:  "srv:/export /mnt/nfs nfs rw 0 0\n",
			out: map[string]string{
				"df -akP -- /mnt/nfs": dfHead + "srv:/export 400 100 300 25% /mnt/nfs\n",
			},
			exec: func(t *testing.T) source.Runner { return noLsblkExec(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 1)
				s.Equal(uint64(400*1024), i.Mounts[0].Total)
			},
		},
		{
			name:    "linux: ZFS datasets parsed with pool + nested + mountpoints",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
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
			name:    "linux: zfs binary present but empty output yields no datasets",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkZFSExec(t, "", "")
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Empty(i.ZFSDatasets)
			},
		},
		{
			name:    "linux: zfs malformed line skipped, well-formed line parsed",
			variant: "linux",
			mounts:  baseMounts,
			out:     baseDF,
			exec: func(t *testing.T) source.Runner {
				return lsblkZFSExec(
					t,
					"",
					"garbage\nonly two\tfields\nname\tprop\tval\tsource\n\t\t\t\n",
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.ZFSDatasets, 1)
				s.Equal("name", i.ZFSDatasets[0].Name)
				s.Equal("val", i.ZFSDatasets[0].Properties["prop"].Value)
			},
		},
		{
			name:    "linux: btrfs mount enriched with allocation + raid profile",
			variant: "linux",
			mounts:  btrfsMount,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"abc-123","mountpoint":"/"}
				]}`)
			},
			fs: btrfsFS(s, "abc-123", map[string]btrfsBgFixture{
				"data":     {raid: "raid1", total: 1073741824, used: 524288000},
				"metadata": {raid: "raid1", total: 268435456, used: 134217728},
				"system":   {raid: "raid1", total: 33554432, used: 16384},
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 1)
				m := i.Mounts[0]
				s.Equal("btrfs", m.Type)
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
			mounts:  btrfsMount,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"single-fs","mountpoint":"/"}
				]}`)
			},
			fs: btrfsFS(s, "single-fs", map[string]btrfsBgFixture{
				"data":     {raid: "single", totalRaw: "not-a-number"},
				"metadata": {raid: "single", total: 1024, used: 256},
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
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
			mounts:  btrfsMount,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"missing","mountpoint":"/"}
				]}`)
			},
			fs: memfs.New(),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:    "linux: btrfs bg dirs exist but with no profile + missing files",
			variant: "linux",
			mounts:  btrfsMount,
			exec: func(t *testing.T) source.Runner {
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
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
			mounts:  btrfsMount,
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"btrfs","uuid":"empty-bg","mountpoint":"/"}
				]}`)
			},
			fs: func() avfs.VFS {
				fs := memfs.New()
				s.Require().NoError(fs.MkdirAll("/sys/fs/btrfs/empty-bg/allocation", 0o755))
				return fs
			}(),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:    "linux: non-btrfs mount left alone even with FS configured",
			variant: "linux",
			mounts:  "/dev/sda1 / ext4 rw 0 0\n",
			exec: func(t *testing.T) source.Runner {
				return lsblkExec(t, `{"blockdevices":[
					{"name":"sda1","fstype":"ext4","uuid":"ext-uuid","mountpoint":"/"}
				]}`)
			},
			fs: memfs.New(),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Nil(i.Mounts[0].Btrfs)
			},
		},
		{
			name:    "darwin: APFS root populated",
			variant: "darwin",
			out:     darwinOut,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 1)
				s.Equal("/dev/disk3s1", i.Mounts[0].Device)
				s.Equal("apfs", i.Mounts[0].Type)
				s.Equal(uint64(500*1024), i.Mounts[0].Total)
				s.Equal([]string{"local", "journaled"}, i.Mounts[0].Opts)
				s.Equal(uint64(100), i.Mounts[0].InodesTotal)
			},
		},
		{
			name:    "darwin: an SMB server that does not answer costs only its own mount",
			variant: "darwin",
			out: map[string]string{
				"mount": "/dev/disk3s1 on / (apfs, local, journaled)\n" +
					"//u@srv/share on /Volumes/share (smbfs, nodev, nosuid)\n" +
					"/dev/disk4s1 on /Volumes/ext (apfs, local)\n",
				"df -ki": "Filesystem 1024-blocks Used Available Capacity iused ifree %iused Mounted on\n" +
					"/dev/disk3s1 500 200 300 40% 10 90 10% /\n",
				"df -ki -- / /Volumes/ext": "Filesystem 1024-blocks Used Available Capacity iused ifree %iused Mounted on\n" +
					"/dev/disk3s1 500 200 300 40% 10 90 10% /\n" +
					"/dev/disk4s1 800 100 700 13% 5 95 5% /Volumes/ext\n",
			},
			errs: map[string]error{
				"df -ki":                   errors.New("exec df: stopped after 30s"),
				"df -ki -- /Volumes/share": errors.New("exec df: stopped after 30s"),
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*filesystem.Info)
				s.Require().True(ok)
				s.Require().Len(i.Mounts, 3)
				s.Zero(i.Mounts[1].Total)
				s.Equal(uint64(800*1024), i.Mounts[2].Total)
			},
		},
		{
			name:    "darwin: mount fails",
			variant: "darwin",
			errs:    map[string]error{"mount": errors.New("mount failed")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c filesystem.Collector
			switch tt.variant {
			case "linux":
				fs := tt.fs
				if fs == nil {
					fs = memfs.New()
				}
				if tt.mounts != "" {
					s.Require().NoError(fs.MkdirAll("/proc", 0o755))
					s.Require().NoError(fs.WriteFile("/proc/mounts", []byte(tt.mounts), 0o644))
				}
				var exec source.Runner
				if tt.exec != nil {
					if mock, ok := tt.exec(s.T()).(*sourcemocks.MockRunner); ok && mock != nil {
						// The row's mock answers lsblk and zfs; df answers
						// come from the row's out and errs.
						df := sourcetest.Commands{Out: tt.out, Err: tt.errs}.Mock(s.T())
						mock.EXPECT().
							Execute(gomock.Any(), "df", gomock.Any()).
							DoAndReturn(df.Execute).
							AnyTimes()
						exec = mock
					}
				}
				c = &filesystem.Linux{FS: fs, Exec: exec}
			case "darwin":
				c = &filesystem.Darwin{
					Exec: sourcetest.Commands{Out: tt.out, Err: tt.errs}.Mock(s.T()),
				}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
