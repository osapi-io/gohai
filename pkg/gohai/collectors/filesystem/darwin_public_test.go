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

package filesystem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
)

type FilesystemDarwinPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemDarwinPublicTestSuite))
}

func (s *FilesystemDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		mounts  []filesystem.Mount
		fnErr   error
		wantErr bool
		want    filesystem.Info
	}{
		{
			name: "APFS root + data",
			mounts: []filesystem.Mount{
				{
					Device:      "/dev/disk3s1",
					Mountpoint:  "/",
					Fstype:      "apfs",
					Total:       500,
					Used:        200,
					Free:        300,
					UsedPercent: 40,
				},
			},
			want: filesystem.Info{Mounts: []filesystem.Mount{
				{
					Device:      "/dev/disk3s1",
					Mountpoint:  "/",
					Fstype:      "apfs",
					Total:       500,
					Used:        200,
					Free:        300,
					UsedPercent: 40,
				},
			}},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fnErr:   errors.New("getfsstat failed"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &filesystem.Darwin{
				MountsFn: func(context.Context) ([]filesystem.Mount, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.mounts, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*filesystem.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
