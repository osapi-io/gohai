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
	"github.com/osapi-io/gohai/internal/collector/mocks"
	"go.uber.org/mock/gomock"
)

type RegistryPublicTestSuite struct {
	suite.Suite
	reg  *collector.Registry
	ctrl *gomock.Controller
}

func TestRegistryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

func (s *RegistryPublicTestSuite) SetupTest() {
	s.reg = collector.NewRegistry()
	s.ctrl = gomock.NewController(s.T())
}

// newCollector returns a generated Collector mock reporting the given identity.
// Collect succeeds, returning "<name>-result".
func newCollector(
	ctrl *gomock.Controller,
	name string,
	category string,
	defaultEnabled bool,
	deps ...string,
) *mocks.MockCollector {
	if category == "" {
		category = "misc"
	}

	if len(deps) == 0 {
		deps = nil
	}

	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(category).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(defaultEnabled).AnyTimes()
	m.EXPECT().Dependencies().Return(deps).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		Return(name+"-result", nil).
		AnyTimes()

	return m
}

// newFailingCollector is newCollector with Collect returning err.
func newFailingCollector(
	ctrl *gomock.Controller,
	name string,
	err error,
	deps ...string,
) *mocks.MockCollector {
	if len(deps) == 0 {
		deps = nil
	}

	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return("misc").AnyTimes()
	m.EXPECT().DefaultEnabled().Return(true).AnyTimes()
	m.EXPECT().Dependencies().Return(deps).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		Return(nil, err).
		AnyTimes()

	return m
}

func (s *RegistryPublicTestSuite) TestRegister() {
	tests := []struct {
		name      string
		collector collector.Collector
		wantErr   bool
	}{
		{
			name:      "registers a new collector",
			collector: newCollector(s.ctrl, "alpha", "", true),
			wantErr:   false,
		},
		{
			name:      "rejects empty name",
			collector: newCollector(s.ctrl, "", "", true),
			wantErr:   true,
		},
		{
			name:      "rejects duplicate registration",
			collector: newCollector(s.ctrl, "dup", "", true),
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
					NoError(reg.Register(newCollector(s.ctrl, tt.lookup, "", true)))
			}
			_, ok := reg.Get(tt.lookup)
			s.Equal(tt.wantOK, ok)
		})
	}
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "a", "cloud", false)))
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "b", "cloud", false)))
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "c", "system", false)))

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
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "b", "", true)))
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "a", "", true)))
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
				NoError(reg.Register(newCollector(s.ctrl, "core1", "", true)))
			s.Require().
				NoError(reg.Register(newCollector(s.ctrl, "core2", "", true)))
			s.Require().
				NoError(reg.Register(newCollector(s.ctrl, "ext", "", true)))
			s.Require().
				NoError(reg.Register(newCollector(s.ctrl, "opt", "", false)))

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
					NoError(reg.Register(newCollector(s.ctrl, "a", "", true)))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "b", "", true, "a")))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "c", "", true, "b")))
			},
			names:       []string{"a", "b", "c"},
			wantResults: []string{"a", "b", "c"},
		},
		{
			name: "auto-includes dependencies",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", false)))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "b", "", false, "a")))
			},
			names:       []string{"b"},
			wantResults: []string{"a", "b"},
		},
		{
			name: "detects cycle",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", true, "b")))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "b", "", true, "a")))
			},
			names:   []string{"a", "b"},
			wantErr: true,
		},
		{
			name: "missing dependency errors",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", true, "missing")))
			},
			names:   []string{"a"},
			wantErr: true,
		},
		{
			name: "collector error omits from results",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newFailingCollector(s.ctrl, "bad", errors.New("boom"))))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "good", "", true)))
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
				s.Require().
					NoError(reg.Register(newFailingCollector(s.ctrl, "bad", errors.New("boom"))))
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
				s.Require().
					NoError(reg.Register(newFailingCollector(s.ctrl, "bad", errors.New("boom"))))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "good", "", true)))
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
