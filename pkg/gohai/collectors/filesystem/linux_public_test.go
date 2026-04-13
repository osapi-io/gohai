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

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
)

type FilesystemLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemLinuxPublicTestSuite))
}

// noLsblkExec returns a mock Executor that errors on every call —
// simulates lsblk absent from PATH or a minimal container.
func noLsblkExec(
	s *FilesystemLinuxPublicTestSuite,
) executor.Executor {
	ctrl := gomock.NewController(s.T())
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsblk", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("not found")).
		AnyTimes()
	return m
}

// lsblkExec returns a mock Executor that returns canned lsblk JSON.
func lsblkExec(
	s *FilesystemLinuxPublicTestSuite,
	out string,
) executor.Executor {
	ctrl := gomock.NewController(s.T())
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsblk", gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]byte(out), nil).
		AnyTimes()
	return m
}

func (s *FilesystemLinuxPublicTestSuite) TestCollect() {
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

	tests := []struct {
		name         string
		partitionsFn func(context.Context, bool) ([]gpdisk.PartitionStat, error)
		usageFn      func(context.Context, string) (*gpdisk.UsageStat, error)
		exec         executor.Executor
		wantErr      bool
		validate     func(*filesystem.Info)
	}{
		{
			name:         "no lsblk: mounts unchanged, no unmounted",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         noLsblkExec(s),
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
			},
		},
		{
			name:         "lsblk merges uuid/label into matching mounts",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: lsblkExec(s, `{"blockdevices":[
				{"name":"sda","fstype":null,"children":[
					{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"part-1","partlabel":""},
					{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"EFI","mountpoint":"/boot","partuuid":"part-2","partlabel":"EFI"}
				]}
			]}`),
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
			name:         "lsblk unmounted entry surfaces as Unmounted",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: lsblkExec(s, `{"blockdevices":[
				{"name":"sda","fstype":null,"children":[
					{"name":"sda1","fstype":"ext4","uuid":"root-uuid","label":"","mountpoint":"/","partuuid":"","partlabel":""},
					{"name":"sda2","fstype":"ext4","uuid":"boot-uuid","label":"","mountpoint":"/boot","partuuid":"","partlabel":""},
					{"name":"sdb1","fstype":"crypto_LUKS","uuid":"luks-uuid","label":"data","mountpoint":"","partuuid":"","partlabel":""}
				]}
			]}`),
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
			name:         "lsblk node with empty fstype ignored (raw disk)",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: lsblkExec(s, `{"blockdevices":[
				{"name":"sda","fstype":"","children":[
					{"name":"sda1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/","partuuid":"","partlabel":""}
				]}
			]}`),
			validate: func(i *filesystem.Info) {
				s.Equal("u", i.Mounts[0].UUID)
				s.Empty(i.Unmounted)
			},
		},
		{
			name:         "lsblk entry with mountpoint but not in gopsutil mounts is ignored (not unmounted)",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec: lsblkExec(s, `{"blockdevices":[
				{"name":"sdc1","fstype":"ext4","uuid":"u","label":"","mountpoint":"/mnt/foo","partuuid":"","partlabel":""}
			]}`),
			validate: func(i *filesystem.Info) {
				s.Empty(i.Unmounted)
			},
		},
		{
			name:         "malformed lsblk json: extension silently skipped",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         lsblkExec(s, `not json`),
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
				s.Equal("", i.Mounts[0].UUID)
			},
		},
		{
			name:         "nil Exec: extension skipped cleanly",
			partitionsFn: okPartitions,
			usageFn:      okUsage,
			exec:         nil,
			validate: func(i *filesystem.Info) {
				s.Len(i.Mounts, 2)
				s.Empty(i.Unmounted)
			},
		},
		{
			name: "gopsutil partitions error propagated",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return nil, errors.New("partitions error")
			},
			usageFn: okUsage,
			exec:    noLsblkExec(s),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreP := filesystem.SetPartitionsFn(tt.partitionsFn)
			defer restoreP()
			restoreU := filesystem.SetUsageFn(tt.usageFn)
			defer restoreU()

			c := &filesystem.Linux{Exec: tt.exec}
			got, err := c.Collect(context.Background())
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

func (s *FilesystemLinuxPublicTestSuite) TestListMounts() {
	okParts := []gpdisk.PartitionStat{
		{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4", Opts: []string{"rw"}},
	}

	tests := []struct {
		name         string
		partitionsFn func(context.Context, bool) ([]gpdisk.PartitionStat, error)
		usageFn      func(context.Context, string) (*gpdisk.UsageStat, error)
		wantErr      bool
		wantLen      int
		wantTotal    uint64
	}{
		{
			name: "success populates usage",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return okParts, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return &gpdisk.UsageStat{Total: 100, Used: 50, Free: 50}, nil
			},
			wantLen:   1,
			wantTotal: 100,
		},
		{
			name: "usage error keeps mount without usage",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return okParts, nil
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) {
				return nil, errors.New("usage failed")
			},
			wantLen:   1,
			wantTotal: 0,
		},
		{
			name: "partitions error propagated",
			partitionsFn: func(context.Context, bool) ([]gpdisk.PartitionStat, error) {
				return nil, errors.New("partitions failed")
			},
			usageFn: func(context.Context, string) (*gpdisk.UsageStat, error) { return nil, nil },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreP := filesystem.SetPartitionsFn(tt.partitionsFn)
			defer restoreP()
			restoreU := filesystem.SetUsageFn(tt.usageFn)
			defer restoreU()
			got, err := filesystem.ListMounts(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Len(got, tt.wantLen)
			s.Equal(tt.wantTotal, got[0].Total)
		})
	}
}
