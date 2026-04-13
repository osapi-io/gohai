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
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type PlatformDarwinPublicTestSuite struct {
	suite.Suite
}

func TestPlatformDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformDarwinPublicTestSuite))
}

// swVersExec returns a MockExecutor that canned-answers `sw_vers`
// with the given output and error.
func swVersExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sw_vers").
		Return(out, err).
		AnyTimes()
	return m
}

const swVersWithRSR = `ProductName:		macOS
ProductVersion:		14.4.1
ProductVersionExtra:	(a)
BuildVersion:		23E224
`

const swVersNoRSR = `ProductName:		macOS
ProductVersion:		13.5
BuildVersion:		22G74
`

func (s *PlatformDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		hostInfo func(context.Context) (*host.InfoStat, error)
		exec     executor.Executor
		wantErr  bool
		want     platform.Info
	}{
		{
			name: "macOS with RSR patch: BuildVersion + ProductVersionExtra both populate",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "14.4.1",
					PlatformFamily: "Standalone Workstation",
					KernelVersion:  "fallback-kernel",
				}, nil
			},
			exec: swVersExec(s.T(), []byte(swVersWithRSR), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "14.4.1",
				VersionExtra: "(a)", Family: "Standalone Workstation",
				Architecture: runtime.GOARCH, Build: "23E224",
			},
		},
		{
			name: "macOS without RSR: BuildVersion populates, VersionExtra stays empty",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "13.5",
					KernelVersion: "fallback-kernel",
				}, nil
			},
			exec: swVersExec(s.T(), []byte(swVersNoRSR), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "13.5",
				Architecture: runtime.GOARCH, Build: "22G74",
			},
		},
		{
			name: "sw_vers error: Build falls back to gopsutil KernelVersion",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "12.6",
					KernelVersion: "21G115",
				}, nil
			},
			exec: swVersExec(s.T(), nil, errors.New("not found")),
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "12.6",
				Architecture: runtime.GOARCH, Build: "21G115",
			},
		},
		{
			name: "sw_vers output with no-colon line: skipped, valid lines parsed",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "darwin", PlatformVersion: "14.0"}, nil
			},
			exec: swVersExec(s.T(),
				[]byte("no colon line\nBuildVersion:\t23A344\nProductVersionExtra:\n"), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "14.0",
				Architecture: runtime.GOARCH, Build: "23A344",
			},
		},
		{
			name: "nil Exec: extension skipped, Build from KernelVersion fallback",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "14.0",
					KernelVersion: "23A344",
				}, nil
			},
			exec: nil,
			want: platform.Info{
				OS: runtime.GOOS, Name: "darwin", Version: "14.0",
				Architecture: runtime.GOARCH, Build: "23A344",
			},
		},
		{
			name: "gopsutil error propagated",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("boom")
			},
			exec:    swVersExec(s.T(), nil, nil),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer platform.SetHostInfoFn(tt.hostInfo)()
			c := &platform.Darwin{Exec: tt.exec}
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
