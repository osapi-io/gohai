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

package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

// darwinKernelExec returns a MockExecutor that canned-answers the
// macOS kernel collector's two exec calls: sysctl for Rosetta
// detection, kextstat for the module list. Shared with the darwin
// test file (lives here per project convention).
func darwinKernelExec(
	t *testing.T,
	sysctlOut []byte, sysctlErr error,
	kextOut []byte, kextErr error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sysctl", "-n", "hw.optional.x86_64").
		Return(sysctlOut, sysctlErr).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "kextstat", "-k", "-l").
		Return(kextOut, kextErr).
		AnyTimes()
	return m
}

type KernelLinuxPublicTestSuite struct {
	suite.Suite
}

func TestKernelLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelLinuxPublicTestSuite))
}

// linuxFS returns a memfs populated with the provided
// (path → contents) map. Parent directories are created automatically.
func linuxFS(
	s *KernelLinuxPublicTestSuite,
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	for path, content := range files {
		s.Require().NoError(fs.MkdirAll(dirOf(path), 0o755))
		s.Require().NoError(fs.WriteFile(path, []byte(content), 0o644))
	}
	return fs
}

// dirOf returns the directory of a "/"-rooted path. Inline to avoid
// pulling path/filepath just for this test helper.
func dirOf(p string) string {
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

// fakeUtsname fills a unix.Utsname with name/release/version/machine.
// The field types differ across platforms ([65]byte on Linux,
// [256]byte on darwin); we write through a []byte slice view so the
// helper compiles on both.
func fakeUtsname(
	name, release, version, machine string,
) func(*unix.Utsname) error {
	return func(u *unix.Utsname) error {
		copyBytes(u.Sysname[:], name)
		copyBytes(u.Release[:], release)
		copyBytes(u.Version[:], version)
		copyBytes(u.Machine[:], machine)
		return nil
	}
}

func copyBytes(dst []byte, src string) {
	for i := range dst {
		if i < len(src) {
			dst[i] = src[i]
		} else {
			dst[i] = 0
		}
	}
}

func (s *KernelLinuxPublicTestSuite) TestCollect() {
	okUname := fakeUtsname("Linux", "5.15.0-47-generic",
		"#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022", "x86_64")

	tests := []struct {
		name    string
		uname   func(*unix.Utsname) error
		fs      avfs.VFS
		wantErr bool
		want    kernel.Info
	}{
		{
			name:  "canonical Linux host with modules + versions",
			uname: okUname,
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "nf_tables 217088 25 rfkill,nf_conntrack - Live 0x0000000000000000\n" +
					"ipv6 557056 24 - Live 0x0000000000000000\n",
				"/sys/module/nf_tables/version": "1.2.3\n",
				"/sys/module/ipv6/version":      "\n", // empty → leave Version untouched
			}),
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64", Processor: "x86_64", OS: "GNU/Linux",
				Modules: map[string]kernel.Module{
					"nf_tables": {Size: 217088, RefCount: 25, Version: "1.2.3"},
					"ipv6":      {Size: 557056, RefCount: 24},
				},
			},
		},
		{
			name:  "missing /proc/modules omits Modules field",
			uname: okUname,
			fs:    linuxFS(s, nil),
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64", Processor: "x86_64", OS: "GNU/Linux",
			},
		},
		{
			name:  "malformed module line skipped; versions missing leaves empty",
			uname: okUname,
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "short\nvalid_mod 1024 3 - Live 0x0\n",
			}),
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64", Processor: "x86_64", OS: "GNU/Linux",
				Modules: map[string]kernel.Module{
					"valid_mod": {Size: 1024, RefCount: 3},
				},
			},
		},
		{
			name:  "unparseable size/refcount leaves field zero",
			uname: okUname,
			fs: linuxFS(s, map[string]string{
				"/proc/modules": "broken abc xyz - Live 0x0\n",
			}),
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64", Processor: "x86_64", OS: "GNU/Linux",
				Modules: map[string]kernel.Module{"broken": {}},
			},
		},
		{
			name:    "uname error propagated",
			uname:   func(*unix.Utsname) error { return errors.New("uname failed") },
			fs:      linuxFS(s, nil),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer kernel.SetUnameSyscall(tt.uname)()
			c := &kernel.Linux{FS: tt.fs}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
