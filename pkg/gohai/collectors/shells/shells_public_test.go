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

package shells_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

// Compile-time assertions that each per-OS struct satisfies
// collector.Collector.
var (
	_ collector.Collector = (*shells.Linux)(nil)
	_ collector.Collector = (*shells.Darwin)(nil)
)

type ShellsPublicTestSuite struct {
	suite.Suite
}

func TestShellsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsPublicTestSuite))
}

// TestNew covers the factory: identity methods inherited from base
// (Name / DefaultEnabled / Dependencies) plus dispatch to the right
// concrete type per platform.Detect() value.
func (s *ShellsPublicTestSuite) TestNew() {
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
			c := shells.New()
			s.Equal("shells", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*shells.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*shells.Linux)
				s.True(ok)
			}
		})
	}
}

// TestNewLinuxWiresFS verifies NewLinux() returns a Linux with a
// non-nil FS. No real filesystem reads — just the wiring.
func (s *ShellsPublicTestSuite) TestNewLinuxWiresFS() {
	c := shells.NewLinux()
	s.NotNil(c.FS)
}

// TestNewDarwinWiresFS verifies NewDarwin() returns a Darwin with a
// non-nil FS.
func (s *ShellsPublicTestSuite) TestNewDarwinWiresFS() {
	c := shells.NewDarwin()
	s.NotNil(c.FS)
}
