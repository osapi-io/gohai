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

type CollectorPublicTestSuite struct {
	suite.Suite
}

func TestCollectorPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CollectorPublicTestSuite))
}

func (s *CollectorPublicTestSuite) TestTierString() {
	tests := []struct {
		name string
		tier collector.Tier
		want string
	}{
		{"core", collector.TierCore, "core"},
		{"extended", collector.TierExtended, "extended"},
		{"opt-in", collector.TierOptIn, "opt-in"},
		{"unknown", collector.Tier(99), "unknown"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.tier.String())
		})
	}
}

func (s *CollectorPublicTestSuite) TestTierEnabledByDefault() {
	tests := []struct {
		name string
		tier collector.Tier
		want bool
	}{
		{"core enabled", collector.TierCore, true},
		{"extended enabled", collector.TierExtended, true},
		{"opt-in disabled", collector.TierOptIn, false},
		{"unknown disabled", collector.Tier(99), false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.tier.EnabledByDefault())
		})
	}
}
