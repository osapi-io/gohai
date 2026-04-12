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

func (s *MemoryLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		info    *memory.Info
		fnErr   error
		wantErr bool
		want    memory.Info
	}{
		{
			name: "fully populated with swap",
			info: &memory.Info{
				Total: 17179869184, Available: 8589934592, Used: 8589934592,
				UsedPercent: 50.0, Free: 4294967296, Buffers: 1048576, Cached: 4194304,
				Swap: &memory.Swap{
					Total:       2147483648,
					Used:        1024,
					Free:        2147482624,
					UsedPercent: 0.0001,
				},
			},
			want: memory.Info{
				Total: 17179869184, Available: 8589934592, Used: 8589934592,
				UsedPercent: 50.0, Free: 4294967296, Buffers: 1048576, Cached: 4194304,
				Swap: &memory.Swap{
					Total:       2147483648,
					Used:        1024,
					Free:        2147482624,
					UsedPercent: 0.0001,
				},
			},
		},
		{
			name: "no swap",
			info: &memory.Info{
				Total:       8589934592,
				Used:        4294967296,
				Free:        4294967296,
				UsedPercent: 50.0,
			},
			want: memory.Info{
				Total:       8589934592,
				Used:        4294967296,
				Free:        4294967296,
				UsedPercent: 50.0,
			},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fnErr:   errors.New("mem error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &memory.Linux{
				ReadFn: func(context.Context) (*memory.Info, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.info, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *MemoryLinuxPublicTestSuite) TestReadMemory() {
	tests := []struct {
		name      string
		vmFn      func(context.Context) (*mem.VirtualMemoryStat, error)
		swapFn    func(context.Context) (*mem.SwapMemoryStat, error)
		wantErr   bool
		wantTotal uint64
		wantSwap  bool
	}{
		{
			name: "virtual + swap populated",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100, Used: 50, Free: 50}, nil
			},
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return &mem.SwapMemoryStat{Total: 10, Used: 1, Free: 9}, nil
			},
			wantTotal: 100,
			wantSwap:  true,
		},
		{
			name: "swap error omits swap field",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100}, nil
			},
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return nil, errors.New("swap failed")
			},
			wantTotal: 100,
			wantSwap:  false,
		},
		{
			name: "zero swap omits swap field",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100}, nil
			},
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return &mem.SwapMemoryStat{Total: 0}, nil
			},
			wantTotal: 100,
			wantSwap:  false,
		},
		{
			name: "virtual memory error propagated",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return nil, errors.New("vm failed")
			},
			swapFn:  func(context.Context) (*mem.SwapMemoryStat, error) { return nil, nil },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreVM := memory.SetVirtualMemoryFn(tt.vmFn)
			defer restoreVM()
			restoreSwap := memory.SetSwapMemoryFn(tt.swapFn)
			defer restoreSwap()
			got, err := memory.ReadMemory(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantTotal, got.Total)
			if tt.wantSwap {
				s.NotNil(got.Swap)
			} else {
				s.Nil(got.Swap)
			}
		})
	}
}
