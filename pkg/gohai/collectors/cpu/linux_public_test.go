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

package cpu_test

import (
	"context"
	"errors"
	"testing"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

type CPULinuxPublicTestSuite struct {
	suite.Suite
}

func TestCPULinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPULinuxPublicTestSuite))
}

func (s *CPULinuxPublicTestSuite) TestCollectFromGopsutil() {
	okInfo := func(_ context.Context) ([]gcpu.InfoStat, error) {
		return []gcpu.InfoStat{{
			ModelName: "Intel Core i7",
			VendorID:  "GenuineIntel",
			Family:    "6",
			Model:     "158",
			Stepping:  10,
			Mhz:       3600.0,
			CacheSize: 8192,
			Flags:     []string{"sse", "sse2"},
		}}, nil
	}
	okCounts := func(logical bool) func(context.Context, bool) (int, error) {
		return func(_ context.Context, l bool) (int, error) {
			if l {
				return 8, nil
			}
			return 4, nil
		}
		_ = logical
	}

	tests := []struct {
		name     string
		infoFn   func(context.Context) ([]gcpu.InfoStat, error)
		countsFn func(context.Context, bool) (int, error)
		wantErr  bool
		want     cpu.Info
	}{
		{
			name:     "success with one info",
			infoFn:   okInfo,
			countsFn: okCounts(true),
			want: cpu.Info{
				Total:     8,
				Cores:     4,
				ModelName: "Intel Core i7",
				VendorID:  "GenuineIntel",
				Family:    "6",
				Model:     "158",
				Stepping:  10,
				Mhz:       3600.0,
				CacheSize: 8192,
				Flags:     []string{"sse", "sse2"},
			},
		},
		{
			name: "empty info list",
			infoFn: func(_ context.Context) ([]gcpu.InfoStat, error) {
				return nil, nil
			},
			countsFn: okCounts(true),
			want:     cpu.Info{Total: 8, Cores: 4},
		},
		{
			name:     "info error",
			infoFn:   func(_ context.Context) ([]gcpu.InfoStat, error) { return nil, errors.New("boom") },
			countsFn: okCounts(true),
			wantErr:  true,
		},
		{
			name:   "logical counts error",
			infoFn: okInfo,
			countsFn: func(_ context.Context, logical bool) (int, error) {
				if logical {
					return 0, errors.New("lg")
				}
				return 4, nil
			},
			wantErr: true,
		},
		{
			name:   "physical counts error",
			infoFn: okInfo,
			countsFn: func(_ context.Context, logical bool) (int, error) {
				if logical {
					return 8, nil
				}
				return 0, errors.New("ph")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := cpu.CollectFromGopsutil(context.Background(), tt.infoFn, tt.countsFn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*cpu.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *CPULinuxPublicTestSuite) TestCollectDefault() {
	got, err := cpu.Collect(context.Background())
	s.Require().NoError(err)
	info, ok := got.(*cpu.Info)
	s.Require().True(ok)
	s.NotZero(info.Total)
}
