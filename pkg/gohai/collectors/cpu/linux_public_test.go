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

	gpcpu "github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

type CPULinuxPublicTestSuite struct {
	suite.Suite
}

func TestCPULinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPULinuxPublicTestSuite))
}

func (s *CPULinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		info    *cpu.Info
		fnErr   error
		wantErr bool
		want    cpu.Info
	}{
		{
			name: "8-core Xeon",
			info: &cpu.Info{
				Total: 16, Real: 1, Cores: 8, ModelName: "Intel(R) Xeon(R) CPU",
				VendorID: "GenuineIntel", Family: "6", Model: "85",
				Stepping: 7, Mhz: 2400, CacheSize: 25600,
				Flags: []string{"fpu", "vme"},
			},
			want: cpu.Info{
				Total: 16, Real: 1, Cores: 8, ModelName: "Intel(R) Xeon(R) CPU",
				VendorID: "GenuineIntel", Family: "6", Model: "85",
				Stepping: 7, Mhz: 2400, CacheSize: 25600,
				Flags: []string{"fpu", "vme"},
			},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fnErr:   errors.New("cpuinfo error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &cpu.Linux{
				ReadFn: func(context.Context) (*cpu.Info, error) {
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
			info, ok := got.(*cpu.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *CPULinuxPublicTestSuite) TestReadCPU() {
	okStats := []gpcpu.InfoStat{{
		PhysicalID: "0", Cores: 8, ModelName: "Intel(R) Xeon(R) CPU",
		VendorID: "GenuineIntel", Family: "6", Model: "85",
		Stepping: 7, Mhz: 2400, CacheSize: 25600,
		Flags: []string{"fpu", "vme"},
	}}

	tests := []struct {
		name      string
		infoFn    func(context.Context) ([]gpcpu.InfoStat, error)
		countsFn  func(context.Context, bool) (int, error)
		wantErr   bool
		wantTotal int
		wantCores int
	}{
		{
			name:      "success maps stats",
			infoFn:    func(context.Context) ([]gpcpu.InfoStat, error) { return okStats, nil },
			countsFn:  func(context.Context, bool) (int, error) { return 16, nil },
			wantTotal: 16,
			wantCores: 8,
		},
		{
			name:      "counts error ignored, info still populated",
			infoFn:    func(context.Context) ([]gpcpu.InfoStat, error) { return okStats, nil },
			countsFn:  func(context.Context, bool) (int, error) { return 0, errors.New("counts failed") },
			wantTotal: 0,
			wantCores: 8,
		},
		{
			name:     "info error propagated",
			infoFn:   func(context.Context) ([]gpcpu.InfoStat, error) { return nil, errors.New("info failed") },
			countsFn: func(context.Context, bool) (int, error) { return 0, nil },
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreInfo := cpu.SetInfoFn(tt.infoFn)
			defer restoreInfo()
			restoreCounts := cpu.SetCountsFn(tt.countsFn)
			defer restoreCounts()
			got, err := cpu.ReadCPU(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantTotal, got.Total)
			s.Equal(tt.wantCores, got.Cores)
		})
	}
}
