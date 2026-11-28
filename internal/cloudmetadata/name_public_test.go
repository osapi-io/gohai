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

package cloudmetadata_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
)

type NamePublicTestSuite struct {
	suite.Suite
}

func TestNamePublicTestSuite(t *testing.T) {
	suite.Run(t, new(NamePublicTestSuite))
}

func (s *NamePublicTestSuite) TestSafeName() {
	tests := []struct {
		name         string
		in           string
		validateFunc func(got bool)
	}{
		{
			name: "a plain child name is safe",
			in:   "ephemeral0",
			validateFunc: func(got bool) {
				s.True(got)
			},
		},
		{
			name: "a MAC with a trailing directory slash is safe",
			in:   "0e:1a:2b:3c:4d:5e/",
			validateFunc: func(got bool) {
				s.True(got)
			},
		},
		{
			name: "a name with underscore, dot and dash is safe",
			in:   "instance-id_v1.2",
			validateFunc: func(got bool) {
				s.True(got)
			},
		},
		{
			name: "a parent-directory name is not safe",
			in:   "..",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a parent-directory name with a slash is not safe",
			in:   "../",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a current-directory name is not safe",
			in:   ".",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a name with a path separator is not safe",
			in:   "a/b",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a name with a query is not safe",
			in:   "root?x=1",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a name with a fragment is not safe",
			in:   "root#x",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "a name with a space is not safe",
			in:   "a b",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
		{
			name: "an empty name is not safe",
			in:   "",
			validateFunc: func(got bool) {
				s.False(got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(cloudmetadata.SafeName(tt.in))
		})
	}
}
