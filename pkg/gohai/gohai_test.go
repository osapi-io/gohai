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

package gohai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type GohaiTestSuite struct {
	suite.Suite
}

func TestGohaiTestSuite(t *testing.T) {
	suite.Run(t, new(GohaiTestSuite))
}

type cycleCollector struct {
	name string
	deps []string
}

func (c *cycleCollector) Name() string {
	return c.name
}

func (c *cycleCollector) Tier() collector.Tier {
	return collector.TierCore
}

func (c *cycleCollector) Dependencies() []string {
	return c.deps
}

func (c *cycleCollector) Collect(
	_ context.Context,
) (any, error) {
	return nil, nil
}

func (s *GohaiTestSuite) TestCollectPropagatesRunError() {
	reg := collector.NewRegistry()
	s.Require().NoError(reg.Register(&cycleCollector{name: "a", deps: []string{"b"}}))
	s.Require().NoError(reg.Register(&cycleCollector{name: "b", deps: []string{"a"}}))

	g := &Gohai{registry: reg}
	sel, err := reg.Selected(nil, nil)
	s.Require().NoError(err)
	g.selected = sel

	_, err = g.Collect(context.Background())
	s.Error(err)
}
