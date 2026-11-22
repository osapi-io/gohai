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

package services_test

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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/services"
)

var (
	_ collector.Collector = (*services.Linux)(nil)
	_ collector.Collector = (*services.Darwin)(nil)
)

// systemctlOut is a representative `systemctl list-units --type=service
// --all --no-pager --plain` output fragment.
const systemctlOut = `UNIT                        LOAD   ACTIVE   SUB     DESCRIPTION
ssh.service                 loaded active   running OpenSSH server daemon
cron.service                loaded active   running Regular background program
NetworkManager.service      loaded inactive dead    Network Manager

LOAD   = Reflects whether the unit definition was properly loaded.
`

// unitFilesOut is list-unit-files: ssh starts at boot, cron is enabled at
// runtime only, and NetworkManager is disabled although it is installed.
const unitFilesOut = `ssh.service                 enabled         enabled
cron.service                enabled-runtime enabled
NetworkManager.service      disabled        enabled
getty@.service              static          -
not-a-unit
`

type ServicesPublicTestSuite struct {
	suite.Suite
}

func TestServicesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServicesPublicTestSuite))
}

func systemctlExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	return systemctlExecWith(t, out, err, []byte(unitFilesOut), nil)
}

// systemctlExecWith answers list-units with out and err, and
// list-unit-files with files and filesErr.
func systemctlExecWith(
	t *testing.T,
	out []byte,
	err error,
	files []byte,
	filesErr error,
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().Execute(
		gomock.Any(),
		"systemctl",
		"list-units",
		"--type=service",
		"--all",
		"--no-pager",
		"--plain",
	).Return(out, err).AnyTimes()
	m.EXPECT().Execute(
		gomock.Any(),
		"systemctl",
		"list-unit-files",
		"--type=service",
		"--no-legend",
		"--no-pager",
	).Return(files, filesErr).AnyTimes()
	return m
}

func (s *ServicesPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(services.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c services.Collector) {
				_, ok := c.(*services.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c services.Collector) {
				_, ok := c.(*services.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c services.Collector) {
				_, ok := c.(*services.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c services.Collector) {
				_, ok := c.(*services.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c services.Collector) {
				_, ok := c.(*services.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := services.New()
			s.Equal("services", c.Name())
			s.Equal("software", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *ServicesPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		exec         executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: systemctl returns services",
			variant: "linux",
			exec:    systemctlExec(s.T(), []byte(systemctlOut), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{
					{
						Name:          "ssh",
						State:         "running",
						Enabled:       true,
						UnitFileState: "enabled",
						Description:   "OpenSSH server daemon",
					},
					{
						Name:          "cron",
						State:         "running",
						Enabled:       true,
						UnitFileState: "enabled-runtime",
						Description:   "Regular background program",
					},
					{
						Name:          "NetworkManager",
						State:         "dead",
						Enabled:       false,
						UnitFileState: "disabled",
						Description:   "Network Manager",
					},
				}, info.Services)
			},
		},
		{
			name:    "linux: a unit that systemd has not loaded is in the list, from its unit file",
			variant: "linux",
			exec: systemctlExecWith(s.T(),
				[]byte("ssh.service loaded active running OpenSSH server daemon\n"), nil,
				[]byte("ssh.service enabled enabled\nlabcheck.service disabled enabled\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{
					{
						Name:          "ssh",
						State:         "running",
						Enabled:       true,
						UnitFileState: "enabled",
						Description:   "OpenSSH server daemon",
					},
					{Name: "labcheck", State: "dead", Enabled: false, UnitFileState: "disabled"},
				}, info.Services)
			},
		},
		{
			name:    "linux: an alias, a template and a unit listed twice are in the list once",
			variant: "linux",
			exec: systemctlExecWith(s.T(), []byte(""), nil,
				[]byte("dbus-org.freedesktop.resolve1.service alias -\n"+
					"getty@.service static -\n"+
					"masked.service masked enabled\n"+
					"masked.service masked enabled\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{
					{Name: "masked", State: "dead", Enabled: false, UnitFileState: "masked"},
				}, info.Services)
			},
		},
		{
			name:    "linux: a unit that runs is not enabled unless its unit file is",
			variant: "linux",
			exec: systemctlExecWith(s.T(), []byte(systemctlOut), nil,
				[]byte("ssh.service disabled enabled\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.False(info.Services[0].Enabled, "ssh runs but is disabled")
			},
		},
		{
			name:    "linux: list-unit-files fails, no unit is enabled",
			variant: "linux",
			exec: systemctlExecWith(s.T(), []byte(systemctlOut), nil,
				nil, errors.New("boom")),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				for _, svc := range info.Services {
					s.False(svc.Enabled, svc.Name)
				}
				s.Equal("OpenSSH server daemon", info.Services[0].Description)
			},
		},
		{
			name:    "linux: systemctl fails, empty list",
			variant: "linux",
			exec:    systemctlExec(s.T(), nil, errors.New("not found")),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{}, info.Services)
			},
		},
		{
			name:    "linux: nil Exec returns empty list",
			variant: "linux",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{}, info.Services)
			},
		},
		{
			name:    "linux: empty output returns empty list",
			variant: "linux",
			exec:    systemctlExecWith(s.T(), []byte(""), nil, nil, nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{}, info.Services)
			},
		},
		{
			name:    "linux: lines without .service are skipped",
			variant: "linux",
			exec: systemctlExec(
				s.T(),
				[]byte("not-a-service-line\nssh.service loaded active running OpenSSH\n"),
				nil,
			),
			// The unit files of the other tests add cron and NetworkManager.
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{
					{
						Name:          "ssh",
						State:         "running",
						Enabled:       true,
						UnitFileState: "enabled",
						Description:   "OpenSSH",
					},
					{Name: "cron", State: "dead", Enabled: true, UnitFileState: "enabled-runtime"},
					{
						Name:          "NetworkManager",
						State:         "dead",
						Enabled:       false,
						UnitFileState: "disabled",
					},
				}, info.Services, "getty@ is a template, so it is not in the list")
			},
		},
		{
			name:    "linux: short field lines skipped",
			variant: "linux",
			exec:    systemctlExecWith(s.T(), []byte("ssh.service loaded\n"), nil, nil, nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{}, info.Services)
			},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service(nil), info.Services)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c services.Collector
			switch tt.variant {
			case "linux":
				c = &services.Linux{Exec: tt.exec}
			case "darwin":
				c = services.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
