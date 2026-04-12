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

package uptime_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
)

type UptimeDarwinPublicTestSuite struct {
	suite.Suite
}

func TestUptimeDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UptimeDarwinPublicTestSuite))
}

func (s *UptimeDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		baseFn  func(context.Context) (*uptime.Info, error)
		wantErr bool
		want    uptime.Info
	}{
		{
			name: "uptime returned",
			baseFn: func(context.Context) (*uptime.Info, error) {
				return &uptime.Info{Seconds: 7200, BootTime: 1_700_000_000, Human: "2h 0m 0s"}, nil
			},
			want: uptime.Info{Seconds: 7200, BootTime: 1_700_000_000, Human: "2h 0m 0s"},
		},
		{
			name:    "BaseFn error propagated",
			baseFn:  func(context.Context) (*uptime.Info, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &uptime.Darwin{BaseFn: tt.baseFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*uptime.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
