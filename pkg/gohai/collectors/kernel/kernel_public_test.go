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

package kernel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

var (
	_ collector.Collector = (*kernel.Linux)(nil)
	_ collector.Collector = (*kernel.Darwin)(nil)
)

type KernelPublicTestSuite struct {
	suite.Suite
}

func TestKernelPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelPublicTestSuite))
}

func (s *KernelPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := kernel.New()
			s.Equal("kernel", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*kernel.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*kernel.Linux)
				s.True(ok)
			}
		})
	}
}

// TestCollectOnHost exercises the real syscall.Uname via the wired
// UnameFn. Asserts shape — specific values vary by test host.
func (s *KernelPublicTestSuite) TestCollectOnHost() {
	tests := []struct{ name string }{
		{name: "host reports non-empty uname fields"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := kernel.NewLinux()
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)
			s.NotEmpty(info.Name)
			s.NotEmpty(info.Release)
			s.NotEmpty(info.Machine)
		})
	}
}
