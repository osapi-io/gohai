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
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type GohaiPublicTestSuite struct {
	suite.Suite
}

func TestGohaiPublicTestSuite(t *testing.T) {
	suite.Run(t, new(GohaiPublicTestSuite))
}

func (s *GohaiPublicTestSuite) TestNew() {
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

func (s *GohaiPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		opts       []gohai.Option
		wantKey    string
		notWantKey string
	}{
		{
			name:    "default collects platform",
			wantKey: "platform",
		},
		{
			name:       "disabled platform is absent",
			opts:       []gohai.Option{gohai.WithDisabled("platform")},
			notWantKey: "platform",
		},
		{
			name:    "only platform",
			opts:    []gohai.Option{gohai.WithCollectors("platform")},
			wantKey: "platform",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(tt.opts...)
			s.Require().NoError(err)
			facts, err := g.Collect(context.Background())
			s.Require().NoError(err)
			s.NotNil(facts)
			if tt.wantKey != "" {
				s.Contains(facts.Data, tt.wantKey)
			}
			if tt.notWantKey != "" {
				s.NotContains(facts.Data, tt.notWantKey)
			}
		})
	}
}

func (s *GohaiPublicTestSuite) TestNewRegistry() {
	reg := gohai.NewRegistry()
	s.Contains(reg.Names(), "platform")
}
