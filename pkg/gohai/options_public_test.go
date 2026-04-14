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

type OptionsPublicTestSuite struct {
	suite.Suite
}

func TestOptionsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OptionsPublicTestSuite))
}

func (s *OptionsPublicTestSuite) TestNew() {
	tests := []struct {
		name    string
		opts    []gohai.Option
		wantErr bool
	}{
		{"defaults", nil, false},
		{"with disabled", []gohai.Option{gohai.WithDisabled("platform")}, false},
		{"with collectors only", []gohai.Option{gohai.WithCollectors("platform")}, false},
		{"unknown enabled errors", []gohai.Option{gohai.WithEnabled("nope")}, true},
		{"unknown disabled errors", []gohai.Option{gohai.WithDisabled("nope")}, true},
		{"unknown only errors", []gohai.Option{gohai.WithCollectors("nope")}, true},
		{"with category cloud", []gohai.Option{gohai.WithCategory("cloud")}, false},
		{
			"with category hardware stacks",
			[]gohai.Option{gohai.WithCategory("hardware", "cloud")},
			false,
		},
		{"unknown category errors", []gohai.Option{gohai.WithCategory("nope")}, true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(tt.opts...)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotNil(g)
		})
	}
}
