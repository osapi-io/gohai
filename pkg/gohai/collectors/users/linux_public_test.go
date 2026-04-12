//go:build linux

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
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
)

type UsersLinuxPublicTestSuite struct {
	suite.Suite
}

func TestUsersLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UsersLinuxPublicTestSuite))
}

func (s *UsersLinuxPublicTestSuite) TestCollectFromGopsutil() {
	two := func(_ context.Context) ([]host.UserStat, error) {
		return []host.UserStat{
			{User: "alice", Terminal: "tty1", Host: "desktop", Started: 1700000000},
			{User: "bob", Terminal: "pts/0", Host: "192.168.1.5", Started: 1700100000},
		}, nil
	}
	empty := func(_ context.Context) ([]host.UserStat, error) {
		return nil, nil
	}
	errFn := func(_ context.Context) ([]host.UserStat, error) {
		return nil, errors.New("boom")
	}

	tests := []struct {
		name    string
		fn      func(context.Context) ([]host.UserStat, error)
		wantErr bool
		wantLen int
	}{
		{"two users", two, false, 2},
		{"empty", empty, false, 0},
		{"error", errFn, true, 0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := users.CollectFromGopsutil(context.Background(), tt.fn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*users.Info)
			s.Require().True(ok)
			s.Len(info.LoggedIn, tt.wantLen)
			if tt.wantLen == 2 {
				s.Equal("alice", info.LoggedIn[0].User)
			}
		})
	}
}

func (s *UsersLinuxPublicTestSuite) TestCollectDefault() {
	got, err := users.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*users.Info)
	s.True(ok)
}
