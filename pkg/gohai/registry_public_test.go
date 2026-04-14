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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type RegistryPublicTestSuite struct {
	suite.Suite
}

func TestRegistryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

func (s *RegistryPublicTestSuite) TestNewRegistry() {
	reg := gohai.NewRegistry()
	s.Contains(reg.Names(), "platform")
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	reg := gohai.NewRegistry()
	tests := []struct {
		name      string
		category  string
		contains  string // expected member of the category; empty when wantEmpty
		wantEmpty bool
	}{
		{"cloud contains gce", "cloud", "gce", false},
		{"hardware contains dmi", "hardware", "dmi", false},
		{"unknown category returns empty", "nope", "", true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := reg.NamesInCategory(tt.category)
			if tt.wantEmpty {
				s.Empty(got)
				return
			}
			s.Contains(got, tt.contains)
		})
	}
}

func (s *RegistryPublicTestSuite) TestCategoryOf() {
	reg := gohai.NewRegistry()
	tests := []struct {
		name      string
		collector string
		want      string
	}{
		{"gce is cloud", "gce", "cloud"},
		{"dmi is hardware", "dmi", "hardware"},
		{"missing collector returns empty", "nope", ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, reg.CategoryOf(tt.collector))
		})
	}
}
