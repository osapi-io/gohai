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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type RegistryPublicTestSuite struct {
	suite.Suite
	reg *collector.Registry
}

func TestRegistryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

func (s *RegistryPublicTestSuite) SetupTest() {
	s.reg = collector.NewRegistry()
}

type fakeCollector struct {
	name           string
	category       string
	defaultEnabled bool
	deps           []string
}

func (f *fakeCollector) Name() string {
	return f.name
}

func (f *fakeCollector) Category() string {
	if f.category == "" {
		return "misc"
	}
	return f.category
}

func (f *fakeCollector) DefaultEnabled() bool {
	return f.defaultEnabled
}

func (f *fakeCollector) Dependencies() []string {
	return f.deps
}

func (f *fakeCollector) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return f.name + "-result", nil
}

type errCollector struct {
	name string
}

func (e *errCollector) Name() string {
	return e.name
}

func (e *errCollector) Category() string {
	return "misc"
}

func (e *errCollector) DefaultEnabled() bool {
	return true
}

func (e *errCollector) Dependencies() []string {
	return nil
}

func (e *errCollector) Collect(
	_ context.Context,
	_ collector.PriorResults,
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
			collector: &fakeCollector{name: "alpha", defaultEnabled: true},
			wantErr:   false,
		},
		{
			name:      "rejects empty name",
			collector: &fakeCollector{name: "", defaultEnabled: true},
			wantErr:   true,
		},
		{
			name:      "rejects duplicate registration",
			collector: &fakeCollector{name: "dup", defaultEnabled: true},
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
					NoError(reg.Register(&fakeCollector{name: tt.lookup, defaultEnabled: true}))
			}
			_, ok := reg.Get(tt.lookup)
			s.Equal(tt.wantOK, ok)
		})
	}
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", category: "cloud"}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", category: "cloud"}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "c", category: "system"}))

	tests := []struct {
		name     string
		category string
		want     []string
	}{
		{"multiple collectors in category", "cloud", []string{"a", "b"}},
		{"single collector in category", "system", []string{"c"}},
		{"unknown category returns empty", "missing", []string{}},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := s.reg.NamesInCategory(tt.category)
			sort.Strings(got)
			s.Equal(tt.want, got)
		})
	}
}

func (s *RegistryPublicTestSuite) TestGetDep() {
	prior := collector.PriorResults{
		"typed":    "hello",
		"wrong":    42,
		"nil-slot": nil,
	}

	tests := []struct {
		name    string
		lookup  string
		wantOK  bool
		wantVal string // only checked when wantOK is true
	}{
		{
			name:    "matching type returns the value",
			lookup:  "typed",
			wantOK:  true,
			wantVal: "hello",
		},
		{
			name:   "missing key returns ok=false",
			lookup: "missing",
		},
		{
			name:   "type mismatch returns ok=false",
			lookup: "wrong",
		},
		{
			name:   "nil-valued any does not type-assert",
			lookup: "nil-slot",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := collector.GetDep[string](prior, tt.lookup)
			s.Equal(tt.wantOK, ok)
			if tt.wantOK {
				s.Equal(tt.wantVal, got)
			}
		})
	}
}

func (s *RegistryPublicTestSuite) TestNames() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", defaultEnabled: true}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", defaultEnabled: true}))
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
				NoError(reg.Register(&fakeCollector{name: "core1", defaultEnabled: true}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "core2", defaultEnabled: true}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "ext", defaultEnabled: true}))
			s.Require().
				NoError(reg.Register(&fakeCollector{name: "opt", defaultEnabled: false}))

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
		name            string
		setup           func(reg *collector.Registry)
		names           []string
		hooks           func(mu *sync.Mutex, onErr *[]string, onComp *[]string) collector.Hooks
		wantResults     []string
		wantErrNames    []string
		wantCompleteAll []string
		wantErr         bool
	}{
		{
			name: "orders by dependency",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", defaultEnabled: true}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", defaultEnabled: true, deps: []string{"a"}}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "c", defaultEnabled: true, deps: []string{"b"}}))
			},
			names:       []string{"a", "b", "c"},
			wantResults: []string{"a", "b", "c"},
		},
		{
			name: "auto-includes dependencies",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", defaultEnabled: false}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", defaultEnabled: false, deps: []string{"a"}}))
			},
			names:       []string{"b"},
			wantResults: []string{"a", "b"},
		},
		{
			name: "detects cycle",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", defaultEnabled: true, deps: []string{"b"}}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "b", defaultEnabled: true, deps: []string{"a"}}))
			},
			names:   []string{"a", "b"},
			wantErr: true,
		},
		{
			name: "missing dependency errors",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "a", defaultEnabled: true, deps: []string{"missing"}}))
			},
			names:   []string{"a"},
			wantErr: true,
		},
		{
			name: "collector error omits from results",
			setup: func(reg *collector.Registry) {
				s.Require().NoError(reg.Register(&errCollector{name: "bad"}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "good", defaultEnabled: true}))
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
		{
			name: "zero-value hooks tolerates error without handler",
			setup: func(reg *collector.Registry) {
				s.Require().NoError(reg.Register(&errCollector{name: "bad"}))
			},
			names: []string{"bad"},
			hooks: func(*sync.Mutex, *[]string, *[]string) collector.Hooks {
				return collector.Hooks{}
			},
			wantResults: nil, // "bad" drops silently
		},
		{
			name: "OnComplete fires for every collector (success and failure)",
			setup: func(reg *collector.Registry) {
				s.Require().NoError(reg.Register(&errCollector{name: "bad"}))
				s.Require().
					NoError(reg.Register(&fakeCollector{name: "good", defaultEnabled: true}))
			},
			names: []string{"bad", "good"},
			hooks: func(
				mu *sync.Mutex,
				onErr *[]string,
				onComp *[]string,
			) collector.Hooks {
				return collector.Hooks{
					OnError: func(n string, _ error) {
						mu.Lock()
						defer mu.Unlock()
						*onErr = append(*onErr, n)
					},
					OnComplete: func(n string, _ time.Duration, _ error) {
						mu.Lock()
						defer mu.Unlock()
						*onComp = append(*onComp, n)
					},
				}
			},
			wantResults:     []string{"good"},
			wantErrNames:    []string{"bad"},
			wantCompleteAll: []string{"bad", "good"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			tt.setup(reg)

			var (
				mu                      sync.Mutex
				errNames, completeNames []string
			)
			hooks := collector.Hooks{
				OnError: func(n string, _ error) {
					mu.Lock()
					defer mu.Unlock()
					errNames = append(errNames, n)
				},
			}
			if tt.hooks != nil {
				hooks = tt.hooks(&mu, &errNames, &completeNames)
			}
			results, err := reg.Run(context.Background(), tt.names, hooks)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			mu.Lock()
			defer mu.Unlock()
			for _, name := range tt.wantResults {
				s.Contains(results, name)
			}
			for _, name := range tt.wantErrNames {
				s.Contains(errNames, name)
			}
			for _, name := range tt.wantCompleteAll {
				s.Contains(completeNames, name)
			}
		})
	}
}
