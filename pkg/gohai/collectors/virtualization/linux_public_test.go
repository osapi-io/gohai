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

package virtualization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

type VirtLinuxPublicTestSuite struct {
	suite.Suite
}

func TestVirtLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(VirtLinuxPublicTestSuite))
}

func (s *VirtLinuxPublicTestSuite) TestCollectWithHost() {
	tests := []struct {
		name    string
		stub    func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    virtualization.Info
	}{
		{
			name: "docker guest",
			stub: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					VirtualizationSystem: "docker",
					VirtualizationRole:   "guest",
				}, nil
			},
			want: virtualization.Info{System: "docker", Role: "guest"},
		},
		{
			name: "bare metal",
			stub: func(_ context.Context) (*host.InfoStat, error) { return &host.InfoStat{}, nil },
			want: virtualization.Info{},
		},
		{
			name:    "host.Info error",
			stub:    func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := virtualization.CollectWithHost(context.Background(), tt.stub)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*virtualization.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *VirtLinuxPublicTestSuite) TestCollectDefault() {
	got, err := virtualization.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*virtualization.Info)
	s.True(ok)
}
