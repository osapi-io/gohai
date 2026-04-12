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

package collector_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type RegistryPublicTestSuite struct {
	suite.Suite
	reg *collector.Registry
}

func TestRegistryPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

func (s *RegistryPublicTestSuite) SetupTest() {
	s.reg = collector.NewRegistry()
}

type fakeCollector struct {
	name string
	tier collector.Tier
	deps []string
}

func (f *fakeCollector) Name() string {
	return f.name
}

func (f *fakeCollector) Tier() collector.Tier {
	return f.tier
}

func (f *fakeCollector) Dependencies() []string {
	return f.deps
}

func (f *fakeCollector) Collect(
	_ context.Context,
) (any, error) {
	return f.name + "-result", nil
}

type errCollector struct {
	name string
}

func (e *errCollector) Name() string {
	return e.name
}

func (e *errCollector) Tier() collector.Tier {
	return collector.TierCore
}

func (e *errCollector) Dependencies() []string {
	return nil
}

func (e *errCollector) Collect(
	_ context.Context,
) (any, error) {
	return nil, errors.New("boom")
}

func (s *RegistryPublicTestSuite) TestRegister() {
	tests := []struct {
		name      string
		collector collector.Collector
		wantErr   bool
	}{
		{
			name:      "registers a new collector",
			collector: &fakeCollector{name: "alpha", tier: collector.TierCore},
			wantErr:   false,
		},
		{
			name:      "rejects empty name",
			collector: &fakeCollector{name: "", tier: collector.TierCore},
			wantErr:   true,
		},
		{
			name:      "rejects duplicate registration",
			collector: &fakeCollector{name: "dup", tier: collector.TierCore},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			if tt.name == "rejects duplicate registration" {
				s.Require().NoError(reg.Register(tt.collector))
			}
			err := reg.Register(tt.collector)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			got, ok := reg.Get(tt.collector.Name())
			s.True(ok)
			s.Equal(tt.collector, got)
		})
	}
}

func (s *RegistryPublicTestSuite) TestGet() {
	tests := []struct {
		name     string
		register bool
		lookup   string
		wantOK   bool
	}{
		{"registered collector found", true, "known", true},
		{"missing collector not found", false, "missing", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			if tt.register {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: tt.lookup, tier: collector.TierCore}))
			}
			_, ok := reg.Get(tt.lookup)
			s.Equal(tt.wantOK, ok)
		})
	}
}

func (s *RegistryPublicTestSuite) TestNames() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", tier: collector.TierCore}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: collector.TierCore}))
	names := s.reg.Names()
	sort.Strings(names)
	s.Equal([]string{"a", "b"}, names)
}

func (s *RegistryPublicTestSuite) TestSelected() {
	tests := []struct {
		name    string
		enable  []string
		disable []string
		unknown bool
		want    []string
		wantErr bool
	}{
		{
			name: "defaults: core+extended on, opt-in off",
			want: []string{"core1", "core2", "ext"},
		},
		{
			name:    "disable a default-on collector",
			disable: []string{"core1"},
			want:    []string{"core2", "ext"},
		},
		{
			name:   "enable an opt-in collector",
			enable: []string{"opt"},
			want:   []string{"core1", "core2", "ext", "opt"},
		},
		{
			name:    "disable wins over enable for same name",
			enable:  []string{"opt"},
			disable: []string{"opt"},
			want:    []string{"core1", "core2", "ext"},
		},
		{
			name:    "unknown in enable list errors",
			enable:  []string{"missing"},
			wantErr: true,
		},
		{
			name:    "unknown in disable list errors",
			disable: []string{"missing"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "core1", tier: collector.TierCore}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "core2", tier: collector.TierCore}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "ext", tier: collector.TierExtended}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "opt", tier: collector.TierOptIn}))

			got, err := reg.Selected(tt.enable, tt.disable)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			names := make([]string, 0, len(got))
			for _, c := range got {
				names = append(names, c.Name())
			}
			sort.Strings(names)
			s.Equal(tt.want, names)
		})
	}
}

func (s *RegistryPublicTestSuite) TestRun() {
	tests := []struct {
		name         string
		setup        func(reg *collector.Registry)
		names        []string
		wantResults  []string
		wantErrNames []string
		wantErr      bool
	}{
		{
			name: "orders by dependency",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", tier: collector.TierCore}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", tier: collector.TierCore, deps: []string{"a"}}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "c", tier: collector.TierCore, deps: []string{"b"}}))
			},
			names:       []string{"a", "b", "c"},
			wantResults: []string{"a", "b", "c"},
		},
		{
			name: "auto-includes dependencies",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", tier: collector.TierOptIn}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", tier: collector.TierOptIn, deps: []string{"a"}}))
			},
			names:       []string{"b"},
			wantResults: []string{"a", "b"},
		},
		{
			name: "detects cycle",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", tier: collector.TierCore, deps: []string{"b"}}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", tier: collector.TierCore, deps: []string{"a"}}))
			},
			names:   []string{"a", "b"},
			wantErr: true,
		},
		{
			name: "missing dependency errors",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", tier: collector.TierCore, deps: []string{"missing"}}))
			},
			names:   []string{"a"},
			wantErr: true,
		},
		{
			name: "collector error omits from results",
			setup: func(reg *collector.Registry) {
				s.Require().NoError(reg.Register(&errCollector{name: "bad"}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "good", tier: collector.TierCore}))
			},
			names:        []string{"bad", "good"},
			wantResults:  []string{"good"},
			wantErrNames: []string{"bad"},
		},
		{
			name: "unknown collector errors",
			setup: func(_ *collector.Registry) {
			},
			names:   []string{"missing"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			tt.setup(reg)

			var errNames []string
			results, err := reg.Run(context.Background(), tt.names, func(n string, _ error) {
				errNames = append(errNames, n)
			})
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			for _, name := range tt.wantResults {
				s.Contains(results, name)
			}
			for _, name := range tt.wantErrNames {
				s.Contains(errNames, name)
			}
		})
	}
}

func (s *RegistryPublicTestSuite) TestRunErrorsWithoutHandler() {
	s.Require().NoError(s.reg.Register(&errCollector{name: "bad"}))
	results, err := s.reg.Run(context.Background(), []string{"bad"}, nil)
	s.Require().NoError(err)
	s.NotContains(results, "bad")
}
