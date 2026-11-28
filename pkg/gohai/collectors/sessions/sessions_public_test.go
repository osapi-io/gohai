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

package sessions_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sessions"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*sessions.Linux)(nil)
	_ collector.Collector = (*sessions.Darwin)(nil)
)

type SessionsPublicTestSuite struct {
	suite.Suite
}

func TestSessionsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SessionsPublicTestSuite))
}

// loginctlExec returns a MockRunner that canned-answers the
// `loginctl list-sessions` command.
func loginctlExec(
	t *testing.T,
	out []byte, err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
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

func (s *SessionsPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(sessions.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c sessions.Collector) {
				_, ok := c.(*sessions.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c sessions.Collector) {
				_, ok := c.(*sessions.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c sessions.Collector) {
				_, ok := c.(*sessions.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c sessions.Collector) {
				_, ok := c.(*sessions.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c sessions.Collector) {
				_, ok := c.(*sessions.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := sessions.New()
			s.Equal("sessions", c.Name())
			s.Equal("users", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SessionsPublicTestSuite) TestCollect() {
	loginctlOutput := []byte(
		"   c1   1000 john     seat0\n" +
			"   2    0    root\n" +
			"\n" +
			"  bad-line\n",
	)
	utmp := map[string]string{"/var/run/utmp": string(linuxUtmp(
		utmpRecord{typ: 8, user: "gone"},
		utmpRecord{typ: 7, user: "fallback", line: "pts/0", host: "10.0.0.1", sec: 1712908800},
	))}

	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) source.Runner
		files        map[string]string
		validateFunc func(any, error)
	}{
		{
			name:    "linux: loginctl present, parses sessions ignores utmp",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return loginctlExec(t, loginctlOutput, nil) },
			files:   utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
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
			name:    "linux: loginctl missing, falls back to utmp",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return loginctlExec(t, nil, errors.New("not found"))
			},
			files: utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
				s.Equal("pts/0", i.LoggedIn[0].Terminal)
				s.Empty(i.LoggedIn[0].SessionID)
			},
		},
		{
			name:    "linux: nil Exec, utmp path direct",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			files:   utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
			},
		},
		{
			name:    "linux: loginctl returns empty output, an empty list, not null, and no fallback",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return loginctlExec(t, []byte(""), nil) },
			files:   utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Equal([]sessions.Session{}, i.LoggedIn)
				b, err := json.Marshal(i)
				s.Require().NoError(err)
				s.Contains(string(b), `"logged_in":[]`)
			},
		},
		{
			name:    "linux: loginctl missing AND no utmp file",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return loginctlExec(t, nil, errors.New("not found"))
			},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: console session",
			variant: "darwin",
			files: map[string]string{"/var/run/utmpx": string(darwinUtmpx(
				utmpRecord{typ: 2},
				utmpRecord{typ: 7, user: "john", line: "console", sec: 1712908800},
			))},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("john", i.LoggedIn[0].User)
				s.Equal("console", i.LoggedIn[0].Terminal)
				s.Equal(uint64(1712908800), i.LoggedIn[0].Started)
			},
		},
		{
			name:    "darwin: no utmpx file",
			variant: "darwin",
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			var c sessions.Collector
			switch tt.variant {
			case "linux":
				c = &sessions.Linux{Exec: tt.exec(s.T()), FS: fs}
			case "darwin":
				c = &sessions.Darwin{FS: fs}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}

// utmpRecord is one session record to write into a test utmp file.
type utmpRecord struct {
	typ              uint16
	user, line, host string
	sec              uint32
}

// record lays r out in size bytes at the given offsets.
func record(
	r utmpRecord,
	size, typ, line, user, host, sec int,
) []byte {
	b := make([]byte, size)
	binary.LittleEndian.PutUint16(b[typ:], r.typ)
	copy(b[line:], r.line)
	copy(b[user:], r.user)
	copy(b[host:], r.host)
	binary.LittleEndian.PutUint32(b[sec:], r.sec)

	return b
}

// linuxUtmp is a glibc utmp file of rs.
func linuxUtmp(
	rs ...utmpRecord,
) []byte {
	b := make([]byte, 0, 384*len(rs))
	for _, r := range rs {
		b = append(b, record(r, 384, 0, 8, 44, 76, 340)...)
	}

	return b
}

// darwinUtmpx is a Darwin utmpx file of rs: a header record, then rs.
func darwinUtmpx(
	rs ...utmpRecord,
) []byte {
	b := make([]byte, 628, 628*(len(rs)+1))
	for _, r := range rs {
		b = append(b, record(r, 628, 296, 260, 0, 308, 300)...)
	}

	return b
}
