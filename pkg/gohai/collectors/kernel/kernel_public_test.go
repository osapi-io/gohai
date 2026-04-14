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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

var (
	_ collector.Collector = (*kernel.Linux)(nil)
	_ collector.Collector = (*kernel.Darwin)(nil)
)

type KernelPublicTestSuite struct {
	suite.Suite
}

func TestKernelPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KernelPublicTestSuite))
}

// fakeUtsname fills a unix.Utsname with name/release/version/machine.
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

func copyBytes(
	dst []byte,
	src string,
) {
	for i := range dst {
		if i < len(src) {
			dst[i] = src[i]
		} else {
			dst[i] = 0
		}
	}
}

// rosettaExec returns a mock that canned-answers the Rosetta sysctl.
func rosettaExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sysctl", "-n", "hw.optional.x86_64").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *KernelPublicTestSuite) TestNew() {
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
			c := kernel.New()
			s.Equal("kernel", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*kernel.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*kernel.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *KernelPublicTestSuite) TestBytesToString() {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"NUL-terminated C string", []byte{'L', 'i', 'n', 'u', 'x', 0, 0, 0}, "Linux"},
		{"no trailing NUL (full array used)", []byte{'a', 'b', 'c'}, "abc"},
		{"empty input", []byte{}, ""},
		{"leading NUL truncates to empty", []byte{0, 'x', 'y'}, ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, kernel.BytesToString(tt.in))
		})
	}
}

func (s *KernelPublicTestSuite) TestDefaultUname() {
	tests := []struct {
		name    string
		fn      func(*unix.Utsname) error
		wantErr bool
	}{
		{
			name: "success returns populated fields",
			fn: func(u *unix.Utsname) error {
				copy(u.Sysname[:], "Linux")
				copy(u.Release[:], "6.1.0")
				copy(u.Version[:], "#1 SMP")
				copy(u.Machine[:], "x86_64")
				return nil
			},
		},
		{
			name:    "syscall error propagated",
			fn:      func(*unix.Utsname) error { return errors.New("uname failed") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := kernel.SetUnameSyscall(tt.fn)
			defer restore()
			name, release, _, machine, err := kernel.DefaultUname()
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal("Linux", name)
			s.Equal("6.1.0", release)
			s.Equal("x86_64", machine)
		})
	}
}

func (s *KernelPublicTestSuite) TestCollect() {
	linuxOK := fakeUtsname("Linux", "5.15.0-47-generic",
		"#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022", "x86_64")
	darwinARM := fakeUtsname(
		"Darwin", "23.4.0",
		"Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
		"arm64",
	)
	darwinIntel := fakeUtsname(
		"Darwin", "22.6.0",
		"Darwin Kernel Version 22.6.0",
		"x86_64",
	)

	tests := []struct {
		name     string
		variant  string
		uname    func(*unix.Utsname) error
		exec     func(*testing.T) executor.Executor
		wantErr  bool
		validate func(*kernel.Info)
	}{
		{
			name:    "linux: canonical identity fields",
			variant: "linux",
			uname:   linuxOK,
			validate: func(i *kernel.Info) {
				s.Equal("Linux", i.Name)
				s.Equal("5.15.0-47-generic", i.Release)
				s.Equal("x86_64", i.Machine)
				s.Equal("x86_64", i.Processor)
				s.Equal("GNU/Linux", i.OS)
			},
		},
		{
			name:    "linux: uname error propagated",
			variant: "linux",
			uname:   func(*unix.Utsname) error { return errors.New("uname failed") },
			wantErr: true,
		},
		{
			name:    "darwin: native arm64 Apple Silicon — no rosetta",
			variant: "darwin",
			uname:   darwinARM,
			exec: func(t *testing.T) executor.Executor {
				return rosettaExec(t, []byte("0\n"), nil)
			},
			validate: func(i *kernel.Info) {
				s.Equal("Darwin", i.Name)
				s.Equal("arm64", i.Machine)
				s.Equal("arm64", i.Processor)
				s.Equal("Darwin", i.OS)
				s.False(i.RosettaTranslated)
			},
		},
		{
			name:    "darwin: native Intel Mac, sysctl returns 0 machine stays x86_64",
			variant: "darwin",
			uname:   darwinIntel,
			exec: func(t *testing.T) executor.Executor {
				return rosettaExec(t, []byte("0\n"), nil)
			},
			validate: func(i *kernel.Info) {
				s.Equal("x86_64", i.Machine)
				s.Equal("x86_64", i.Processor)
				s.False(i.RosettaTranslated)
			},
		},
		{
			name:    "darwin: native Intel Mac, sysctl errors no rosetta",
			variant: "darwin",
			uname:   darwinIntel,
			exec: func(t *testing.T) executor.Executor {
				return rosettaExec(t, nil, errors.New("no sysctl"))
			},
			validate: func(i *kernel.Info) {
				s.Equal("x86_64", i.Machine)
				s.False(i.RosettaTranslated)
			},
		},
		{
			name:    "darwin: Rosetta on Apple Silicon, machine corrected to arm64",
			variant: "darwin",
			uname:   darwinIntel,
			exec: func(t *testing.T) executor.Executor {
				return rosettaExec(t, []byte("1\n"), nil)
			},
			validate: func(i *kernel.Info) {
				s.Equal("arm64", i.Machine)
				s.Equal("arm64", i.Processor)
				s.True(i.RosettaTranslated)
			},
		},
		{
			name:    "darwin: nil Exec, no rosetta",
			variant: "darwin",
			uname:   darwinARM,
			exec:    func(*testing.T) executor.Executor { return nil },
			validate: func(i *kernel.Info) {
				s.Equal("arm64", i.Machine)
				s.False(i.RosettaTranslated)
			},
		},
		{
			name:    "darwin: uname error propagated",
			variant: "darwin",
			uname:   func(*unix.Utsname) error { return errors.New("uname failed") },
			exec:    func(*testing.T) executor.Executor { return nil },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer kernel.SetUnameSyscall(tt.uname)()
			var c kernel.Collector
			switch tt.variant {
			case "linux":
				c = &kernel.Linux{}
			case "darwin":
				d := &kernel.Darwin{}
				if tt.exec != nil {
					d.Exec = tt.exec(s.T())
				}
				c = d
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
