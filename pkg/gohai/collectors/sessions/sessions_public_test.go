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
	"fmt"
	"os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sessions"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*sessions.Linux)(nil)
	_ collector.Collector = (*sessions.Darwin)(nil)
)

// The command lines that the rows answer, as sourcetest.Commands keys them.
const (
	loginctlLine = "loginctl --no-pager --no-legend --no-ask-password list-sessions"
)

type SessionsPublicTestSuite struct {
	suite.Suite
}

func TestSessionsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SessionsPublicTestSuite))
}

func (s *SessionsPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c sessions.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := sessions.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = sessions.New()
			}
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
			"   3    1001 jane     -     pts/1\n" +
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
		dirs         []string
		validateFunc func(any, error)
	}{
		{
			name:    "linux: loginctl present, parses sessions ignores utmp",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{loginctlLine: string(loginctlOutput)},
				}.Mock(
					t,
				)
			},
			files: utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Require().Len(i.LoggedIn, 3)
				s.Equal("c1", i.LoggedIn[0].SessionID)
				s.Equal("1000", i.LoggedIn[0].UID)
				s.Equal("john", i.LoggedIn[0].User)
				s.Equal("seat0", i.LoggedIn[0].Seat)
				s.Equal("2", i.LoggedIn[1].SessionID)
				s.Equal("root", i.LoggedIn[1].User)
				s.Empty(i.LoggedIn[1].Seat)
				s.Equal("jane", i.LoggedIn[2].User)
				s.Empty(i.LoggedIn[2].Seat)
			},
		},
		{
			name:    "linux: loginctl missing, falls back to utmp",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						loginctlLine: fmt.Errorf("exec loginctl: %w", exec.ErrNotFound),
					},
				}.Mock(
					t,
				)
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{loginctlLine: ""}}.Mock(t)
			},
			files: utmp,
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
			name:    "linux: a failed loginctl falls back to utmp",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{loginctlLine: errors.New("exit status 1")},
				}.Mock(
					t,
				)
			},
			files: utmp,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*sessions.Info)
				s.Require().True(ok)
				s.Require().Len(i.LoggedIn, 1)
				s.Equal("fallback", i.LoggedIn[0].User)
			},
		},
		{
			name:    "linux: no loginctl and no utmp file, no sessions",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						loginctlLine: fmt.Errorf("exec loginctl: %w", exec.ErrNotFound),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&sessions.Info{LoggedIn: []sessions.Session{}}, got)
			},
		},
		{
			name:    "linux: a failed loginctl and no utmp file, the loginctl error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{loginctlLine: errors.New("exit status 1")},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "loginctl list-sessions: exit status 1")
			},
		},
		{
			name:    "linux: a utmp file that cannot be read is an error",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			dirs:    []string{"/var/run/utmp"},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read /var/run/utmp")
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
			name:    "darwin: no utmpx file, no sessions",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&sessions.Info{LoggedIn: []sessions.Session{}}, got)
			},
		},
		{
			name:    "darwin: a utmpx file that cannot be read is an error",
			variant: "darwin",
			dirs:    []string{"/var/run/utmpx"},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read /var/run/utmpx")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			for _, dir := range tt.dirs {
				s.Require().NoError(fs.MkdirAll(dir, 0o755))
			}
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
	binary.NativeEndian.PutUint16(b[typ:], r.typ)
	copy(b[line:], r.line)
	copy(b[user:], r.user)
	copy(b[host:], r.host)
	binary.NativeEndian.PutUint32(b[sec:], r.sec)

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
