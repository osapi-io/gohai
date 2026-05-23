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

package libvirt_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
)

var (
	_ collector.Collector = (*libvirt.Linux)(nil)
	_ collector.Collector = (*libvirt.Darwin)(nil)
)

// Sample virsh output fixtures.
const (
	virshVersionOutput = `Compiled against library: libvirt 10.0.0
Using library: libvirt 10.0.0
Using API: QEMU 10.0.0
Running hypervisor: QEMU 8.2.1
Running against daemon: 10.0.0
`

	virshVersionNoDeamon = `Compiled against library: libvirt 10.0.0
Using library: libvirt 10.0.0
Using API: QEMU 10.0.0
Running hypervisor: QEMU 8.2.1
`

	virshURIOutput = "qemu:///system\n"

	virshListOutput = ` Id   Name        State
-----------------------------
 1    myvm        running
 -    stopped-vm  shut off
`

	virshListEmptyOutput = ` Id   Name   State
-----------------------
`

	virshDominfoMyvm = `Id:             1
Name:           myvm
UUID:           aaaabbbb-cccc-dddd-eeee-ffffffffffff
OS Type:        hvm
State:          running
CPU(s):         4
Max memory:     2097152 KiB
Used memory:    2097152 KiB
Persistent:     yes
Autostart:      enable
Managed save:   no
Security model: none
Security DOI:   0
`

	virshDominfoStopped = `Id:             -
Name:           stopped-vm
UUID:           11112222-3333-4444-5555-666677778888
OS Type:        hvm
State:          shut off
CPU(s):         2
Max memory:     1048576 KiB
Used memory:    0 KiB
Persistent:     yes
Autostart:      disable
Managed save:   no
Security model: none
Security DOI:   0
`
)

type LibvirtPublicTestSuite struct {
	suite.Suite
}

func TestLibvirtPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LibvirtPublicTestSuite))
}

// fullVirshMock returns a MockExecutor wired for the canonical happy-path:
// version, uri, list --all, and dominfo for each domain.
func fullVirshMock(
	t *testing.T,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().Execute(gomock.Any(), "virsh", "version").
		Return([]byte(virshVersionOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
		Return([]byte(virshURIOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
		Return([]byte(virshListOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "dominfo", "myvm").
		Return([]byte(virshDominfoMyvm), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "dominfo", "stopped-vm").
		Return([]byte(virshDominfoStopped), nil).AnyTimes()
	return m
}

func (s *LibvirtPublicTestSuite) TestNew() {
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
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := libvirt.New()
			s.Equal("libvirt", c.Name())
			s.Equal("virtualization", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*libvirt.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*libvirt.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *LibvirtPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		exec     func(*testing.T) executor.Executor
		wantNil  bool
		validate func(*libvirt.Info)
	}{
		// ----- Linux -----
		{
			name:    "linux: full happy path — version, URI, two domains with dominfo",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return fullVirshMock(t) },
			validate: func(i *libvirt.Info) {
				s.Equal("10.0.0", i.Version)
				s.Equal("qemu:///system", i.URI)
				s.Len(i.Domains, 2)

				myvm := i.Domains[0]
				s.Equal("myvm", myvm.Name)
				s.Equal("running", myvm.State)
				s.Equal("aaaabbbb-cccc-dddd-eeee-ffffffffffff", myvm.UUID)
				s.Equal(4, myvm.VCPUs)
				s.Equal("2097152 KiB", myvm.MaxMemory)
				s.True(myvm.Autostart)

				stopped := i.Domains[1]
				s.Equal("stopped-vm", stopped.Name)
				s.Equal("shut off", stopped.State)
				s.Equal("11112222-3333-4444-5555-666677778888", stopped.UUID)
				s.Equal(2, stopped.VCPUs)
				s.False(stopped.Autostart)
			},
		},
		{
			name:    "linux: version without daemon line uses library fallback",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionNoDeamon), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Equal("10.0.0", i.Version)
				s.Nil(i.Domains)
			},
		},
		{
			name:    "linux: library line without libvirt prefix — raw value used",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				// "Using library:" with a value that does NOT start with "libvirt "
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte("Using library: somevendor 9.0.0\n"), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Equal("somevendor 9.0.0", i.Version)
			},
		},
		{
			name:    "linux: dominfo with no-colon lines skipped, valid fields parsed",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(" Id   Name   State\n-----------\n 1    testvm   running\n"), nil).
					AnyTimes()
				// dominfo with a no-colon line mixed in
				m.EXPECT().Execute(gomock.Any(), "virsh", "dominfo", "testvm").
					Return([]byte("no colon line here\nUUID: deadbeef-dead-beef-dead-beefdeadbeef\nAutostart: enable\n"), nil).
					AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Len(i.Domains, 1)
				s.Equal("deadbeef-dead-beef-dead-beefdeadbeef", i.Domains[0].UUID)
				s.True(i.Domains[0].Autostart)
			},
		},
		{
			name:    "linux: virsh version fails — returns nil",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return(nil, errors.New("virsh: command not found")).AnyTimes()
				return m
			},
			wantNil: true,
		},
		{
			name:    "linux: nil Exec returns nil",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			wantNil: true,
		},
		{
			name:    "linux: virsh uri fails — URI stays empty, domains still collected",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return(nil, errors.New("connection refused")).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Empty(i.URI)
				s.Equal("10.0.0", i.Version)
			},
		},
		{
			name:    "linux: virsh list fails — returns Info with version only",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return(nil, errors.New("permission denied")).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Equal("10.0.0", i.Version)
				s.Nil(i.Domains)
			},
		},
		{
			name:    "linux: virsh dominfo fails — domain kept with name+state only",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(" Id   Name   State\n-----------\n 1    myvm   running\n"), nil).
					AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "dominfo", "myvm").
					Return(nil, errors.New("permission denied")).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Len(i.Domains, 1)
				s.Equal("myvm", i.Domains[0].Name)
				s.Equal("running", i.Domains[0].State)
				s.Empty(i.Domains[0].UUID)
			},
		},
		{
			name:    "linux: empty domain list — Domains field nil",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				ctrl := gomock.NewController(t)
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validate: func(i *libvirt.Info) {
				s.Nil(i.Domains)
			},
		},
		// ----- Darwin -----
		{
			name:    "darwin always returns nil",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c libvirt.Collector
			switch tt.variant {
			case "linux":
				c = &libvirt.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = libvirt.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*libvirt.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
