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

package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
)

var (
	_ collector.Collector = (*command.Linux)(nil)
	_ collector.Collector = (*command.Darwin)(nil)
)

type CommandPublicTestSuite struct {
	suite.Suite
}

func TestCommandPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CommandPublicTestSuite))
}

// psExec returns a MockExecutor that answers `ps -ef` with the
// provided output and error.
func psExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ps", "-ef").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *CommandPublicTestSuite) TestNew() {
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
			c := command.New()
			s.Equal("command", c.Name())
			s.Equal("misc", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*command.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*command.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *CommandPublicTestSuite) TestCollect() {
	psOutput := []byte(`UID        PID  PPID  C STIME TTY          TIME CMD
root         1     0  0 10:00 ?        00:00:01 /sbin/init
root         2     0  0 10:00 ?        00:00:00 [kthreadd]
user      1234     1  0 10:01 pts/0    00:00:00 bash
`)

	tests := []struct {
		name    string
		variant string
		exec    func(*testing.T) executor.Executor
		want    []string
	}{
		{
			name:    "linux: ps output parsed and trailing whitespace trimmed",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return psExec(t, psOutput, nil) },
			want: []string{
				"UID        PID  PPID  C STIME TTY          TIME CMD",
				"root         1     0  0 10:00 ?        00:00:01 /sbin/init",
				"root         2     0  0 10:00 ?        00:00:00 [kthreadd]",
				"user      1234     1  0 10:01 pts/0    00:00:00 bash",
			},
		},
		{
			name:    "linux: empty output yields empty slice",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return psExec(t, []byte{}, nil) },
			want:    []string{},
		},
		{
			name:    "linux: blank lines in output are skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return psExec(t, []byte("header\n\nprocess1\n\n"), nil)
			},
			want: []string{"header", "process1"},
		},
		{
			name:    "linux: exec error yields empty slice, no error",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return psExec(t, nil, errors.New("not found")) },
			want:    []string{},
		},
		{
			name:    "linux: nil Exec yields empty slice, no error",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    []string{},
		},
		{
			name:    "darwin: ps output parsed",
			variant: "darwin",
			exec:    func(t *testing.T) executor.Executor { return psExec(t, psOutput, nil) },
			want: []string{
				"UID        PID  PPID  C STIME TTY          TIME CMD",
				"root         1     0  0 10:00 ?        00:00:01 /sbin/init",
				"root         2     0  0 10:00 ?        00:00:00 [kthreadd]",
				"user      1234     1  0 10:01 pts/0    00:00:00 bash",
			},
		},
		{
			name:    "darwin: exec error yields empty slice, no error",
			variant: "darwin",
			exec:    func(t *testing.T) executor.Executor { return psExec(t, nil, errors.New("not found")) },
			want:    []string{},
		},
		{
			name:    "darwin: nil Exec yields empty slice, no error",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    []string{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c command.Collector
			switch tt.variant {
			case "linux":
				c = &command.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &command.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*command.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.PS)
		})
	}
}
