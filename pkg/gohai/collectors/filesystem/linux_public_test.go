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

	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
)

type FilesystemLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemLinuxPublicTestSuite))
}

func (s *FilesystemLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		mounts  []filesystem.Mount
		fnErr   error
		wantErr bool
		want    filesystem.Info
	}{
		{
			name: "root + /boot mounts",
			mounts: []filesystem.Mount{
				{
					Device:      "/dev/sda1",
					Mountpoint:  "/",
					Fstype:      "ext4",
					Total:       100,
					Used:        50,
					Free:        50,
					UsedPercent: 50,
				},
				{Device: "/dev/sda2", Mountpoint: "/boot", Fstype: "ext4"},
			},
			want: filesystem.Info{Mounts: []filesystem.Mount{
				{
					Device:      "/dev/sda1",
					Mountpoint:  "/",
					Fstype:      "ext4",
					Total:       100,
					Used:        50,
					Free:        50,
					UsedPercent: 50,
				},
				{Device: "/dev/sda2", Mountpoint: "/boot", Fstype: "ext4"},
			}},
		},
		{
			name:   "empty mount list",
			mounts: []filesystem.Mount{},
			want:   filesystem.Info{Mounts: []filesystem.Mount{}},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fnErr:   errors.New("mounts error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &filesystem.Linux{
				MountsFn: func(context.Context) ([]filesystem.Mount, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.mounts, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*filesystem.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
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
