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

package disk_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
)

type DiskDarwinPublicTestSuite struct {
	suite.Suite
}

func TestDiskDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DiskDarwinPublicTestSuite))
}

func (s *DiskDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		devs    []disk.Device
		fnErr   error
		wantErr bool
		want    disk.Info
	}{
		{
			name: "disk0 snapshot",
			devs: []disk.Device{
				{
					Name:       "disk0",
					ReadCount:  200,
					WriteCount: 100,
					ReadBytes:  204800,
					WriteBytes: 102400,
				},
			},
			want: disk.Info{Devices: []disk.Device{
				{
					Name:       "disk0",
					ReadCount:  200,
					WriteCount: 100,
					ReadBytes:  204800,
					WriteBytes: 102400,
				},
			}},
		},
		{
			name:    "iokit error propagated",
			fnErr:   errors.New("iokit unavailable"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &disk.Darwin{
				DevicesFn: func(context.Context) ([]disk.Device, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.devs, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*disk.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
