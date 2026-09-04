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

	gpload "github.com/shirou/gopsutil/v4/load"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
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
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", "debian", osLinux},
		{"rhel dispatches to Linux", "rhel", osLinux},
		{"arch dispatches to Linux", "arch", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := load.New()
			s.Equal("load", c.Name())
			s.Equal("misc", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*load.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*load.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *LoadPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		fn      func(context.Context) (*gpload.AvgStat, error)
		wantErr bool
		want    load.Info
	}{
		{
			name:    "linux: averages returned",
			variant: osLinux,
			fn: func(context.Context) (*gpload.AvgStat, error) {
				return &gpload.AvgStat{Load1: 0.25, Load5: 0.5, Load15: 1.0}, nil
			},
			want: load.Info{One: 0.25, Five: 0.5, Fifteen: 1.0},
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: osLinux,
			fn:      func(context.Context) (*gpload.AvgStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
		{
			name:    "darwin: averages returned",
			variant: osDarwin,
			fn: func(context.Context) (*gpload.AvgStat, error) {
				return &gpload.AvgStat{Load1: 1.5, Load5: 2.0, Load15: 2.5}, nil
			},
			want: load.Info{One: 1.5, Five: 2.0, Fifteen: 2.5},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: osDarwin,
			fn:      func(context.Context) (*gpload.AvgStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer load.SetAvgFn(tt.fn)()
			var c load.Collector
			switch tt.variant {
			case osLinux:
				c = &load.Linux{}
			case osDarwin:
				c = &load.Darwin{}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*load.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
