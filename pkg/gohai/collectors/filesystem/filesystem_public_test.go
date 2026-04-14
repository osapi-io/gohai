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
	return m
}

// lsblkExec returns a mock Executor that returns canned lsblk JSON.
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
			return &gpdisk.UsageStat{Total: 100, Used: 50, Free: 50, UsedPercent: 50}, nil
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
				c = &filesystem.Linux{Exec: tt.exec(s.T())}
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
