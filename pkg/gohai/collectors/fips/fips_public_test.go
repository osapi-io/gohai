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

package fips_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
)

type FipsPublicTestSuite struct {
	suite.Suite
}

func TestFipsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FipsPublicTestSuite))
}

func (s *FipsPublicTestSuite) TestNew() {
	c := fips.New()
	s.Equal("fips", c.Name())
	s.Equal(true, c.DefaultEnabled())
	s.Empty(c.Dependencies())
}

func (s *FipsPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = fips.New()
}

func (s *FipsPublicTestSuite) TestCollect() {
	c := fips.New()
	got, err := c.Collect(context.Background())
	if err != nil {
		return
	}
	if got == nil {
		return
	}
	_, ok := got.(*fips.Info)
	s.True(ok)
}
