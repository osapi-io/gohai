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

package cmd_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type CollectorPublicTestSuite struct {
	suite.Suite
}

func TestCollectorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CollectorPublicTestSuite))
}

func (s *CollectorPublicTestSuite) TestWriteProblems() {
	problems := []gohai.Problem{{
		Collector: "bad", Rule: "name", Message: "is not lowercase.", Fix: "rename it.",
	}}
	tests := []struct {
		name         string
		asJSON       bool
		validateFunc func(out string, err error)
	}{
		{
			name: "a problem prints its fix and fails",
			validateFunc: func(out string, err error) {
				s.Error(err)
				s.Equal("bad [name]: is not lowercase. Fix: rename it.\n", out)
			},
		},
		{
			name:   "a problem as JSON fails",
			asJSON: true,
			validateFunc: func(out string, err error) {
				s.Error(err)
				s.Contains(out, `"collector": "bad"`)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer
			err := cmd.WriteProblems(&out, problems, tt.asJSON)
			tt.validateFunc(out.String(), err)
		})
	}
}
