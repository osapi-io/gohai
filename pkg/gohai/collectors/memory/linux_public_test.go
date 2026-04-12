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

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
)

type MemoryLinuxPublicTestSuite struct {
	suite.Suite
}

func TestMemoryLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryLinuxPublicTestSuite))
}

func (s *MemoryLinuxPublicTestSuite) TestCollectFromGopsutil() {
	okVM := func(_ context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{
			Total:       16 * 1024 * 1024 * 1024,
			Available:   8 * 1024 * 1024 * 1024,
			Used:        8 * 1024 * 1024 * 1024,
			UsedPercent: 50.0,
			Free:        4 * 1024 * 1024 * 1024,
			Buffers:     1024,
			Cached:      2048,
		}, nil
	}
	okSwap := func(_ context.Context) (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{
			Total:       2 * 1024 * 1024 * 1024,
			Used:        1024,
			Free:        2*1024*1024*1024 - 1024,
			UsedPercent: 0.1,
		}, nil
	}
	zeroSwap := func(_ context.Context) (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{}, nil
	}
	nilSwap := func(_ context.Context) (*mem.SwapMemoryStat, error) {
		return nil, nil
	}

	tests := []struct {
		name     string
		vmFn     func(context.Context) (*mem.VirtualMemoryStat, error)
		swapFn   func(context.Context) (*mem.SwapMemoryStat, error)
		wantErr  bool
		wantSwap bool
	}{
		{"happy path with swap", okVM, okSwap, false, true},
		{"happy path zero swap", okVM, zeroSwap, false, false},
		{"happy path nil swap", okVM, nilSwap, false, false},
		{
			"vm error",
			func(_ context.Context) (*mem.VirtualMemoryStat, error) { return nil, errors.New("boom") },
			okSwap,
			true,
			false,
		},
		{
			"swap error",
			okVM,
			func(_ context.Context) (*mem.SwapMemoryStat, error) { return nil, errors.New("boom") },
			true,
			false,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := memory.CollectFromGopsutil(context.Background(), tt.vmFn, tt.swapFn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			s.NotZero(info.Total)
			if tt.wantSwap {
				s.NotNil(info.Swap)
			} else {
				s.Nil(info.Swap)
			}
		})
	}
}

func (s *MemoryLinuxPublicTestSuite) TestCollectDefault() {
	got, err := memory.Collect(context.Background())
	s.Require().NoError(err)
	info, ok := got.(*memory.Info)
	s.Require().True(ok)
	s.NotZero(info.Total)
}
