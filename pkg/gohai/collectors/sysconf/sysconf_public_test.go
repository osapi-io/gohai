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
	"testing"

	"github.com/osapi-io/gohai/internal/sourcetest"

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
	with := func(key, value string) map[string]string {
		out := map[string]string{}
		for k, v := range all {
			out[k] = v
		}
		if value == "" {
			delete(out, key)
		} else {
			out[key] = value
		}
		return out
	}
	tests := []struct {
		name    string
		variant string
		out     map[string]string
		wantErr string
	}{
		{name: "linux: the four values", variant: "linux", out: all},
		{name: "darwin: the four values", variant: "darwin", out: all},
		{
			name: "linux: getconf fails", variant: "linux",
			out: with("getconf PAGESIZE", ""), wantErr: "getconf PAGESIZE",
		},
		{
			name:    "linux: not a number",
			variant: "linux",
			out: with(
				"getconf _NPROCESSORS_ONLN",
				"undefined\n",
			),
			wantErr: "getconf _NPROCESSORS_ONLN",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			r := sourcetest.Commands{Out: tt.out}
			var c sysconf.Collector = &sysconf.Linux{Exec: r.Mock(s.T())}
			if tt.variant == "darwin" {
				c = &sysconf.Darwin{Exec: r.Mock(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(
				&sysconf.Info{ClkTck: 100, Pagesize: 4096, NprocessorsConf: 8, NprocessorsOnln: 6},
				got,
			)
		})
	}
}
