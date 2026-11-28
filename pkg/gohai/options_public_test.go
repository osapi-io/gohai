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

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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
		name         string
		opts         []gohai.Option
		builtins     func(source.Sources) []collector.Collector
		validateFunc func(any, error)
	}{
		{
			name: "a built-in collector that does not register errors",
			builtins: func(src source.Sources) []collector.Collector {
				return []collector.Collector{platform.NewWith(src), platform.NewWith(src)}
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, `register built-in collector "platform"`)
			},
		},
		{
			name: "defaults",
			opts: nil,
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with disabled",
			opts: []gohai.Option{gohai.WithDisabled("platform")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with collectors only",
			opts: []gohai.Option{gohai.WithCollectors("platform")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "unknown enabled errors",
			opts: []gohai.Option{gohai.WithEnabled("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "unknown disabled errors",
			opts: []gohai.Option{gohai.WithDisabled("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "with collectors and disabled together errors",
			opts: []gohai.Option{
				gohai.WithCollectors("platform"),
				gohai.WithDisabled("hostname"),
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "WithCollectors and WithDisabled together")
			},
		},
		{
			name: "unknown only errors",
			opts: []gohai.Option{gohai.WithCollectors("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "with category cloud",
			opts: []gohai.Option{gohai.WithCategory("cloud")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with category hardware stacks",
			opts: []gohai.Option{gohai.WithCategory("hardware", "cloud")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "unknown category errors",
			opts: []gohai.Option{gohai.WithCategory("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.builtins != nil {
				defer gohai.SetBuiltinCollectorsFn(tt.builtins)()
			}
			tt.validateFunc(gohai.New(tt.opts...))
		})
	}
}
