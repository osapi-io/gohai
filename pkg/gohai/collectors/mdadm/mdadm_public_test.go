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
	"fmt"
	"io/fs"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*mdadm.Linux)(nil)
	_ collector.Collector = (*mdadm.Darwin)(nil)
)

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

// mdstatInactive is /proc/mdstat with an active md0 and an inactive md127,
// for which `mdadm --detail` exits non-zero.
var mdstatInactive = []byte(
	"Personalities : [raid1]\n" +
		"md0 : active raid1 sda1[0] sdb1[1]\n" +
		"      1046528 blocks super 1.2 [2/2] [UU]\n" +
		"\n" +
		"md127 : inactive sdc1[1](S)\n" +
		"      1046528 blocks super 1.2\n" +
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
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(mdadm.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c mdadm.Collector) {
				_, ok := c.(*mdadm.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c mdadm.Collector) {
				_, ok := c.(*mdadm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c mdadm.Collector) {
				_, ok := c.(*mdadm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c mdadm.Collector) {
				_, ok := c.(*mdadm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c mdadm.Collector) {
				_, ok := c.(*mdadm.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c mdadm.Collector) {
				// New picks the variant for the host the test runs on.
				_, darwin := c.(*mdadm.Darwin)
				s.Equal(runtime.GOOS == "darwin", darwin)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := mdadm.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = mdadm.New()
			}
			s.Equal("mdadm", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *MdadmPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() avfs.VFS
		// setupEx returns a source.Runner to inject. Return nil to test
		// the no-executor path (Exec field left nil on the Linux struct).
		setupEx func(*testing.T) *mocks.MockRunner
		// cancel runs Collect with a context that is already cancelled.
		cancel       bool
		validateFunc func(any, error)
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
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return(detailRaid1, nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{
					{
						Device:       "md0",
						Level:        "raid1",
						State:        "clean",
						UUID:         "a5d3:1234:dead:beef",
						ActiveDisks:  2,
						TotalDisks:   2,
						SpareDisks:   0,
						WorkingDisks: 2,
						Members:      []string{"sda1", "sdb1"},
						Spares:       []string{},
						Faulty:       []string{},
						Journal:      []string{},
					},
				}, info.Arrays)
			},
		},
		{
			name:    "linux: lines over 64 KiB in mdstat and --detail keep the lines after them",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				content := append([]byte("junk "+strings.Repeat("x", 70000)+"\n"), mdstatRaid1...)
				_ = f.WriteFile("/proc/mdstat", content, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				detail := append([]byte("Junk : "+strings.Repeat("x", 70000)+"\n"), detailRaid1...)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return(detail, nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Require().Len(info.Arrays, 1)
				s.Equal("md0", info.Arrays[0].Device)
				s.Equal("a5d3:1234:dead:beef", info.Arrays[0].UUID)
			},
		},
		{
			name:    "linux: a degraded raid1 keeps its whole state, and counts its failed disk",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatRaid1, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return([]byte(
						"/dev/md0:\n"+
							"        Raid Level : raid1\n"+
							"     Total Devices : 2\n"+
							"             State : clean, degraded \n"+
							"    Active Devices : 1\n"+
							"   Working Devices : 1\n"+
							"    Failed Devices : 1\n"+
							"     Spare Devices : 0\n"+
							"              UUID : a5d3:1234:dead:beef\n"+
							"       Array State : A. ('A' == active, '.' == missing)\n",
					), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Require().Len(info.Arrays, 1)
				s.Equal("clean, degraded", info.Arrays[0].State)
				s.Equal(1, info.Arrays[0].FailedDisks)
				s.Equal(1, info.Arrays[0].WorkingDisks)
				s.Equal(1, info.Arrays[0].ActiveDisks)
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
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md1").
					Return(nil, fmt.Errorf("exec mdadm: %w", exec.ErrNotFound))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{
					{
						Device:  "md1",
						Members: []string{"sda1", "sdb1"},
						Spares:  []string{"sdc1"},
						Faulty:  []string{},
						Journal: []string{},
					},
				}, info.Arrays)
			},
		},
		{
			name:    "linux: faulty and journal devices are not data members",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", []byte(
					"md2 : active raid5 sda1[0] sdb1[1](F) sdc1[2] nvme0n1p1[3](J)\n",
				), fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md2").
					Return(nil, fmt.Errorf("exec mdadm: %w", exec.ErrNotFound))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{
					{
						Device:  "md2",
						Members: []string{"sda1", "sdc1"},
						Spares:  []string{},
						Faulty:  []string{"sdb1"},
						Journal: []string{"nvme0n1p1"},
					},
				}, info.Arrays)
			},
		},
		{
			name:    "linux: a failed --detail on one inactive array keeps the others",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatInactive, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return(detailRaid1, nil)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md127").
					Return(nil, errors.New("exit status 1"))
				return m
			},
			validateFunc: func(got any, err error) {
				// The failed detail is the error, and the result is partial.
				s.EqualError(err, "mdadm --detail /dev/md127: exit status 1")
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Require().Len(info.Arrays, 2)
				s.Equal("md0", info.Arrays[0].Device)
				s.Equal("raid1", info.Arrays[0].Level)
				s.Empty(info.Arrays[0].DetailError)
				s.Equal("md127", info.Arrays[1].Device)
				s.Empty(info.Arrays[1].Level)
				s.Equal([]string{"sdc1"}, info.Arrays[1].Spares)
				s.Equal(
					"mdadm --detail /dev/md127: exit status 1",
					info.Arrays[1].DetailError,
				)
			},
		},
		{
			name:    "linux: a cancelled context during --detail is an error",
			variant: "linux",
			cancel:  true,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc", 0o755)
				_ = f.WriteFile("/proc/mdstat", mdstatRaid1, fs.FileMode(0o444))
				return f
			},
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "mdadm", "--detail", "/dev/md0").
					Return(nil, context.Canceled)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "mdadm --detail /dev/md0: context canceled")
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
			setupEx: func(t *testing.T) *mocks.MockRunner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "mdadm", "--detail", gomock.Any()).
					Return(nil, fmt.Errorf("exec mdadm: %w", exec.ErrNotFound)).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{
					{
						Device: "md0", Members: []string{"sda1", "sdd1"},
						Spares: []string{}, Faulty: []string{}, Journal: []string{},
					},
					{
						Device: "md1", Members: []string{"sdb1", "sdc1"},
						Spares: []string{}, Faulty: []string{}, Journal: []string{},
					},
				}, info.Arrays)
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
			setupEx: func(_ *testing.T) *mocks.MockRunner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{}, info.Arrays)
			},
		},
		{
			name:    "linux: /proc/mdstat absent returns empty list",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			setupEx: func(_ *testing.T) *mocks.MockRunner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{}, info.Arrays)
			},
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
			setupEx: func(_ *testing.T) *mocks.MockRunner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*mdadm.Info)
				s.Require().True(ok)
				s.Equal([]mdadm.Array{
					{
						Device: "md0", Members: []string{"sda1", "sdb1"},
						Spares: []string{}, Faulty: []string{}, Journal: []string{},
					},
				}, info.Arrays)
			},
		},
		{
			name:    "linux: read error on /proc/mdstat propagates",
			variant: "linux",
			setupFS: func() avfs.VFS { return sourcetest.FailRead(s.T(), memfs.New(), "/proc/mdstat") },
			setupEx: func(_ *testing.T) *mocks.MockRunner { return nil },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: unsupported",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
			},
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			tt.validateFunc(c.Collect(ctx, nil))
		})
	}
}
