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

package gohai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai"
)

// failingCollector is a test double that always returns a canned error
// from Collect. Used to exercise the error-path branch of
// Facts.Timings without depending on a real collector's upstream seam.
type failingCollector struct {
	name string
	err  error
}

func (f *failingCollector) Name() string           { return f.name }
func (f *failingCollector) Category() string       { return "misc" }
func (f *failingCollector) DefaultEnabled() bool   { return false }
func (f *failingCollector) Dependencies() []string { return nil }
func (f *failingCollector) Collect(context.Context, collector.PriorResults) (any, error) {
	return nil, f.err
}

var _ collector.Collector = (*failingCollector)(nil)

type GohaiPublicTestSuite struct {
	suite.Suite
}

func TestGohaiPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GohaiPublicTestSuite))
}

func (s *GohaiPublicTestSuite) TestCollect() {
	tests := []struct {
		name           string
		opts           []gohai.Option
		injectFailing  bool
		wantPlatform   bool
		wantHostname   bool
		wantTimings    bool
		wantFailingErr bool
	}{
		{
			name:         "no opts collects nothing",
			opts:         nil,
			wantPlatform: false,
			wantHostname: false,
		},
		{
			name:         "WithDefaults collects platform and hostname",
			opts:         []gohai.Option{gohai.WithDefaults()},
			wantPlatform: true,
			wantHostname: true,
		},
		{
			name:         "WithDefaults + WithDisabled subtracts platform",
			opts:         []gohai.Option{gohai.WithDefaults(), gohai.WithDisabled("platform")},
			wantPlatform: false,
			wantHostname: true,
		},
		{
			name:         "WithEnabled platform without defaults still collects platform",
			opts:         []gohai.Option{gohai.WithEnabled("platform")},
			wantPlatform: true,
			wantHostname: false,
		},
		{
			name:         "only platform",
			opts:         []gohai.Option{gohai.WithCollectors("platform")},
			wantPlatform: true,
			wantHostname: false,
		},
		{
			name: "WithTimings populates Facts.Timings per collector",
			opts: []gohai.Option{
				gohai.WithCollectors("platform"),
				gohai.WithTimings(),
			},
			wantPlatform: true,
			wantTimings:  true,
		},
		{
			name:         "without WithTimings Facts.Timings is nil",
			opts:         []gohai.Option{gohai.WithCollectors("platform")},
			wantPlatform: true,
			wantTimings:  false,
		},
		{
			name: "failed collector drops from typed output, surfaces in Timings",
			opts: []gohai.Option{
				gohai.WithTimings(),
			},
			injectFailing:  true,
			wantTimings:    true,
			wantFailingErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(tt.opts...)
			s.Require().NoError(err)
			if tt.injectFailing {
				s.Require().NoError(
					g.Register(&failingCollector{
						name: "failtest",
						err:  errors.New("simulated collector failure"),
					}),
				)
				s.Require().NoError(g.Select("failtest"))
			}
			facts, err := g.Collect(context.Background())
			s.Require().NoError(err)
			s.Require().NotNil(facts)
			s.Equal(tt.wantPlatform, facts.Platform != nil)
			s.Equal(tt.wantHostname, facts.Hostname != nil)
			if tt.wantTimings {
				s.Require().NotNil(facts.Timings)
				s.Require().NotEmpty(facts.Timings.Collectors)
				if tt.wantFailingErr {
					entry, ok := facts.Timings.Collectors["failtest"]
					s.Require().True(ok)
					s.Equal("err", entry.Status)
					s.Contains(entry.Error, "simulated collector failure")
					s.Greater(entry.DurationNs, int64(0))
				} else {
					entry, ok := facts.Timings.Collectors["platform"]
					s.Require().True(ok)
					s.Equal("ok", entry.Status)
					s.Empty(entry.Error)
					s.Greater(entry.DurationNs, int64(0))
				}
			} else {
				s.Nil(facts.Timings)
			}
		})
	}
}
