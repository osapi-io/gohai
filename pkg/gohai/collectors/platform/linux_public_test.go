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

package platform_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type PlatformLinuxPublicTestSuite struct {
	suite.Suite
}

func TestPlatformLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformLinuxPublicTestSuite))
}

func (s *PlatformLinuxPublicTestSuite) TestCollectWithHost() {
	tests := []struct {
		name    string
		stub    func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    platform.Info
	}{
		{
			name: "ubuntu 24.04",
			stub: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "ubuntu",
					PlatformVersion: "24.04",
					PlatformFamily:  "debian",
					KernelArch:      "x86_64",
				}, nil
			},
			want: platform.Info{
				OS:           "linux",
				Name:         "ubuntu",
				Version:      "24.04",
				Family:       "debian",
				Architecture: "x86_64",
			},
		},
		{
			name: "rhel 9",
			stub: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "rhel",
					PlatformVersion: "9.3",
					PlatformFamily:  "rhel",
					KernelArch:      "aarch64",
				}, nil
			},
			want: platform.Info{
				OS:           "linux",
				Name:         "rhel",
				Version:      "9.3",
				Family:       "rhel",
				Architecture: "aarch64",
			},
		},
		{
			name: "host.Info error",
			stub: func(_ context.Context) (*host.InfoStat, error) {
				return nil, errors.New("boom")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := platform.CollectWithHost(context.Background(), tt.stub)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*platform.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *PlatformLinuxPublicTestSuite) TestCollectReal() {
	got, err := platform.Collect(context.Background())
	s.Require().NoError(err)
	info, ok := got.(*platform.Info)
	s.Require().True(ok)
	s.NotEmpty(info.Architecture)
}
