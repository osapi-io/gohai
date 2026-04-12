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

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
)

type ProcessPublicTestSuite struct {
	suite.Suite
}

func TestProcessPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessPublicTestSuite))
}

// fakeProc is a lightweight ProcSnapshot stub used by Collect tests to
// avoid exercising real /proc data. Every field has an "err" variant
// the test can set to simulate partial-read scenarios.
type fakeProc struct {
	pid       int32
	ppid      int32
	ppidErr   error
	name      string
	nameErr   error
	username  string
	userErr   error
	cmdline   string
	cmdErr    error
	status    []string
	statusErr error
	ctime     int64
	ctimeErr  error
}

func (f fakeProc) Pid() int32                 { return f.pid }
func (f fakeProc) Ppid() (int32, error)       { return f.ppid, f.ppidErr }
func (f fakeProc) Name() (string, error)      { return f.name, f.nameErr }
func (f fakeProc) Username() (string, error)  { return f.username, f.userErr }
func (f fakeProc) Cmdline() (string, error)   { return f.cmdline, f.cmdErr }
func (f fakeProc) Status() ([]string, error)  { return f.status, f.statusErr }
func (f fakeProc) CreateTime() (int64, error) { return f.ctime, f.ctimeErr }

func (s *ProcessPublicTestSuite) TestNew() {
	c := process.New()
	s.Equal("process", c.Name())
	s.True(c.DefaultEnabled())
	s.Empty(c.Dependencies())
}

func (s *ProcessPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = process.NewLinux()
	var _ collector.Collector = process.NewDarwin()
}

func (s *ProcessPublicTestSuite) TestNewDispatch() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin", "darwin", "darwin"},
		{"debian", "debian", "linux"},
		{"rhel", "rhel", "linux"},
		{"arch", "arch", "linux"},
		{"unknown", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			got := process.New()
			switch tt.wantKind {
			case "darwin":
				_, ok := got.(*process.Darwin)
				s.True(ok)
			case "linux":
				_, ok := got.(*process.Linux)
				s.True(ok)
			}
		})
	}
}

// TestSnapshotOf is exercised indirectly via Linux/Darwin Collect
// tables. This entry-level test here covers shared helper behavior
// through a public seam: running Collect() with a fake proc and
// asserting the populated fields.
func (s *ProcessPublicTestSuite) TestCollectMapsFields() {
	tests := []struct {
		name    string
		procs   []process.ProcSnapshot
		wantErr bool
		want    process.Info
	}{
		{
			name: "fully-populated process",
			procs: []process.ProcSnapshot{
				fakeProc{
					pid:      1,
					ppid:     0,
					name:     "systemd",
					username: "root",
					cmdline:  "/sbin/init",
					status:   []string{"S"},
					ctime:    1_700_000_000_000,
				},
			},
			want: process.Info{
				Count: 1,
				Processes: []process.Process{
					{
						PID:       1,
						PPID:      0,
						Name:      "systemd",
						Username:  "root",
						CmdLine:   "/sbin/init",
						State:     "S",
						StartTime: 1_700_000_000,
					},
				},
			},
		},
		{
			name: "access-denied errors fall through to empty fields",
			procs: []process.ProcSnapshot{
				fakeProc{
					pid: 42, ppidErr: errors.New("perm"), nameErr: errors.New("perm"),
					userErr: errors.New("perm"), cmdErr: errors.New("perm"),
					statusErr: errors.New("perm"), ctimeErr: errors.New("perm"),
				},
			},
			want: process.Info{
				Count:     1,
				Processes: []process.Process{{PID: 42}},
			},
		},
		{
			name: "negative create time is skipped",
			procs: []process.ProcSnapshot{
				fakeProc{pid: 99, name: "unset-ctime", ctime: -1},
			},
			want: process.Info{
				Count:     1,
				Processes: []process.Process{{PID: 99, Name: "unset-ctime"}},
			},
		},
		{
			name: "empty status slice leaves state empty",
			procs: []process.ProcSnapshot{
				fakeProc{pid: 100, name: "weird", status: []string{}},
			},
			want: process.Info{
				Count:     1,
				Processes: []process.Process{{PID: 100, Name: "weird"}},
			},
		},
		{
			name:  "empty process list",
			procs: []process.ProcSnapshot{},
			want:  process.Info{Count: 0, Processes: []process.Process{}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &process.Linux{
				ProcessesFn: func(context.Context) ([]process.ProcSnapshot, error) {
					return tt.procs, nil
				},
			}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*process.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
