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

package shard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
)

var (
	_ collector.Collector = (*shard.Linux)(nil)
	_ collector.Collector = (*shard.Darwin)(nil)
)

const (
	machinename = "somehost004"
	serial      = "234du3m4i498xdjr2"
	uuid        = "48555CF4-5BB1-21D9-BC4C-E8B73DDE5801"
)

func priorWithDMI() collector.PriorResults {
	return collector.PriorResults{
		"hostname": &hostname.Info{MachineName: machinename},
		"dmi": &dmi.Info{
			Product: &dmi.Product{SerialNumber: serial, UUID: uuid},
		},
	}
}

// Ohai test vector: MD5("somehost004" + "234du3m4i498xdjr2" +
// "48555CF4-5BB1-21D9-BC4C-E8B73DDE5801")[0:7] → 27767217
const ohaiDefaultSeed = 27767217

const sysProfJSON = `{"SPHardwareDataType": [{"serial_number": "234du3m4i498xdjr2"}]}`

type ShardPublicTestSuite struct {
	suite.Suite
}

func TestShardPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShardPublicTestSuite))
}

func (s *ShardPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := shard.New()
			s.Equal("shard", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Equal([]string{"hostname", "dmi"}, c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*shard.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*shard.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *ShardPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		prior    collector.PriorResults
		hostFn   func(context.Context) (*host.InfoStat, error)
		spOut    []byte
		spErr    error
		wantSeed int
	}{
		{
			name:     "linux: matches Ohai test vector (machinename + serial + uuid)",
			variant:  "linux",
			prior:    priorWithDMI(),
			wantSeed: ohaiDefaultSeed,
		},
		{
			name:    "linux: baseboard serial fallback when product serial empty",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
				"dmi": &dmi.Info{
					Product:   &dmi.Product{UUID: uuid},
					Baseboard: &dmi.Baseboard{SerialNumber: serial},
				},
			},
			wantSeed: ohaiDefaultSeed,
		},
		{
			name:    "linux: chassis serial fallback",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
				"dmi": &dmi.Info{
					Product: &dmi.Product{UUID: uuid},
					Chassis: &dmi.Chassis{SerialNumber: serial},
				},
			},
			wantSeed: ohaiDefaultSeed,
		},
		{
			name:    "linux: no dmi → seed from machinename only",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			wantSeed: -1,
		},
		{
			name:    "linux: no hostname prior → seed from serial+uuid only",
			variant: "linux",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{
					Product: &dmi.Product{SerialNumber: serial, UUID: uuid},
				},
			},
			wantSeed: -1,
		},
		{
			name:     "linux: empty prior → deterministic zero-input seed",
			variant:  "linux",
			prior:    collector.PriorResults{},
			wantSeed: -1,
		},
		{
			name:    "linux: nil dmi sub-structs → empty serial+uuid",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
				"dmi":      &dmi.Info{},
			},
			wantSeed: -1,
		},
		{
			name:    "darwin: matches Ohai test vector",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: uuid}, nil
			},
			spOut:    []byte(sysProfJSON),
			wantSeed: ohaiDefaultSeed,
		},
		{
			name:    "darwin: gopsutil error → empty uuid, seed still computes",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			hostFn:   func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			spOut:    []byte(sysProfJSON),
			wantSeed: -1,
		},
		{
			name:    "darwin: malformed system_profiler JSON → empty serial",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: uuid}, nil
			},
			spOut:    []byte("not json"),
			wantSeed: -1,
		},
		{
			name:    "darwin: empty items array → empty serial",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: uuid}, nil
			},
			spOut:    []byte(`{"SPHardwareDataType": []}`),
			wantSeed: -1,
		},
		{
			name:    "darwin: system_profiler error → empty serial",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: uuid}, nil
			},
			spErr:    errors.New("not found"),
			wantSeed: -1,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c shard.Collector
			switch tt.variant {
			case "linux":
				c = &shard.Linux{}
			case "darwin":
				defer shard.SetHostInfoFn(tt.hostFn)()
				ctrl := gomock.NewController(s.T())
				mockExec := execmocks.NewMockExecutor(ctrl)
				if tt.spErr != nil {
					mockExec.EXPECT().
						Execute(gomock.Any(), "system_profiler", "SPHardwareDataType", "-json").
						Return(nil, tt.spErr)
				} else {
					mockExec.EXPECT().
						Execute(gomock.Any(), "system_profiler", "SPHardwareDataType", "-json").
						Return(tt.spOut, nil)
				}
				c = &shard.Darwin{Exec: mockExec}
			}

			got, err := c.Collect(context.Background(), tt.prior)
			s.Require().NoError(err)
			info, ok := got.(*shard.Info)
			s.Require().True(ok)

			if tt.wantSeed >= 0 {
				s.Equal(tt.wantSeed, info.Seed)
			} else {
				s.GreaterOrEqual(info.Seed, 0)
			}
		})
	}
}
