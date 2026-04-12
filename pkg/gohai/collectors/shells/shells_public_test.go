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

type ShellsPublicTestSuite struct {
	suite.Suite
}

func TestShellsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsPublicTestSuite))
}

// TestNew verifies the factory returns a collector whose interface
// contract matches what the registry expects. The actual OS-specific
// type it returns depends on the test host; on dev machines it will be
// *shells.Linux or *shells.Darwin.
func (s *ShellsPublicTestSuite) TestNew() {
	c := shells.New()
	s.Equal("shells", c.Name())
	s.Equal(true, c.DefaultEnabled())
	s.Empty(c.Dependencies())
}

// TestImplementsCollectorInterface confirms both variants satisfy
// collector.Collector (compile-time check).
func (s *ShellsPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = shells.NewLinux()
	var _ collector.Collector = shells.NewDarwin()
}

// TestNewDispatch verifies every platform-detect branch returns the
// expected concrete type. Stubs platform.Detect directly (a swappable
// var) so this test needs no gopsutil import — the gopsutil dependency
// stays sealed inside internal/platform.
func (s *ShellsPublicTestSuite) TestNewDispatch() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string // "linux" or "darwin"
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux (no shells-specific debian variant)", "debian", "linux"},
		{"rhel dispatches to Linux (no shells-specific rhel variant)", "rhel", "linux"},
		{"arch dispatches to Linux (generic)", "arch", "linux"},
		{"empty (unknown) dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			got := shells.New()
			switch tt.wantKind {
			case "darwin":
				_, ok := got.(*shells.Darwin)
				s.True(ok, "expected *shells.Darwin")
			case "linux":
				_, ok := got.(*shells.Linux)
				s.True(ok, "expected *shells.Linux")
			}
		})
	}
}
