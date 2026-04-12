//go:build darwin

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

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
)

type FilesystemDarwinPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemDarwinPublicTestSuite))
}

func (s *FilesystemDarwinPublicTestSuite) TestCollectFromGopsutil() {
	twoParts := func(_ context.Context, _ bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4", Opts: []string{"rw"}},
			{Device: "tmpfs", Mountpoint: "/tmp", Fstype: "tmpfs"},
		}, nil
	}
	okUsage := func(_ context.Context, mp string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        mp,
			Total:       1000,
			Used:        300,
			Free:        700,
			UsedPercent: 30,
			InodesTotal: 1000,
			InodesUsed:  50,
			InodesFree:  950,
		}, nil
	}
	errUsage := func(_ context.Context, _ string) (*disk.UsageStat, error) {
		return nil, errors.New("no usage")
	}

	tests := []struct {
		name     string
		partFn   func(context.Context, bool) ([]disk.PartitionStat, error)
		usageFn  func(context.Context, string) (*disk.UsageStat, error)
		wantErr  bool
		wantLen  int
		wantUsed uint64
	}{
		{"happy path", twoParts, okUsage, false, 2, 300},
		{"usage error leaves zero values", twoParts, errUsage, false, 2, 0},
		{
			"partitions error",
			func(_ context.Context, _ bool) ([]disk.PartitionStat, error) { return nil, errors.New("boom") },
			okUsage,
			true,
			0,
			0,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := filesystem.CollectFromGopsutil(context.Background(), tt.partFn, tt.usageFn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*filesystem.Info)
			s.Require().True(ok)
			s.Len(info.Mounts, tt.wantLen)
			if tt.wantLen > 0 {
				s.Equal(tt.wantUsed, info.Mounts[0].Used)
			}
		})
	}
}

func (s *FilesystemDarwinPublicTestSuite) TestCollectDefault() {
	got, err := filesystem.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*filesystem.Info)
	s.True(ok)
}
