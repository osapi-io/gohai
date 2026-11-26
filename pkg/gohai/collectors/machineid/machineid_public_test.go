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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/machineid"
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
		name         string
		detect       string
		validateFunc func(machineid.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c machineid.Collector) {
				_, ok := c.(*machineid.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c machineid.Collector) {
				_, ok := c.(*machineid.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c machineid.Collector) {
				_, ok := c.(*machineid.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c machineid.Collector) {
				_, ok := c.(*machineid.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c machineid.Collector) {
				_, ok := c.(*machineid.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := machineid.New()
			s.Equal("machine_id", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *MachineIDPublicTestSuite) TestCollect() {
	const ioregLine = "ioreg -rd1 -c IOPlatformExpertDevice"
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		runner  sourcetest.Commands
		want    *machineid.Info
		wantErr string
	}{
		{
			name: "linux: /etc/machine-id first", variant: "linux",
			files: map[string]string{
				"/etc/machine-id":          "abc123\n",
				"/var/lib/dbus/machine-id": "dbus456\n",
			},
			want: &machineid.Info{ID: "abc123"},
		},
		{
			name: "linux: an empty /etc/machine-id falls to dbus", variant: "linux",
			files: map[string]string{
				"/etc/machine-id":          "\n",
				"/var/lib/dbus/machine-id": "dbus456\n",
			},
			want: &machineid.Info{ID: "dbus456"},
		},
		{
			name: "linux: the DMI product UUID, in lower case", variant: "linux",
			files: map[string]string{"/sys/class/dmi/id/product_uuid": "4C4C4544-0042\n"},
			want:  &machineid.Info{ID: "4c4c4544-0042"},
		},
		{name: "linux: no ID anywhere", variant: "linux", want: &machineid.Info{}},
		{
			name:    "darwin: IOPlatformUUID",
			variant: "darwin",
			runner: sourcetest.Commands{
				Out: map[string]string{ioregLine: "+-o J316sAP  <class IOPlatformExpertDevice>\n" +
					"    {\n      \"IOPlatformUUID\" = \"052A2E50-1749-5FAA-0000-000000000000\"\n    }\n"},
			},
			want: &machineid.Info{ID: "052A2E50-1749-5FAA-0000-000000000000"},
		},
		{
			name: "darwin: no UUID in the output", variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{ioregLine: "{}\n"}},
			want:   &machineid.Info{},
		},
		{
			name: "darwin: ioreg fails", variant: "darwin",
			runner:  sourcetest.Commands{Err: map[string]error{ioregLine: errors.New("boom")}},
			wantErr: "ioreg IOPlatformExpertDevice: boom",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c machineid.Collector
			switch tt.variant {
			case "linux":
				c = &machineid.Linux{FS: sourcetest.FS(s.T(), tt.files)}
			case "darwin":
				c = &machineid.Darwin{Exec: tt.runner.Mock(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
