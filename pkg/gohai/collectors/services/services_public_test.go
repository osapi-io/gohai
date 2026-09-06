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
					{Name: "ssh", State: "running", Enabled: true},
					{Name: "cron", State: "running", Enabled: true},
					{Name: "NetworkManager", State: "dead", Enabled: false},
				}, info.Services)
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
			exec:    systemctlExec(s.T(), []byte(""), nil),
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*services.Info)
				s.Require().True(ok)
				s.Equal([]services.Service{
					{Name: "ssh", State: "running", Enabled: true},
				}, info.Services)
			},
		},
		{
			name:    "linux: short field lines skipped",
			variant: "linux",
			exec:    systemctlExec(s.T(), []byte("ssh.service loaded\n"), nil),
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
