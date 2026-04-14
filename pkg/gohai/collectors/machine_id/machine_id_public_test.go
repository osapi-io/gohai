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

package machineid_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
)

var (
	_ collector.Collector = (*machineid.Linux)(nil)
	_ collector.Collector = (*machineid.Darwin)(nil)
)

type MachineIDPublicTestSuite struct {
	suite.Suite
}

func TestMachineIDPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MachineIDPublicTestSuite))
}

func (s *MachineIDPublicTestSuite) TestNew() {
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
			c := machineid.New()
			s.Equal("machine_id", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*machineid.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*machineid.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *MachineIDPublicTestSuite) TestCollect() {
	const dbusPath = "/var/lib/dbus/machine-id"
	tests := []struct {
		name    string
		variant string
		hostFn  func(context.Context) (*host.InfoStat, error)
		dbus    string // empty → file absent
		wantErr bool
		wantID  string
	}{
		{
			name:    "linux: gopsutil returns /etc/machine-id → use it",
			variant: "linux",
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: "gopsutil-id"}, nil
			},
			wantID: "gopsutil-id",
		},
		{
			name:    "linux: gopsutil empty, dbus fallback wins",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return &host.InfoStat{}, nil },
			dbus:    "dbus-id\n",
			wantID:  "dbus-id",
		},
		{
			name:    "linux: gopsutil empty, dbus missing → empty ID (no error)",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return &host.InfoStat{}, nil },
			wantID:  "",
		},
		{
			name:    "linux: gopsutil empty, dbus whitespace-only → empty ID",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return &host.InfoStat{}, nil },
			dbus:    "   \n",
			wantID:  "",
		},
		{
			name:    "linux: gopsutil nil info returns empty",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, nil },
			wantID:  "",
		},
		{
			name:    "linux: gopsutil error wrapped and returned",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
		{
			name:    "darwin: gopsutil returns IOPlatformUUID",
			variant: "darwin",
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: "iokit-uuid-1234"}, nil
			},
			wantID: "iokit-uuid-1234",
		},
		{
			name:    "darwin: nil info returns empty",
			variant: "darwin",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, nil },
			wantID:  "",
		},
		{
			name:    "darwin: gopsutil error wrapped and returned",
			variant: "darwin",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer machineid.SetHostInfoFn(tt.hostFn)()
			var c machineid.Collector
			switch tt.variant {
			case "linux":
				var vfs avfs.VFS = memfs.New()
				if tt.dbus != "" {
					f := memfs.New()
					_ = f.MkdirAll("/var/lib/dbus", 0o755)
					_ = f.WriteFile(dbusPath, []byte(tt.dbus), fs.FileMode(0o644))
					vfs = f
				}
				c = &machineid.Linux{FS: vfs}
			case "darwin":
				c = &machineid.Darwin{}
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*machineid.Info)
			s.Require().True(ok)
			s.Equal(tt.wantID, info.ID)
		})
	}
}
