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

package gohai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cloud/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware/dmi"
)

type GohaiTestSuite struct {
	suite.Suite
}

func TestGohaiTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GohaiTestSuite))
}

type cycleCollector struct {
	name string
	deps []string
}

func (c *cycleCollector) Name() string {
	return c.name
}

func (c *cycleCollector) Category() string {
	return "misc"
}

func (c *cycleCollector) DefaultEnabled() bool {
	return true
}

func (c *cycleCollector) Dependencies() []string {
	return c.deps
}

func (c *cycleCollector) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return nil, nil
}

func (s *GohaiTestSuite) TestFactsSet() {
	tests := []struct {
		name   string
		key    string
		value  any
		verify func(s *GohaiTestSuite, f *Facts)
	}{
		{
			name:  "gce populates Facts.Gce",
			key:   "gce",
			value: &gce.Info{Name: "vm-1"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Gce)
				s.Equal("vm-1", f.Gce.Name)
			},
		},
		{
			name:  "dmi populates Facts.DMI",
			key:   "dmi",
			value: &dmi.Info{Product: &dmi.Product{Name: "Google Compute Engine"}},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.DMI)
				s.Equal("Google Compute Engine", f.DMI.Product.Name)
			},
		},
		{
			name:  "mismatched type for gce is silently ignored",
			key:   "gce",
			value: "not-gce-info",
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.Gce)
			},
		},
		{
			name:  "mismatched type for dmi is silently ignored",
			key:   "dmi",
			value: 42,
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.DMI)
			},
		},
		{
			name:  "unknown key is a no-op",
			key:   "nope",
			value: &dmi.Info{},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.DMI)
				s.Nil(f.Gce)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := &Facts{}
			f.set(tt.key, tt.value)
			tt.verify(s, f)
		})
	}
}

func (s *GohaiTestSuite) TestFactsCountPopulated() {
	tests := []struct {
		name  string
		facts *Facts
		want  int
	}{
		{"empty", &Facts{}, 0},
		{"gce only", &Facts{Gce: &gce.Info{}}, 1},
		{"dmi only", &Facts{DMI: &dmi.Info{}}, 1},
		{"gce + dmi", &Facts{Gce: &gce.Info{}, DMI: &dmi.Info{}}, 2},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.facts.countPopulated())
		})
	}
}

func (s *GohaiTestSuite) TestCollectPropagatesRunError() {
	reg := collector.NewRegistry()
	s.Require().NoError(reg.Register(&cycleCollector{name: "a", deps: []string{"b"}}))
	s.Require().NoError(reg.Register(&cycleCollector{name: "b", deps: []string{"a"}}))

	g := &Gohai{registry: reg}
	sel, err := reg.Selected(nil, nil)
	s.Require().NoError(err)
	g.selected = sel

	_, err = g.Collect(context.Background())
	s.Error(err)
}
