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
		name          string
		collector     collector.Collector
		registerTwice bool
		validateFunc  func(*collector.Registry, collector.Collector, error)
	}{
		{
			name:      "registers a new collector",
			collector: newCollector(s.ctrl, "alpha", "", true),
			validateFunc: func(reg *collector.Registry, c collector.Collector, err error) {
				s.NoError(err)
				got, ok := reg.Get(c.Name())
				s.True(ok)
				s.Equal(c, got)
			},
		},
		{
			name:      "rejects empty name",
			collector: newCollector(s.ctrl, "", "", true),
			validateFunc: func(_ *collector.Registry, _ collector.Collector, err error) {
				s.Error(err)
			},
		},
		{
			name:          "rejects duplicate registration",
			collector:     newCollector(s.ctrl, "dup", "", true),
			registerTwice: true,
			validateFunc: func(_ *collector.Registry, _ collector.Collector, err error) {
				s.Error(err)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			if tt.registerTwice {
				s.Require().NoError(reg.Register(tt.collector))
			}

			tt.validateFunc(reg, tt.collector, reg.Register(tt.collector))
		})
	}
}

func (s *RegistryPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		register     bool
		lookup       string
		validateFunc func(bool)
	}{
		{
			name:     "registered collector found",
			register: true,
			lookup:   "known",
			validateFunc: func(ok bool) {
				s.True(ok)
			},
		},
		{
			name:     "missing collector not found",
			register: false,
			lookup:   "missing",
			validateFunc: func(ok bool) {
				s.False(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			if tt.register {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, tt.lookup, "", true)))
			}
			_, ok := reg.Get(tt.lookup)

			tt.validateFunc(ok)
		})
	}
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "a", "cloud", false)))
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "b", "cloud", false)))
	s.Require().NoError(s.reg.Register(newCollector(s.ctrl, "c", "system", false)))

	tests := []struct {
		name         string
		category     string
		validateFunc func([]string)
	}{
		{
			name:     "multiple collectors in category",
			category: "cloud",
			validateFunc: func(got []string) {
				s.Equal([]string{"a", "b"}, got)
			},
		},
		{
			name:     "single collector in category",
			category: "system",
			validateFunc: func(got []string) {
				s.Equal([]string{"c"}, got)
			},
		},
		{
			name:     "unknown category returns empty",
			category: "missing",
			validateFunc: func(got []string) {
				s.Equal([]string{}, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := s.reg.NamesInCategory(tt.category)
			sort.Strings(got)

			tt.validateFunc(got)
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
		name         string
		lookup       string
		validateFunc func(string, bool)
	}{
		{
			name:   "matching type returns the value",
			lookup: "typed",
			validateFunc: func(got string, ok bool) {
				s.True(ok)
				s.Equal("hello", got)
			},
		},
		{
			name:   "missing key returns ok=false",
			lookup: "missing",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:   "type mismatch returns ok=false",
			lookup: "wrong",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:   "nil-valued any does not type-assert",
			lookup: "nil-slot",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.GetDep[string](prior, tt.lookup))
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
		name         string
		enable       []string
		disable      []string
		unknown      bool
		validateFunc func([]collector.Collector, error)
	}{
		{
			name: "defaults: core+extended on, opt-in off",
			validateFunc: func(got []collector.Collector, err error) {
				s.Require().NoError(err)
				names := make([]string, 0, len(got))
				for _, c := range got {
					names = append(names, c.Name())
				}
				sort.Strings(names)
				s.Equal([]string{"core1", "core2", "ext"}, names)
			},
		},
		{
			name:    "disable a default-on collector",
			disable: []string{"core1"},
			validateFunc: func(got []collector.Collector, err error) {
				s.Require().NoError(err)
				names := make([]string, 0, len(got))
				for _, c := range got {
					names = append(names, c.Name())
				}
				sort.Strings(names)
				s.Equal([]string{"core2", "ext"}, names)
			},
		},
		{
			name:   "enable an opt-in collector",
			enable: []string{"opt"},
			validateFunc: func(got []collector.Collector, err error) {
				s.Require().NoError(err)
				names := make([]string, 0, len(got))
				for _, c := range got {
					names = append(names, c.Name())
				}
				sort.Strings(names)
				s.Equal([]string{"core1", "core2", "ext", "opt"}, names)
			},
		},
		{
			name:    "disable wins over enable for same name",
			enable:  []string{"opt"},
			disable: []string{"opt"},
			validateFunc: func(got []collector.Collector, err error) {
				s.Require().NoError(err)
				names := make([]string, 0, len(got))
				for _, c := range got {
					names = append(names, c.Name())
				}
				sort.Strings(names)
				s.Equal([]string{"core1", "core2", "ext"}, names)
			},
		},
		{
			name:   "unknown in enable list errors",
			enable: []string{"missing"},
			validateFunc: func(_ []collector.Collector, err error) {
				s.Error(err)
			},
		},
		{
			name:    "unknown in disable list errors",
			disable: []string{"missing"},
			validateFunc: func(_ []collector.Collector, err error) {
				s.Error(err)
			},
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

			tt.validateFunc(reg.Selected(tt.enable, tt.disable))
		})
	}
}

func (s *RegistryPublicTestSuite) TestRun() {
	tests := []struct {
		name         string
		setup        func(reg *collector.Registry)
		names        []string
		hooks        func(mu *sync.Mutex, onErr *[]string, onComp *[]string) collector.Hooks
		validateFunc func(map[string]any, []string, []string, error)
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
			names: []string{"a", "b", "c"},
			validateFunc: func(results map[string]any, _, _ []string, err error) {
				s.Require().NoError(err)
				s.Contains(results, "a")
				s.Contains(results, "b")
				s.Contains(results, "c")
			},
		},
		{
			name: "auto-includes dependencies",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", false)))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "b", "", false, "a")))
			},
			names: []string{"b"},
			validateFunc: func(results map[string]any, _, _ []string, err error) {
				s.Require().NoError(err)
				s.Contains(results, "a")
				s.Contains(results, "b")
			},
		},
		{
			name: "detects cycle",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", true, "b")))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "b", "", true, "a")))
			},
			names: []string{"a", "b"},
			validateFunc: func(_ map[string]any, _, _ []string, err error) {
				s.Error(err)
			},
		},
		{
			name: "missing dependency errors",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "a", "", true, "missing")))
			},
			names: []string{"a"},
			validateFunc: func(_ map[string]any, _, _ []string, err error) {
				s.Error(err)
			},
		},
		{
			name: "collector error omits from results",
			setup: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newFailingCollector(s.ctrl, "bad", errors.New("boom"))))
				s.Require().
					NoError(reg.Register(newCollector(s.ctrl, "good", "", true)))
			},
			names: []string{"bad", "good"},
			validateFunc: func(results map[string]any, errNames, _ []string, err error) {
				s.Require().NoError(err)
				s.Contains(results, "good")
				s.Contains(errNames, "bad")
			},
		},
		{
			name: "unknown collector errors",
			setup: func(_ *collector.Registry) {
			},
			names: []string{"missing"},
			validateFunc: func(_ map[string]any, _, _ []string, err error) {
				s.Error(err)
			},
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
			validateFunc: func(_ map[string]any, _, _ []string, err error) {
				s.Require().NoError(err)
			},
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
			validateFunc: func(results map[string]any, errNames, completeNames []string, err error) {
				s.Require().NoError(err)
				s.Contains(results, "good")
				s.Contains(errNames, "bad")
				s.Contains(completeNames, "bad")
				s.Contains(completeNames, "good")
			},
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

			mu.Lock()
			defer mu.Unlock()

			tt.validateFunc(results, errNames, completeNames, err)
		})
	}
}
