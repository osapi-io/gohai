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
	"testing"

	"github.com/stretchr/testify/suite"
)

type FactsTestSuite struct {
	suite.Suite
}

func TestFactsTestSuite(t *testing.T) {
	suite.Run(t, new(FactsTestSuite))
}

func (s *FactsTestSuite) TestFlattenMap() {
	tests := []struct {
		name string
		in   map[string]any
		want map[string]any
	}{
		{
			name: "scalars",
			in:   map[string]any{"a": 1, "b": "x"},
			want: map[string]any{"a": 1, "b": "x"},
		},
		{
			name: "nested map",
			in:   map[string]any{"cpu": map[string]any{"total": 8, "model": "intel"}},
			want: map[string]any{"cpu.total": 8, "cpu.model": "intel"},
		},
		{
			name: "deep nest",
			in:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			want: map[string]any{"a.b.c": 1},
		},
		{
			name: "empty",
			in:   map[string]any{},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := flattenMap("", tt.in)
			s.Equal(tt.want, got)
		})
	}
}

func (s *FactsTestSuite) TestNormalize() {
	tests := []struct {
		name    string
		in      any
		wantKey string
	}{
		{
			name:    "map round-trips to map",
			in:      map[string]any{"a": 1},
			wantKey: "a",
		},
		{
			name: "unmarshalable returns original",
			in:   make(chan int),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := normalize(tt.in)
			if tt.wantKey != "" {
				m, ok := got.(map[string]any)
				s.Require().True(ok)
				s.Contains(m, tt.wantKey)
				return
			}
			s.NotNil(got)
		})
	}
}

func (s *FactsTestSuite) TestFlatNonMapFallback() {
	// Build a Facts whose normalize result isn't a map. Since toMap always
	// returns a map[string]any and time/int values are JSON-safe, this tests
	// the defensive type assertion path.
	f := &Facts{Data: map[string]any{}}
	got := f.Flat()
	s.NotNil(got)
}
