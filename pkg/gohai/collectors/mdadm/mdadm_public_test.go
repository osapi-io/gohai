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

package mdadm_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
)

var (
	_ collector.Collector = (*mdadm.Linux)(nil)
	_ collector.Collector = (*mdadm.Darwin)(nil)
)

// errorFS forces a non-ErrNotExist read error from ReadFile.
type errorFS struct {
	avfs.VFS
}

func (errorFS) ReadFile(
	string,
) ([]byte, error) {
	return nil, errors.New("permission denied")
}

// mdstatRaid1 is a minimal /proc/mdstat with one RAID-1 array.
var mdstatRaid1 = []byte(
	"Personalities : [raid1]\n" +
		"md0 : active raid1 sda1[0] sdb1[1]\n" +
		"      976760832 blocks super 1.2 [2/2] [UU]\n" +
		"\n" +
		"unused devices: <none>\n",
)

// mdstatWithSpare has one RAID-5 array with an active spare.
var mdstatWithSpare = []byte(
	"Personalities : [raid5]\n" +
		"md1 : active raid5 sda1[0] sdb1[1] sdc1[2](S)\n" +
		"      976760832 blocks super 1.2 [3/3] [UUU]\n" +
		"\n" +
		"unused devices: <none>\n",
)

// mdstatTwo has two arrays for sort-order testing.
var mdstatTwo = []byte(
	"Personalities : [raid1]\n" +
		"md1 : active raid1 sdb1[0] sdc1[1]\n" +
		"      512000000 blocks [2/2] [UU]\n" +
		"md0 : active raid1 sda1[0] sdd1[1]\n" +
		"      512000000 blocks [2/2] [UU]\n" +
		"\n" +
		"unused devices: <none>\n",
)

// detailRaid1 is sample `mdadm --detail` output for md0.
var detailRaid1 = []byte(
	"/dev/md0:\n" +
		"           Version : 1.2\n" +
		"     Creation Time : Mon Jan  1 00:00:00 2024\n" +
		"        Raid Level : raid1\n" +
		"        Array Size : 976760832 (931.39 GiB 1000.07 GB)\n" +
		"     Used Dev Size : 976760832 (931.39 GiB 1000.07 GB)\n" +
		"      Raid Devices : 2\n" +
		"     Total Devices : 2\n" +
		"       Persistence : Superblock is persistent\n" +
		"             State : clean\n" +
		"    Active Devices : 2\n" +
		"   Working Devices : 2\n" +
		"    Failed Devices : 0\n" +
		"     Spare Devices : 0\n" +
		"              UUID : a5d3:1234:dead:beef\n",
)

type MdadmPublicTestSuite struct {
	suite.Suite
}

func TestMdadmPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MdadmPublicTestSuite))
}

func (s *MdadmPublicTestSuite) TestNew() {
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
			c := mdadm.New()
			s.Equal("mdadm", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*mdadm.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*mdadm.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *MdadmPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() avfs.VFS
		// setupEx returns an executor.Executor to inject. Return nil to test
		// the no-executor path (Exec field left nil on the Linux struct).
		setupEx func(*testing.T) *execmocks.MockExecutor
		wantErr bool
		wantNil bool
		want    []mdadm.Array
	}{
		{
			name:    "linux: raid1 array enriched by mdadm --detail",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatRaid1, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *execmocks.MockExecutor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return(detailRaid1, nil)
				return m
			},
			want: []mdadm.Array{
				{
					Device:      "md0",
					Level:       "raid1",
					State:       "clean",
					UUID:        "a5d3:1234:dead:beef",
					ActiveDisks: 2,
					TotalDisks:  2,
					SpareDisks:  0,
					Members:     []string{"sda1", "sdb1"},
					Spares:      []string{},
				},
			},
		},
		{
			name:    "linux: array with spare member",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatWithSpare, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *execmocks.MockExecutor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md1").
					Return(nil, errors.New("mdadm not found"))
				return m
			},
			want: []mdadm.Array{
				{
					Device:  "md1",
					Members: []string{"sda1", "sdb1"},
					Spares:  []string{"sdc1"},
				},
			},
		},
		{
			name:    "linux: two arrays returned in sorted order",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatTwo, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *execmocks.MockExecutor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "mdadm", "--detail", gomock.Any()).
					Return(nil, errors.New("not found")).AnyTimes()
				return m
			},
			want: []mdadm.Array{
				{Device: "md0", Members: []string{"sda1", "sdd1"}, Spares: []string{}},
				{Device: "md1", Members: []string{"sdb1", "sdc1"}, Spares: []string{}},
			},
		},
		{
			name:    "linux: no arrays in /proc/mdstat returns empty list",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat",
					[]byte("Personalities : []\nunused devices: <none>\n"),
					fs.FileMode(0o444))
				return f
			},
			setupEx: func(_ *testing.T) *execmocks.MockExecutor { return nil },
			want:    []mdadm.Array{},
		},
		{
			name:    "linux: /proc/mdstat absent returns empty list",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			setupEx: func(_ *testing.T) *execmocks.MockExecutor { return nil },
			want:    []mdadm.Array{},
		},
		{
			name:    "linux: nil Exec skips mdadm --detail enrichment",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatRaid1, fs.FileMode(0o444))
				return f
			},
			setupEx: func(_ *testing.T) *execmocks.MockExecutor { return nil },
			want: []mdadm.Array{
				{Device: "md0", Members: []string{"sda1", "sdb1"}, Spares: []string{}},
			},
		},
		{
			name:    "linux: read error on /proc/mdstat propagates",
			variant: "linux",
			setupFS: func() avfs.VFS { return errorFS{memfs.New()} },
			setupEx: func(_ *testing.T) *execmocks.MockExecutor { return nil },
			wantErr: true,
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c mdadm.Collector
			switch tt.variant {
			case "linux":
				lc := &mdadm.Linux{FS: tt.setupFS()}
				if tt.setupEx != nil {
					if m := tt.setupEx(s.T()); m != nil {
						lc.Exec = m
					}
				}
				c = lc
			case "darwin":
				c = mdadm.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*mdadm.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Arrays)
		})
	}
}
