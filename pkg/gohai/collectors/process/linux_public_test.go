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
	"os"
	"testing"

	gpprocess "github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
)

type ProcessLinuxPublicTestSuite struct {
	suite.Suite
}

func TestProcessLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessLinuxPublicTestSuite))
}

func (s *ProcessLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		procs   []process.Process
		fnErr   error
		wantErr bool
		want    process.Info
	}{
		{
			name: "snapshot with populated processes",
			procs: []process.Process{
				{
					PID:       1,
					Name:      "systemd",
					Username:  "root",
					CmdLine:   "/sbin/init",
					State:     "S",
					StartTime: 1_700_000_000,
				},
				{PID: 1247, Name: "nginx", Username: "www-data"},
			},
			want: process.Info{
				Count: 2,
				Processes: []process.Process{
					{
						PID:       1,
						Name:      "systemd",
						Username:  "root",
						CmdLine:   "/sbin/init",
						State:     "S",
						StartTime: 1_700_000_000,
					},
					{PID: 1247, Name: "nginx", Username: "www-data"},
				},
			},
		},
		{
			name:  "empty process list",
			procs: []process.Process{},
			want:  process.Info{Count: 0, Processes: []process.Process{}},
		},
		{
			name:    "processes error propagated",
			fnErr:   errors.New("boom"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &process.Linux{
				ProcessesFn: func(context.Context) ([]process.Process, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.procs, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*process.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *ProcessLinuxPublicTestSuite) TestListProcesses() {
	// NewProcess with our own PID yields a valid gopsutil Process whose
	// field readers succeed — exercises the success branch of
	// snapshotFromGopsutil.
	ownPID := int32(os.Getpid())
	tests := []struct {
		name    string
		fn      func(context.Context) ([]*gpprocess.Process, error)
		wantErr bool
		wantLen int
	}{
		{
			name: "success wraps gopsutil processes",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				p, _ := gpprocess.NewProcess(ownPID)
				return []*gpprocess.Process{p}, nil
			},
			wantLen: 1,
		},
		{
			name:    "gopsutil error wrapped and returned",
			fn:      func(context.Context) ([]*gpprocess.Process, error) { return nil, errors.New("processes failed") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := process.SetProcessesFn(tt.fn)
			defer restore()
			got, err := process.ListProcesses(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Len(got, tt.wantLen)
		})
	}
}
