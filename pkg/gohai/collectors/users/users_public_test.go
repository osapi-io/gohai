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

package users_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
)

type UsersPublicTestSuite struct {
	suite.Suite
}

func TestUsersPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UsersPublicTestSuite))
}

func (s *UsersPublicTestSuite) TestNew() {
	c := users.New()
	s.Equal("users", c.Name())
	s.Equal(collector.TierCore, c.Tier())
	s.Empty(c.Dependencies())
}

func (s *UsersPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = users.New()
}

func (s *UsersPublicTestSuite) TestCollect() {
	c := users.New()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	if got == nil {
		return
	}
	_, ok := got.(*users.Info)
	s.True(ok)
}
