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
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysctl"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*sysctl.Linux)(nil)
	_ collector.Collector = (*sysctl.Darwin)(nil)
)

// The command lines that the rows answer, as sourcetest.Commands keys them.
const (
	sysctlLine = "sysctl -a"
)

type SysctlPublicTestSuite struct {
	suite.Suite
}

func TestSysctlPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlPublicTestSuite))
}

func (s *SysctlPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c sysctl.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := sysctl.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = sysctl.New()
			}
			s.Equal("sysctl", c.Name())
			s.Equal("linux", string(c.Category()))
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
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: canonical key=value output parsed",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{sysctlLine: string(linuxOutput)},
				}.Mock(
					t,
				)
			},
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{sysctlLine: ""}}.Mock(t)
			},
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{sysctlLine: "no_separator\nkernel.panic = 1\n"},
				}.Mock(
					t,
				)
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
			exec: func(t *testing.T) source.Runner {
				// A line with a separator but no key: the key is empty, so it is skipped.
				return sourcetest.Commands{
					Out: map[string]string{sysctlLine: " = value\nkernel.panic = 2\n"},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"kernel.panic": "2"}, info.Params)
			},
		},
		{
			name:    "linux: no sysctl yields empty params, no error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{sysctlLine: fmt.Errorf("sysctl: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Params)
			},
		},
		{
			name:    "linux: a value with \": \" keeps its text",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						sysctlLine: "dev.cdrom.info = CD-ROM information, Id: cdrom.c 3.20\n",
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"dev.cdrom.info": "CD-ROM information, Id: cdrom.c 3.20",
				}, info.Params)
			},
		},
		{
			name:    "linux: a failed run that printed part of the table gives that part",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{sysctlLine: "kernel.panic = 1\n"},
					Err: map[string]error{sysctlLine: errors.New("exit status 255")},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysctl.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"kernel.panic": "1"}, info.Params)
			},
		},
		{
			name:    "linux: a failed run that printed nothing is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{sysctlLine: errors.New("signal: killed")},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "sysctl -a: signal: killed")
				s.Nil(got)
			},
		},
		{
			name:    "linux: nil Exec yields empty params, no error",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{sysctlLine: string(darwinOutput)},
				}.Mock(
					t,
				)
			},
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
			name:    "darwin: no sysctl yields empty params, no error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{sysctlLine: fmt.Errorf("sysctl: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
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
			exec:    func(*testing.T) source.Runner { return nil },
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
