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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

type CPUDarwinPublicTestSuite struct {
	suite.Suite
}

func TestCPUDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPUDarwinPublicTestSuite))
}

func (s *CPUDarwinPublicTestSuite) TestCollect() {
	baseInfo := func() *cpu.Info {
		// What gopsutil's ReadFn returns on Darwin today:
		// Cores == Total (hyperthreaded Intel — wrong), Mhz may be 0.
		return &cpu.Info{
			Count: 12, Sockets: 1, Cores: 12,
			ModelName: "Apple M2 Pro",
			Mhz:       0,
		}
	}

	// setupMock returns an Executor that expects specific sysctl calls.
	// Each entry in `returns` is keyed by the sysctl arg (e.g.
	// "hw.physicalcpu"); value is the stdout string or empty for error.
	type sysctlRet struct {
		out string
		err error
	}

	tests := []struct {
		name     string
		readFn   func(context.Context) (*cpu.Info, error)
		sysctls  map[string]sysctlRet
		wantErr  bool
		validate func(*cpu.Info)
	}{
		{
			name:   "Intel Mac with hyperthreading: all sysctls override",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "6\n"},
				"hw.packages":         {out: "1\n"},
				"hw.cpufrequency_max": {out: "2600000000\n"},
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Count)   // unchanged from gopsutil
				s.Equal(6, i.Cores)    // overridden
				s.Equal(1, i.Sockets)  // set
				s.Equal(2600.0, i.Mhz) // 2600 MHz
			},
		},
		{
			name:   "Apple Silicon: cpufrequency_max absent, cpufrequency also absent, Mhz untouched",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "10\n"},
				"hw.packages":         {out: "1\n"},
				"hw.cpufrequency_max": {err: errors.New("unknown oid")},
				"hw.cpufrequency":     {err: errors.New("unknown oid")},
			},
			validate: func(i *cpu.Info) {
				s.Equal(10, i.Cores)
				s.Equal(1, i.Sockets)
				s.Equal(0.0, i.Mhz) // stayed at gopsutil's 0
			},
		},
		{
			name:   "Apple Silicon fallback: cpufrequency_max absent but cpufrequency present",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "8\n"},
				"hw.packages":         {out: "1\n"},
				"hw.cpufrequency_max": {err: errors.New("unknown oid")},
				"hw.cpufrequency":     {out: "3200000000\n"},
			},
			validate: func(i *cpu.Info) {
				s.Equal(3200.0, i.Mhz)
			},
		},
		{
			name:   "hw.physicalcpu fails: Cores unchanged",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {err: errors.New("unknown oid")},
				"hw.packages":         {out: "1\n"},
				"hw.cpufrequency_max": {err: errors.New("unknown oid")},
				"hw.cpufrequency":     {err: errors.New("unknown oid")},
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Cores) // unchanged from base
			},
		},
		{
			name:   "hw.packages fails: Sockets unchanged",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "6\n"},
				"hw.packages":         {err: errors.New("unknown oid")},
				"hw.cpufrequency_max": {err: errors.New("unknown oid")},
				"hw.cpufrequency":     {err: errors.New("unknown oid")},
			},
			validate: func(i *cpu.Info) {
				s.Equal(1, i.Sockets) // unchanged from base
			},
		},
		{
			name:   "hw.physicalcpu non-numeric: Cores unchanged",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "xyz\n"},
				"hw.packages":         {err: errors.New("unknown oid")},
				"hw.cpufrequency_max": {err: errors.New("unknown oid")},
				"hw.cpufrequency":     {err: errors.New("unknown oid")},
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Cores)
			},
		},
		{
			name:   "hw.cpufrequency_max non-numeric falls through to hw.cpufrequency",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			sysctls: map[string]sysctlRet{
				"hw.physicalcpu":      {out: "6\n"},
				"hw.packages":         {out: "1\n"},
				"hw.cpufrequency_max": {out: "not-a-number\n"},
				"hw.cpufrequency":     {out: "2800000000\n"},
			},
			validate: func(i *cpu.Info) {
				s.Equal(2800.0, i.Mhz)
			},
		},
		{
			name:    "ReadFn error propagated",
			readFn:  func(context.Context) (*cpu.Info, error) { return nil, errors.New("sysctl error") },
			sysctls: map[string]sysctlRet{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockExec := execmocks.NewMockExecutor(ctrl)
			for key, ret := range tt.sysctls {
				call := mockExec.EXPECT().
					Execute(gomock.Any(), "sysctl", "-n", key)
				if ret.err != nil {
					call.Return(nil, ret.err).AnyTimes()
				} else {
					call.Return([]byte(ret.out), nil).AnyTimes()
				}
			}

			c := &cpu.Darwin{ReadFn: tt.readFn, Exec: mockExec}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*cpu.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
