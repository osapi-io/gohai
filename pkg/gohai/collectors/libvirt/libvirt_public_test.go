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
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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

// fullVirshMock returns a MockRunner wired for the canonical happy-path:
// version, uri, list --all, and dominfo for each domain.
func fullVirshMock(
	t *testing.T,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
		Return([]byte(virshVersionOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
		Return([]byte(virshURIOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
		Return([]byte(virshListOutput), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", "myvm").
		Return([]byte(virshDominfoMyvm), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", "stopped-vm").
		Return([]byte(virshDominfoStopped), nil).AnyTimes()
	return m
}

func (s *LibvirtPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(libvirt.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c libvirt.Collector) {
				_, ok := c.(*libvirt.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c libvirt.Collector) {
				_, ok := c.(*libvirt.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c libvirt.Collector) {
				_, ok := c.(*libvirt.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c libvirt.Collector) {
				_, ok := c.(*libvirt.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := libvirt.New()
			s.Equal("libvirt", c.Name())
			s.Equal("virtualization", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *LibvirtPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		exec    func(*testing.T) source.Runner
		// files, when not nil, is the host's filesystem, where the
		// collector looks for the daemons' sockets.
		files        map[string]string
		validateFunc func(any, error)
	}{
		// ----- Linux -----
		{
			name:    "linux: reads the version, the URI and two domains with dominfo",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return fullVirshMock(t) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
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
			name:    "linux: lines over 64 KiB in version, list and dominfo keep the lines after them",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				long := "Junk: " + strings.Repeat("x", 70000) + "\n"
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(long+virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(long+virshListOutput), nil).AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", gomock.Any()).
					Return([]byte(long+virshDominfoMyvm), nil).
					AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Equal("10.0.0", i.Version)
				s.Require().Len(i.Domains, 2)
				s.Equal("aaaabbbb-cccc-dddd-eeee-ffffffffffff", i.Domains[0].UUID)
			},
		},
		{
			name:    "linux: version without daemon line uses library fallback",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionNoDeamon), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Equal("10.0.0", i.Version)
				s.Nil(i.Domains)
			},
		},
		{
			name:    "linux: a library line without the libvirt prefix keeps its raw value",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				// "Using library:" with a value that does NOT start with "libvirt "
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte("Using library: somevendor 9.0.0\n"), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Equal("somevendor 9.0.0", i.Version)
			},
		},
		{
			name:    "linux: dominfo with no-colon lines skipped, valid fields parsed",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(" Id   Name   State\n-----------\n 1    testvm   running\n"), nil).
					AnyTimes()
				// dominfo with a no-colon line mixed in
				m.EXPECT().
					Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", "testvm").
					Return([]byte("no colon line here\nUUID: deadbeef-dead-beef-dead-beefdeadbeef\nAutostart: enable\n"), nil).
					AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Len(i.Domains, 1)
				s.Equal("deadbeef-dead-beef-dead-beefdeadbeef", i.Domains[0].UUID)
				s.True(i.Domains[0].Autostart)
			},
		},
		{
			name:    "linux: dominfo with an empty CPU(s) value keeps the domain and its other fields",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(" Id   Name   State\n-----------\n 1    testvm   running\n"), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", "testvm").
					Return([]byte("CPU(s):\nAutostart: enable\n"), nil).
					AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Require().Len(i.Domains, 1)
				s.Zero(i.Domains[0].VCPUs)
				s.True(i.Domains[0].Autostart)
			},
		},
		{
			name:    "linux: no virsh on the host gives no fact",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, fmt.Errorf("exec virsh: %w", exec.ErrNotFound))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a daemon that is not running gives no fact",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, errors.New("exec virsh: exit status 1: error: failed to connect to the hypervisor\n"+
						"error: Failed to connect socket to '/var/run/libvirt/virtqemud-sock': No such file or directory"))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a daemon that refuses the connection gives no fact",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, errors.New("exec virsh: exit status 1: error: failed to connect to the hypervisor\n"+
						"error: Failed to connect socket to '/var/run/libvirt/libvirt-sock': Connection refused"))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a translated error with no daemon socket on the host gives no fact",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, errors.New("exec virsh: exit status 1: Fehler: Verbindung zum Hypervisor fehlgeschlagen\n"+
						"Fehler: Verbindung zum Socket '/var/run/libvirt/virtqemud-sock' fehlgeschlagen: Datei oder Verzeichnis nicht gefunden"))
				return m
			},
			files: map[string]string{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a translated error with a daemon socket on the host is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, errors.New("exec virsh: exit status 1: Fehler: Verbindung zum Hypervisor fehlgeschlagen\n"+
						"Fehler: Verbindung zum Socket '/run/libvirt/libvirt-sock' fehlgeschlagen: Keine Berechtigung"))
				return m
			},
			files: map[string]string{"/run/libvirt/libvirt-sock": ""},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read virsh version:")
				s.ErrorContains(err, "Keine Berechtigung")
			},
		},
		{
			name:    "linux: no permission on the daemon's socket is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return(nil, errors.New("exec virsh: exit status 1: error: failed to connect to the hypervisor\n"+
						"error: Failed to connect socket to '/var/run/libvirt/libvirt-sock': Permission denied"))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read virsh version:")
				s.ErrorContains(err, "Permission denied")
			},
		},
		{
			name:    "linux: nil Exec returns nil",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: virsh uri fails after the daemon answered is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return(nil, errors.New("connection refused")).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read virsh uri")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: virsh list fails after the daemon answered is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return(nil, errors.New("permission denied")).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read virsh list")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: when virsh dominfo fails keeps the domain with its name and state only",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(" Id   Name   State\n-----------\n 1    myvm   running\n"), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "dominfo", "myvm").
					Return(nil, errors.New("permission denied")).
					AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Len(i.Domains, 1)
				s.Equal("myvm", i.Domains[0].Name)
				s.Equal("running", i.Domains[0].State)
				s.Empty(i.Domains[0].UUID)
			},
		},
		{
			name:    "linux: an empty domain list leaves Domains nil",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				ctrl := gomock.NewController(t)
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "version").
					Return([]byte(virshVersionOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "uri").
					Return([]byte(virshURIOutput), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "virsh", "-c", "qemu:///system", "list", "--all").
					Return([]byte(virshListEmptyOutput), nil).AnyTimes()
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*libvirt.Info)
				s.Require().True(ok)
				s.Nil(i.Domains)
			},
		},
		// ----- Darwin -----
		{
			name:    "darwin: unsupported",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c libvirt.Collector
			switch tt.variant {
			case "linux":
				l := &libvirt.Linux{Exec: tt.exec(s.T())}
				if tt.files != nil {
					l.FS = sourcetest.FS(s.T(), tt.files)
				}
				c = l
			case "darwin":
				c = libvirt.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
