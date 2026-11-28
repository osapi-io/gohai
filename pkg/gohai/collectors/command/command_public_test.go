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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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

// psExec returns a MockRunner that answers `ps -ef` with the
// provided output and error.
func psExec(
	t *testing.T,
	out []byte,
	err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
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
		name         string
		detect       string
		validateFunc func(command.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c command.Collector) {
				_, ok := c.(*command.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c command.Collector) {
				_, ok := c.(*command.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c command.Collector) {
				_, ok := c.(*command.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c command.Collector) {
				_, ok := c.(*command.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c command.Collector) {
				_, ok := c.(*command.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := command.New()
			s.Equal("command", c.Name())
			s.Equal("misc", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *CommandPublicTestSuite) TestCollect() {
	// secretOutput holds arguments that look like secrets, which must
	// never reach the facts.
	secretOutput := []byte(`UID        PID  PPID  C STIME TTY          TIME CMD
app        700     1  0 10:02 ?        00:00:03 /usr/bin/app --password=fake-pw-123 --token fake-tok-456
app        701     1  0 10:02 ?        00:00:00 sh -c echo fake-env-789
`)
	psOutput := []byte(`UID        PID  PPID  C STIME TTY          TIME CMD
root         1     0  0 10:00 ?        00:00:01 /sbin/init  	
root         2     0  0 10:00 ?        00:00:00 [kthreadd]
user      1234     1  0 10:01 pts/0    00:00:00 bash
`)

	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: the arguments of a command never reach the facts",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return psExec(t, secretOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{
					"UID        PID  PPID  C STIME TTY          TIME CMD",
					"app        700     1  0 10:02 ?        00:00:03 /usr/bin/app",
					"app        701     1  0 10:02 ?        00:00:00 sh",
				}, info.PS)
				facts, err := json.Marshal(got)
				s.Require().NoError(err)
				s.NotContains(string(facts), "fake-pw-123")
				s.NotContains(string(facts), "fake-tok-456")
				s.NotContains(string(facts), "fake-env-789")
			},
		},
		{
			name:    "darwin: the arguments of a command never reach the facts",
			variant: "darwin",
			exec:    func(t *testing.T) source.Runner { return psExec(t, secretOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				facts, err := json.Marshal(got)
				s.Require().NoError(err)
				s.Contains(string(facts), "/usr/bin/app")
				s.NotContains(string(facts), "fake-pw-123")
				s.NotContains(string(facts), "fake-tok-456")
				s.NotContains(string(facts), "fake-env-789")
			},
		},
		{
			name:    "linux: the BusyBox layout keeps no arguments",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("PID   USER     TIME  COMMAND\n"+
					"    1 root      0:00 /sbin/init --fake-pw-123\n"+
					"   42 app       0:01 app --token fake-tok-456\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{
					"PID   USER     TIME  COMMAND",
					"    1 root      0:00 /sbin/init",
					"   42 app       0:01 app",
				}, info.PS)
			},
		},
		{
			name:    "linux: columns split by tabs keep no arguments",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(
					t,
					[]byte("PID\tTIME\tCMD\n7\t0:00\tapp\t--token\tfake-tok-456\n"),
					nil,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{"PID\tTIME\tCMD", "7\t0:00\tapp"}, info.PS)
			},
		},
		{
			name:    "linux: a line with no field at the command column is dropped",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("UID PID PPID C STIME TTY TIME CMD\n"+
					"root 1 0 0 10:00 ?\n"+
					"app 7 1 0 10:02 ? 00:00:00 app --note\n"+
					"fake-pw-123 rest of an argument that held a newline\n"+
					"x fake-tok-456\n"+
					"root 8 1 0 10:03 ? 00:00:00 sshd\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{
					"UID PID PPID C STIME TTY TIME CMD",
					"app 7 1 0 10:02 ? 00:00:00 app",
					"root 8 1 0 10:03 ? 00:00:00 sshd",
				}, info.PS)
				facts, err := json.Marshal(got)
				s.Require().NoError(err)
				s.NotContains(string(facts), "fake-pw-123")
				s.NotContains(string(facts), "fake-tok-456")
			},
		},
		{
			name:    "linux: a header with no PID column gives no lines",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("USER TIME CMD\nroot 0:00 init --fake-pw-123\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "linux: a header with the command before another column gives no lines",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("PID CMD TIME\n1 init --fake-pw-123 0:00\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "linux: output with no CMD or COMMAND header gives no lines",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("PID USER ARGS\n1 root /sbin/init --fake-pw-123\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "linux: ps output parsed and trailing whitespace trimmed",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return psExec(t, psOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{
					"UID        PID  PPID  C STIME TTY          TIME CMD",
					"root         1     0  0 10:00 ?        00:00:01 /sbin/init",
					"root         2     0  0 10:00 ?        00:00:00 [kthreadd]",
					"user      1234     1  0 10:01 pts/0    00:00:00 bash",
				}, info.PS)
			},
		},
		{
			name:    "linux: a command line longer than 64 KiB keeps the processes after it",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				long := "root  9  1  0 10:00 ?  00:00:01 java " + strings.Repeat("-Dx=y ", 20000)
				out := "UID PID PPID C STIME TTY TIME CMD\n" + long + "\nroot 10 1 0 10:00 ? 00:00:00 sshd\n"
				return psExec(t, []byte(out), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Require().Len(info.PS, 3)
				s.Equal("root  9  1  0 10:00 ?  00:00:01 java", info.PS[1])
				s.Equal("root 10 1 0 10:00 ? 00:00:00 sshd", info.PS[2])
			},
		},
		{
			name:    "linux: empty output yields empty slice",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return psExec(t, []byte{}, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "linux: blank lines in output are skipped",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, []byte("PID CMD\n\n1 process1\n\n"), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{"PID CMD", "1 process1"}, info.PS)
			},
		},
		{
			name:    "linux: no ps on the host yields empty slice, no error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, nil, fmt.Errorf("exec ps: %w", exec.ErrNotFound))
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "linux: a ps that fails is an error",
			variant: "linux",
			exec:    func(t *testing.T) source.Runner { return psExec(t, nil, errors.New("exit status 1")) },
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Exactly(nil, got)
				s.Contains(err.Error(), "read ps")
			},
		},
		{
			name:    "linux: nil Exec yields empty slice, no error",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "darwin: ps output parsed",
			variant: "darwin",
			exec:    func(t *testing.T) source.Runner { return psExec(t, psOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{
					"UID        PID  PPID  C STIME TTY          TIME CMD",
					"root         1     0  0 10:00 ?        00:00:01 /sbin/init",
					"root         2     0  0 10:00 ?        00:00:00 [kthreadd]",
					"user      1234     1  0 10:01 pts/0    00:00:00 bash",
				}, info.PS)
			},
		},
		{
			name:    "darwin: no ps on the host yields empty slice, no error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return psExec(t, nil, fmt.Errorf("exec ps: %w", exec.ErrNotFound))
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
		},
		{
			name:    "darwin: a ps that fails is an error",
			variant: "darwin",
			exec:    func(t *testing.T) source.Runner { return psExec(t, nil, errors.New("exit status 1")) },
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Exactly(nil, got)
				s.Contains(err.Error(), "read ps")
			},
		},
		{
			name:    "darwin: nil Exec yields empty slice, no error",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*command.Info)
				s.Require().True(ok)
				s.Equal([]string{}, info.PS)
			},
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
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
