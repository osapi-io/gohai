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

package cpu_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

var (
	_ collector.Collector = (*cpu.Linux)(nil)
	_ collector.Collector = (*cpu.Darwin)(nil)
)

type CPUPublicTestSuite struct {
	suite.Suite
}

func TestCPUPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPUPublicTestSuite))
}

func (s *CPUPublicTestSuite) TestNew() {
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
			c := cpu.New()
			s.Equal("cpu", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*cpu.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			}
		})
	}
}

// TestCollectOnHost exercises the real gopsutil-backed readCPU
// bridge. gopsutil's InfoStat values aren't unit-constructable.
func (s *CPUPublicTestSuite) TestCollectOnHost() {
	tests := []struct{ name string }{
		{name: "host reports non-zero logical CPU count"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := cpu.NewLinux()
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*cpu.Info)
			s.Require().True(ok)
			s.Greater(info.Count, 0)
		})
	}
}
