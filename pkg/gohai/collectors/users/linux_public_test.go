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
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
)

type UsersLinuxPublicTestSuite struct {
	suite.Suite
}

func TestUsersLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UsersLinuxPublicTestSuite))
}

// loginctlExec returns a MockExecutor that canned-answers the
// `loginctl list-sessions` command.
func loginctlExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(
			gomock.Any(),
			"loginctl",
			"--no-pager",
			"--no-legend",
			"--no-ask-password",
			"list-sessions",
		).
		Return(out, err).
		AnyTimes()
	return m
}

func (s *UsersLinuxPublicTestSuite) TestCollect() {
	loginctlOutput := []byte(
		"   c1   1000 john     seat0\n" +
			"   2    0    root\n" +
			"\n" +
			"  bad-line\n",
	)
	utmpUsers := []host.UserStat{
		{User: "fallback", Terminal: "pts/0", Host: "10.0.0.1", Started: 1712908800},
	}

	tests := []struct {
		name     string
		exec     executor.Executor
		usersFn  func(context.Context) ([]host.UserStat, error)
		wantErr  bool
		validate func(*users.Info)
	}{
		{
			name:    "loginctl present: parses sessions, ignores utmp",
			exec:    loginctlExec(s.T(), loginctlOutput, nil),
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 2)
				s.Equal("c1", i.LoggedIn[0].SessionID)
				s.Equal("1000", i.LoggedIn[0].UID)
				s.Equal("john", i.LoggedIn[0].User)
				s.Equal("seat0", i.LoggedIn[0].Seat)
				s.Equal("2", i.LoggedIn[1].SessionID)
				s.Equal("root", i.LoggedIn[1].User)
				s.Empty(i.LoggedIn[1].Seat)
			},
		},
		{
			name:    "loginctl missing: falls back to utmp via gopsutil",
			exec:    loginctlExec(s.T(), nil, errors.New("not found")),
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
				s.Equal("pts/0", i.LoggedIn[0].Terminal)
				s.Empty(i.LoggedIn[0].SessionID)
			},
		},
		{
			name:    "nil Exec: utmp path direct",
			exec:    nil,
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
			},
		},
		{
			name:    "loginctl returns empty output: empty session list (not fallback)",
			exec:    loginctlExec(s.T(), []byte(""), nil),
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Empty(i.LoggedIn)
			},
		},
		{
			name:    "loginctl missing AND gopsutil errors: error propagated",
			exec:    loginctlExec(s.T(), nil, errors.New("not found")),
			usersFn: func(context.Context) ([]host.UserStat, error) { return nil, errors.New("utmp boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer users.SetUsersFn(tt.usersFn)()
			c := &users.Linux{Exec: tt.exec}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*users.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}

func (s *UsersLinuxPublicTestSuite) TestListSessions() {
	tests := []struct {
		name    string
		fn      func(context.Context) ([]host.UserStat, error)
		wantLen int
		wantErr bool
	}{
		{
			name: "success maps UserStat",
			fn: func(context.Context) ([]host.UserStat, error) {
				return []host.UserStat{
					{User: "john", Terminal: "pts/0", Host: "10.0.0.1", Started: 1712908800},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:    "gopsutil error wrapped",
			fn:      func(context.Context) ([]host.UserStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer users.SetUsersFn(tt.fn)()
			ss, err := users.ListSessions(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Len(ss, tt.wantLen)
		})
	}
}
