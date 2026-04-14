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

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
)

var (
	_ collector.Collector = (*users.Linux)(nil)
	_ collector.Collector = (*users.Darwin)(nil)
)

type UsersPublicTestSuite struct {
	suite.Suite
}

func TestUsersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UsersPublicTestSuite))
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

func (s *UsersPublicTestSuite) TestNew() {
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
			c := users.New()
			s.Equal("users", c.Name())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*users.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*users.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *UsersPublicTestSuite) TestCollect() {
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
		variant  string
		exec     func(*testing.T) executor.Executor
		usersFn  func(context.Context) ([]host.UserStat, error)
		wantErr  bool
		validate func(*users.Info)
	}{
		{
			name:    "linux: loginctl present, parses sessions ignores utmp",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return loginctlExec(t, loginctlOutput, nil) },
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
			name:    "linux: loginctl missing, falls back to utmp via gopsutil",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return loginctlExec(t, nil, errors.New("not found"))
			},
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
				s.Equal("pts/0", i.LoggedIn[0].Terminal)
				s.Empty(i.LoggedIn[0].SessionID)
			},
		},
		{
			name:    "linux: nil Exec, utmp path direct",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
			},
		},
		{
			name:    "linux: loginctl returns empty output, empty session list not fallback",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return loginctlExec(t, []byte(""), nil) },
			usersFn: func(context.Context) ([]host.UserStat, error) { return utmpUsers, nil },
			validate: func(i *users.Info) {
				s.Empty(i.LoggedIn)
			},
		},
		{
			name:    "linux: loginctl missing AND gopsutil errors propagate",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return loginctlExec(t, nil, errors.New("not found"))
			},
			usersFn: func(context.Context) ([]host.UserStat, error) {
				return nil, errors.New("utmp boom")
			},
			wantErr: true,
		},
		{
			name:    "darwin: console session",
			variant: "darwin",
			usersFn: func(context.Context) ([]host.UserStat, error) {
				return []host.UserStat{
					{User: "john", Terminal: "console", Started: 1712908800},
				}, nil
			},
			validate: func(i *users.Info) {
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("john", i.LoggedIn[0].User)
				s.Equal("console", i.LoggedIn[0].Terminal)
				s.Equal(uint64(1712908800), i.LoggedIn[0].Started)
			},
		},
		{
			name:    "darwin: gopsutil error wrapped and returned",
			variant: "darwin",
			usersFn: func(context.Context) ([]host.UserStat, error) {
				return nil, errors.New("utmpx error")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer users.SetUsersFn(tt.usersFn)()
			var c users.Collector
			switch tt.variant {
			case "linux":
				c = &users.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &users.Darwin{}
			}
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
