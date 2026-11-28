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

package process_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
)

var (
	_ collector.Collector = (*process.Linux)(nil)
	_ collector.Collector = (*process.Darwin)(nil)
)

type ProcessPublicTestSuite struct {
	suite.Suite
}

func TestProcessPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessPublicTestSuite))
}

func (s *ProcessPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(process.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := process.New()
			s.Equal("process", c.Name())
			s.Equal("misc", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *ProcessPublicTestSuite) TestCollect() {
	started, err := time.ParseInLocation(
		"Mon Jan 2 15:04:05 2006",
		"Thu Nov 20 12:04:00 2026",
		time.Local,
	)
	s.Require().NoError(err)
	const psMain = "ps -ax -o pid=,ppid=,user=,state=,pcpu=,pmem=,rss=,time=,lstart="
	const psName = "ps -ax -o pid=,ucomm="
	const psArgs = "ps -ax -ww -o pid=,args="

	tests := []struct {
		name      string
		variant   string
		files     map[string]string
		runner    sourcetest.Commands
		cancelled bool // the context is cancelled before Collect
		nilExec   bool
		want      *process.Info
		wantErr   string
	}{
		{name: "darwin: no Exec gives no fact", variant: "darwin", nilExec: true},
		{
			name:    "linux: /proc, with owners and start times",
			variant: "linux",
			files: map[string]string{
				"/proc/stat":      "cpu 1 2 3\nbtime 1700000000\n",
				"/proc/meminfo":   "MemFree: 1 kB\nMemTotal:        4096 kB\n",
				"/etc/passwd":     "root:x:0:0::/root:/bin/sh\nbad line\n",
				"/proc/1/stat":    "1 (my (proc)) S 0 1 1 0 -1 4194560 1 2 3 4 5 6 7 8 20 0 1 0 500 9 10\n",
				"/proc/1/status":  "Name:\tmy (proc)\nUid:\t0\t0\t0\t0\nVmRSS:\t    1024 kB\n",
				"/proc/1/cmdline": "/sbin/init\x00splash\x00",
				"/proc/42/stat":   "42 (kworker) R\n",
				"/proc/42/status": "Uid:\t1000\t1000\t1000\t1000\n",
				"/proc/99/status": "Uid:\t0\n",
				"/proc/77/stat":   "no parens here\n",
				"/proc/self/stat": "not a pid\n",
			},
			want: &process.Info{Count: 2, Processes: []process.Process{
				{
					PID: 1, Name: "my (proc)", State: "sleep", Owner: "root",
					CmdLine: "/sbin/init splash", CreationTime: 1700000005,
					CPUPercent: 1, RSS: 1024 * 1024, MemPercent: 25, CPUSeconds: 0.11,
				},
				// uid 1000 is not in /etc/passwd, so it is the owner.
				{PID: 42, Name: "kworker", Owner: "1000"},
			}},
		},
		{
			name:    "linux: no boot time, no passwd, and no MemTotal",
			variant: "linux",
			files: map[string]string{
				"/proc/stat":    "cpu 1 2 3\n",
				"/proc/meminfo": "MemFree: 1 kB\n",
				"/proc/1/stat":  "1 (init) Z 0 1 1 0 -1 4194560 1 2 3 4 5 6 7 8 20 0 1 0 500 9 10\n",
			},
			want: &process.Info{Count: 1, Processes: []process.Process{
				{PID: 1, Name: "init", State: "zombie"},
			}},
		},
		{
			name:    "linux: no /proc/stat",
			variant: "linux",
			files:   map[string]string{"/proc/5/stat": "5 (x) S 1\n"},
			want: &process.Info{Count: 1, Processes: []process.Process{
				{PID: 5, Name: "x", State: "sleep", ParentPID: 1},
			}},
		},
		{name: "linux: no /proc", variant: "linux", wantErr: "read /proc"},
		{
			name:    "darwin: ps, joined by pid",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				psMain: "    1     0 root     Ss    0.0  0.1   1024 1-02:03:04.50 Thu Nov 20 12:04:00 2026\n" +
					"  500     1 john     R+   12.5  3.0   2048 0:01.25 not a date at all\n" +
					"  x 1 bad R\n" +
					"short\n",
				psName: "    1 launchd\n  500 Google Chrome Helper\nnospace\n",
				psArgs: "    1 /sbin/launchd\n  500 /Applications/Google Chrome.app/x --type=gpu\n",
			}},
			want: &process.Info{Count: 2, Processes: []process.Process{
				{
					PID: 1, Name: "launchd", Owner: "root", State: "sleep",
					CmdLine: "/sbin/launchd", CreationTime: started.Unix(),
					MemPercent: 0.1, RSS: 1024 * 1024, CPUSeconds: 86400 + 2*3600 + 3*60 + 4.5,
				},
				{
					PID:        500,
					ParentPID:  1,
					Name:       "Google Chrome Helper",
					Owner:      "john",
					State:      "running",
					CmdLine:    "/Applications/Google Chrome.app/x --type=gpu",
					CPUPercent: 12.5,
					MemPercent: 3,
					RSS:        2048 * 1024,
					CPUSeconds: 1.25,
				},
			}},
		},
		{
			name:    "darwin: the name and the args fail",
			variant: "darwin",
			runner: sourcetest.Commands{
				Out: map[string]string{
					psMain: "    7     1 john     U     0.0  0.0      0 x-1:00 Thu Nov 20 12:04:00 2026\n",
				},
				Err: map[string]error{psName: errors.New("boom"), psArgs: errors.New("boom")},
			},
			want: &process.Info{Count: 1, Processes: []process.Process{
				{PID: 7, ParentPID: 1, Owner: "john", CreationTime: started.Unix()},
			}},
		},
		{
			name:    "darwin: ps fails",
			variant: "darwin",
			runner:  sourcetest.Commands{Err: map[string]error{psMain: errors.New("boom")}},
			wantErr: "ps: boom",
		},
		{
			name:      "linux: a cancelled context is the error, with no snapshot",
			variant:   "linux",
			files:     map[string]string{"/proc/1/stat": "1 (init) S 0\n"},
			cancelled: true,
			wantErr:   context.Canceled.Error(),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c process.Collector
			switch tt.variant {
			case "linux":
				c = &process.Linux{
					FS:  sourcetest.FS(s.T(), tt.files),
					Now: func() time.Time { return time.Unix(1700000016, 0) },
				}
			case "darwin":
				d := &process.Darwin{Exec: tt.runner.Mock(s.T())}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancelled {
				cancel()
			}
			got, err := c.Collect(ctx, nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			if tt.nilExec {
				s.Nil(got)

				return
			}
			s.Equal(tt.want, got)
		})
	}
}

func (s *ProcessPublicTestSuite) TestRead() {
	const one = "ps -p 7 -o pid=,ppid=,user=,state=,pcpu=,pmem=,rss=,time=,lstart="
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		runner  sourcetest.Commands
		nilExec bool
		want    *process.Process
		wantErr string
	}{
		{
			name:    "darwin: no Exec is an error",
			variant: "darwin",
			nilExec: true,
			wantErr: "no process 7: no command runner",
		},
		{
			name:    "linux: one /proc directory",
			variant: "linux",
			files: map[string]string{
				"/proc/stat":     "btime 1700000000\n",
				"/proc/7/stat":   "7 (agent) S 1 7 7 0 -1 4194560 1 2 3 4 300 200 7 8 20 0 1 0 1000 9 10\n",
				"/proc/7/status": "VmRSS:\t2048 kB\n",
				"/proc/8/stat":   "8 (other) S 1\n",
			},
			want: &process.Process{
				PID: 7, ParentPID: 1, Name: "agent", State: "sleep",
				CreationTime: 1700000010, CPUPercent: 50, RSS: 2048 * 1024, CPUSeconds: 5,
			},
		},
		{name: "linux: no such process", variant: "linux", wantErr: "no process 7"},
		{
			name:    "darwin: ps -p",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				one:                         "    7     1 john     S     2.5  0.4   4096 0:03.00 not a date at all\n",
				"ps -p 7 -o pid=,ucomm=":    "    7 osapi\n",
				"ps -p 7 -ww -o pid=,args=": "    7 /usr/local/bin/osapi agent start\n",
			}},
			want: &process.Process{
				PID: 7, ParentPID: 1, Name: "osapi", Owner: "john", State: "sleep",
				CmdLine: "/usr/local/bin/osapi agent start", CPUPercent: 2.5, MemPercent: 0.4,
				RSS: 4096 * 1024, CPUSeconds: 3,
			},
		},
		{
			name:    "darwin: a time that is not a time",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				one: "    7     1 john     S     0.0  0.0      0 n/a\n",
			}},
			want: &process.Process{PID: 7, ParentPID: 1, Owner: "john", State: "sleep"},
		},
		{
			name:    "darwin: ps prints nothing",
			variant: "darwin",
			runner:  sourcetest.Commands{Out: map[string]string{one: ""}},
			wantErr: "no process 7",
		},
		{
			name:    "darwin: ps fails",
			variant: "darwin",
			runner:  sourcetest.Commands{Err: map[string]error{one: errors.New("exit status 1")}},
			wantErr: "ps: exit status 1",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c process.Collector
			switch tt.variant {
			case "linux":
				c = &process.Linux{
					FS:  sourcetest.FS(s.T(), tt.files),
					Now: func() time.Time { return time.Unix(1700000020, 0) },
				}
			case "darwin":
				d := &process.Darwin{Exec: tt.runner.Mock(s.T())}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			}
			got, err := c.Read(context.Background(), 7)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
