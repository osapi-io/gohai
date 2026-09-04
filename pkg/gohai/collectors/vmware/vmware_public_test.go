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

package vmware_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/vmware"
)

var (
	_ collector.Collector = (*vmware.Linux)(nil)
	_ collector.Collector = (*vmware.Darwin)(nil)
)

// vmwareToolboxPath is duplicated here to avoid exporting the private const.
const vmwareToolboxPath = "/usr/bin/vmware-toolbox-cmd"

type VMwarePublicTestSuite struct {
	suite.Suite
}

func TestVMwarePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VMwarePublicTestSuite))
}

// toolboxExec returns a MockExecutor whose Execute expectations are set by
// the caller via the returned recorder — convenience for multi-call mocks.
func toolboxExec(
	t *testing.T,
) (*execmocks.MockExecutor, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	return m, ctrl
}

// fullToolboxMock sets up a mock executor that answers all the
// vmware-toolbox-cmd calls made by collectVMwareTools, plus the
// initial -v probe used by the darwin/linux variants.
func fullToolboxMock(
	t *testing.T,
) executor.Executor {
	m, _ := toolboxExec(t)
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
		Return([]byte("12.3.0 build-21581411\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
		Return([]byte("01 Jan 2026 12:00:00\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
		Return([]byte("2600 MHz\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
		Return([]byte("1\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
		Return([]byte("4096 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
		Return([]byte("0 MHz\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
		Return([]byte("unlimited\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
		Return([]byte("VMware Tools are up-to-date\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
		Return([]byte("Timesync is disabled\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
		Return([]byte(""), nil).AnyTimes()
	return m
}

// vsphereMock is like fullToolboxMock but returns a vSphere session JSON.
func vsphereMock(
	t *testing.T,
) executor.Executor {
	m, _ := toolboxExec(t)
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
		Return([]byte("12.3.0 build-21581411\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
		Return([]byte("01 Jan 2026 12:00:00\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
		Return([]byte("2600 MHz\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
		Return([]byte("1\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
		Return([]byte("4096 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
		Return([]byte("0 MB\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
		Return([]byte("0 MHz\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
		Return([]byte("unlimited\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
		Return([]byte("VMware Tools are up-to-date\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
		Return([]byte("Timesync is disabled\n"), nil).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
		Return([]byte(`{"version":"7.0.3"}`), nil).AnyTimes()
	return m
}

func (s *VMwarePublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", "debian", osLinux},
		{"rhel dispatches to Linux", "rhel", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := vmware.New()
			s.Equal("vmware", c.Name())
			s.Equal("virtualization", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*vmware.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*vmware.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *VMwarePublicTestSuite) TestCollect() {
	tests := []struct {
		name        string
		variant     string
		setupLinux  func() *vmware.Linux
		setupDarwin func() *vmware.Darwin
		wantNil     bool
		validate    func(*vmware.Info)
	}{
		// ----- Linux -----
		{
			name:    "linux: vmware SCSI in /proc/scsi/scsi + tools cmd",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll(pathProcScsi, 0o755)
				_ = f.WriteFile(
					pathProcScsiScsi,
					[]byte(
						"Host: scsi0 Channel: 00 Id: 00 Lun: 00\n  Vendor: VMware   Model: Virtual disk\n",
					),
					fs.FileMode(0o444),
				)
				return &vmware.Linux{FS: f, Exec: fullToolboxMock(s.T())}
			},
			validate: func(i *vmware.Info) {
				s.Equal("12.3.0 build-21581411", i.Version)
				s.Equal(virtVMware, i.HostType)
				s.Equal("01 Jan 2026 12:00:00", i.Hosttime)
			},
		},
		{
			name:    "linux: vSphere session JSON sets host_type and host_version",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll(pathProcScsi, 0o755)
				_ = f.WriteFile(pathProcScsiScsi,
					[]byte("VMware SCSI\n"),
					fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: vsphereMock(s.T())}
			},
			validate: func(i *vmware.Info) {
				s.Equal("vmware_vsphere", i.HostType)
				s.Equal("7.0.3", i.HostVersion)
			},
		},
		{
			name:    "linux: /proc/scsi/scsi missing but toolbox-cmd responds",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				// probe call (no /proc/scsi/scsi present)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte("12.3.0\n"), nil).AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
					Return([]byte("time\n"), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
					Return([]byte(""), nil).
					AnyTimes()
				return &vmware.Linux{FS: memfs.New(), Exec: m}
			},
			validate: func(i *vmware.Info) {
				s.Equal("12.3.0", i.Version)
				s.Equal(virtVMware, i.HostType)
			},
		},
		{
			name:    "linux: UpdateInfo failed filtered from stat output",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte("12.3.0\n"), nil).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
					Return([]byte("UpdateInfo failed\n"), nil).AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
					Return([]byte(""), nil).
					AnyTimes()
				f := memfs.New()
				_ = f.MkdirAll(pathProcScsi, 0o755)
				_ = f.WriteFile(pathProcScsiScsi, []byte("VMware\n"), fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: m}
			},
			validate: func(i *vmware.Info) {
				s.Empty(i.Hosttime)
			},
		},
		{
			name:    "linux: invalid vSphere session JSON still sets host_type",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte("12.3.0\n"), nil).AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
					Return([]byte(""), nil).
					AnyTimes()
				// non-empty but not valid JSON — triggers the json.Unmarshal error branch
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
					Return([]byte("not-json"), nil).
					AnyTimes()
				f := memfs.New()
				_ = f.MkdirAll(pathProcScsi, 0o755)
				_ = f.WriteFile(pathProcScsiScsi, []byte("VMware\n"), fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: m}
			},
			validate: func(i *vmware.Info) {
				s.Equal("vmware_vsphere", i.HostType)
				s.Empty(i.HostVersion)
			},
		},
		{
			name:    "linux: no VMware signals returns nil",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, errors.New("not found")).AnyTimes()
				return &vmware.Linux{FS: memfs.New(), Exec: m}
			},
			wantNil: true,
		},
		{
			name:    "linux: nil Exec and no /proc/scsi/scsi returns nil",
			variant: osLinux,
			setupLinux: func() *vmware.Linux {
				return &vmware.Linux{FS: memfs.New(), Exec: nil}
			},
			wantNil: true,
		},
		// ----- Darwin -----
		{
			name:    "darwin: toolbox-cmd present returns Info",
			variant: osDarwin,
			setupDarwin: func() *vmware.Darwin {
				return &vmware.Darwin{Exec: fullToolboxMock(s.T())}
			},
			validate: func(i *vmware.Info) {
				s.Equal("12.3.0 build-21581411", i.Version)
				s.Equal(virtVMware, i.HostType)
			},
		},
		{
			name:    "darwin: stat command error leaves field empty",
			variant: osDarwin,
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte("12.3.0\n"), nil).AnyTimes()
				// speed fails — field should be empty string.
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
					Return(nil, errors.New("exec error")).AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "sessionid").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "balloon").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "swap").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memlimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "memres").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpures").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "cpulimit").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "upgrade", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "timesync", "status").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "raw", "json", "session").
					Return([]byte(""), nil).
					AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			validate: func(i *vmware.Info) {
				s.Empty(i.Speed)
				s.Equal(virtVMware, i.HostType)
			},
		},
		{
			name:    "darwin: toolbox-cmd absent returns nil",
			variant: osDarwin,
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, errors.New("not found")).AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			wantNil: true,
		},
		{
			name:    "darwin: toolbox-cmd returns empty output — nil",
			variant: osDarwin,
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte{}, nil).AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			wantNil: true,
		},
		{
			name:    "darwin: nil Exec returns nil",
			variant: osDarwin,
			setupDarwin: func() *vmware.Darwin {
				return &vmware.Darwin{Exec: nil}
			},
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c vmware.Collector
			switch tt.variant {
			case osLinux:
				c = tt.setupLinux()
			case osDarwin:
				c = tt.setupDarwin()
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*vmware.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
