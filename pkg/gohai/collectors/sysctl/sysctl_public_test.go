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

package sysctl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysctl"
)

var (
	_ collector.Collector = (*sysctl.Linux)(nil)
	_ collector.Collector = (*sysctl.Darwin)(nil)
)

type SysctlPublicTestSuite struct {
	suite.Suite
}

func TestSysctlPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlPublicTestSuite))
}

// sysctlExec returns a MockExecutor that answers `sysctl -a` with the
// provided output and error.
func sysctlExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sysctl", "-a").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *SysctlPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(sysctl.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c sysctl.Collector) {
				_, ok := c.(*sysctl.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c sysctl.Collector) {
				_, ok := c.(*sysctl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c sysctl.Collector) {
				_, ok := c.(*sysctl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c sysctl.Collector) {
				_, ok := c.(*sysctl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c sysctl.Collector) {
				_, ok := c.(*sysctl.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := sysctl.New()
			s.Equal("sysctl", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SysctlPublicTestSuite) TestCollect() {
	linuxOutput := []byte(`kernel.hostname = myhost
kernel.ostype = Linux
net.ipv4.ip_forward = 0
vm.swappiness = 60
`)
	darwinOutput := []byte(`kern.ostype: Darwin
kern.osrelease: 23.5.0
hw.ncpu: 10
vm.swapusage: total = 1024.00M  used = 512.00M  free = 512.00M  (encrypted)
`)

	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: canonical key=value output parsed",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return sysctlExec(t, linuxOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"kernel.hostname":     "myhost",
					"kernel.ostype":       "Linux",
					"net.ipv4.ip_forward": "0",
					"vm.swappiness":       "60",
				}, info.Params)
			},
		},
		{
			name:    "linux: empty output yields empty params map",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return sysctlExec(t, []byte{}, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
		{
			name:    "linux: lines without separator skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return sysctlExec(t, []byte("no_separator\nkernel.panic = 1\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"kernel.panic": "1"}, info.Params)
			},
		},
		{
			name:    "linux: empty key skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				// Line with separator but no key portion — key becomes empty, skipped.
				return sysctlExec(t, []byte(": value\nkernel.panic = 2\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"kernel.panic": "2"}, info.Params)
			},
		},
		{
			name:    "linux: exec error yields empty params, no error",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return sysctlExec(t, nil, errors.New("not found")) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
		{
			name:    "linux: nil Exec yields empty params, no error",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
		{
			name:    "darwin: colon-separated output parsed",
			variant: "darwin",
			exec:    func(t *testing.T) executor.Executor { return sysctlExec(t, darwinOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"kern.ostype":    "Darwin",
					"kern.osrelease": "23.5.0",
					"hw.ncpu":        "10",
					"vm.swapusage":   "total = 1024.00M  used = 512.00M  free = 512.00M  (encrypted)",
				}, info.Params)
			},
		},
		{
			name:    "darwin: exec error yields empty params, no error",
			variant: "darwin",
			exec:    func(t *testing.T) executor.Executor { return sysctlExec(t, nil, errors.New("not found")) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
		{
			name:    "darwin: nil Exec yields empty params, no error",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c sysctl.Collector
			switch tt.variant {
			case "linux":
				c = &sysctl.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &sysctl.Darwin{Exec: tt.exec(s.T())}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
