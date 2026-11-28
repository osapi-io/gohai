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

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysconf"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c sysconf.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := sysconf.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = sysconf.New()
			}
			s.Equal("sysconf", c.Name())
			s.Equal("misc", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SysconfPublicTestSuite) TestCollect() {
	all := map[string]string{
		"getconf CLK_TCK":           "100\n",
		"getconf PAGESIZE":          "4096\n",
		"getconf _NPROCESSORS_CONF": "8\n",
		"getconf _NPROCESSORS_ONLN": "6\n",
	}
	full := &sysconf.Info{ClkTck: 100, Pagesize: 4096, NprocessorsConf: 8, NprocessorsOnln: 6}
	tests := []struct {
		name         string
		variant      string
		exec         sourcetest.Commands
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux: the four values",
			variant: "linux",
			exec:    sourcetest.Commands{Out: all},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(full, got)
			},
		},
		{
			name:    "darwin: the four values",
			variant: "darwin",
			exec:    sourcetest.Commands{Out: all},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(full, got)
			},
		},
		{
			name:    "linux: no Exec gives no fact",
			variant: "linux",
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: no Exec gives no fact",
			variant: "darwin",
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: no getconf on the host gives no fact and no error",
			variant: "linux",
			exec:    sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a getconf that fails keeps the other three values",
			variant: "linux",
			exec: sourcetest.Commands{
				Out: all,
				Err: map[string]error{"getconf PAGESIZE": errors.New("signal: killed")},
			},
			validateFunc: func(got any, err error) {
				s.EqualError(err, "getconf PAGESIZE: signal: killed")
				info, ok := got.(*sysconf.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal(int64(100), info.ClkTck)
				s.Equal(int64(0), info.Pagesize)
				s.Equal(int64(6), info.NprocessorsOnln)
			},
		},
		{
			name:    "linux: a value that is not a number keeps the other three",
			variant: "linux",
			exec: sourcetest.Commands{Out: map[string]string{
				"getconf CLK_TCK":           "100\n",
				"getconf PAGESIZE":          "4096\n",
				"getconf _NPROCESSORS_CONF": "8\n",
				"getconf _NPROCESSORS_ONLN": "undefined\n",
			}},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "getconf _NPROCESSORS_ONLN")
				info, ok := got.(*sysconf.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal(int64(8), info.NprocessorsConf)
			},
		},
		{
			name:    "linux: every getconf failing is an error and no fact",
			variant: "linux",
			exec: sourcetest.Commands{Err: map[string]error{
				"getconf CLK_TCK":           errors.New("boom"),
				"getconf PAGESIZE":          errors.New("boom"),
				"getconf _NPROCESSORS_CONF": errors.New("boom"),
				"getconf _NPROCESSORS_ONLN": errors.New("boom"),
			}},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "getconf CLK_TCK: boom")
				s.ErrorContains(err, "getconf _NPROCESSORS_ONLN: boom")
				s.Exactly(nil, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var exec source.Runner = tt.exec.Mock(s.T())
			if tt.nilExec {
				exec = nil
			}
			var c sysconf.Collector = &sysconf.Linux{Exec: exec}
			if tt.variant == "darwin" {
				c = &sysconf.Darwin{Exec: exec}
			}
			got, err := c.Collect(context.Background(), nil)
			tt.validateFunc(got, err)
		})
	}
}
