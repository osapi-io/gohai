// Copyright (c) 2024 John Dewey

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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type FactsPublicTestSuite struct {
	suite.Suite
	facts *gohai.Facts
}

func TestFactsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FactsPublicTestSuite))
}

func (s *FactsPublicTestSuite) SetupTest() {
	s.facts = &gohai.Facts{
		Data: map[string]any{
			"platform": map[string]any{"name": "ubuntu", "version": "24.04"},
			"cpu":      map[string]any{"total": 8},
		},
		CollectTime:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		CollectDuration: 100 * time.Millisecond,
	}
}

func (s *FactsPublicTestSuite) TestJSON() {
	tests := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"compact", s.facts.JSON},
		{"pretty", s.facts.PrettyJSON},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			b, err := tt.fn()
			s.Require().NoError(err)
			var got map[string]any
			s.Require().NoError(json.Unmarshal(b, &got))
			s.Contains(got, "platform")
			s.Contains(got, "cpu")
			s.Contains(got, "collect_time")
		})
	}
}

func (s *FactsPublicTestSuite) TestFlat() {
	flat := s.facts.Flat()
	s.Equal("ubuntu", flat["platform.name"])
	s.Equal("24.04", flat["platform.version"])
	s.EqualValues(8, flat["cpu.total"])
}

func (s *FactsPublicTestSuite) TestGet() {
	tests := []struct {
		path string
		want any
	}{
		{"platform.name", "ubuntu"},
		{"cpu.total", float64(8)},
		{"missing", nil},
	}
	for _, tt := range tests {
		s.Run(tt.path, func() {
			s.Equal(tt.want, s.facts.Get(tt.path))
		})
	}
}

func (s *FactsPublicTestSuite) TestString() {
	s.Contains(s.facts.String(), "Facts{")
}
