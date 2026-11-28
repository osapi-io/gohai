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
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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

// The seed of Ohai's test input: SHA-256("somehost004" + "234du3m4i498xdjr2" +
// "48555CF4-5BB1-21D9-BC4C-E8B73DDE5801")[0:7] → 117055036
const defaultSeed = 117055036

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
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(shard.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c shard.Collector) {
				_, ok := c.(*shard.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c shard.Collector) {
				_, ok := c.(*shard.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c shard.Collector) {
				_, ok := c.(*shard.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c shard.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := shard.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = shard.New()
			}
			s.Equal("shard", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Equal([]string{"hostname", "dmi"}, c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

// seedOf is the seed of the machine name with serial and uuid, as the
// Linux variant computes it from dmi, so a macOS row can name the seed of
// the inputs it expects.
func seedOf(
	t *testing.T,
	serial string,
	uuid string,
) int {
	got, err := (&shard.Linux{}).Collect(
		context.Background(),
		collector.PriorResults{
			"hostname": &hostname.Info{MachineName: machinename},
			"dmi":      &dmi.Info{Product: &dmi.Product{SerialNumber: serial, UUID: uuid}},
		},
	)
	require.NoError(t, err)
	info, ok := got.(*shard.Info)
	require.True(t, ok)

	return info.Seed
}

func (s *ShardPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		prior        collector.PriorResults
		ioregOut     []byte
		ioregErr     error
		spOut        []byte
		spErr        error
		noExec       bool
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:    "darwin: no Exec seeds from the machine name alone",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				// The same seed as a host with an empty serial and UUID.
				want, wantErr := (&shard.Linux{}).Collect(
					context.Background(),
					collector.PriorResults{
						"hostname": &hostname.Info{MachineName: machinename},
						"dmi":      &dmi.Info{},
					},
				)
				s.Require().NoError(wantErr)
				s.Equal(want, got)
			},
		},
		{
			name:    "linux: gives the seed of Ohai's test input (machinename + serial + uuid)",
			variant: "linux",
			prior:   priorWithDMI(),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal([]string{"machinename", "serial", "uuid"}, info.Inputs)

				if defaultSeed >= 0 {
					s.Equal(defaultSeed, info.Seed)
				} else {
					s.GreaterOrEqual(info.Seed, 0)
				}
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)

				if defaultSeed >= 0 {
					s.Equal(defaultSeed, info.Seed)
				} else {
					s.GreaterOrEqual(info.Seed, 0)
				}
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)

				if defaultSeed >= 0 {
					s.Equal(defaultSeed, info.Seed)
				} else {
					s.GreaterOrEqual(info.Seed, 0)
				}
			},
		},
		{
			name:    "linux: no dmi → seed from machinename only",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal([]string{"machinename"}, info.Inputs)

				if -1 >= 0 {
					s.Equal(-1, info.Seed)
				} else {
					s.GreaterOrEqual(info.Seed, 0)
				}
			},
		},
		{
			name:    "linux: no hostname prior → ErrNoMachineName, no seed",
			variant: "linux",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{
					Product: &dmi.Product{SerialNumber: serial, UUID: uuid},
				},
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, shard.ErrNoMachineName)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: empty prior → ErrNoMachineName, no seed",
			variant: "linux",
			prior:   collector.PriorResults{},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, shard.ErrNoMachineName)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an empty machine name → ErrNoMachineName, no seed",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: "  "},
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, shard.ErrNoMachineName)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: no hostname prior → ErrNoMachineName, and no command runs",
			variant: "darwin",
			prior:   collector.PriorResults{},
			noExec:  true,
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, shard.ErrNoMachineName)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: nil dmi sub-structs → empty serial+uuid",
			variant: "linux",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
				"dmi":      &dmi.Info{},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)

				if -1 >= 0 {
					s.Equal(-1, info.Seed)
				} else {
					s.GreaterOrEqual(info.Seed, 0)
				}
			},
		},
		{
			name:    "darwin: gives the seed of Ohai's test input",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte(`"IOPlatformUUID" = "` + uuid + `"`),
			spOut:    []byte(sysProfJSON),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal(defaultSeed, info.Seed)
			},
		},
		{
			name:    "darwin: an ioreg that fails is an error and no seed",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregErr: errors.New("signal: killed"),
			spOut:    []byte(sysProfJSON),
			validateFunc: func(got any, err error) {
				s.EqualError(err, "ioreg: signal: killed")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: no ioreg on the host seeds without the uuid",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregErr: fmt.Errorf("ioreg: %w", osexec.ErrNotFound),
			spOut:    []byte(sysProfJSON),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal(seedOf(s.T(), serial, ""), info.Seed)
			},
		},
		{
			name:    "darwin: no UUID in ioreg's output seeds without the uuid",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte("{}"),
			spOut:    []byte(sysProfJSON),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal(seedOf(s.T(), serial, ""), info.Seed)
			},
		},
		{
			name:    "darwin: system_profiler JSON that does not parse is an error and no seed",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte(`"IOPlatformUUID" = "` + uuid + `"`),
			spOut:    []byte("not json"),
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "system_profiler: invalid character")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: an empty items array seeds without the serial",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte(`"IOPlatformUUID" = "` + uuid + `"`),
			spOut:    []byte(`{"SPHardwareDataType": []}`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal(seedOf(s.T(), "", uuid), info.Seed)
			},
		},
		{
			name:    "darwin: a system_profiler that times out is an error and no seed",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte(`"IOPlatformUUID" = "` + uuid + `"`),
			spErr:    context.DeadlineExceeded,
			validateFunc: func(got any, err error) {
				s.EqualError(err, "system_profiler: context deadline exceeded")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: no system_profiler on the host seeds without the serial",
			variant: "darwin",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{MachineName: machinename},
			},
			ioregOut: []byte(`"IOPlatformUUID" = "` + uuid + `"`),
			spErr:    fmt.Errorf("system_profiler: %w", osexec.ErrNotFound),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*shard.Info)
				s.Require().True(ok)
				s.Equal(seedOf(s.T(), "", uuid), info.Seed)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c shard.Collector
			switch tt.variant {
			case "linux":
				c = &shard.Linux{}
			case "darwin":
				ctrl := gomock.NewController(s.T())
				mockExec := mocks.NewMockRunner(ctrl)
				if tt.nilExec {
					c = &shard.Darwin{}
					break
				}
				if tt.noExec {
					c = &shard.Darwin{Exec: mockExec}
					break
				}
				mockExec.EXPECT().
					Execute(gomock.Any(), "ioreg", "-rd1", "-c", "IOPlatformExpertDevice").
					Return(tt.ioregOut, tt.ioregErr)
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

			tt.validateFunc(c.Collect(context.Background(), tt.prior))
		})
	}
}
