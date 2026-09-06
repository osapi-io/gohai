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

package sysconf_test

import (
	"context"
	"errors"
	"testing"

	gosysconf "github.com/tklauser/go-sysconf"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysconf"
)

var (
	_ collector.Collector = (*sysconf.Linux)(nil)
	_ collector.Collector = (*sysconf.Darwin)(nil)
)

type SysconfPublicTestSuite struct {
	suite.Suite
}

func TestSysconfPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysconfPublicTestSuite))
}

func (s *SysconfPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(sysconf.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c sysconf.Collector) {
				_, ok := c.(*sysconf.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c sysconf.Collector) {
				_, ok := c.(*sysconf.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c sysconf.Collector) {
				_, ok := c.(*sysconf.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c sysconf.Collector) {
				_, ok := c.(*sysconf.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c sysconf.Collector) {
				_, ok := c.(*sysconf.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := sysconf.New()
			s.Equal("sysconf", c.Name())
			s.Equal("misc", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SysconfPublicTestSuite) TestCollect() {
	// stubFn returns a func(int)(int64,error) that maps SC_* constants
	// to the provided values in order: CLK_TCK, PAGESIZE, NPROCESSORS_CONF,
	// NPROCESSORS_ONLN.
	stubFn := func(
		clkTck, pagesize, nconf, nonln int64,
	) func(int) (int64, error) {
		return func(name int) (int64, error) {
			switch name {
			case gosysconf.SC_CLK_TCK:
				return clkTck, nil
			case gosysconf.SC_PAGESIZE:
				return pagesize, nil
			case gosysconf.SC_NPROCESSORS_CONF:
				return nconf, nil
			case gosysconf.SC_NPROCESSORS_ONLN:
				return nonln, nil
			default:
				return 0, errors.New("unknown constant")
			}
		}
	}

	// errorAt returns a func that errors on the n-th call (0-indexed by SC_* order).
	errorAt := func(failOn int) func(int) (int64, error) {
		calls := 0
		return func(_ int) (int64, error) {
			result := calls
			calls++
			if result == failOn {
				return 0, errors.New("sysconf error")
			}
			return 100, nil
		}
	}

	tests := []struct {
		name         string
		variant      string
		stub         func(int) (int64, error)
		validateFunc func(any, error)
	}{
		{
			name:    "linux: all values returned",
			variant: "linux",
			stub:    stubFn(100, 4096, 4, 4),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysconf.Info)
				s.Require().True(ok)
				s.Equal(sysconf.Info{
					ClkTck:          100,
					Pagesize:        4096,
					NprocessorsConf: 4,
					NprocessorsOnln: 4,
				}, *info)
			},
		},
		{
			name:    "linux: CLK_TCK error propagated",
			variant: "linux",
			stub:    errorAt(0),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: PAGESIZE error propagated",
			variant: "linux",
			stub:    errorAt(1),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: NPROCESSORS_CONF error propagated",
			variant: "linux",
			stub:    errorAt(2),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: NPROCESSORS_ONLN error propagated",
			variant: "linux",
			stub:    errorAt(3),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: all values returned",
			variant: "darwin",
			stub:    stubFn(100, 16384, 8, 8),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*sysconf.Info)
				s.Require().True(ok)
				s.Equal(sysconf.Info{
					ClkTck:          100,
					Pagesize:        16384,
					NprocessorsConf: 8,
					NprocessorsOnln: 8,
				}, *info)
			},
		},
		{
			name:    "darwin: CLK_TCK error propagated",
			variant: "darwin",
			stub:    errorAt(0),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := sysconf.SetSysconfFn(tt.stub)
			defer restore()

			var c sysconf.Collector
			switch tt.variant {
			case "linux":
				c = sysconf.NewLinux()
			case "darwin":
				c = sysconf.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
