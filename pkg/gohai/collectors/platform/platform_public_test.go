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

package platform_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	plat "github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

func (s *PlatformPublicTestSuite) TestNew() {
	c := platform.New()
	s.Equal("platform", c.Name())
	s.True(c.DefaultEnabled())
	s.Empty(c.Dependencies())
}

func (s *PlatformPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = platform.NewLinux()
	var _ collector.Collector = platform.NewDarwin()
}

func (s *PlatformPublicTestSuite) TestNewDispatch() {
	orig := plat.Detect
	defer func() { plat.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin", "darwin", "darwin"},
		{"debian", "debian", "linux"},
		{"rhel", "rhel", "linux"},
		{"arch", "arch", "linux"},
		{"unknown", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			plat.Detect = func() string { return tt.detect }
			got := platform.New()
			switch tt.wantKind {
			case "darwin":
				_, ok := got.(*platform.Darwin)
				s.True(ok)
			case "linux":
				_, ok := got.(*platform.Linux)
				s.True(ok)
			}
		})
	}
}
