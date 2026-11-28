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

package pattern_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/pattern"
)

type PatternPublicTestSuite struct {
	suite.Suite
}

func (s *PatternPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		expr         string
		validateFunc func(*pattern.Pattern)
	}{
		{
			name: "a good expression compiles and matches",
			expr: `^a+$`,
			validateFunc: func(p *pattern.Pattern) {
				s.True(p.MatchString("aaa"))
			},
		},
		{
			name: "a bad expression gives a pattern that matches nothing",
			expr: `(`,
			validateFunc: func(p *pattern.Pattern) {
				s.False(p.MatchString("("))
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr))
		})
	}
}

func (s *PatternPublicTestSuite) TestErrors() {
	tests := []struct {
		name         string
		expr         string
		validateFunc func(error)
	}{
		{
			name: "a bad expression is reported with its text",
			expr: `[z-a]`,
			validateFunc: func(err error) {
				s.Require().Error(err)
				s.ErrorContains(err, `pattern "[z-a]"`)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			pattern.New(tt.expr)
			tt.validateFunc(pattern.Errors())
		})
	}
}

func (s *PatternPublicTestSuite) TestMatchString() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got bool)
	}{
		{
			name: "a match",
			expr: `b`,
			in:   "abc",
			validateFunc: func(got bool) {
				s.True(got)
			},
		},
		{
			name: "no match",
			expr: `z`,
			in:   "abc",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a bad expression matches nothing",
			expr: `(`,
			in:   "(",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).MatchString(tt.in))
		})
	}
}

func (s *PatternPublicTestSuite) TestFindStringSubmatch() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got []string)
	}{
		{
			name: "the match and its group",
			expr: `a(\d)`,
			in:   "xa7",
			validateFunc: func(got []string) {
				s.Equal([]string{"a7", "7"}, got)
			},
		},
		{
			name: "a bad expression finds nothing",
			expr: `(`,
			in:   "(",
			validateFunc: func(got []string) {
				s.Nil(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).FindStringSubmatch(tt.in))
		})
	}
}

func (s *PatternPublicTestSuite) TestFindSubmatch() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got [][]byte)
	}{
		{
			name: "the match and its group",
			expr: `a(\d)`,
			in:   "xa7",
			validateFunc: func(got [][]byte) {
				s.Equal([][]byte{[]byte("a7"), []byte("7")}, got)
			},
		},
		{
			name: "a bad expression finds nothing",
			expr: `(`,
			in:   "(",
			validateFunc: func(got [][]byte) {
				s.Nil(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).FindSubmatch([]byte(tt.in)))
		})
	}
}

func (s *PatternPublicTestSuite) TestFindAllString() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got []string)
	}{
		{
			name: "each match",
			expr: `\d`,
			in:   "a1b2",
			validateFunc: func(got []string) {
				s.Equal([]string{"1", "2"}, got)
			},
		},
		{
			name: "a bad expression finds nothing",
			expr: `(`,
			in:   "(",
			validateFunc: func(got []string) {
				s.Nil(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).FindAllString(tt.in, -1))
		})
	}
}

func (s *PatternPublicTestSuite) TestFindAllStringSubmatch() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got [][]string)
	}{
		{
			name: "each match and its group",
			expr: `(\d)`,
			in:   "a1b2",
			validateFunc: func(got [][]string) {
				s.Equal([][]string{{"1", "1"}, {"2", "2"}}, got)
			},
		},
		{
			name: "a bad expression finds nothing",
			expr: `(`,
			in:   "(",
			validateFunc: func(got [][]string) {
				s.Nil(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).FindAllStringSubmatch(tt.in, -1))
		})
	}
}

func (s *PatternPublicTestSuite) TestReplaceAllString() {
	tests := []struct {
		name         string
		expr         string
		in           string
		validateFunc func(got string)
	}{
		{
			name: "each match is replaced",
			expr: `\d`,
			in:   "a1b2",
			validateFunc: func(got string) {
				s.Equal("a_b_", got)
			},
		},
		{
			name: "a bad expression leaves the text as it is",
			expr: `(`,
			in:   "a1b2",
			validateFunc: func(got string) {
				s.Equal("a1b2", got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(pattern.New(tt.expr).ReplaceAllString(tt.in, "_"))
		})
	}
}

func TestPatternPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PatternPublicTestSuite))
}
