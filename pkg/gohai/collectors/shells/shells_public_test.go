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
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

type ShellsPublicTestSuite struct {
	suite.Suite
}

func TestShellsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsPublicTestSuite))
}

func (s *ShellsPublicTestSuite) TestNew() {
	c := shells.New()
	s.Equal("shells", c.Name())
	s.Equal(collector.TierCore, c.Tier())
	s.Empty(c.Dependencies())
}

func (s *ShellsPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = shells.New()
}

func (s *ShellsPublicTestSuite) TestCollect() {
	c := shells.New()
	got, err := c.Collect(context.Background())
	if err != nil {
		return // /etc/shells absent on host is OK
	}
	if got == nil {
		return
	}
	_, ok := got.(*shells.Info)
	s.True(ok)
}
