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
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

type RegistryPublicTestSuite struct {
	suite.Suite
}

func TestRegistryPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

func (s *RegistryPublicTestSuite) TestNewRegistry() {
	tests := []struct {
		name         string
		builtins     func(source.Sources) []collector.Collector
		validateFunc func(gohai.Registry, error)
	}{
		{
			name: "has the built-in collectors",
			validateFunc: func(reg gohai.Registry, err error) {
				s.Require().NoError(err)
				s.Contains(reg.Names(), "platform")
			},
		},
		{
			name: "the names of the collectors are sorted",
			validateFunc: func(reg gohai.Registry, err error) {
				s.Require().NoError(err)
				s.True(slices.IsSorted(reg.Names()))
			},
		},
		{
			name: "a built-in collector that does not register is an error",
			builtins: func(src source.Sources) []collector.Collector {
				return []collector.Collector{platform.NewWith(src), platform.NewWith(src)}
			},
			validateFunc: func(_ gohai.Registry, err error) {
				s.ErrorContains(err, `register built-in collector "platform"`)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.builtins != nil {
				defer gohai.SetBuiltinCollectorsFn(tt.builtins)()
			}
			tt.validateFunc(gohai.NewRegistry())
		})
	}
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	reg, err := gohai.NewRegistry()
	s.Require().NoError(err)
	tests := []struct {
		name         string
		category     string
		validateFunc func([]string)
	}{
		{
			name:     "cloud contains gce",
			category: "cloud",
			validateFunc: func(got []string) {
				s.Contains(got, "gce")
			},
		},
		{
			name:     "hardware contains dmi",
			category: "hardware",
			validateFunc: func(got []string) {
				s.Contains(got, "dmi")
			},
		},
		{
			name:     "unknown category returns empty",
			category: "nope",
			validateFunc: func(got []string) {
				s.Empty(got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(reg.NamesInCategory(tt.category))
		})
	}
}

func (s *RegistryPublicTestSuite) TestCategoryOf() {
	reg, err := gohai.NewRegistry()
	s.Require().NoError(err)
	tests := []struct {
		name         string
		collector    string
		validateFunc func(string)
	}{
		{
			name:      "gce is cloud",
			collector: "gce",
			validateFunc: func(got string) {
				s.Equal("cloud", got)
			},
		},
		{
			name:      "dmi is hardware",
			collector: "dmi",
			validateFunc: func(got string) {
				s.Equal("hardware", got)
			},
		},
		{
			name:      "missing collector returns empty",
			collector: "nope",
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(reg.CategoryOf(tt.collector))
		})
	}
}
