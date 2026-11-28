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
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/vmware"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*vmware.Linux)(nil)
	_ collector.Collector = (*vmware.Darwin)(nil)
)

// vmwareToolboxPath is the command name the collector runs, duplicated to
// keep the const private.
const vmwareToolboxPath = "vmware-toolbox-cmd"

type VMwarePublicTestSuite struct {
	suite.Suite
}

func TestVMwarePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VMwarePublicTestSuite))
}

// toolboxExec returns a MockRunner whose Execute expectations are set by
// the caller via the returned recorder — convenience for multi-call mocks.
func toolboxExec(
	t *testing.T,
) (*mocks.MockRunner, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	return m, ctrl
}

// fullToolboxMock sets up a mock executor that answers all the
// vmware-toolbox-cmd calls made by collectVMwareTools, plus the
// initial -v probe used by the darwin/linux variants.
func fullToolboxMock(
	t *testing.T,
) source.Runner {
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
) source.Runner {
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

// failingToolboxMock answers each vmware-toolbox-cmd call as
// fullToolboxMock does, but the call whose args join to failing returns
// err.
func failingToolboxMock(
	t *testing.T,
	failing string,
	err error,
) source.Runner {
	m, _ := toolboxExec(t)
	answers := map[string]string{
		"-v":                    "12.3.0 build-21581411\n",
		"stat hosttime":         "01 Jan 2026 12:00:00\n",
		"stat speed":            "2600 MHz\n",
		"stat sessionid":        "1\n",
		"stat balloon":          "0 MB\n",
		"stat swap":             "0 MB\n",
		"stat memlimit":         "4096 MB\n",
		"stat memres":           "0 MB\n",
		"stat cpures":           "0 MHz\n",
		"stat cpulimit":         "unlimited\n",
		"upgrade status":        "VMware Tools are up-to-date\n",
		"timesync status":       "Timesync is disabled\n",
		"stat raw json session": `{"version":"7.0.3"}`,
	}
	for args, out := range answers {
		call := m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, strings.Fields(args)).AnyTimes()
		if args == failing {
			call.Return(nil, err)
			continue
		}
		call.Return([]byte(out), nil)
	}
	return m
}

