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
	ownPID := int32(os.Getpid())
	tests := []struct {
		name    string
		fn      func(context.Context) ([]*gpprocess.Process, error)
		wantErr bool
		wantLen int
	}{
		{
			name: "snapshot wraps gopsutil processes",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				p, _ := gpprocess.NewProcess(ownPID)
				return []*gpprocess.Process{p}, nil
			},
			wantLen: 1,
		},
		{
			name: "empty process list",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				return []*gpprocess.Process{}, nil
			},
			wantLen: 0,
		},
		{
			name:    "gopsutil error propagated",
			fn:      func(context.Context) ([]*gpprocess.Process, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer process.SetProcessesFn(tt.fn)()
			c := &process.Linux{}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*process.Info)
			s.Require().True(ok)
			s.Equal(tt.wantLen, info.Count)
			s.Len(info.Processes, tt.wantLen)
		})
	}
}
