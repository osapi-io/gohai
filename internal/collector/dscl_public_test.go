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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type DsclPublicTestSuite struct {
	suite.Suite
}

func TestDsclPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DsclPublicTestSuite))
}

func (s *DsclPublicTestSuite) TestDsclPath() {
	tests := []struct {
		name         string
		line         string
		validateFunc func(string, string, bool)
	}{
		{
			name: "a name and one path, with padding",
			line: "john                /Users/john",
			validateFunc: func(name, path string, ok bool) {
				s.True(ok)
				s.Equal("john", name)
				s.Equal("/Users/john", path)
			},
		},
		{
			name: "a path with a space is one value",
			line: "jo      /Users/Jo Smith",
			validateFunc: func(name, path string, ok bool) {
				s.True(ok)
				s.Equal("jo", name)
				s.Equal("/Users/Jo Smith", path)
			},
		},
		{
			name: "a path with a space on another volume is one value",
			line: "jo /Volumes/Macintosh HD/Users/jo",
			validateFunc: func(_, path string, ok bool) {
				s.True(ok)
				s.Equal("/Volumes/Macintosh HD/Users/jo", path)
			},
		},
		{
			name: "two paths keep the first",
			line: "root /var/root /private/var/root",
			validateFunc: func(name, path string, ok bool) {
				s.True(ok)
				s.Equal("root", name)
				s.Equal("/var/root", path)
			},
		},
		{
			name: "a name with no value is not a record",
			line: "guest",
			validateFunc: func(name, path string, ok bool) {
				s.False(ok)
				s.Empty(name)
				s.Empty(path)
			},
		},
		{
			name: "a name with only padding is not a record",
			line: "guest    ",
			validateFunc: func(_, _ string, ok bool) {
				s.False(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.DsclPath(tt.line))
		})
	}
}
