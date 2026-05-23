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

package ipc_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
)

var (
	_ collector.Collector = (*ipc.Linux)(nil)
	_ collector.Collector = (*ipc.Darwin)(nil)
)

type IPCPublicTestSuite struct {
	suite.Suite
}

func TestIPCPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(IPCPublicTestSuite))
}

func (s *IPCPublicTestSuite) TestNew() {
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
			c := ipc.New()
			s.Equal("ipc", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*ipc.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*ipc.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *IPCPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() *memfs.MemFS
		wantNil bool
		want    *ipc.Info
	}{
		{
			name:    "linux: all sysctl files present",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc/sys/kernel", 0o755)
				_ = f.WriteFile(
					"/proc/sys/kernel/sem",
					[]byte("250\t32000\t32\t128\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile("/proc/sys/kernel/msgmnb", []byte("65536\n"), fs.FileMode(0o444))
				_ = f.WriteFile("/proc/sys/kernel/msgmni", []byte("32000\n"), fs.FileMode(0o444))
				_ = f.WriteFile("/proc/sys/kernel/msgmax", []byte("8192\n"), fs.FileMode(0o444))
				_ = f.WriteFile(
					"/proc/sys/kernel/shmall",
					[]byte("18446744073692774399\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/proc/sys/kernel/shmmax",
					[]byte("18446744073692774399\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile("/proc/sys/kernel/shmmni", []byte("4096\n"), fs.FileMode(0o444))
				return f
			},
			want: &ipc.Info{
				Sem: ipc.SemLimits{SEMMSL: "250", SEMMNS: "32000", SEMOPM: "32", SEMMNI: "128"},
				Msg: ipc.MsgLimits{MSGMNB: "65536", MSGMNI: "32000", MSGMAX: "8192"},
				Shm: ipc.ShmLimits{
					SHMALL: "18446744073692774399",
					SHMMAX: "18446744073692774399",
					SHMMNI: "4096",
				},
			},
		},
		{
			name:    "linux: missing sysctl files yield empty strings",
			variant: "linux",
			setupFS: func() *memfs.MemFS { return memfs.New() },
			want: &ipc.Info{
				Sem: ipc.SemLimits{},
				Msg: ipc.MsgLimits{},
				Shm: ipc.ShmLimits{},
			},
		},
		{
			name:    "linux: sem file with fewer than 4 fields is partial",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc/sys/kernel", 0o755)
				_ = f.WriteFile("/proc/sys/kernel/sem", []byte("250\t32000\n"), fs.FileMode(0o444))
				return f
			},
			want: &ipc.Info{
				Sem: ipc.SemLimits{SEMMSL: "250", SEMMNS: "32000"},
				Msg: ipc.MsgLimits{},
				Shm: ipc.ShmLimits{},
			},
		},
		{
			name:    "linux: sem file with only one field",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc/sys/kernel", 0o755)
				_ = f.WriteFile("/proc/sys/kernel/sem", []byte("250\n"), fs.FileMode(0o444))
				return f
			},
			want: &ipc.Info{
				Sem: ipc.SemLimits{SEMMSL: "250"},
				Msg: ipc.MsgLimits{},
				Shm: ipc.ShmLimits{},
			},
		},
		{
			name:    "linux: sem file with three fields",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/proc/sys/kernel", 0o755)
				_ = f.WriteFile(
					"/proc/sys/kernel/sem",
					[]byte("250 32000 32\n"),
					fs.FileMode(0o444),
				)
				return f
			},
			want: &ipc.Info{
				Sem: ipc.SemLimits{SEMMSL: "250", SEMMNS: "32000", SEMOPM: "32"},
				Msg: ipc.MsgLimits{},
				Shm: ipc.ShmLimits{},
			},
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c ipc.Collector
			switch tt.variant {
			case "linux":
				c = &ipc.Linux{FS: tt.setupFS()}
			case "darwin":
				c = ipc.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*ipc.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info)
		})
	}
}
