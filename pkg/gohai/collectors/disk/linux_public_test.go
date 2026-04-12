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

package disk_test

import (
	"context"
	"errors"
	"testing"

	gdisk "github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
)

type DiskLinuxPublicTestSuite struct {
	suite.Suite
}

func TestDiskLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DiskLinuxPublicTestSuite))
}

func (s *DiskLinuxPublicTestSuite) TestCollectFromGopsutil() {
	two := func(_ context.Context, _ ...string) (map[string]gdisk.IOCountersStat, error) {
		return map[string]gdisk.IOCountersStat{
			"sda": {Name: "sda", ReadCount: 100, WriteCount: 50, ReadBytes: 4096, WriteBytes: 8192},
			"sdb": {Name: "sdb", ReadCount: 10, WriteCount: 5},
		}, nil
	}
	empty := func(_ context.Context, _ ...string) (map[string]gdisk.IOCountersStat, error) {
		return map[string]gdisk.IOCountersStat{}, nil
	}
	errFn := func(_ context.Context, _ ...string) (map[string]gdisk.IOCountersStat, error) {
		return nil, errors.New("boom")
	}

	tests := []struct {
		name    string
		fn      func(context.Context, ...string) (map[string]gdisk.IOCountersStat, error)
		wantErr bool
		wantLen int
	}{
		{"two devices sorted", two, false, 2},
		{"empty", empty, false, 0},
		{"error", errFn, true, 0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := disk.CollectFromGopsutil(context.Background(), tt.fn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*disk.Info)
			s.Require().True(ok)
			s.Len(info.Devices, tt.wantLen)
			if tt.wantLen == 2 {
				s.Equal("sda", info.Devices[0].Name) // sorted
				s.Equal("sdb", info.Devices[1].Name)
			}
		})
	}
}

func (s *DiskLinuxPublicTestSuite) TestCollectDefault() {
	got, err := disk.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*disk.Info)
	s.True(ok)
}
