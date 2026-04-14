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

package kernelmodules_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	kernelmodules "github.com/osapi-io/gohai/pkg/gohai/collectors/kernel_modules"
)

var (
	_ collector.Collector = (*kernelmodules.Linux)(nil)
	_ collector.Collector = (*kernelmodules.Darwin)(nil)
)

type KernelModulesPublicTestSuite struct {
	suite.Suite
}

func TestKernelModulesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KernelModulesPublicTestSuite))
}

// linuxFS returns a memfs populated with the provided (path → contents) map.
func linuxFS(
	s *KernelModulesPublicTestSuite,
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	for path, content := range files {
		s.Require().NoError(fs.MkdirAll(dirOf(path), 0o755))
		s.Require().NoError(fs.WriteFile(path, []byte(content), 0o644))
	}
	return fs
}

func dirOf(
	p string,
) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "/"
}

// kextstatExec returns a mock that maps the kextstat call to the given
// (output, error) pair.
func kextstatExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "kextstat", "-k", "-l").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *KernelModulesPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := kernelmodules.New()
			s.Equal("kernel_modules", c.Name())
			s.Equal("system", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*kernelmodules.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*kernelmodules.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *KernelModulesPublicTestSuite) TestCollect() {
	kextstatOut := []byte(
		`Index Refs Address            Size       Wired      Name (Version) <Linked Against>
    1    0 0xffffff7f80000000 0x8a8      0xa8       com.apple.iokit.IOPCIFamily (2.9)
    2   12 0xffffff7f80001000 0x1000     0x100      com.apple.driver.AppleACPIPlatform (6.1)
`)

	tests := []struct {
		name     string
		variant  string
		fs       avfs.VFS
		exec     func(*testing.T) executor.Executor
		validate func(*kernelmodules.Info)
	}{
		{
			name:    "linux: canonical modules with versions",
			variant: "linux",
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "nf_tables 217088 25 rfkill,nf_conntrack - Live 0x0000000000000000\n" +
					"ipv6 557056 24 - Live 0x0000000000000000\n",
				"/sys/module/nf_tables/version": "1.2.3\n",
				"/sys/module/ipv6/version":      "\n",
			}),
			validate: func(i *kernelmodules.Info) {
				s.Len(i.Modules, 2)
				s.Equal(uint64(217088), i.Modules["nf_tables"].Size)
				s.Equal(25, i.Modules["nf_tables"].RefCount)
				s.Equal("1.2.3", i.Modules["nf_tables"].Version)
				s.Equal(uint64(557056), i.Modules["ipv6"].Size)
				s.Empty(i.Modules["ipv6"].Version)
			},
		},
		{
			name:    "linux: missing /proc/modules yields empty Info",
			variant: "linux",
			fs:      linuxFS(s, nil),
			validate: func(i *kernelmodules.Info) {
				s.Nil(i.Modules)
			},
		},
		{
			name:    "linux: malformed line skipped, remaining rows parsed",
			variant: "linux",
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "short\nvalid_mod 1024 3 - Live 0x0\n",
			}),
			validate: func(i *kernelmodules.Info) {
				s.Len(i.Modules, 1)
				s.Equal(uint64(1024), i.Modules["valid_mod"].Size)
				s.Equal(3, i.Modules["valid_mod"].RefCount)
			},
		},
		{
			name:    "linux: unparseable size/refcount leaves field zero",
			variant: "linux",
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "broken abc xyz - Live 0x0\n",
			}),
			validate: func(i *kernelmodules.Info) {
				s.Contains(i.Modules, "broken")
				s.Equal(uint64(0), i.Modules["broken"].Size)
				s.Equal(0, i.Modules["broken"].RefCount)
			},
		},
		{
			name:    "darwin: kextstat output parsed into modules",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return kextstatExec(t, kextstatOut, nil)
			},
			validate: func(i *kernelmodules.Info) {
				s.Len(i.Modules, 2)
				s.Equal("2.9", i.Modules["com.apple.iokit.IOPCIFamily"].Version)
				s.Equal(uint64(0x8a8), i.Modules["com.apple.iokit.IOPCIFamily"].Size)
				s.Equal(12, i.Modules["com.apple.driver.AppleACPIPlatform"].RefCount)
			},
		},
		{
			name:    "darwin: kextstat error yields empty modules",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return kextstatExec(t, nil, errors.New("not found"))
			},
			validate: func(i *kernelmodules.Info) {
				s.Empty(i.Modules)
			},
		},
		{
			name:    "darwin: unparseable line skipped leaves empty",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return kextstatExec(t, []byte("garbage line that cannot match\n"), nil)
			},
			validate: func(i *kernelmodules.Info) {
				s.Empty(i.Modules)
			},
		},
		{
			name:    "darwin: nil Exec yields empty modules",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			validate: func(i *kernelmodules.Info) {
				s.Empty(i.Modules)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c kernelmodules.Collector
			switch tt.variant {
			case "linux":
				c = &kernelmodules.Linux{FS: tt.fs}
			case "darwin":
				d := &kernelmodules.Darwin{}
				if tt.exec != nil {
					d.Exec = tt.exec(s.T())
				}
				c = d
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*kernelmodules.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