func (s *VMwarePublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(vmware.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c vmware.Collector) {
				_, ok := c.(*vmware.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c vmware.Collector) {
				_, ok := c.(*vmware.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c vmware.Collector) {
				_, ok := c.(*vmware.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c vmware.Collector) {
				_, ok := c.(*vmware.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := vmware.New()
			s.Equal("vmware", c.Name())
			s.Equal("virtualization", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *VMwarePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		setupLinux   func() *vmware.Linux
		setupDarwin  func() *vmware.Darwin
		validateFunc func(any, error)
	}{
		// ----- Linux -----
		{
			name:    "linux: vmware SCSI in /proc/scsi/scsi + tools cmd",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile(
					"/proc/scsi/scsi",
					[]byte(
						"Host: scsi0 Channel: 00 Id: 00 Lun: 00\n  Vendor: VMware   Model: Virtual disk\n",
					),
					fs.FileMode(0o444),
				)
				return &vmware.Linux{FS: f, Exec: fullToolboxMock(s.T())}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("12.3.0 build-21581411", i.Version)
				s.Equal("vmware_desktop", i.HostType)
				s.Equal("01 Jan 2026 12:00:00", i.Hosttime)
			},
		},
		{
			name:    "linux: vSphere session JSON sets host_type and host_version",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi",
					[]byte("VMware SCSI\n"),
					fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: vsphereMock(s.T())}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("vmware_vsphere", i.HostType)
				s.Equal("7.0.3", i.HostVersion)
			},
		},
		{
			name:    "linux: /proc/scsi/scsi missing but toolbox-cmd responds",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("12.3.0", i.Version)
				s.Equal("vmware_desktop", i.HostType)
			},
		},
		{
			name:    "linux: UpdateInfo failed filtered from stat output",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
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
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware\n"), fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Empty(i.Hosttime)
			},
		},
		{
			name:    "linux: invalid vSphere session JSON still sets host_type",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
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
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware\n"), fs.FileMode(0o444))
				return &vmware.Linux{FS: f, Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("vmware_vsphere", i.HostType)
				s.Empty(i.HostVersion)
			},
		},
		{
			name:    "linux: no VMware signals returns nil",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, fmt.Errorf("exec: %w", exec.ErrNotFound)).AnyTimes()
				return &vmware.Linux{FS: memfs.New(), Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "linux: a VMware controller but no executor, no fact",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware SCSI\n"), fs.FileMode(0o444))
				return &vmware.Linux{FS: f}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "linux: /proc/scsi/scsi that cannot be read is an error",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi/scsi", 0o755)
				return &vmware.Linux{FS: f, Exec: fullToolboxMock(s.T())}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.ErrorContains(err, "read /proc/scsi/scsi")
			},
		},
		{
			name:    "linux: vmware-toolbox-cmd -v exits with an error, no fact",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, &exec.ExitError{}).AnyTimes()
				return &vmware.Linux{FS: memfs.New(), Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "linux: vmware-toolbox-cmd -v that does not finish is an error",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				ctrl := gomock.NewController(s.T())
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, context.DeadlineExceeded).AnyTimes()
				return &vmware.Linux{FS: memfs.New(), Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.EqualError(err, "vmware-toolbox-cmd -v: context deadline exceeded")
			},
		},
		{
			name:    "linux: a raw session query that does not finish is an error, not a desktop guest",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware SCSI\n"), fs.FileMode(0o444))
				return &vmware.Linux{
					FS: f,
					Exec: failingToolboxMock(
						s.T(),
						"stat raw json session",
						context.DeadlineExceeded,
					),
				}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.EqualError(
					err,
					"vmware-toolbox-cmd stat raw json session: context deadline exceeded",
				)
			},
		},
		{
			name:    "linux: a raw session query that exits with an error is a desktop guest",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware SCSI\n"), fs.FileMode(0o444))
				return &vmware.Linux{
					FS:   f,
					Exec: failingToolboxMock(s.T(), "stat raw json session", &exec.ExitError{}),
				}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("vmware_desktop", i.HostType)
				s.Empty(i.HostVersion)
				s.Equal("2600 MHz", i.Speed)
			},
		},
		{
			name:    "linux: a stat that does not finish is an error",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				f := memfs.New()
				_ = f.MkdirAll("/proc/scsi", 0o755)
				_ = f.WriteFile("/proc/scsi/scsi", []byte("VMware SCSI\n"), fs.FileMode(0o444))
				return &vmware.Linux{
					FS:   f,
					Exec: failingToolboxMock(s.T(), "stat speed", errors.New("signal: killed")),
				}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.EqualError(err, "vmware-toolbox-cmd stat speed: signal: killed")
			},
		},
		{
			name:    "linux: nil Exec and no /proc/scsi/scsi returns nil",
			variant: "linux",
			setupLinux: func() *vmware.Linux {
				return &vmware.Linux{FS: memfs.New(), Exec: nil}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		// ----- Darwin -----
		{
			name:    "darwin: toolbox-cmd present returns Info",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				return &vmware.Darwin{Exec: fullToolboxMock(s.T())}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Equal("12.3.0 build-21581411", i.Version)
				s.Equal("vmware_desktop", i.HostType)
			},
		},
		{
			name:    "darwin: stat command error leaves field empty",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte("12.3.0\n"), nil).AnyTimes()
				// speed exits with an error, as a stat the guest does not
				// offer does, so its field is empty.
				m.EXPECT().
					Execute(gomock.Any(), vmwareToolboxPath, "stat", "hosttime").
					Return([]byte(""), nil).
					AnyTimes()
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "stat", "speed").
					Return(nil, &exec.ExitError{}).AnyTimes()
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*vmware.Info)
				s.Require().True(ok)
				s.Empty(i.Speed)
				s.Equal("vmware_desktop", i.HostType)
			},
		},
		{
			name:    "darwin: toolbox-cmd absent returns nil",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, fmt.Errorf("exec: %w", exec.ErrNotFound)).AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "darwin: toolbox-cmd returns empty output — nil",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return([]byte{}, nil).AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "darwin: vmware-toolbox-cmd -v that does not finish is an error",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				ctrl := gomock.NewController(s.T())
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().Execute(gomock.Any(), vmwareToolboxPath, "-v").
					Return(nil, context.DeadlineExceeded).AnyTimes()
				return &vmware.Darwin{Exec: m}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.EqualError(err, "vmware-toolbox-cmd -v: context deadline exceeded")
			},
		},
		{
			name:    "darwin: a stat that does not finish is an error",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				return &vmware.Darwin{
					Exec: failingToolboxMock(s.T(), "stat hosttime", context.DeadlineExceeded),
				}
			},
			validateFunc: func(got any, err error) {
				s.Nil(got)
				s.EqualError(err, "vmware-toolbox-cmd stat hosttime: context deadline exceeded")
			},
		},
		{
			name:    "darwin: nil Exec returns nil",
			variant: "darwin",
			setupDarwin: func() *vmware.Darwin {
				return &vmware.Darwin{Exec: nil}
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c vmware.Collector
			switch tt.variant {
			case "linux":
				c = tt.setupLinux()
			case "darwin":
				c = tt.setupDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
