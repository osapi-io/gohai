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

package hostnamectl_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*hostnamectl.Linux)(nil)
	_ collector.Collector = (*hostnamectl.Darwin)(nil)
)

type HostnamectlPublicTestSuite struct {
	suite.Suite
}

func TestHostnamectlPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnamectlPublicTestSuite))
}

// hostnamectlExec returns a MockRunner that answers `hostnamectl`
// with the provided output and error.
func hostnamectlExec(
	t *testing.T,
	out []byte,
	err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "hostnamectl").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *HostnamectlPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(hostnamectl.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c hostnamectl.Collector) {
				_, ok := c.(*hostnamectl.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c hostnamectl.Collector) {
				_, ok := c.(*hostnamectl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c hostnamectl.Collector) {
				_, ok := c.(*hostnamectl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c hostnamectl.Collector) {
				_, ok := c.(*hostnamectl.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c hostnamectl.Collector) {
				_, ok := c.(*hostnamectl.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := hostnamectl.New()
			s.Equal("hostnamectl", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *HostnamectlPublicTestSuite) TestCollect() {
	// Full modern systemd output with Unicode decorators.
	fullOutput := []byte(` Static hostname: myhost
         Icon name: computer-vm
           Chassis: vm 🖥
        Deployment: production
          Location: rack-42
            Kernel: Linux 5.15.0-91-generic
   Operating System: Ubuntu 22.04.3 LTS
        CPE OS Name: cpe:/o:ubuntu:ubuntu:22.04
         Virtualization: kvm
       Hardware Vendor: QEMU
        Hardware Model: Standard PC (Q35 + ICH9, 2009)
      Firmware Version: 2.5+dfsg-4
`)

	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: full output parsed correctly",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t, fullOutput, nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{
					StaticHostname:            "myhost",
					IconName:                  "computer-vm",
					Chassis:                   "vm",
					Deployment:                "production",
					Location:                  "rack-42",
					KernelName:                "Linux",
					KernelRelease:             "5.15.0-91-generic",
					OperatingSystemPrettyName: "Ubuntu 22.04.3 LTS",
					OperatingSystemCPEName:    "cpe:/o:ubuntu:ubuntu:22.04",
					Virtualization:            "kvm",
					HardwareVendor:            "QEMU",
					HardwareModel:             "Standard PC (Q35 + ICH9, 2009)",
					FirmwareVersion:           "2.5+dfsg-4",
				}, *info)
			},
		},
		{
			name:    "linux: minimal output, only static_hostname",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t, []byte(" Static hostname: node1\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{StaticHostname: "node1"}, *info)
			},
		},
		{
			name:    "linux: line without colon-space separator skipped",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t,
					[]byte("no separator here\n Static hostname: host2\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{StaticHostname: "host2"}, *info)
			},
		},
		{
			name:    "linux: operating_system key (no pretty_name) sets PrettyName",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t,
					[]byte(" Operating System: Debian GNU/Linux 12 (bookworm)\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(
					hostnamectl.Info{OperatingSystemPrettyName: "Debian GNU/Linux 12 (bookworm)"},
					*info,
				)
			},
		},
		{
			name:    "linux: cpe_os_name key maps to OperatingSystemCPEName",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t,
					[]byte(" CPE OS Name: cpe:/o:debian:debian_linux:12\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(
					hostnamectl.Info{OperatingSystemCPEName: "cpe:/o:debian:debian_linux:12"},
					*info,
				)
			},
		},
		{
			name:    "linux: non-ASCII value collapses double spaces",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				// An emoji between two words leaves two adjacent spaces after stripping.
				return hostnamectlExec(t,
					[]byte(" Chassis: vm \U0001F5A5 server\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{Chassis: "vm server"}, *info)
			},
		},
		{
			name:    "linux: UTF-8 in a location or a vendor name is kept",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(
					t,
					[]byte(
						" Static hostname: node1\n        Location: Büro München\n Hardware Vendor: Société Générale\n",
					),
					nil,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{
					StaticHostname: "node1",
					Location:       "Büro München",
					HardwareVendor: "Société Générale",
				}, *info)
			},
		},
		{
			name:    "linux: the transient hostname is read",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t,
					[]byte(" Static hostname: n/a\nTransient hostname: dhcp-42\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(
					hostnamectl.Info{StaticHostname: "n/a", TransientHostname: "dhcp-42"},
					*info,
				)
			},
		},
		{
			name:    "linux: no hostnamectl on the host yields empty Info, no error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t, nil, fmt.Errorf("exec hostnamectl: %w", exec.ErrNotFound))
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{}, *info)
			},
		},
		{
			name:    "linux: a hostnamectl that fails is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return hostnamectlExec(t, nil, errors.New("exit status 1"))
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Exactly(nil, got)
				s.Contains(err.Error(), "read hostnamectl")
			},
		},
		{
			name:    "linux: nil Exec yields empty Info, no error",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{}, *info)
			},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Exactly(nil, got)
					return
				}
				info, ok := got.(*hostnamectl.Info)
				s.Require().True(ok)
				s.Equal(hostnamectl.Info{}, *info)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c hostnamectl.Collector
			switch tt.variant {
			case "linux":
				c = &hostnamectl.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = hostnamectl.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
