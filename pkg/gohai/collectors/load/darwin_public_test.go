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

package load_test

import (
	"context"
	"errors"
	"testing"

	gpload "github.com/shirou/gopsutil/v4/load"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
)

type LoadDarwinPublicTestSuite struct {
	suite.Suite
}

func TestLoadDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadDarwinPublicTestSuite))
}

func (s *LoadDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		avgFn   func(context.Context) (*gpload.AvgStat, error)
		wantErr bool
		want    load.Info
	}{
		{
			name: "averages returned",
			avgFn: func(_ context.Context) (*gpload.AvgStat, error) {
				return &gpload.AvgStat{Load1: 1.5, Load5: 2.0, Load15: 2.5}, nil
			},
			want: load.Info{One: 1.5, Five: 2.0, Fifteen: 2.5},
		},
		{
			name: "error propagated",
			avgFn: func(_ context.Context) (*gpload.AvgStat, error) {
				return nil, errors.New("boom")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &load.Darwin{AvgFn: tt.avgFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*load.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
