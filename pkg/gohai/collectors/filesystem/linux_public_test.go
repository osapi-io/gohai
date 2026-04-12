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

type FilesystemLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFilesystemLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemLinuxPublicTestSuite))
}

func (s *FilesystemLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		mounts  []filesystem.Mount
		fnErr   error
		wantErr bool
		want    filesystem.Info
	}{
		{
			name: "root + /boot mounts",
			mounts: []filesystem.Mount{
				{
					Device:      "/dev/sda1",
					Mountpoint:  "/",
					Fstype:      "ext4",
					Total:       100,
					Used:        50,
					Free:        50,
					UsedPercent: 50,
				},
				{Device: "/dev/sda2", Mountpoint: "/boot", Fstype: "ext4"},
			},
			want: filesystem.Info{Mounts: []filesystem.Mount{
				{
					Device:      "/dev/sda1",
					Mountpoint:  "/",
					Fstype:      "ext4",
					Total:       100,
					Used:        50,
					Free:        50,
					UsedPercent: 50,
				},
				{Device: "/dev/sda2", Mountpoint: "/boot", Fstype: "ext4"},
			}},
		},
		{
			name:   "empty mount list",
			mounts: []filesystem.Mount{},
			want:   filesystem.Info{Mounts: []filesystem.Mount{}},
		},
		{
			name:    "gopsutil error propagated",
			fnErr:   errors.New("mounts error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &filesystem.Linux{
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
