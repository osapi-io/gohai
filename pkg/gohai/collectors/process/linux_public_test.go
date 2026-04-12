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

package process_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
)

type fakeSnap struct {
	pid             int32
	name, user, cmd string
	nameErr         error
	userErr         error
	cmdErr          error
}

func (f fakeSnap) Pid() int32                { return f.pid }
func (f fakeSnap) Name() (string, error)     { return f.name, f.nameErr }
func (f fakeSnap) Username() (string, error) { return f.user, f.userErr }
func (f fakeSnap) Cmdline() (string, error)  { return f.cmd, f.cmdErr }

type ProcessLinuxPublicTestSuite struct {
	suite.Suite
}

func TestProcessLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessLinuxPublicTestSuite))
}

func (s *ProcessLinuxPublicTestSuite) TestCollectFromGopsutil() {
	ok := func(_ context.Context) ([]process.Snapshot, error) {
		return []process.Snapshot{
			fakeSnap{pid: 1, name: "init", user: "root", cmd: "/sbin/init"},
			fakeSnap{
				pid:     42,
				name:    "",
				user:    "",
				cmd:     "",
				nameErr: errors.New("denied"),
				userErr: errors.New("denied"),
				cmdErr:  errors.New("denied"),
			},
		}, nil
	}
	fail := func(_ context.Context) ([]process.Snapshot, error) {
		return nil, errors.New("boom")
	}

	tests := []struct {
		name    string
		fn      func(context.Context) ([]process.Snapshot, error)
		wantErr bool
		wantLen int
	}{
		{"happy path with permission-denied second process", ok, false, 2},
		{"error from list", fail, true, 0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := process.CollectFromGopsutil(context.Background(), tt.fn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*process.Info)
			s.Require().True(ok)
			s.Equal(tt.wantLen, info.Count)
			s.Len(info.Processes, tt.wantLen)
			if tt.wantLen == 2 {
				s.Equal("init", info.Processes[0].Name)
				s.Empty(info.Processes[1].Name) // permission-denied left empty
			}
		})
	}
}

func (s *ProcessLinuxPublicTestSuite) TestCollectDefault() {
	got, err := process.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*process.Info)
	s.True(ok)
}
