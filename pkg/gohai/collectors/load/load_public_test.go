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

package load_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
)

var (
	_ collector.Collector = (*load.Linux)(nil)
	_ collector.Collector = (*load.Darwin)(nil)
)

type LoadPublicTestSuite struct {
	suite.Suite
}

func TestLoadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LoadPublicTestSuite))
}

func (s *LoadPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(load.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c load.Collector) {
				_, ok := c.(*load.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c load.Collector) {
				_, ok := c.(*load.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c load.Collector) {
				_, ok := c.(*load.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c load.Collector) {
				_, ok := c.(*load.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c load.Collector) {
				_, ok := c.(*load.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := load.New()
			s.Equal("load", c.Name())
			s.Equal("misc", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *LoadPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		runner  sourcetest.Commands
		want    *load.Info
		wantErr string
	}{
		{
			name:    "linux: /proc/loadavg",
			variant: "linux",
			files:   map[string]string{"/proc/loadavg": "0.25 0.50 1.00 1/412 12345\n"},
			want:    &load.Info{One: 0.25, Five: 0.5, Fifteen: 1.0},
		},
		{name: "linux: no /proc/loadavg", variant: "linux", wantErr: "read /proc/loadavg"},
		{
			name: "linux: too few fields", variant: "linux",
			files: map[string]string{"/proc/loadavg": "0.25 0.50"}, wantErr: "no load averages",
		},
		{
			name: "linux: a field that is not a number", variant: "linux",
			files: map[string]string{"/proc/loadavg": "0.25 x 1.00"}, wantErr: `load average "x"`,
		},
		{
			name:    "darwin: vm.loadavg",
			variant: "darwin",
			runner: sourcetest.Commands{
				Out: map[string]string{"sysctl -n vm.loadavg": "{ 1.50 2.00 2.50 }\n"},
			},
			want: &load.Info{One: 1.5, Five: 2.0, Fifteen: 2.5},
		},
		{
			name:    "darwin: sysctl fails",
			variant: "darwin",
			runner: sourcetest.Commands{
				Err: map[string]error{"sysctl -n vm.loadavg": errors.New("boom")},
			},
			wantErr: "sysctl vm.loadavg: boom",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c load.Collector
			switch tt.variant {
			case "linux":
				c = &load.Linux{FS: sourcetest.FS(s.T(), tt.files)}
			case "darwin":
				c = &load.Darwin{Exec: tt.runner.Mock(s.T())}
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
