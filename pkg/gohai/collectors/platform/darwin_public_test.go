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

package platform_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

// TestMain enables self-exec subprocess testing for runSwVers.
// When GOHAI_PLATFORM_SWVERS_TEST is set, the test binary emulates
// the sw_vers output we want — no /bin/echo or real external process.
func TestMain(m *testing.M) {
	if os.Getenv("GOHAI_PLATFORM_SWVERS_TEST") == "1" {
		fmt.Print("(a)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type PlatformDarwinPublicTestSuite struct {
	suite.Suite
}

func TestPlatformDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformDarwinPublicTestSuite))
}

func (s *PlatformDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		hostFn  func(context.Context) (*host.InfoStat, error)
		runCmd  func(string, ...string) ([]byte, error)
		wantErr bool
		want    platform.Info
	}{
		{
			name: "macOS with RSR patch version",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "14.4.1",
					PlatformFamily: "Standalone Workstation", KernelVersion: "23E224",
				}, nil
			},
			runCmd: func(string, ...string) ([]byte, error) { return []byte("(a)\n"), nil },
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "14.4.1",
				VersionExtra: "(a)", Family: "Standalone Workstation",
				Architecture: runtime.GOARCH, Build: "23E224",
			},
		},
		{
			name: "macOS without RSR patch (sw_vers empty)",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "darwin",
					PlatformVersion: "13.5",
					KernelVersion:   "22G74",
				}, nil
			},
			runCmd: func(string, ...string) ([]byte, error) { return []byte("\n"), nil },
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "13.5",
				Architecture: runtime.GOARCH, Build: "22G74",
			},
		},
		{
			name: "sw_vers error omits version_extra",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "darwin",
					PlatformVersion: "12.6",
					KernelVersion:   "21G115",
				}, nil
			},
			runCmd: func(string, ...string) ([]byte, error) { return nil, errors.New("not found") },
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "12.6",
				Architecture: runtime.GOARCH, Build: "21G115",
			},
		},
		{
			name:   "nil info",
			hostFn: func(_ context.Context) (*host.InfoStat, error) { return nil, nil },
			runCmd: func(string, ...string) ([]byte, error) { return nil, errors.New("no host info") },
			want:   platform.Info{OS: runtime.GOOS, Architecture: runtime.GOARCH},
		},
		{
			name:    "host.Info error propagated",
			hostFn:  func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			runCmd:  func(string, ...string) ([]byte, error) { return nil, nil },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &platform.Darwin{HostInfoFn: tt.hostFn, RunCmdFn: tt.runCmd}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*platform.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

// TestRunSwVers covers the exec wrapper via Go's standard self-exec
// pattern: the test binary re-invokes itself with an env-var flag
// that TestMain handles, so we exercise the real
// exec.Command().Output() path against a subprocess we fully control.
// No real sw_vers, no external binary (/bin/echo, /usr/bin/true, etc.).
func (s *PlatformDarwinPublicTestSuite) TestRunSwVers() {
	tests := []struct {
		name    string
		envVal  string
		cmd     string
		wantOut string
		wantErr bool
	}{
		{
			name:    "subprocess writes expected output",
			envVal:  "1",
			cmd:     os.Args[0],
			wantOut: "(a)",
		},
		{
			name:    "missing binary returns error",
			envVal:  "",
			cmd:     "/gohai-test-does-not-exist",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Setenv("GOHAI_PLATFORM_SWVERS_TEST", tt.envVal)
			c := platform.NewDarwin()
			out, err := c.RunCmdFn(tt.cmd)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantOut, string(out))
		})
	}
}
